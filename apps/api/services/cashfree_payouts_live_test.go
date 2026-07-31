//go:build cfsandbox

package services

// Live sandbox round-trip. Excluded from the normal build by the cfsandbox tag —
// it talks to Cashfree's real sandbox and must never run in CI.
//   go test ./services/ -tags cfsandbox -run TestCashfreeSandbox -v
//
// Deliberately re-runnable: beneficiary ids are the production scheme (payee +
// instrument digest), so a second run resolves to the same beneficiaries instead
// of minting timestamped ones. The sandbox enforces one beneficiary per
// account/IFSC across the merchant account, so accounts burned by other ids are
// skipped until a usable one is found.

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/homechef/api/models"
	"github.com/homechef/api/payouts"
)

// The fixed fake payee this test registers beneficiaries under.
var sandboxTestPayee = payouts.PayeeRef{
	Type: payouts.PayeeChef,
	ID:   uuid.MustParse("e2e00000-0000-4000-8000-00000000cafe"),
}

// Cashfree's documented sandbox accounts that simulate SUCCESS transfers.
var sandboxSuccessAccounts = []payouts.Instrument{
	{Kind: payouts.MethodBankAccount, AccountNumber: "00011020001772", IFSC: "HDFC0000001"},
	{Kind: payouts.MethodBankAccount, AccountNumber: "026291800001191", IFSC: "YESB0000262"},
	{Kind: payouts.MethodBankAccount, AccountNumber: "1233943142", IFSC: "ICIC0000009"},
	{Kind: payouts.MethodBankAccount, AccountNumber: "388108022658", IFSC: "ICIC0000009"},
	{Kind: payouts.MethodBankAccount, AccountNumber: "000890289871772", IFSC: "SCBL0036078"},
	{Kind: payouts.MethodBankAccount, AccountNumber: "000100289877623", IFSC: "SBIN0008752"},
}

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

	// Register (or resolve) a beneficiary, skipping accounts already owned by
	// some other beneficiary id in this shared sandbox account.
	var benID string
	var instrument payouts.Instrument
	for _, in := range sandboxSuccessAccounts {
		id := payouts.BeneficiaryIDForInstrument(sandboxTestPayee, in)
		res, err := c.CreateBeneficiary(ctx, payouts.BeneficiaryRequest{
			BeneficiaryID: id,
			Name:          "Test Chef",
			Instrument:    in,
			Email:         "test@fe3dr.com",
			Phone:         "9999999999",
		})
		if errors.Is(err, payouts.ErrBeneficiaryRejected) {
			t.Logf("account %s unavailable (%v) — trying the next", payouts.MaskAccountNumber(in.AccountNumber), err)
			continue
		}
		if err != nil {
			t.Fatalf("create beneficiary %s: %v", id, err)
		}
		benID, instrument = id, in
		t.Logf("beneficiary resolved: id=%s status=%s", res.BeneficiaryID, res.Status)
		break
	}
	if benID == "" {
		t.Fatalf("every documented sandbox success account is owned by another beneficiary id — clean up the sandbox beneficiaries")
	}

	// Idempotency: the same payee + instrument must resolve to the same
	// beneficiary, not a second one.
	again, err := c.CreateBeneficiary(ctx, payouts.BeneficiaryRequest{
		BeneficiaryID: benID, Name: "Test Chef", Instrument: instrument,
	})
	if err != nil {
		t.Fatalf("re-create beneficiary (should be idempotent): %v", err)
	}
	if again.BeneficiaryID != benID {
		t.Fatalf("re-registration resolved to %s, want %s", again.BeneficiaryID, benID)
	}
	t.Logf("re-registration idempotent: id=%s status=%s", again.BeneficiaryID, again.Status)

	// A fresh transfer every run — timestamped key, same beneficiary.
	key := "payout:sandbox:" + time.Now().Format("20060102150405")
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
