package services

import (
	"encoding/json"
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
// accepted and settled. Only ACTIVE counts — IN_BENE_CREATION means Cashfree
// is still verifying the destination.
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
			log.Printf("cashfree[%s]: vendor %s update refused (%d) — returning current state", c.mode, req.VendorID, status)
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
