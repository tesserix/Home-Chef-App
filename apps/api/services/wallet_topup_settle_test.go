package services

// wallet_topup_settle_test.go — #554. The platform-funded wallet top-up transfer had
// no idempotency guard, so a retried VerifyPayment re-issued the same real money
// transfer. settleWalletTopUpsWith now claims each (order, account) once and only
// transfers on the winning claim, releasing on failure so a retry re-attempts.
//
// Relocated from handlers/wallet_topup_settle_test.go (#872 step 2, Task 2) —
// settleWalletTopUpsWith moved to services alongside the rest of the
// wallet-settlement cluster. settleWalletTopUpsWith reads the package-global
// database.DB (not a parameter, since ClaimWalletTopUp/ReleaseWalletTopUp are
// hard-coded to it), so each test swaps it to the fresh in-memory db and
// restores it on cleanup — setupTopUpDedupDB itself does not do this swap,
// since its own tests (wallet_topup_dedup_test.go) pass db explicitly.

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/homechef/api/database"
)

func withTopUpDedupDB(t *testing.T) {
	t.Helper()
	db := setupTopUpDedupDB(t)
	prev := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = prev })
}

func TestSettleWalletTopUps_TransfersOncePerAccountAcrossRetries(t *testing.T) {
	withTopUpDedupDB(t)
	orderID := uuid.New()
	topUps := []TransferSpec{
		{Account: "acc_chef", Amount: 1000, Currency: "INR"},
		{Account: "acc_driver", Amount: 500, Currency: "INR"},
	}
	calls := map[string]int{}
	doTransfer := func(_ int, ts TransferSpec) error { calls[ts.Account]++; return nil }

	settleWalletTopUpsWith(orderID, "ORD-1", topUps, doTransfer)
	settleWalletTopUpsWith(orderID, "ORD-1", topUps, doTransfer) // retried verify

	require.Equal(t, 1, calls["acc_chef"], "chef top-up transferred exactly once")
	require.Equal(t, 1, calls["acc_driver"], "driver top-up transferred exactly once")
}

// #558: two legs sharing ONE Razorpay payout account (same person as chef AND driver) must each
// be paid — keying on account alone silently deduped the second leg into a no-op.
func TestSettleWalletTopUps_SameAccountBothLegsPaid(t *testing.T) {
	withTopUpDedupDB(t)
	orderID := uuid.New()
	topUps := []TransferSpec{
		{Account: "acc_shared", Amount: 1000, Currency: "INR"}, // chef leg
		{Account: "acc_shared", Amount: 500, Currency: "INR"},  // driver leg, same account
	}
	var total int
	doTransfer := func(_ int, ts TransferSpec) error { total += ts.Amount; return nil }

	settleWalletTopUpsWith(orderID, "ORD-1", topUps, doTransfer)
	settleWalletTopUpsWith(orderID, "ORD-1", topUps, doTransfer) // retried verify — still idempotent

	require.Equal(t, 1500, total, "both legs paid exactly once despite the shared account (per-leg key)")
}

func TestSettleWalletTopUps_FailedTransferIsRetried(t *testing.T) {
	withTopUpDedupDB(t)
	orderID := uuid.New()
	topUps := []TransferSpec{{Account: "acc_chef", Amount: 1000, Currency: "INR"}}

	attempts := 0
	failing := func(int, TransferSpec) error { attempts++; return errors.New("gateway down") }
	ok := func(int, TransferSpec) error { attempts++; return nil }

	settleWalletTopUpsWith(orderID, "ORD-1", topUps, failing) // fails → claim released
	settleWalletTopUpsWith(orderID, "ORD-1", topUps, ok)      // retries → succeeds

	require.Equal(t, 2, attempts, "a failed transfer is retried on the next settlement")

	// And once it succeeds, a further settlement is a no-op.
	settleWalletTopUpsWith(orderID, "ORD-1", topUps, ok)
	require.Equal(t, 2, attempts, "no re-transfer after success")
}
