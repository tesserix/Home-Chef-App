//go:build cfsandbox

package services

// Live Easy Split sandbox round-trip. Excluded from CI by the cfsandbox tag —
// it talks to Cashfree's real sandbox:
//   CF_PG_ID=… CF_PG_SECRET=… go test ./services/ -tags cfsandbox -run TestCashfreeEasySplitSandbox -v
//
// Re-runnable: the vendor id is deterministic, so a second run updates the same
// vendor instead of minting a new one.

import (
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/homechef/api/models"
)

var easySplitTestChefID = uuid.MustParse("e2e00000-0000-4000-8000-00000000beef")

func easySplitSandboxClient(t *testing.T) *CashfreeClient {
	t.Helper()
	id, secret := os.Getenv("CF_PG_ID"), os.Getenv("CF_PG_SECRET")
	if id == "" || secret == "" {
		t.Skip("CF_PG_ID / CF_PG_SECRET not set")
	}
	return NewCashfreeTestClient("", id, secret, "", models.ChefModeTest)
}

func TestCashfreeEasySplitSandboxRoundTrip(t *testing.T) {
	c := easySplitSandboxClient(t)
	vendorID := EasySplitVendorIDFor(easySplitTestChefID)

	vendor, err := c.CreateVendor(&CashfreeVendorRequest{
		VendorID:      vendorID,
		Status:        CashfreeVendorActive,
		Name:          "E2E Split Kitchen",
		Email:         "e2e-split@fe3dr.com",
		Phone:         "9999999901",
		VerifyAccount: true,
		Bank: &CashfreeVendorBank{
			AccountNumber: "00011020001772",
			AccountHolder: "E2E Split Kitchen",
			IFSC:          "HDFC0000001",
		},
		KYC: CashfreeVendorKYC{
			AccountType:  "savings",
			BusinessType: "Food and Beverages",
			PAN:          "ABCDE1234F",
		},
	})
	if err != nil {
		t.Fatalf("create vendor: %v", err)
	}
	t.Logf("vendor %s status=%s", vendor.VendorID, vendor.Status)
	if vendor.VendorID != vendorID {
		t.Fatalf("vendor id mismatch: got %s want %s", vendor.VendorID, vendorID)
	}

	fetched, err := c.FetchVendor(vendorID)
	if err != nil {
		t.Fatalf("fetch vendor: %v", err)
	}
	t.Logf("fetched status=%s payable=%v", fetched.Status, fetched.SplitPayable())

	// An order carrying a split for the vendor. 585.00 with a 505.55 vendor
	// share — the remainder stays with the merchant account.
	orderID := "e2e-split-" + uuid.NewString()[:8]
	order, err := c.CreateOrder(&CashfreeOrderRequest{
		OrderID:     orderID,
		AmountPaise: CashfreeAmountFromPaise(58500),
		Currency:    "INR",
		Customer: CashfreeCustomerDetails{
			CustomerID:    strings.ReplaceAll(uuid.NewString(), "-", ""),
			CustomerPhone: "9999999902",
			CustomerName:  "E2E Split Customer",
			CustomerEmail: "e2e-split-customer@fe3dr.com",
		},
		Splits: []CashfreeOrderSplit{
			{VendorID: vendorID, AmountPaise: CashfreeAmountFromPaise(50555)},
		},
		OrderNote: "easy-split sandbox validation",
	})
	if err != nil {
		t.Fatalf("create split order: %v", err)
	}
	if order.PaymentSessionID == "" {
		t.Fatalf("split order %s has no payment session (status=%s)", order.OrderID, order.OrderStatus)
	}
	t.Logf("split order %s status=%s session=%s…", order.OrderID, order.OrderStatus, order.PaymentSessionID[:20])
}
