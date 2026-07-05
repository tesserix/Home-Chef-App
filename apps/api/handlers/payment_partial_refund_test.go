package handlers

// payment_partial_refund_test.go — #549. InitiateRefund must treat a PARTIAL refund
// (amount < remaining) as NON-terminal: it may NOT flip status / payment_status →
// Refunded nor stamp refunded_at, because every release-side payout guard blocks the
// WHOLE chef hold on `refunded_at IS NOT NULL`. Only a FULL refund is terminal. These
// drive the to-wallet branch (GetRazorpay()==nil in tests, so the gateway branch would
// 503 before persist; the wallet branch reaches the persist that carries the flip).

import (
	"database/sql"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// addWalletTables adds the wallet ledger tables so the to-wallet refund branch
// (CreditWallet) actually executes under setupPayDB.
func addWalletTables(t *testing.T, db *gorm.DB) {
	t.Helper()
	require.NoError(t, db.Exec(`CREATE TABLE wallets (id TEXT PRIMARY KEY, user_id TEXT UNIQUE, balance REAL DEFAULT 0,
		currency TEXT DEFAULT 'INR', created_at DATETIME, updated_at DATETIME)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE wallet_txns (id TEXT PRIMARY KEY, wallet_id TEXT, user_id TEXT, type TEXT, source TEXT,
		amount REAL, balance_after REAL, currency TEXT, order_id TEXT, reason TEXT, created_by TEXT,
		idempotency_key TEXT UNIQUE, created_at DATETIME)`).Error)
}

type refundState struct {
	PaymentStatus string
	Status        string
	RefundAmount  float64
	RefundedAt    sql.NullTime
}

func loadRefundState(t *testing.T, db *gorm.DB, orderID uuid.UUID) refundState {
	t.Helper()
	var s refundState
	require.NoError(t, db.Raw(
		`SELECT payment_status, status, refund_amount, refunded_at FROM orders WHERE id = ?`,
		orderID.String()).Scan(&s).Error)
	return s
}

// A partial to-wallet refund records the credit but leaves the order NON-terminal:
// payment_status stays completed, status unchanged, refunded_at NULL — so the chef's
// payout hold stays releasable for the remainder (#549).
func TestInitiateRefund_PartialToWallet_NotTerminal(t *testing.T) {
	db := setupPayDB(t)
	addWalletTables(t, db)
	cust := payUser(t, db, "customer")
	chefUser := payUser(t, db, "chef")
	chef := payChef(t, db, chefUser)
	orderID := payOrder(t, db, cust, chef, "completed", 500, "rzp_order_x", "pay_x")

	w := callPay(chefUser, http.MethodPost, "/payments/order/"+orderID.String()+"/refund", regRefund,
		map[string]any{"reason": "partial goodwill", "amount": 100.0, "toWallet": true})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	s := loadRefundState(t, db, orderID)
	require.Equal(t, "completed", s.PaymentStatus, "partial refund must NOT flip payment_status → refunded")
	require.NotEqual(t, "refunded", s.Status, "partial refund must NOT flip status → refunded")
	require.False(t, s.RefundedAt.Valid, "partial refund must NOT stamp refunded_at (would re-block the whole hold)")
	require.Equal(t, 100.0, s.RefundAmount, "the partial refund amount is recorded")
}

// A full to-wallet refund IS terminal: payment_status + status → refunded and
// refunded_at stamped — the whole payout is correctly blocked.
func TestInitiateRefund_FullToWallet_Terminal(t *testing.T) {
	db := setupPayDB(t)
	addWalletTables(t, db)
	cust := payUser(t, db, "customer")
	chefUser := payUser(t, db, "chef")
	chef := payChef(t, db, chefUser)
	orderID := payOrder(t, db, cust, chef, "completed", 500, "rzp_order_x", "pay_x")

	w := callPay(chefUser, http.MethodPost, "/payments/order/"+orderID.String()+"/refund", regRefund,
		map[string]any{"reason": "full refund", "amount": 500.0, "toWallet": true})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	s := loadRefundState(t, db, orderID)
	require.Equal(t, "refunded", s.PaymentStatus, "a full refund flips payment_status → refunded")
	require.Equal(t, "refunded", s.Status, "a full refund flips status → refunded")
	require.True(t, s.RefundedAt.Valid, "a full refund stamps refunded_at")
	require.Equal(t, 500.0, s.RefundAmount)
}

// #549 makes partial refunds repeatable (the order stays non-terminal), so two
// sequential partial to-wallet refunds must EACH credit the customer — the wallet
// idempotency key is per refund-instance, not per order. A per-order key would make
// the second refund silently no-op the credit while still recording it (customer
// shorted) and clawing back the chef.
func TestInitiateRefund_RepeatPartialToWallet_CreditsEachTime(t *testing.T) {
	db := setupPayDB(t)
	addWalletTables(t, db)
	cust := payUser(t, db, "customer")
	chefUser := payUser(t, db, "chef")
	chef := payChef(t, db, chefUser)
	orderID := payOrder(t, db, cust, chef, "completed", 500, "rzp_order_x", "pay_x")

	w1 := callPay(chefUser, http.MethodPost, "/payments/order/"+orderID.String()+"/refund", regRefund,
		map[string]any{"reason": "goodwill 1", "amount": 100.0, "toWallet": true})
	require.Equal(t, http.StatusOK, w1.Code, w1.Body.String())
	w2 := callPay(chefUser, http.MethodPost, "/payments/order/"+orderID.String()+"/refund", regRefund,
		map[string]any{"reason": "goodwill 2", "amount": 150.0, "toWallet": true})
	require.Equal(t, http.StatusOK, w2.Code, w2.Body.String())

	var balance float64
	require.NoError(t, db.Raw(`SELECT balance FROM wallets WHERE user_id = ?`, cust.String()).Scan(&balance).Error)
	require.Equal(t, 250.0, balance, "both partial refunds credit the wallet (no per-order key collision)")

	s := loadRefundState(t, db, orderID)
	require.Equal(t, 250.0, s.RefundAmount, "both partial amounts accumulate")
	require.Equal(t, "completed", s.PaymentStatus, "two partials keep the order non-terminal")
	require.False(t, s.RefundedAt.Valid)

	var n int64
	require.NoError(t, db.Raw(`SELECT COUNT(*) FROM wallet_txns WHERE user_id = ?`, cust.String()).Scan(&n).Error)
	require.Equal(t, int64(2), n, "two distinct wallet credit rows, not one deduped")
}

// The wallet idempotency key is stable for a given prior-refunded snapshot (so a
// lost-response retry of the SAME refund dedups) but distinct once prior-refunded
// advances (so a genuinely later refund credits).
func TestRefundWalletIdempotencyKey_StablePerSnapshotDistinctAcrossRefunds(t *testing.T) {
	id := uuid.New()
	require.Equal(t, refundWalletIdempotencyKey("refund", id, 0), refundWalletIdempotencyKey("refund", id, 0),
		"same prior-refunded ⇒ same key ⇒ retry dedups")
	require.NotEqual(t, refundWalletIdempotencyKey("refund", id, 0), refundWalletIdempotencyKey("refund", id, 100),
		"advanced prior-refunded ⇒ distinct key ⇒ the next refund credits")
	require.NotEqual(t, refundWalletIdempotencyKey("refund", id, 0), refundWalletIdempotencyKey("refund-wallet", id, 0),
		"distinct prefixes never collide")
}

// The default (amount omitted) refunds the full remaining balance and is terminal —
// the omitted-amount convenience path must land in the FULL branch, not partial.
func TestInitiateRefund_DefaultAmountToWallet_Terminal(t *testing.T) {
	db := setupPayDB(t)
	addWalletTables(t, db)
	cust := payUser(t, db, "customer")
	chefUser := payUser(t, db, "chef")
	chef := payChef(t, db, chefUser)
	orderID := payOrder(t, db, cust, chef, "completed", 500, "rzp_order_x", "pay_x")

	w := callPay(chefUser, http.MethodPost, "/payments/order/"+orderID.String()+"/refund", regRefund,
		map[string]any{"reason": "whole order", "toWallet": true})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	s := loadRefundState(t, db, orderID)
	require.Equal(t, "refunded", s.PaymentStatus, "omitted amount = full remaining → terminal")
	require.True(t, s.RefundedAt.Valid)
	require.Equal(t, 500.0, s.RefundAmount)
}
