package services

// meal_plan_verify_advance_test.go — #200 (tiffin E2E). Covers VerifyMealPlanAdvance, the
// meal-plan payment-capture entry point (previously zero coverage). It is the anti-under-payment
// gate: it binds the fetched gateway payment to THIS plan's advance order + amount and verifies the
// Checkout signature before stamping EscrowPaymentID — without which a ₹1 payment reused across
// plans could mark a large plan "paid" out of the platform escrow. Mirrors the order/tip/group
// verify (#395·4). Drives FetchPayment + VerifyPaymentSignature against the shared httptest seam.

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

func setupAdvanceDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TABLE meal_plans (id TEXT PRIMARY KEY, meal_plan_number TEXT,
		razorpay_order_id TEXT, total REAL, escrow_payment_id TEXT, created_at DATETIME, updated_at DATETIME)`).Error)
	prev := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = prev })
	return db
}

// seedAdvancePlan inserts a plan row (escrow_payment_id blank) and returns the matching struct the
// verify acts on (the handler passes a freshly-loaded plan; the row lets us read the stamp back).
func seedAdvancePlan(t *testing.T, db *gorm.DB, rzOrderID string, total float64) models.MealPlan {
	t.Helper()
	id := uuid.New()
	require.NoError(t, db.Exec(
		`INSERT INTO meal_plans (id, meal_plan_number, razorpay_order_id, total, escrow_payment_id)
		 VALUES (?,?,?,?,?)`,
		id.String(), "MP-"+id.String()[:8], rzOrderID, total, "").Error)
	return models.MealPlan{ID: id, RazorpayOrderID: rzOrderID, Total: total}
}

// cannedAdvancePayment stubs GET /payments/{id} with the given captured status, order binding and
// amount (paise), and points GetRazorpay at it (keySecret "secret" — see advanceSignature).
func cannedAdvancePayment(t *testing.T, status, orderID string, amountPaise int) {
	t.Helper()
	withRazorpayTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/payments/") {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id": "pay_adv", "status": status, "captured": status == "captured",
				"order_id": orderID, "amount": amountPaise,
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
}

// advanceSignature computes the Checkout HMAC the way VerifyPaymentSignature does, keyed on the
// test client's secret ("secret", set by withRazorpayTestServer).
func advanceSignature(orderID, paymentID string) string {
	mac := hmac.New(sha256.New, []byte("secret"))
	mac.Write([]byte(orderID + "|" + paymentID))
	return hex.EncodeToString(mac.Sum(nil))
}

func escrowPaymentIDOf(t *testing.T, db *gorm.DB, id uuid.UUID) string {
	t.Helper()
	var s string
	require.NoError(t, db.Raw(`SELECT escrow_payment_id FROM meal_plans WHERE id = ?`, id.String()).Scan(&s).Error)
	return s
}

// Escrow OFF → pure no-op: no gateway call, no stamp.
func TestVerifyMealPlanAdvance_EscrowOff_NoOp(t *testing.T) {
	escrowFlag(t, false)
	db := setupAdvanceDB(t)
	plan := seedAdvancePlan(t, db, "order_adv1", 240)
	require.NoError(t, VerifyMealPlanAdvance(db, &plan, "pay_adv", ""))
	require.Empty(t, escrowPaymentIDOf(t, db, plan.ID), "escrow off → not stamped")
}

// Captured payment bound to the plan's order for the full amount → stamps EscrowPaymentID.
func TestVerifyMealPlanAdvance_HappyPath_Stamps(t *testing.T) {
	escrowFlag(t, true)
	db := setupAdvanceDB(t)
	plan := seedAdvancePlan(t, db, "order_adv1", 240)
	cannedAdvancePayment(t, "captured", "order_adv1", 24000) // 240.00 → 24000 paise, exact
	require.NoError(t, VerifyMealPlanAdvance(db, &plan, "pay_adv", ""))
	require.Equal(t, "pay_adv", escrowPaymentIDOf(t, db, plan.ID), "persisted")
	require.Equal(t, "pay_adv", plan.EscrowPaymentID, "struct updated")
}

// No advance order on the plan → reject before any gateway trust.
func TestVerifyMealPlanAdvance_NoAdvanceOrder_Errors(t *testing.T) {
	escrowFlag(t, true)
	db := setupAdvanceDB(t)
	plan := seedAdvancePlan(t, db, "", 240)
	cannedAdvancePayment(t, "captured", "order_adv1", 24000)
	require.ErrorContains(t, VerifyMealPlanAdvance(db, &plan, "pay_adv", ""), "no advance order")
}

// Payment not captured (e.g. only authorized) → reject.
func TestVerifyMealPlanAdvance_NotCaptured_Errors(t *testing.T) {
	escrowFlag(t, true)
	db := setupAdvanceDB(t)
	plan := seedAdvancePlan(t, db, "order_adv1", 240)
	cannedAdvancePayment(t, "authorized", "order_adv1", 24000)
	require.ErrorContains(t, VerifyMealPlanAdvance(db, &plan, "pay_adv", ""), "not captured")
	require.Empty(t, escrowPaymentIDOf(t, db, plan.ID))
}

// Payment belongs to a DIFFERENT gateway order → reject (cross-plan reuse guard).
func TestVerifyMealPlanAdvance_OrderMismatch_Errors(t *testing.T) {
	escrowFlag(t, true)
	db := setupAdvanceDB(t)
	plan := seedAdvancePlan(t, db, "order_adv1", 240)
	cannedAdvancePayment(t, "captured", "order_OTHER", 24000)
	require.ErrorContains(t, VerifyMealPlanAdvance(db, &plan, "pay_adv", ""), "does not belong")
	require.Empty(t, escrowPaymentIDOf(t, db, plan.ID))
}

// Captured amount below the plan total → reject (the anti-under-payment gate: a ₹1 short here
// must never mark the plan paid out of platform escrow).
func TestVerifyMealPlanAdvance_AmountTooLow_Errors(t *testing.T) {
	escrowFlag(t, true)
	db := setupAdvanceDB(t)
	plan := seedAdvancePlan(t, db, "order_adv1", 240)
	cannedAdvancePayment(t, "captured", "order_adv1", 23999) // 1 paise short of 24000
	require.ErrorContains(t, VerifyMealPlanAdvance(db, &plan, "pay_adv", ""), "amount does not match")
	require.Empty(t, escrowPaymentIDOf(t, db, plan.ID))
}

// A present-but-wrong signature → reject (the client always sends it; when present it is enforced).
func TestVerifyMealPlanAdvance_BadSignature_Errors(t *testing.T) {
	escrowFlag(t, true)
	db := setupAdvanceDB(t)
	plan := seedAdvancePlan(t, db, "order_adv1", 240)
	cannedAdvancePayment(t, "captured", "order_adv1", 24000)
	require.ErrorContains(t, VerifyMealPlanAdvance(db, &plan, "pay_adv", "deadbeef"), "signature verification failed")
	require.Empty(t, escrowPaymentIDOf(t, db, plan.ID))
}

// A correctly-computed signature passes the gate and stamps.
func TestVerifyMealPlanAdvance_ValidSignature_Stamps(t *testing.T) {
	escrowFlag(t, true)
	db := setupAdvanceDB(t)
	plan := seedAdvancePlan(t, db, "order_adv1", 240)
	cannedAdvancePayment(t, "captured", "order_adv1", 24000)
	require.NoError(t, VerifyMealPlanAdvance(db, &plan, "pay_adv", advanceSignature("order_adv1", "pay_adv")))
	require.Equal(t, "pay_adv", escrowPaymentIDOf(t, db, plan.ID))
}
