package services

// meal_plan_confirm_advance_test.go — #395·3 durability. ConfirmMealPlanAdvance is the
// SHARED confirm seam behind BOTH the client verify-payment path AND the
// payment.captured webhook fallback. The webhook fallback exists because a dropped
// client verify (the RN Razorpay SDK returning dismiss on the success auto-redirect)
// would otherwise strand a CAPTURED advance — money taken, plan left awaiting_customer,
// chef payout never held. These tests drive the webhook path (signature "") end to end:
// validate → confirm plan + days → stamp EscrowPaymentID → hold one transfer per
// accepted day → emit the confirmed event, and prove it is idempotent + escrow-gated.

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

func setupConfirmAdvanceDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	require.NoError(t, err)
	for _, s := range []string{
		`CREATE TABLE meal_plans (id TEXT PRIMARY KEY, meal_plan_number TEXT, customer_id TEXT, chef_id TEXT,
			status TEXT, razorpay_order_id TEXT, escrow_payment_id TEXT, subtotal REAL, tax REAL, total REAL,
			currency TEXT, confirmed_at DATETIME, created_at DATETIME, updated_at DATETIME)`,
		`CREATE TABLE meal_plan_days (id TEXT PRIMARY KEY, meal_plan_id TEXT, status TEXT,
			payout_transfer_id TEXT DEFAULT '', commission_rate REAL DEFAULT 0, price REAL,
			created_at DATETIME, updated_at DATETIME)`,
		`CREATE TABLE chef_profiles (id TEXT PRIMARY KEY, user_id TEXT, razorpay_account_id TEXT DEFAULT '',
			created_at DATETIME, updated_at DATETIME)`,
		`CREATE TABLE platform_settings (key TEXT PRIMARY KEY, value TEXT)`,
		`CREATE TABLE audit_logs (id TEXT PRIMARY KEY, user_id TEXT, action TEXT, entity_type TEXT,
			entity_id TEXT, old_value TEXT, new_value TEXT, ip_address TEXT, user_agent TEXT,
			correlation_id TEXT, created_at DATETIME)`,
		`CREATE TABLE outbox_events (id TEXT PRIMARY KEY, subject TEXT, msg_id TEXT, aggregate_type TEXT,
			aggregate_id TEXT, payload TEXT, status TEXT, attempts INT, last_error TEXT,
			next_retry_at DATETIME, created_at DATETIME, updated_at DATETIME, published_at DATETIME)`,
	} {
		require.NoError(t, db.Exec(s).Error)
	}
	prev := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = prev })
	return db
}

// confirmAdvanceStub serves BOTH GET /payments/{id} (captured, bound to orderID/amount —
// for VerifyMealPlanAdvance) AND POST /transfers (on-hold — for HoldChefPayouts),
// counting transfer creates so a test can assert one hold per accepted day.
func confirmAdvanceStub(t *testing.T, orderID string, amountPaise int, creates *int32) {
	t.Helper()
	withRazorpayTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/payments/"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id": "pay_adv", "status": "captured", "captured": true,
				"order_id": orderID, "amount": amountPaise,
			})
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/transfers"):
			n := atomic.AddInt32(creates, 1)
			_ = json.NewEncoder(w).Encode(map[string]any{"id": fmt.Sprintf("trf_hold_%d", n), "on_hold": true})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})
}

// seedConfirmPlan inserts an awaiting_customer plan with an advance order + N accepted
// days, plus the chef, and returns the in-memory struct ConfirmMealPlanAdvance acts on.
func seedConfirmPlan(t *testing.T, db *gorm.DB, rzOrderID string, dayPrices []float64) (*models.MealPlan, uuid.UUID, []uuid.UUID) {
	t.Helper()
	planID, chefID, custID := uuid.New(), uuid.New(), uuid.New()
	chefUserID := uuid.New()
	total := 0.0
	for _, p := range dayPrices {
		total += p
	}
	total += 32 // GST + delivery, > sum of day prices (matches the real snapshot)
	require.NoError(t, db.Exec(`INSERT INTO meal_plans
		(id, meal_plan_number, customer_id, chef_id, status, razorpay_order_id, escrow_payment_id, subtotal, tax, total, currency)
		VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		planID.String(), "MP-conf1", custID.String(), chefID.String(), string(models.MealPlanAwaitingCustomer),
		rzOrderID, "", total-32, 25.6, total, "INR").Error)
	require.NoError(t, db.Exec(`INSERT INTO chef_profiles (id, user_id, razorpay_account_id) VALUES (?,?,?)`,
		chefID.String(), chefUserID.String(), "acc_chef_conf").Error)
	days := make([]models.MealPlanDay, 0, len(dayPrices))
	dayIDs := make([]uuid.UUID, 0, len(dayPrices))
	for _, p := range dayPrices {
		dayID := uuid.New()
		require.NoError(t, db.Exec(`INSERT INTO meal_plan_days (id, meal_plan_id, status, price) VALUES (?,?,?,?)`,
			dayID.String(), planID.String(), string(models.MealPlanDayAccepted), p).Error)
		days = append(days, models.MealPlanDay{ID: dayID, MealPlanID: planID, Status: models.MealPlanDayAccepted, Price: p})
		dayIDs = append(dayIDs, dayID)
	}
	plan := &models.MealPlan{
		ID: planID, MealPlanNumber: "MP-conf1", CustomerID: custID, ChefID: chefID,
		Status: models.MealPlanAwaitingCustomer, RazorpayOrderID: rzOrderID, Total: total, Days: days,
	}
	return plan, planID, dayIDs
}

func planField(t *testing.T, db *gorm.DB, id uuid.UUID, col string) string {
	t.Helper()
	var s string
	require.NoError(t, db.Raw(`SELECT COALESCE(CAST(`+col+` AS TEXT),'') FROM meal_plans WHERE id = ?`, id.String()).Scan(&s).Error)
	return s
}

func confDayStatus(t *testing.T, db *gorm.DB, id uuid.UUID) string {
	t.Helper()
	var s string
	require.NoError(t, db.Raw(`SELECT status FROM meal_plan_days WHERE id = ?`, id.String()).Scan(&s).Error)
	return s
}

func outboxCount(t *testing.T, db *gorm.DB, subject string) int {
	t.Helper()
	var n int64
	require.NoError(t, db.Raw(`SELECT COUNT(*) FROM outbox_events WHERE subject = ?`, subject).Scan(&n).Error)
	return int(n)
}

// The webhook path (signature "") confirms a captured advance end to end: plan + days →
// confirmed, EscrowPaymentID stamped, one hold transfer per accepted day, event emitted.
func TestConfirmMealPlanAdvance_WebhookPath_ConfirmsHoldsStamps(t *testing.T) {
	escrowFlag(t, true)
	db := setupConfirmAdvanceDB(t)
	var creates int32
	confirmAdvanceStub(t, "order_conf1", 35200, &creates) // total 352.00 → 35200 paise
	plan, planID, dayIDs := seedConfirmPlan(t, db, "order_conf1", []float64{160, 160})

	confirmed, err := ConfirmMealPlanAdvance(db, plan, "pay_adv", "") // webhook path: no signature
	require.NoError(t, err)
	require.True(t, confirmed, "the awaiting_customer → confirmed transition happened")

	require.Equal(t, string(models.MealPlanConfirmed), planField(t, db, planID, "status"), "plan confirmed")
	require.Equal(t, "pay_adv", planField(t, db, planID, "escrow_payment_id"), "captured payment stamped")
	require.NotEmpty(t, planField(t, db, planID, "confirmed_at"), "confirmed_at set")
	for _, d := range dayIDs {
		require.Equal(t, string(models.MealPlanDayConfirmed), confDayStatus(t, db, d), "accepted day → confirmed")
	}
	require.Equal(t, int32(2), atomic.LoadInt32(&creates), "one hold transfer per accepted day")
	require.Equal(t, 1, outboxCount(t, db, SubjectMealPlanConfirmed), "confirmed event emitted once")
}

// A second call (client verify racing the webhook, or a webhook re-delivery) is a no-op:
// no re-confirm, no second hold transfer, returns confirmed=false.
func TestConfirmMealPlanAdvance_Idempotent_NoDoubleHold(t *testing.T) {
	escrowFlag(t, true)
	db := setupConfirmAdvanceDB(t)
	var creates int32
	confirmAdvanceStub(t, "order_conf1", 35200, &creates)
	plan, planID, _ := seedConfirmPlan(t, db, "order_conf1", []float64{160, 160})

	c1, err := ConfirmMealPlanAdvance(db, plan, "pay_adv", "")
	require.NoError(t, err)
	require.True(t, c1)
	require.Equal(t, int32(2), atomic.LoadInt32(&creates))

	// Re-run against a fresh in-memory struct (as a second delivery would load).
	plan2, _, _ := seedConfirmPlanFrom(t, db, planID)
	c2, err := ConfirmMealPlanAdvance(db, plan2, "pay_adv", "")
	require.NoError(t, err)
	require.False(t, c2, "already confirmed → no transition")
	require.Equal(t, int32(2), atomic.LoadInt32(&creates), "no second hold transfer")
	require.Equal(t, 1, outboxCount(t, db, SubjectMealPlanConfirmed), "event not re-emitted")
}

// seedConfirmPlanFrom rebuilds the in-memory struct from the current DB row (simulating
// a fresh load on a webhook re-delivery), so the idempotency guard is exercised against
// real persisted state rather than a stale struct.
func seedConfirmPlanFrom(t *testing.T, db *gorm.DB, planID uuid.UUID) (*models.MealPlan, uuid.UUID, []uuid.UUID) {
	t.Helper()
	var plan models.MealPlan
	require.NoError(t, db.Preload("Days").First(&plan, "id = ?", planID).Error)
	ids := make([]uuid.UUID, 0, len(plan.Days))
	for _, d := range plan.Days {
		ids = append(ids, d.ID)
	}
	return &plan, planID, ids
}

// Escrow OFF → pure no-op ack: nothing confirmed, no hold, no event.
func TestConfirmMealPlanAdvance_EscrowOff_NoOp(t *testing.T) {
	escrowFlag(t, false)
	db := setupConfirmAdvanceDB(t)
	var creates int32
	confirmAdvanceStub(t, "order_conf1", 35200, &creates)
	plan, planID, _ := seedConfirmPlan(t, db, "order_conf1", []float64{160, 160})

	confirmed, err := ConfirmMealPlanAdvance(db, plan, "pay_adv", "")
	require.NoError(t, err)
	require.False(t, confirmed)
	require.Equal(t, string(models.MealPlanAwaitingCustomer), planField(t, db, planID, "status"), "unchanged")
	require.Equal(t, int32(0), atomic.LoadInt32(&creates), "no hold when escrow off")
}

// A captured payment bound to a DIFFERENT gateway order must NOT confirm the plan — the
// anti-cross-plan-reuse gate in VerifyMealPlanAdvance propagates through and blocks it.
func TestConfirmMealPlanAdvance_OrderMismatch_NoConfirm(t *testing.T) {
	escrowFlag(t, true)
	db := setupConfirmAdvanceDB(t)
	var creates int32
	confirmAdvanceStub(t, "order_OTHER", 35200, &creates) // payment belongs to another order
	plan, planID, _ := seedConfirmPlan(t, db, "order_conf1", []float64{160, 160})

	confirmed, err := ConfirmMealPlanAdvance(db, plan, "pay_adv", "")
	require.Error(t, err)
	require.False(t, confirmed)
	require.Equal(t, string(models.MealPlanAwaitingCustomer), planField(t, db, planID, "status"), "not confirmed on mismatch")
	require.Equal(t, int32(0), atomic.LoadInt32(&creates))
}
