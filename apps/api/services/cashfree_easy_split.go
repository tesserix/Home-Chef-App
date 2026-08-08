package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
)

// cashfree_easy_split.go — Easy Split vendors on the PG client.
//
// Easy Split is Cashfree's marketplace settlement: an order created with
// order_splits pays each vendor's share straight from capture, the remainder
// stays with the platform merchant account, and Cashfree settles both sides to
// their own bank accounts. The platform never holds vendor money — which is
// the regulatory point of using it.
//
// A vendor here is a CHEF registered with the split rail. Distinct from the
// Payouts-rail beneficiary (cashfree_payouts.go): that rail moves money the
// platform already holds; this one prevents the platform holding it at all.

// Cashfree vendor status values.
const (
	CashfreeVendorActive         = "ACTIVE"
	CashfreeVendorInBeneCreation = "IN_BENE_CREATION"
	CashfreeVendorBlocked        = "BLOCKED"
	CashfreeVendorDeleted        = "DELETED"
	// CashfreeVendorInBankValidation is undocumented but real: the sandbox
	// returns it while penny-dropping the account (#1082).
	CashfreeVendorInBankValidation = "IN_BANK_VALIDATION"
	// CashfreeVendorBankValidationFailed is a refusal, not a stage: the penny
	// drop bounced and nothing about it resolves on its own (#1083).
	CashfreeVendorBankValidationFailed = "BANK_VALIDATION_FAILED"
)

type CashfreeVendorBank struct {
	AccountNumber string `json:"account_number"`
	AccountHolder string `json:"account_holder"`
	IFSC          string `json:"ifsc"`
}

type CashfreeVendorUPI struct {
	VPA           string `json:"vpa"`
	AccountHolder string `json:"account_holder"`
}

type CashfreeVendorKYC struct {
	AccountType  string `json:"account_type,omitempty"`
	BusinessType string `json:"business_type,omitempty"`
	PAN          string `json:"pan,omitempty"`
	GST          string `json:"gst,omitempty"`
}

type CashfreeVendorRequest struct {
	VendorID        string              `json:"vendor_id"`
	Status          string              `json:"status,omitempty"`
	Name            string              `json:"name"`
	Email           string              `json:"email"`
	Phone           string              `json:"phone"`
	VerifyAccount   bool                `json:"verify_account"`
	DashboardAccess bool                `json:"dashboard_access"`
	ScheduleOption  int                 `json:"schedule_option,omitempty"`
	Bank            *CashfreeVendorBank `json:"bank,omitempty"`
	UPI             *CashfreeVendorUPI  `json:"upi,omitempty"`
	KYC             CashfreeVendorKYC   `json:"kyc_details"`
}

type CashfreeVendorResponse struct {
	VendorID string              `json:"vendor_id"`
	Status   string              `json:"status"`
	Name     string              `json:"name"`
	Email    string              `json:"email"`
	Bank     *CashfreeVendorBank `json:"bank,omitempty"`
	UPI      *CashfreeVendorUPI  `json:"upi,omitempty"`
}

// SplitPayable reports whether an order split naming this vendor will be
// accepted and settled. Only ACTIVE counts — every other state, documented or
// not, means Cashfree has not confirmed the destination.
func (v *CashfreeVendorResponse) SplitPayable() bool {
	return v != nil && strings.EqualFold(v.Status, CashfreeVendorActive)
}

// CreateVendor registers (or updates) an Easy Split vendor.
//
// An existing vendor_id is re-applied with PATCH so changed bank details reach
// Cashfree instead of being silently ignored — Easy Split vendors are mutable,
// unlike Payouts beneficiaries. The sandbox reports the duplicate as a plain
// 400 "vendor already exists" rather than a 409, so both are treated as
// "exists" (verified against the live sandbox; see the cfsandbox test).
func (c *CashfreeClient) CreateVendor(req *CashfreeVendorRequest) (*CashfreeVendorResponse, error) {
	if req.Status == "" {
		req.Status = CashfreeVendorActive
	}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("cashfree: marshal vendor request: %w", err)
	}

	resp, status, err := c.do("POST", "/easy-split/vendors", body, nil)
	if err != nil {
		return nil, err
	}
	exists := status == http.StatusConflict ||
		(status == http.StatusBadRequest && strings.Contains(strings.ToLower(string(resp)), "already exists"))
	if exists {
		log.Printf("cashfree[%s]: vendor %s exists — updating in place", c.mode, req.VendorID)
		// Status transitions are gated while Cashfree validates the account, so
		// the update carries details only, never a state change.
		patch := *req
		patch.Status = ""
		pbody, mErr := json.Marshal(&patch)
		if mErr != nil {
			return nil, fmt.Errorf("cashfree: marshal vendor update: %w", mErr)
		}
		resp, status, err = c.do("PATCH", "/easy-split/vendors/"+req.VendorID, pbody, nil)
		if err != nil {
			return nil, err
		}
		if status >= 400 {
			// Mid-validation the sandbox refuses updates outright. The vendor is
			// registered — report its live state instead of a phantom failure.
			log.Printf("cashfree[%s]: vendor %s update refused, returning current state: %v",
				c.mode, req.VendorID, cashfreeError(status, resp))
			return c.FetchVendor(req.VendorID)
		}
	}
	if status >= 400 {
		return nil, cashfreeError(status, resp)
	}

	var result CashfreeVendorResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("cashfree: parse vendor response: %w", err)
	}
	return &result, nil
}

// ErrEasySplitRetryable — Cashfree has not finished syncing the payment yet.
// The same call a couple of minutes later succeeds, so the caller must retry
// rather than give up on the split rail.
var ErrEasySplitRetryable = errors.New("cashfree: order not ready to split yet")

type cashfreeSplitRequest struct {
	Split []CashfreeVendorSplit `json:"split"`
	// DisableSplit closes the split window for this order. We split once, for
	// one vendor, so leaving it open would hold the vendor's balance
	// provisional until the delay lapses for no gain.
	DisableSplit bool `json:"disable_split"`
}

// SplitOrderAfterPayment allocates a vendor's share of an order that has
// already been paid — the ADR-0003 rail, replacing order_splits at capture so
// the release governor decides before the money moves.
//
// idempotencyKey is what makes a retry after a timeout safe; Cashfree dedupes
// on it. An already-applied split comes back as "transaction already
// processed" (409, and 400 in the sandbox) and is reported as success: the chef
// has been paid, and calling it a failure would send the order down the payout
// rail and pay them twice.
func (c *CashfreeClient) SplitOrderAfterPayment(orderID string, splits []CashfreeVendorSplit, idempotencyKey string) error {
	if orderID == "" || len(splits) == 0 {
		return fmt.Errorf("cashfree: split for order %q needs at least one vendor share", orderID)
	}
	body, err := json.Marshal(cashfreeSplitRequest{Split: splits, DisableSplit: true})
	if err != nil {
		return fmt.Errorf("cashfree: marshal split request: %w", err)
	}

	resp, status, err := c.do("POST", "/easy-split/orders/"+orderID+"/split", body,
		map[string]string{"x-idempotency-key": idempotencyKey})
	if err != nil {
		return err
	}
	if status < 400 {
		return nil
	}

	message := strings.ToLower(string(resp))
	switch {
	case strings.Contains(message, "already processed"):
		log.Printf("cashfree[%s]: order %s was already split — treating as done", c.mode, orderID)
		return nil
	case strings.Contains(message, "not synced"):
		return ErrEasySplitRetryable
	}
	return cashfreeError(status, resp)
}

// FetchVendor reads a vendor's current state — the answer to "has Cashfree
// finished verifying this chef's destination yet?".
func (c *CashfreeClient) FetchVendor(vendorID string) (*CashfreeVendorResponse, error) {
	resp, status, err := c.do("GET", "/easy-split/vendors/"+vendorID, nil, nil)
	if err != nil {
		return nil, err
	}
	if status >= 400 {
		return nil, cashfreeError(status, resp)
	}
	var result CashfreeVendorResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("cashfree: parse vendor response: %w", err)
	}
	return &result, nil
}
