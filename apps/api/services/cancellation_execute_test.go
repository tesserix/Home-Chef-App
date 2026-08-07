package services

// cancellation_execute_test.go — #636. ExecuteCancellationRefund persisted refund_amount via the
// caller's STALE in-memory `order.RefundAmount + refund`. Its payment_status claim excludes the
// ReserveRefund family, but NOT RefundIssueToWallet (which never flips payment_status) — so a
// partial issue refund that atomically incremented refund_amount between the caller's read and
// this write was clobbered. The fix increments in-SQL (COALESCE(refund_amount,0) + refund).

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

// addCancellationRequestsTable + a seeded wallet let setupCancelRefundDB drive
// ExecuteCancellationRefund's wallet-destination path.
func seedCancelExecFixtures(t *testing.T, db *gorm.DB, o *models.Order, refundTotalPaise int) *models.CancellationRequest {
	t.Helper()
	require.NoError(t, db.Exec(`CREATE TABLE IF NOT EXISTS cancellation_requests (
		id TEXT PRIMARY KEY, order_id TEXT, customer_id TEXT, refund_destination TEXT DEFAULT 'wallet',
		refund_total_paise INTEGER DEFAULT 0, food_refund_paise INTEGER DEFAULT 0, delivery_refund_paise INTEGER DEFAULT 0,
		tax_refund_paise INTEGER DEFAULT 0, vendor_kept_paise INTEGER DEFAULT 0, platform_kept_paise INTEGER DEFAULT 0,
		refund_executed BOOLEAN DEFAULT 0, refund_ref TEXT DEFAULT '',
		resolved_at DATETIME, created_at DATETIME, updated_at DATETIME)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO wallets (id, user_id, balance) VALUES (?,?,0)`,
		uuid.NewString(), o.CustomerID.String()).Error)
	crID := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO cancellation_requests (id, order_id, customer_id, refund_destination, refund_total_paise, refund_executed)
		VALUES (?,?,?, 'wallet', ?, 0)`, crID.String(), o.ID.String(), o.CustomerID.String(), refundTotalPaise).Error)
	return &models.CancellationRequest{
		ID: crID, OrderID: o.ID, CustomerID: o.CustomerID,
		RefundTotalPaise: refundTotalPaise, RefundDestination: "wallet",
	}
}

// seedCancelExecFixturesDest mirrors seedCancelExecFixtures but lets the test pick the
// refund destination — "original" exercises the gateway-split path below (the "wallet"
// destination is already covered by seedCancelExecFixtures).
func seedCancelExecFixturesDest(t *testing.T, db *gorm.DB, o *models.Order, refundTotalPaise int, destination string) *models.CancellationRequest {
	t.Helper()
	require.NoError(t, db.Exec(`CREATE TABLE IF NOT EXISTS cancellation_requests (
		id TEXT PRIMARY KEY, order_id TEXT, customer_id TEXT, refund_destination TEXT DEFAULT 'wallet',
		refund_total_paise INTEGER DEFAULT 0, food_refund_paise INTEGER DEFAULT 0, delivery_refund_paise INTEGER DEFAULT 0,
		tax_refund_paise INTEGER DEFAULT 0, vendor_kept_paise INTEGER DEFAULT 0, platform_kept_paise INTEGER DEFAULT 0,
		refund_executed BOOLEAN DEFAULT 0, refund_ref TEXT DEFAULT '',
		resolved_at DATETIME, created_at DATETIME, updated_at DATETIME)`).Error)
	crID := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO cancellation_requests (id, order_id, customer_id, refund_destination, refund_total_paise, refund_executed)
		VALUES (?,?,?,?,?,0)`, crID.String(), o.ID.String(), o.CustomerID.String(), destination, refundTotalPaise).Error)
	return &models.CancellationRequest{
		ID: crID, OrderID: o.ID, CustomerID: o.CustomerID,
		RefundTotalPaise: refundTotalPaise, RefundDestination: destination,
	}
}

// seedMixedPaymentOrder is a gateway-captured order that ALSO consumed wallet AND
// loyalty store credit at checkout, so the gateway only ever captured
// (Total − WalletApplied − LoyaltyApplied). gatewayOrderID may be "" to model a
// fully-credit-funded order that never touched the gateway at all.
func seedMixedPaymentOrder(t *testing.T, db *gorm.DB, total, walletApplied, loyaltyApplied float64, gatewayOrderID string) *models.Order {
	t.Helper()
	o := &models.Order{
		ID: uuid.New(), OrderNumber: "ORD-MIX", CustomerID: uuid.New(), ChefID: uuid.New(),
		Status: models.OrderStatusPreparing, PaymentStatus: models.PaymentCompleted,
		PaymentProvider: "cashfree", RazorpayOrderID: gatewayOrderID,
		Total: total, WalletApplied: walletApplied, LoyaltyApplied: loyaltyApplied,
	}
	require.NoError(t, db.Exec(`INSERT INTO orders (id, order_number, customer_id, chef_id, status, payment_status,
		payment_provider, razorpay_order_id, total, wallet_applied, loyalty_applied, refund_amount)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`,
		o.ID.String(), o.OrderNumber, o.CustomerID.String(), o.ChefID.String(), string(o.Status),
		string(models.PaymentCompleted), "cashfree", gatewayOrderID, total, walletApplied, loyaltyApplied, 0.0).Error)
	return o
}

// #766: a mixed-payment order (wallet + loyalty credit applied at checkout, so the
// gateway only ever captured Total−WalletApplied−LoyaltyApplied) hit a live 502 on
// POST /orders/:id/cancel-request: ExecuteCancellationRefund sent the FULL
// cr.RefundTotalPaise to Razorpay, which exceeds what was actually captured on the
// card. Razorpay rejects ("amount greater than amount captured"), the sweep re-sends
// the same wrong amount forever, and the customer is never refunded. The fix splits
// the refund across funding rails — wallet + loyalty slices credit back instantly,
// and ONLY the card slice reaches the gateway (mirrors splitCancelRefundAcrossRails,
// the already-correct chef-cancel path).
func TestExecuteCancellationRefund_MixedPayment_SplitsWalletAndLoyaltyOffGateway(t *testing.T) {
	db := setupCancelRefundDB(t)
	o := seedMixedPaymentOrder(t, db, 481.91, 150.40, 3.85, "cf_ord_mix")
	cr := seedCancelExecFixturesDest(t, db, o, 46295, "original") // ₹462.95 refund

	spy := withCashfreeRefundSpy(t, http.StatusOK)

	stale := &models.Order{ID: o.ID, CustomerID: o.CustomerID, OrderNumber: o.OrderNumber,
		PaymentProvider: "cashfree", RazorpayOrderID: "cf_ord_mix",
		Total: 481.91, WalletApplied: 150.40, LoyaltyApplied: 3.85, RefundAmount: 0}
	require.NoError(t, ExecuteCancellationRefund(stale, cr), "must not 502 on a mixed-payment refund")

	require.NotEqual(t, 46295, spy.amountPaise, "the full refund total must never be sent to the gateway")
	require.LessOrEqual(t, spy.amountPaise, 32766, "card slice must never exceed what was actually captured (Total-Wallet-Loyalty = 481.91-150.40-3.85)")
	require.Equal(t, 31478, spy.amountPaise, "gateway gets exactly the pro-rata card slice")

	var status string
	require.NoError(t, db.Raw(`SELECT status FROM orders WHERE id = ?`, o.ID.String()).Scan(&status).Error)
	require.Equal(t, string(models.OrderStatusCancelled), status)

	ps, refundAmount, _, _ := loadRefund(t, db, o.ID)
	require.Equal(t, string(models.PaymentRefunded), ps)
	require.InDelta(t, 462.95, refundAmount, 0.01, "refund_amount is the full total across all rails, unchanged by the split")

	var bal float64
	require.NoError(t, db.Raw(`SELECT balance FROM wallets WHERE user_id = ?`, o.CustomerID.String()).Scan(&bal).Error)
	require.InDelta(t, 148.17, bal, 0.01, "wallet credited the wallet (144.48) + loyalty (3.69) slices")
}

// A fully credit-funded order (wallet + loyalty cover the entire refund, so the card
// slice is 0) must refund entirely to the wallet with NO gateway call at all — even
// when the order has no Razorpay payment on file (nothing was ever captured).
func TestExecuteCancellationRefund_FullyCreditFunded_NoGatewayCall(t *testing.T) {
	db := setupCancelRefundDB(t)
	o := seedMixedPaymentOrder(t, db, 200, 150, 50, "") // wallet+loyalty cover the whole order
	cr := seedCancelExecFixturesDest(t, db, o, 20000, "original")

	spy := withCashfreeRefundSpy(t, http.StatusOK)

	stale := &models.Order{ID: o.ID, CustomerID: o.CustomerID, OrderNumber: o.OrderNumber,
		PaymentProvider: "razorpay", RazorpayPaymentID: "",
		Total: 200, WalletApplied: 150, LoyaltyApplied: 50, RefundAmount: 0}
	require.NoError(t, ExecuteCancellationRefund(stale, cr), "a fully credit-funded refund needs no gateway payment")

	require.False(t, spy.calls > 0, "a fully credit-funded order must not call the gateway")

	ps, refundAmount, _, _ := loadRefund(t, db, o.ID)
	require.Equal(t, string(models.PaymentRefunded), ps)
	require.InDelta(t, 200, refundAmount, 0.01)

	var bal float64
	require.NoError(t, db.Raw(`SELECT balance FROM wallets WHERE user_id = ?`, o.CustomerID.String()).Scan(&bal).Error)
	require.InDelta(t, 200, bal, 0.01, "the entire refund credits to the wallet when nothing was captured at the gateway")
}

// The clobber: a concurrent partial issue refund committed refund_amount=100 (in the DB) AFTER the
// caller loaded `order` (so the caller's snapshot still reads 0). ExecuteCancellationRefund must
// increment the CURRENT value, not overwrite it with stale+refund.
func TestExecuteCancellationRefund_AtomicIncrement_NoClobber(t *testing.T) {
	db := setupCancelRefundDB(t)
	// DB reflects the concurrent issue refund: refund_amount=100, still completed / not refunded.
	o := seedWalletOrder(t, db, models.PaymentCompleted, 1000, 100)
	cr := seedCancelExecFixtures(t, db, o, 50000) // cancellation refunds ₹500

	// The caller's STALE snapshot — loaded before the issue refund committed → RefundAmount 0.
	stale := &models.Order{ID: o.ID, CustomerID: o.CustomerID, PaymentProvider: "wallet", RefundAmount: 0}
	require.NoError(t, ExecuteCancellationRefund(stale, cr))

	_, refundAmount, _, _ := loadRefund(t, db, o.ID)
	require.InDelta(t, 600, refundAmount, 0.01,
		"#636: refund_amount = 100 (concurrent issue) + 500 (cancellation), not the stale 0 + 500")
}

// Happy path (no prior refund): refund_amount = the cancellation amount.
func TestExecuteCancellationRefund_NoPriorRefund(t *testing.T) {
	db := setupCancelRefundDB(t)
	o := seedWalletOrder(t, db, models.PaymentCompleted, 1000, 0)
	cr := seedCancelExecFixtures(t, db, o, 50000)

	stale := &models.Order{ID: o.ID, CustomerID: o.CustomerID, PaymentProvider: "wallet", RefundAmount: 0}
	require.NoError(t, ExecuteCancellationRefund(stale, cr))

	status, refundAmount, _, refundedAt := loadRefund(t, db, o.ID)
	require.InDelta(t, 500, refundAmount, 0.01)
	require.Equal(t, string(models.PaymentRefunded), status)
	require.NotNil(t, refundedAt)
}

// Idempotent: a second run (sweep retry) loses the payment_status claim and must NOT increment
// refund_amount again.
func TestExecuteCancellationRefund_Idempotent_NoDoubleIncrement(t *testing.T) {
	db := setupCancelRefundDB(t)
	o := seedWalletOrder(t, db, models.PaymentCompleted, 1000, 0)
	cr := seedCancelExecFixtures(t, db, o, 50000)

	stale := &models.Order{ID: o.ID, CustomerID: o.CustomerID, PaymentProvider: "wallet", RefundAmount: 0}
	require.NoError(t, ExecuteCancellationRefund(stale, cr))
	_, after1, _, _ := loadRefund(t, db, o.ID)

	// Re-run with a fresh CR struct (the sweep re-drives) — claim is lost (already refunded).
	cr2 := &models.CancellationRequest{ID: cr.ID, OrderID: o.ID, CustomerID: o.CustomerID,
		RefundTotalPaise: 50000, RefundDestination: "wallet"}
	require.NoError(t, ExecuteCancellationRefund(stale, cr2))
	_, after2, _, _ := loadRefund(t, db, o.ID)

	require.InDelta(t, after1, after2, 0.001, "a retry after the claim is lost must not double-increment refund_amount")
	require.InDelta(t, 500, after2, 0.01)
}

// #644: the cancellation refund cap is snapshot-time only. cr.RefundTotalPaise was capped at
// RemainingRefundable at DECISION time — an UNLOCKED read. If a RefundIssueToWallet partial (which
// never flips payment_status, so the payment_status claim doesn't exclude it) commits between that
// snapshot and the money move, the stale snapshot can push cumulative refunds past Total.
// ExecuteCancellationRefund must re-derive the remaining UNDER the order lock and cap the refund.
func TestExecuteCancellationRefund_CapsAtRemainingRefundable(t *testing.T) {
	db := setupCancelRefundDB(t)
	// A concurrent issue refund already returned ₹800 of the ₹1000 order — payment_status stays
	// completed and refunded_at NULL (the issue path never flips payment_status).
	o := seedWalletOrder(t, db, models.PaymentCompleted, 1000, 800)
	// The cancellation was decided earlier to refund ₹500 (snapshot) — stale now that only ₹200 is left.
	cr := seedCancelExecFixtures(t, db, o, 50000)
	cr.FoodRefundPaise = 50000 // the decided breakdown was all-food (scales proportionally on cap)

	stale := &models.Order{ID: o.ID, CustomerID: o.CustomerID, PaymentProvider: "wallet", RefundAmount: 0}
	require.NoError(t, ExecuteCancellationRefund(stale, cr))

	status, refundAmount, _, refundedAt := loadRefund(t, db, o.ID)
	require.InDelta(t, 1000, refundAmount, 0.01,
		"#644: capped at remaining ₹200 (Total 1000 − 800 already refunded) → 800+200, never the stale 800+500=1300")
	require.LessOrEqual(t, refundAmount, 1000.0+0.01, "cumulative refund must never exceed Total")
	require.Equal(t, string(models.PaymentRefunded), status)
	require.NotNil(t, refundedAt)

	// Only the capped ₹200 hit the wallet, not the stale ₹500.
	var bal float64
	require.NoError(t, db.Raw(`SELECT balance FROM wallets WHERE user_id = ?`, o.CustomerID.String()).Scan(&bal).Error)
	require.InDelta(t, 200, bal, 0.01, "#644: wallet credited the capped remaining, not the stale snapshot")

	// #644 verify (finding 4): the persisted request reflects the CAPPED amount + a conserved
	// breakdown, not the stale pre-cap snapshot the customer never received.
	var row struct {
		RefundTotalPaise, FoodRefundPaise, VendorKeptPaise, PlatformKeptPaise int
	}
	require.NoError(t, db.Raw(`SELECT refund_total_paise, food_refund_paise, vendor_kept_paise, platform_kept_paise
		FROM cancellation_requests WHERE order_id = ?`, o.ID.String()).Scan(&row).Error)
	require.Equal(t, 20000, row.RefundTotalPaise, "persisted refund_total_paise is the capped ₹200, not the stale ₹500")
	require.Equal(t, 50000, row.RefundTotalPaise+row.VendorKeptPaise+row.PlatformKeptPaise,
		"breakdown stays conserved: grand (₹500) == refundTotal + vendorKept + platformKept")
}
