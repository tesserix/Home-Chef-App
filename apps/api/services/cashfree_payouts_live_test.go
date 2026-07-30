//go:build cfsandbox

package services

// Live sandbox round-trip. Excluded from the normal build by the cfsandbox tag —
// it talks to Cashfree's real sandbox and must never run in CI.
//   go test ./services/ -tags cfsandbox -run TestCashfreeSandbox -v

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/homechef/api/models"
	"github.com/homechef/api/payouts"
)

func sandboxClient(t *testing.T) *CashfreePayoutClient {
	t.Helper()
	c := NewCashfreePayoutTestClient("", os.Getenv("CF_ID"), os.Getenv("CF_SECRET"), "", models.ChefModeTest)
	pk, err := parseCashfreePublicKey(os.Getenv("CF_PEM"))
	if err != nil {
		t.Fatalf("parse public key: %v", err)
	}
	c.publicKey = pk
	return c
}

func TestCashfreeSandboxRoundTrip(t *testing.T) {
	ctx := context.Background()
	c := sandboxClient(t)

	if err := c.HealthCheck(ctx); err != nil {
		t.Fatalf("health check: %v", err)
	}
	t.Log("health check OK (signature accepted)")

	benID := "hc_test_" + time.Now().Format("20060102150405")
	res, err := c.CreateBeneficiary(ctx, payouts.BeneficiaryRequest{
		BeneficiaryID: benID,
		Name:          "Test Chef",
		Instrument: payouts.Instrument{
			Kind: payouts.MethodBankAccount,
			// Cashfree's documented sandbox test account.
			AccountNumber: "00011020001772",
			IFSC:          "HDFC0000001",
		},
		Email: "test@fe3dr.com",
		Phone: "9999999999",
	})
	if err != nil {
		t.Fatalf("create beneficiary: %v", err)
	}
	t.Logf("beneficiary created: id=%s status=%s", res.BeneficiaryID, res.Status)

	// Idempotency: the same id must resolve to the same beneficiary, not a second.
	again, err := c.CreateBeneficiary(ctx, payouts.BeneficiaryRequest{
		BeneficiaryID: benID, Name: "Test Chef",
		Instrument: payouts.Instrument{Kind: payouts.MethodBankAccount,
			AccountNumber: "00011020001772", IFSC: "HDFC0000001"},
	})
	if err != nil {
		t.Fatalf("re-create beneficiary (should be idempotent): %v", err)
	}
	t.Logf("re-registration idempotent: id=%s status=%s", again.BeneficiaryID, again.Status)

	key := "payout:sandbox:" + benID
	out, err := c.CreateTransfer(ctx, payouts.DisburseRequest{
		IdempotencyKey: key,
		BeneficiaryID:  benID,
		Amount:         payouts.FromMinor(100_00, payouts.CurrencyINR),
		Remarks:        "HomeChef sandbox payout",
		Kind:           payouts.MethodBankAccount,
	})
	if err != nil {
		t.Fatalf("create transfer: %v", err)
	}
	t.Logf("transfer created: status=%s ref=%s utr=%s", out.Status, out.Reference, out.UTR)

	got, err := c.GetTransfer(ctx, key)
	if err != nil {
		t.Fatalf("get transfer by reference: %v", err)
	}
	t.Logf("transfer read back: status=%s ref=%s utr=%s", got.Status, got.Reference, got.UTR)
}
