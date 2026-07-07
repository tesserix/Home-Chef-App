package handlers

// payment_concurrent_refund_test.go — #567. The InitiateRefund to-wallet branch had
// no atomic refund claim (unlike the gateway branch), so two concurrent double-submits
// on one order could both persist (lost-update on refund_amount) and both claw the
// chef. The claim is now taken via claimRefundForProcessing BEFORE the to-wallet /
// gateway split, so exactly one concurrent request wins and the duplicate gets 409.
//
// The race is exercised deterministically (mirroring payment_complete_race_test.go)
// rather than with goroutines: two back-to-back claims against a completed order model
// the concurrent window where both requests read payment_status='completed' — only the
// first flip wins.

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClaimRefundForProcessing_SerializesConcurrentDuplicates(t *testing.T) {
	db := setupPayDB(t)
	cust := payUser(t, db, "customer")
	chef := payChef(t, db, payUser(t, db, "chef"))
	orderID := payOrder(t, db, cust, chef, "completed", 500, "order_claim", "pay_x")

	// Two requests both read the order as completed (the concurrent window).
	won1, err1 := claimRefundForProcessing(db, orderID)
	require.NoError(t, err1)
	require.True(t, won1, "the winning claim flips completed→refunded")

	won2, err2 := claimRefundForProcessing(db, orderID)
	require.NoError(t, err2)
	require.False(t, won2, "a concurrent duplicate claim loses the flip (→ 409), so it can't double-claw")

	// After a PARTIAL refund reverts payment_status to completed, a later legitimate
	// refund re-claims (repeatable partials — #549).
	require.NoError(t, db.Exec(`UPDATE orders SET payment_status = 'completed' WHERE id = ?`, orderID.String()).Error)
	won3, err3 := claimRefundForProcessing(db, orderID)
	require.NoError(t, err3)
	require.True(t, won3, "a new refund re-claims once the previous partial reverted to completed")

	// A claim against a non-completed (e.g. already-refunded) order never wins.
	require.NoError(t, db.Exec(`UPDATE orders SET payment_status = 'refunded' WHERE id = ?`, orderID.String()).Error)
	won4, err4 := claimRefundForProcessing(db, orderID)
	require.NoError(t, err4)
	require.False(t, won4, "a refunded order cannot be claimed for another refund")
}

// #602: a PARTIAL to-wallet refund whose persist tx FAILS must NOT release the reservation.
// services.ReserveRefund already committed `refund_amount += reserved` in its own tx BEFORE
// the wallet credit, so on a persist failure `refund_amount` is CORRECT (the customer got the
// credit). The old code decremented it back (releaseReservation) to unstick the order —
// erasing a refund that actually happened → the next distinct refund over-refunds / collides
// the amount-based idempotency key. The money-safe behavior is to leave the order STUCK at
// refunded with the ledger correct; reconcileStuckRefunds (slice 2) finalizes it later.
func TestInitiateRefund_PartialToWallet_PersistFailure_StaysStuckLedgerCorrect(t *testing.T) {
	db := setupPayDB(t)
	addWalletTables(t, db)
	cust := payUser(t, db, "customer")
	chefUser := payUser(t, db, "chef")
	chef := payChef(t, db, chefUser)
	orderID := payOrder(t, db, cust, chef, "completed", 500, "order_pf", "pay_x")
	require.NoError(t, db.Exec(`DROP TABLE outbox_events`).Error) // force the persist tx to fail

	w := callPay(chefUser, http.MethodPost, "/payments/order/"+orderID.String()+"/refund", regRefund,
		map[string]any{"reason": "partial", "amount": 100.0, "toWallet": true})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String()) // credit committed; persist failure best-effort

	var row struct {
		PaymentStatus string
		RefundAmount  float64
	}
	require.NoError(t, db.Raw(`SELECT payment_status, refund_amount FROM orders WHERE id = ?`, orderID.String()).Scan(&row).Error)
	require.Equal(t, "refunded", row.PaymentStatus, "persist failure leaves the order STUCK at refunded (NOT released) — the reserve durably recorded the refund")
	require.Equal(t, 100.0, row.RefundAmount, "refund_amount stays CORRECT (reflects the committed wallet credit) — never decremented back")

	var balance float64
	require.NoError(t, db.Raw(`SELECT balance FROM wallets WHERE user_id = ?`, cust.String()).Scan(&balance).Error)
	require.Equal(t, 100.0, balance, "the wallet credit committed")
}

// A FULL to-wallet refund whose persist fails legitimately STAYS refunded — its
// unconditional cross-guard is the safety net that must block the chef payout.
func TestInitiateRefund_FullToWallet_PersistFailure_StaysRefunded(t *testing.T) {
	db := setupPayDB(t)
	addWalletTables(t, db)
	cust := payUser(t, db, "customer")
	chefUser := payUser(t, db, "chef")
	chef := payChef(t, db, chefUser)
	orderID := payOrder(t, db, cust, chef, "completed", 500, "order_pf2", "pay_x")
	require.NoError(t, db.Exec(`DROP TABLE outbox_events`).Error)

	w := callPay(chefUser, http.MethodPost, "/payments/order/"+orderID.String()+"/refund", regRefund,
		map[string]any{"reason": "full", "amount": 500.0, "toWallet": true})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var status string
	require.NoError(t, db.Raw(`SELECT payment_status FROM orders WHERE id = ?`, orderID.String()).Scan(&status).Error)
	require.Equal(t, "refunded", status, "a full refund stays refunded even on persist failure (safety net)")
}
