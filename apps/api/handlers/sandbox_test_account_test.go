package handlers

// #1158. The seeded account feeds two rails — the Payouts beneficiary and the
// Easy Split vendor's penny drop — and Easy Split accepts fewer of Cashfree's
// sandbox accounts than Payouts does.

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// Documented Success on the Easy Split sandbox page, and SUCCESS on the Payouts
// one. Any account outside this set strands a seeded chef on one rail or other.
var easySplitSandboxSuccess = map[string]string{
	"00011020001772":  "HDFC0000001",
	"026291800001191": "YESB0000262",
	"000890289871772": "SCBL0036078",
}

func TestSandboxAccountsVerifyOnBothRails(t *testing.T) {
	for _, a := range cashfreeSandboxTestAccounts {
		ifsc, ok := easySplitSandboxSuccess[a.account]
		require.Truef(t, ok, "account %s is not a documented Easy Split success", a.account)
		require.Equal(t, ifsc, a.ifsc)
	}
}

func TestEveryChefDrawsAnAccountBothRailsAccept(t *testing.T) {
	for range 500 {
		account, ifsc := sandboxTestAccountFor(uuid.New())
		require.Equal(t, easySplitSandboxSuccess[account], ifsc,
			"drew %s/%s, which Easy Split does not verify", account, ifsc)
	}
}

func TestSaffronDrawsAnAccountEasySplitVerifies(t *testing.T) {
	// The kitchen that surfaced this: it hashed to SBIN0008752, documented
	// "Failed at the bank", so its vendor could never leave BANK_VALIDATION_FAILED.
	saffron := uuid.MustParse("e150c72a-42e2-4beb-8cb1-389666dd813c")
	account, ifsc := sandboxTestAccountFor(saffron)
	require.NotEqual(t, "000100289877623", account)
	require.Equal(t, easySplitSandboxSuccess[account], ifsc)
}
