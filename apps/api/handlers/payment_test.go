package handlers

// payment_test.go — backend verification for issue #6 (payment happy-path +
// failure paths). The on-device WebView tap-through (Razorpay Standard Checkout)
// and the live-key switch are manual/owner actions (#25). What we pin down here
// is the server-side contract the device depends on, WITHOUT needing a live
// gateway: ownership/IDOR, "already paid" guard, verify-step validation, refund
// authorization + preconditions, and that the webhook fails CLOSED on a bad
// signature (no silent state change). These are the "no silent failure" and
// "no double-charge surface" guarantees the issue calls for.
//
// Self-contained in-memory SQLite (own tables + helpers, distinct names from the
// other handler tests in this package). Not parallel — shares the global DB.

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
)

func setupPayDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	require.NoError(t, err)

	// users + orders carry gorm.DeletedAt → GORM appends `deleted_at IS NULL`;
	// the column must exist or the (preloaded) order load errors → spurious 404.
	require.NoError(t, db.Exec(`CREATE TABLE users (
		id TEXT PRIMARY KEY, email TEXT, first_name TEXT DEFAULT '', last_name TEXT DEFAULT '',
		phone TEXT DEFAULT '', role TEXT DEFAULT 'customer', is_active INTEGER DEFAULT 1,
		created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
	)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE chef_profiles (
		id TEXT PRIMARY KEY, user_id TEXT, business_name TEXT DEFAULT '',
		payment_provider TEXT DEFAULT 'razorpay', razorpay_account_id TEXT DEFAULT '',
		stripe_account_id TEXT DEFAULT '', stripe_charges_enabled INTEGER DEFAULT 0,
		payout_country TEXT DEFAULT 'IN', created_at DATETIME, updated_at DATETIME
	)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE orders (
		id TEXT PRIMARY KEY, order_number TEXT, customer_id TEXT, chef_id TEXT, delivery_id TEXT,
		status TEXT DEFAULT 'pending', payment_status TEXT DEFAULT 'pending',
		payment_method TEXT DEFAULT '', payment_provider TEXT DEFAULT 'razorpay',
		subtotal REAL DEFAULT 0, tax REAL DEFAULT 0, total REAL DEFAULT 0,
		chef_tip REAL DEFAULT 0, driver_tip REAL DEFAULT 0, delivery_fee REAL DEFAULT 0,
		chef_funded_discount REAL DEFAULT 0, commission_rate REAL DEFAULT 0,
		delivery_address_state TEXT DEFAULT '',
		currency TEXT DEFAULT 'INR', razorpay_order_id TEXT DEFAULT '', razorpay_payment_id TEXT DEFAULT '',
		stripe_payment_intent_id TEXT DEFAULT '', refund_id TEXT DEFAULT '', refund_amount REAL DEFAULT 0,
		refund_reason TEXT DEFAULT '', refund_initiated_by TEXT DEFAULT '', refunded_at DATETIME,
		payout_hold_status TEXT DEFAULT '', wallet_applied REAL DEFAULT 0,
		created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
	)`).Error)
	// PlatformSettings backs services.GetCommissionRate — present so the
	// frozen-rate test can set a live rate that differs from the order's column.
	require.NoError(t, db.Exec(`CREATE TABLE platform_settings (
		id TEXT PRIMARY KEY DEFAULT '', key TEXT, value TEXT DEFAULT '',
		type TEXT DEFAULT 'string', updated_by TEXT, updated_at DATETIME
	)`).Error)
	// CreateOrderPayment preloads Delivery.DeliveryPartner — the tables must
	// exist (no rows in these tests, so the nested scan stays empty).
	require.NoError(t, db.Exec(`CREATE TABLE deliveries (
		id TEXT PRIMARY KEY, order_id TEXT, delivery_partner_id TEXT, status TEXT DEFAULT 'pending',
		created_at DATETIME, updated_at DATETIME
	)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE delivery_partners (
		id TEXT PRIMARY KEY, user_id TEXT, razorpay_account_id TEXT DEFAULT '',
		created_at DATETIME, updated_at DATETIME
	)`).Error)
	// #394: InitiateRefund now checks whether the order is refund-managed by a typed
	// escrow flow (meal-plan day / group order). The COUNT probes need the tables to
	// exist or every refund would 500; empty in most tests → the guard passes through.
	// order_items — RemainingRefundable (#560, InitiateRefund) sums cancelled lines; without
	// the table its raw query errors and silently falls back to the old formula.
	require.NoError(t, db.Exec(`CREATE TABLE order_items (id TEXT PRIMARY KEY, order_id TEXT,
		menu_item_id TEXT, quantity INTEGER DEFAULT 0, subtotal REAL DEFAULT 0, is_cancelled BOOLEAN DEFAULT 0, refund_amount REAL DEFAULT 0, created_at DATETIME)`).Error)
	// order_issues — claimOrderItemForCancel (#622) checks whether a resolved customer issue
	// already refunded the target line; the table must exist or the per-line cancel errors.
	require.NoError(t, db.Exec(`CREATE TABLE order_issues (id TEXT PRIMARY KEY, order_id TEXT, chef_id TEXT,
		customer_id TEXT, reason TEXT, affected_item_ids TEXT, requested_amount REAL DEFAULT 0,
		refund_amount REAL DEFAULT 0, status TEXT DEFAULT 'pending', created_at DATETIME, updated_at DATETIME)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE meal_plan_days (id TEXT PRIMARY KEY, order_id TEXT)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE group_orders (id TEXT PRIMARY KEY, order_id TEXT)`).Error)
	// #395: the completion helper stages chef.new_order + order.paid via the outbox.
	require.NoError(t, db.Exec(`CREATE TABLE outbox_events (id TEXT PRIMARY KEY, subject TEXT, msg_id TEXT,
		aggregate_type TEXT, aggregate_id TEXT, payload TEXT, status TEXT, attempts INT, last_error TEXT,
		next_retry_at DATETIME, created_at DATETIME, updated_at DATETIME, published_at DATETIME)`).Error)
	// processed_events (the wallet top-up + webhook dedup ledger) is added on demand by
	// tests that need it via addProcessedEventsTable, to avoid a double-create collision.

	prev := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = prev })
	return db
}

func payUser(t *testing.T, db *gorm.DB, role string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	require.NoError(t, db.Exec(
		`INSERT INTO users (id, email, role, is_active, created_at, updated_at) VALUES (?, ?, ?, 1, ?, ?)`,
		id.String(), id.String()+"@t.com", role, time.Now(), time.Now()).Error)
	return id
}

func payChef(t *testing.T, db *gorm.DB, userID uuid.UUID) uuid.UUID {
	t.Helper()
	id := uuid.New()
	require.NoError(t, db.Exec(
		`INSERT INTO chef_profiles (id, user_id, business_name, payment_provider, payout_country, created_at, updated_at)
		 VALUES (?, ?, 'Test Kitchen', 'razorpay', 'IN', ?, ?)`,
		id.String(), userID.String(), time.Now(), time.Now()).Error)
	return id
}

// payOrder inserts an order. paymentStatus drives the precondition branches.
func payOrder(t *testing.T, db *gorm.DB, customerID, chefID uuid.UUID, paymentStatus string, total float64, rzOrderID, rzPaymentID string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	require.NoError(t, db.Exec(
		`INSERT INTO orders (id, order_number, customer_id, chef_id, status, payment_status, payment_provider, subtotal, tax, total, currency, razorpay_order_id, razorpay_payment_id, created_at, updated_at)
		 VALUES (?, ?, ?, ?, 'pending', ?, 'razorpay', ?, ?, ?, 'INR', ?, ?, ?, ?)`,
		id.String(), "HC-"+id.String()[:8], customerID.String(), chefID.String(), paymentStatus,
		total*0.9, total*0.1, total, rzOrderID, rzPaymentID, time.Now(), time.Now()).Error)
	return id
}

func callPay(userID uuid.UUID, method, path string, register func(*gin.Engine, *PaymentHandler), body any) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("userID", userID); c.Next() })
	register(r, NewPaymentHandler())

	var reader *bytes.Reader
	if body != nil {
		buf, _ := json.Marshal(body)
		reader = bytes.NewReader(buf)
	} else {
		reader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func regCreate(r *gin.Engine, h *PaymentHandler) {
	r.POST("/payments/order/:orderId/create", h.CreateOrderPayment)
}
func regVerify(r *gin.Engine, h *PaymentHandler) {
	r.POST("/payments/order/:orderId/verify", h.VerifyPayment)
}
func regRefund(r *gin.Engine, h *PaymentHandler) {
	r.POST("/payments/order/:orderId/refund", h.InitiateRefund)
}

// ── CreateOrderPayment ───────────────────────────────────────────────────────

func TestCreateOrderPayment_InvalidOrderID_400(t *testing.T) {
	setupPayDB(t)
	w := callPay(uuid.New(), http.MethodPost, "/payments/order/not-a-uuid/create", regCreate, nil)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d (%s)", w.Code, w.Body.String())
	}
}

func TestCreateOrderPayment_OtherCustomersOrder_404_IDOR(t *testing.T) {
	db := setupPayDB(t)
	owner := payUser(t, db, "customer")
	attacker := payUser(t, db, "customer")
	chefUser := payUser(t, db, "chef")
	chef := payChef(t, db, chefUser)
	orderID := payOrder(t, db, owner, chef, "pending", 500, "", "")

	// Attacker (different customer) must not see/pay another customer's order.
	w := callPay(attacker, http.MethodPost, "/payments/order/"+orderID.String()+"/create", regCreate, nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("want 404 for IDOR, got %d (%s)", w.Code, w.Body.String())
	}
}

func TestCreateOrderPayment_AlreadyPaid_400(t *testing.T) {
	db := setupPayDB(t)
	cust := payUser(t, db, "customer")
	chef := payChef(t, db, payUser(t, db, "chef"))
	orderID := payOrder(t, db, cust, chef, "completed", 500, "rzp_order_x", "pay_x")

	w := callPay(cust, http.MethodPost, "/payments/order/"+orderID.String()+"/create", regCreate, nil)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400 already-paid, got %d (%s)", w.Code, w.Body.String())
	}
}

// Note: the "gateway unconfigured → 503" path isn't unit-testable here —
// services.GetRazorpay() lazily resolves credentials via GCP Secret Manager,
// which isn't initialised in the test process. In production InitRazorpay() runs
// at startup. That branch is covered by the live-switch manual step (#25).

// ── VerifyPayment ────────────────────────────────────────────────────────────

func TestVerifyPayment_OrderNotFound_404(t *testing.T) {
	db := setupPayDB(t)
	cust := payUser(t, db, "customer")
	w := callPay(cust, http.MethodPost, "/payments/order/"+uuid.New().String()+"/verify", regVerify,
		map[string]string{"razorpayPaymentId": "pay_1", "razorpayOrderId": "ord_1", "razorpaySignature": "sig"})
	if w.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d (%s)", w.Code, w.Body.String())
	}
}

func TestVerifyPayment_MissingFields_400(t *testing.T) {
	db := setupPayDB(t)
	cust := payUser(t, db, "customer")
	chef := payChef(t, db, payUser(t, db, "chef"))
	orderID := payOrder(t, db, cust, chef, "pending", 500, "rzp_order_x", "")

	// Empty razorpay fields → handler must reject (no silent pass).
	w := callPay(cust, http.MethodPost, "/payments/order/"+orderID.String()+"/verify", regVerify,
		map[string]string{})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400 missing-fields, got %d (%s)", w.Code, w.Body.String())
	}
}

func TestVerifyPayment_OrderIDMismatch_400(t *testing.T) {
	db := setupPayDB(t)
	cust := payUser(t, db, "customer")
	chef := payChef(t, db, payUser(t, db, "chef"))
	orderID := payOrder(t, db, cust, chef, "pending", 500, "rzp_order_REAL", "")

	// Client supplies a razorpay order id that doesn't match the stored one.
	w := callPay(cust, http.MethodPost, "/payments/order/"+orderID.String()+"/verify", regVerify,
		map[string]string{"razorpayPaymentId": "pay_1", "razorpayOrderId": "rzp_order_FORGED", "razorpaySignature": "sig"})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400 order-id mismatch, got %d (%s)", w.Code, w.Body.String())
	}
}

// ── InitiateRefund ───────────────────────────────────────────────────────────

func TestInitiateRefund_Unauthorized_403(t *testing.T) {
	db := setupPayDB(t)
	cust := payUser(t, db, "customer")
	chef := payChef(t, db, payUser(t, db, "chef"))
	orderID := payOrder(t, db, cust, chef, "completed", 500, "rzp_order_x", "pay_x")

	// A random customer (not the chef, not admin) cannot refund.
	stranger := payUser(t, db, "customer")
	w := callPay(stranger, http.MethodPost, "/payments/order/"+orderID.String()+"/refund", regRefund,
		map[string]any{"reason": "changed mind"})
	if w.Code != http.StatusForbidden {
		t.Fatalf("want 403, got %d (%s)", w.Code, w.Body.String())
	}
}

func TestInitiateRefund_NotCompleted_400(t *testing.T) {
	db := setupPayDB(t)
	cust := payUser(t, db, "customer")
	chefUser := payUser(t, db, "chef")
	chef := payChef(t, db, chefUser)
	orderID := payOrder(t, db, cust, chef, "pending", 500, "", "") // not completed

	w := callPay(chefUser, http.MethodPost, "/payments/order/"+orderID.String()+"/refund", regRefund,
		map[string]any{"reason": "test"})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400 not-completed, got %d (%s)", w.Code, w.Body.String())
	}
}

func TestInitiateRefund_ExceedsTotal_400(t *testing.T) {
	db := setupPayDB(t)
	cust := payUser(t, db, "customer")
	chefUser := payUser(t, db, "chef")
	chef := payChef(t, db, chefUser)
	orderID := payOrder(t, db, cust, chef, "completed", 500, "rzp_order_x", "pay_x")

	w := callPay(chefUser, http.MethodPost, "/payments/order/"+orderID.String()+"/refund", regRefund,
		map[string]any{"reason": "test", "amount": 999.0})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400 exceeds-total, got %d (%s)", w.Code, w.Body.String())
	}
}

func TestInitiateRefund_AdminAllowed_NoRazorpayPayment_400(t *testing.T) {
	db := setupPayDB(t)
	cust := payUser(t, db, "customer")
	chef := payChef(t, db, payUser(t, db, "chef"))
	// Completed order but no razorpay_payment_id recorded.
	orderID := payOrder(t, db, cust, chef, "completed", 500, "rzp_order_x", "")
	admin := payUser(t, db, "admin")

	// Admin passes authz + completed + amount-ok, then fails on the missing
	// gateway payment id — proving the admin authorization path is reachable.
	w := callPay(admin, http.MethodPost, "/payments/order/"+orderID.String()+"/refund", regRefund,
		map[string]any{"reason": "ops refund"})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400 no-razorpay-payment (admin authorized past 403), got %d (%s)", w.Code, w.Body.String())
	}
}

// Chef NET payout (#390): the chef's Route transfer is now net of commission and
// TDS (gross = food + tax + chef tip, less any chef-funded promo they bear).
// chefNetPayout reads the FROZEN order.CommissionRate — no rate argument — so the
// transfer can never drift from the settlement statement computed on the same row.
func TestChefNetPayout(t *testing.T) {
	// 1000 food + 50 tax + 20 tip @ 6%: gross 1070, commission 60, tds 10.70,
	// net = 1070 - 60 - 10.70 = 999.30.
	base := &models.Order{Subtotal: 1000, Tax: 50, ChefTip: 20, CommissionRate: 0.06}
	if got := chefNetPayout(base); got != 999.3 {
		t.Fatalf("chef net payout = %.2f, want 999.30", got)
	}
	// Chef-funded promo (#39): itemRevenue 900, gross 970, commission 54, tds 9.70,
	// net = 970 - 54 - 9.70 = 906.30.
	chefFunded := &models.Order{Subtotal: 1000, Tax: 50, ChefTip: 20, ChefFundedDiscount: 100, CommissionRate: 0.06}
	if got := chefNetPayout(chefFunded); got != 906.3 {
		t.Fatalf("chef-funded net payout = %.2f, want 906.30", got)
	}
	// Never negative even if a discount somehow exceeds the food revenue.
	huge := &models.Order{Subtotal: 100, ChefFundedDiscount: 500, CommissionRate: 0.06}
	if got := chefNetPayout(huge); got != 0 {
		t.Fatalf("over-discounted net payout = %.2f, want 0 (floored)", got)
	}
}

func TestOrderSettlements_ChefNetTransfer(t *testing.T) {
	order := &models.Order{
		OrderNumber:        "HC-1",
		Subtotal:           1000,
		Tax:                50,
		ChefTip:            20,
		DeliveryFee:        40,
		ChefFundedDiscount: 100,
		CommissionRate:     0.06,
	}
	order.Chef.RazorpayAccountID = "acc_chef"
	settlements := orderSettlements(order)
	// Chef settlement = NET: itemRevenue 900, gross 970, commission 54, tds 9.70,
	// net = 906.30 → 90630 paise.
	if settlements[0].Amount != services.ToPaise(906.3) {
		t.Fatalf("chef settlement = %d paise, want %d (net)", settlements[0].Amount, services.ToPaise(906.3))
	}
	// Driver settlement (deliveryFee + driverTip) is UNCHANGED — the driver's money.
	if settlements[1].Amount != services.ToPaise(40) {
		t.Fatalf("driver settlement = %d paise, want %d", settlements[1].Amount, services.ToPaise(40))
	}
}

// TestChefTransferEqualsStatementNetPayout (#390, W3) is the non-tautological
// guard against field-mapping drift between the two construction sites: the
// payout side (payment.go → chefNetPayout) and the statement side (statement.go →
// scanned statementOrderRow). It builds ONE order, maps the payout-side input the
// way chefNetPayout does, and the statement-side input the way statement.go maps a
// scanned row (subtotal→ItemRevenue, tax→Tax, delivery_fee→DeliveryFee,
// chef_tip→ChefTip, chef_funded_discount→ChefFundedDiscount,
// delivery_address_state→DeliveryState, commission_rate→CommissionRate), then
// asserts payout NetPayout == statement NetPayout == chefNetPayout(order). A
// mistake like reading Total instead of Subtotal on one side would break this.
// TODO(#390): promote to a live SQL round-trip once a handler test DB is wired.
func TestChefTransferEqualsStatementNetPayout(t *testing.T) {
	order := &models.Order{
		OrderNumber:          "HC-EQ",
		Subtotal:             1000,
		Tax:                  50,
		ChefTip:              20,
		DeliveryFee:          70,
		DriverTip:            0,
		ChefFundedDiscount:   0,
		DeliveryAddressState: "Maharashtra",
		CommissionRate:       0.06,
	}
	order.Chef.State = "Maharashtra"

	// Payout-side input: exactly what chefNetPayout(order) maps.
	payoutInput := services.EarningsInput{
		ItemRevenue:        order.Subtotal,
		Tax:                order.Tax,
		ChefTip:            order.ChefTip,
		DeliveryFee:        order.DeliveryFee,
		ChefFundedDiscount: order.ChefFundedDiscount,
		DeliveryState:      order.DeliveryAddressState,
		CommissionRate:     order.CommissionRate,
	}
	// Statement-side input: exactly what statement.go maps from a scanned
	// statementOrderRow for the SAME order (columns spelled out independently).
	statementInput := services.EarningsInput{
		ItemRevenue:        order.Subtotal,             // o.subtotal
		Tax:                order.Tax,                  // o.tax
		ChefTip:            order.ChefTip,              // o.chef_tip
		DeliveryFee:        order.DeliveryFee,          // o.delivery_fee
		ChefFundedDiscount: order.ChefFundedDiscount,   // o.chef_funded_discount
		DeliveryState:      order.DeliveryAddressState, // o.delivery_address_state
		CommissionRate:     order.CommissionRate,       // o.commission_rate
	}

	payoutNet := services.ComputeOrderEarnings(payoutInput, order.Chef.State).NetPayout
	statementNet := services.ComputeOrderEarnings(statementInput, order.Chef.State).NetPayout
	if payoutNet != statementNet {
		t.Fatalf("payout net %.2f != statement net %.2f — field mapping drifted", payoutNet, statementNet)
	}
	if got := chefNetPayout(order); got != payoutNet {
		t.Fatalf("chefNetPayout %.2f != computed net %.2f", got, payoutNet)
	}
	if payoutNet != 999.3 {
		t.Fatalf("net = %.2f, want 999.30", payoutNet)
	}
}

// TestConservationExcludesGSTOnCommission (#390, B1) locks the money-conservation
// identity against the CAPTURED order total, with GST-on-commission EXCLUDED. GST
// on the platform's commission is a downstream remittance obligation on the
// platform's own revenue — it is NOT money the customer paid, so it is absent from
// order.Total and must NOT appear in the identity (mirrors earnings_test.go which
// asserts commission + tds + net == gross with GST excluded).
func TestConservationExcludesGSTOnCommission(t *testing.T) {
	// 1000 food + 50 tax + 70 delivery + 20 chef tip, intra-state, 6%, no service
	// fee, no refund. Customer pays 1000 + 50 + 20 + 70 = 1140.
	order := &models.Order{
		Subtotal:             1000,
		Tax:                  50,
		DeliveryFee:          70,
		ChefTip:              20,
		DriverTip:            0,
		ServiceFee:           0,
		ChefFundedDiscount:   0,
		DeliveryAddressState: "Maharashtra",
		CommissionRate:       0.06,
		Total:                1140,
	}
	order.Chef.State = "Maharashtra"

	e := services.ComputeOrderEarnings(services.EarningsInput{
		ItemRevenue:        order.Subtotal,
		Tax:                order.Tax,
		ChefTip:            order.ChefTip,
		DeliveryFee:        order.DeliveryFee,
		ChefFundedDiscount: order.ChefFundedDiscount,
		DeliveryState:      order.DeliveryAddressState,
		CommissionRate:     order.CommissionRate,
	}, order.Chef.State)

	// Platform retains commission + TDS + serviceFee — NOT the GST on commission.
	platformRetained := e.PlatformCommission + e.TDS + order.ServiceFee
	driver := order.DeliveryFee + order.DriverTip
	const refunds = 0.0

	sum := services.Round2(e.NetPayout + platformRetained + driver + refunds)
	if sum != order.Total {
		t.Fatalf("conservation broken: net %.2f + retained %.2f + driver %.2f + refunds %.2f = %.2f, want total %.2f",
			e.NetPayout, platformRetained, driver, refunds, sum, order.Total)
	}

	// The chef's actual transfer equals the statement's NetPayout.
	if got := chefNetPayout(order); got != e.NetPayout {
		t.Fatalf("chefNetPayout %.2f != statement net %.2f", got, e.NetPayout)
	}

	// GST-on-commission is genuinely OUTSIDE the captured total: folding it into
	// the identity must OVERSHOOT order.Total (proving it is not customer money).
	gstOnCommission := e.CGST + e.SGST + e.IGST
	if gstOnCommission <= 0 {
		t.Fatalf("expected a non-zero GST-on-commission for this order, got %.2f", gstOnCommission)
	}
	if services.Round2(sum+gstOnCommission) == order.Total {
		t.Fatalf("GST-on-commission (%.2f) must NOT be part of order.Total — it is a downstream carve-out", gstOnCommission)
	}
}

// TestFrozenRateSurvivesRetune (#390, B2) proves chefNetPayout uses the rate
// FROZEN on the order, not the live runtime setting. Even if an admin retunes
// GetCommissionRate after checkout, an order stamped with CommissionRate 0.06
// still settles at 999.30 — the equality with the already-sent transfer holds.
func TestFrozenRateSurvivesRetune(t *testing.T) {
	db := setupPayDB(t)
	// Live setting says 12%, but the order froze 6% at checkout.
	require.NoError(t, db.Exec(
		`INSERT INTO platform_settings (key, value) VALUES ('payout.commission_rate', '0.12')`).Error)
	if live := services.GetCommissionRate(db); live != 0.12 {
		t.Fatalf("live rate = %.2f, want 0.12 (precondition)", live)
	}

	order := &models.Order{Subtotal: 1000, Tax: 50, ChefTip: 20, CommissionRate: 0.06}
	// chefNetPayout must read the frozen 6%, not the live 12% → 999.30 (not 939.30).
	if got := chefNetPayout(order); got != 999.3 {
		t.Fatalf("net = %.2f, want 999.30 (frozen 6%%, not live 12%%)", got)
	}
}
