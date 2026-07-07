package handlers

// wallet_topup_settle_test.go — #554. The platform-funded wallet top-up transfer had
// no idempotency guard, so a retried VerifyPayment re-issued the same real money
// transfer. settleWalletTopUpsWith now claims each (order, account) once and only
// transfers on the winning claim, releasing on failure so a retry re-attempts.

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/homechef/api/services"
	"github.com/stretchr/testify/require"
)

func TestSettleWalletTopUps_TransfersOncePerAccountAcrossRetries(t *testing.T) {
	addProcessedEventsTable(t, setupPayDB(t))
	orderID := uuid.New()
	topUps := []services.TransferSpec{
		{Account: "acc_chef", Amount: 1000, Currency: "INR"},
		{Account: "acc_driver", Amount: 500, Currency: "INR"},
	}
	calls := map[string]int{}
	doTransfer := func(_ int, ts services.TransferSpec) error { calls[ts.Account]++; return nil }

	settleWalletTopUpsWith(orderID, "ORD-1", topUps, doTransfer)
	settleWalletTopUpsWith(orderID, "ORD-1", topUps, doTransfer) // retried verify

	require.Equal(t, 1, calls["acc_chef"], "chef top-up transferred exactly once")
	require.Equal(t, 1, calls["acc_driver"], "driver top-up transferred exactly once")
}

// #558: two legs sharing ONE Razorpay payout account (same person as chef AND driver) must each
// be paid — keying on account alone silently deduped the second leg into a no-op.
func TestSettleWalletTopUps_SameAccountBothLegsPaid(t *testing.T) {
	addProcessedEventsTable(t, setupPayDB(t))
	orderID := uuid.New()
	topUps := []services.TransferSpec{
		{Account: "acc_shared", Amount: 1000, Currency: "INR"}, // chef leg
		{Account: "acc_shared", Amount: 500, Currency: "INR"},  // driver leg, same account
	}
	var total int
	doTransfer := func(_ int, ts services.TransferSpec) error { total += ts.Amount; return nil }

	settleWalletTopUpsWith(orderID, "ORD-1", topUps, doTransfer)
	settleWalletTopUpsWith(orderID, "ORD-1", topUps, doTransfer) // retried verify — still idempotent

	require.Equal(t, 1500, total, "both legs paid exactly once despite the shared account (per-leg key)")
}

func TestSettleWalletTopUps_FailedTransferIsRetried(t *testing.T) {
	addProcessedEventsTable(t, setupPayDB(t))
	orderID := uuid.New()
	topUps := []services.TransferSpec{{Account: "acc_chef", Amount: 1000, Currency: "INR"}}

	attempts := 0
	failing := func(int, services.TransferSpec) error { attempts++; return errors.New("gateway down") }
	ok := func(int, services.TransferSpec) error { attempts++; return nil }

	settleWalletTopUpsWith(orderID, "ORD-1", topUps, failing) // fails → claim released
	settleWalletTopUpsWith(orderID, "ORD-1", topUps, ok)      // retries → succeeds

	require.Equal(t, 2, attempts, "a failed transfer is retried on the next settlement")

	// And once it succeeds, a further settlement is a no-op.
	settleWalletTopUpsWith(orderID, "ORD-1", topUps, ok)
	require.Equal(t, 2, attempts, "no re-transfer after success")
}
