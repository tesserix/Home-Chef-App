package handlers

// meal_plan_orchestration_test.go — #200 (tiffin E2E). HTTP-level coverage of the meal-plan
// negotiation handshake that was previously untested end-to-end:
//   - RespondMealPlan (chef): accept-all → auto-confirm + confirm every day; cherry-pick →
//     awaiting_customer, accepted days accepted, declined days declined, charge basis = accepted.
//   - finalizeByCustomer (ApproveMealPlan / RejectMealPlan): approve → confirm accepted days (charge
//     basis = accepted); reject → cancel. Plus the state guards (wrong status, wrong owner).
//
// Escrow is OFF so RefundDeclinedDays / RefundUndeliveredDays / HoldChefPayouts are no-ops — this
// isolates the orchestration (status/day transitions, charge-basis recompute, event fan-out). The
// money mechanics of those calls are covered by the services-package tests (VerifyMealPlanAdvance,
// HoldChefPayouts, RefundDay atomic/stamp, the cutoff sweep).

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/homechef/api/config"
	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

func escrowOff(t *testing.T) {
	t.Helper()
	saved := config.AppConfig
	t.Cleanup(func() { config.AppConfig = saved })
	config.AppConfig = &config.Config{MealPlanEscrowEnabled: false}
}

func setupOrchestrationDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	for _, s := range []string{
		`CREATE TABLE meal_plans (id text PRIMARY KEY, meal_plan_number text, customer_id text, chef_id text,
			status text, subtotal real, tax real, total real, currency text, escrow_payment_id text,
			razorpay_order_id text, chef_respond_by datetime, customer_approve_by datetime, confirmed_at datetime,
			cancelled_at datetime, cancel_reason text, start_date datetime, end_date datetime,
			created_at datetime, updated_at datetime)`,
		`CREATE TABLE meal_plan_days (id text PRIMARY KEY, meal_plan_id text, status text, price real, date datetime,
			payout_transfer_id text, commission_rate real, payout_hold_status text, refund_txn_id text, order_id text,
			created_at datetime, updated_at datetime)`,
		`CREATE TABLE chef_profiles (address_line1_enc text DEFAULT '', address_line2_enc text DEFAULT '', id text PRIMARY KEY, user_id text, business_name text, profile_image text,
			razorpay_account_id text, payout_country text, is_active integer DEFAULT 1, deleted_at datetime)`,
		`CREATE TABLE users (email_enc text DEFAULT '', email_bidx text DEFAULT '', first_name_enc text DEFAULT '', last_name_enc text DEFAULT '', phone_enc text DEFAULT '', phone_bidx text DEFAULT '', id text PRIMARY KEY, first_name text, last_name text, email text, phone text, deleted_at datetime)`,
		`CREATE TABLE outbox_events (id text PRIMARY KEY, subject text, msg_id text, aggregate_type text,
			aggregate_id text, payload text, status text, attempts int, last_error text, next_retry_at datetime,
			created_at datetime, updated_at datetime, published_at datetime)`,
	} {
		require.NoError(t, db.Exec(s).Error)
	}
	orig := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = orig })
	return db
}

func seedOrchChef(t *testing.T, db *gorm.DB) (chefID, chefUserID uuid.UUID) {
	t.Helper()
	chefID, chefUserID = uuid.New(), uuid.New()
	require.NoError(t, db.Exec(
		`INSERT INTO chef_profiles (id, user_id, business_name, razorpay_account_id, is_active) VALUES (?,?,?,?,1)`,
		chefID.String(), chefUserID.String(), "Tiffin Kitchen", "acc_chef_1").Error)
	return chefID, chefUserID
}

func seedOrchPlan(t *testing.T, db *gorm.DB, status models.MealPlanStatus, customerID, chefID uuid.UUID) uuid.UUID {
	t.Helper()
	planID := uuid.New()
	require.NoError(t, db.Exec(
		`INSERT INTO meal_plans (id, meal_plan_number, customer_id, chef_id, status, total) VALUES (?,?,?,?,?,?)`,
		planID.String(), "MP-"+planID.String()[:8], customerID.String(), chefID.String(), string(status), 0.0).Error)
	return planID
}

func seedOrchDay(t *testing.T, db *gorm.DB, planID uuid.UUID, status models.MealPlanDayStatus, price float64) uuid.UUID {
	t.Helper()
	dayID := uuid.New()
	require.NoError(t, db.Exec(
		`INSERT INTO meal_plan_days (id, meal_plan_id, status, price) VALUES (?,?,?,?)`,
		dayID.String(), planID.String(), string(status), price).Error)
	return dayID
}

func respondReq(t *testing.T, chefUserID, planID uuid.UUID, body any) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("userID", chefUserID); c.Next() })
	r.POST("/chef/meal-plans/:id/respond", (&MealPlanHandler{}).RespondMealPlan)
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/chef/meal-plans/"+planID.String()+"/respond", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func finalizeReq(t *testing.T, customerID, planID uuid.UUID, action string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("userID", customerID); c.Next() })
	h := &MealPlanHandler{}
	r.PUT("/meal-plans/:id/approve", h.ApproveMealPlan)
	r.PUT("/meal-plans/:id/reject", h.RejectMealPlan)
	req := httptest.NewRequest(http.MethodPut, "/meal-plans/"+planID.String()+"/"+action, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func planField(t *testing.T, db *gorm.DB, planID uuid.UUID, col string) string {
	t.Helper()
	var s string
	require.NoError(t, db.Raw(`SELECT COALESCE(CAST(`+col+` AS TEXT), '') FROM meal_plans WHERE id = ?`, planID.String()).Scan(&s).Error)
	return s
}

func dayStatusOf(t *testing.T, db *gorm.DB, dayID uuid.UUID) string {
	t.Helper()
	var s string
	require.NoError(t, db.Raw(`SELECT status FROM meal_plan_days WHERE id = ?`, dayID.String()).Scan(&s).Error)
	return s
}

func planTotalOf(t *testing.T, db *gorm.DB, planID uuid.UUID) float64 {
	t.Helper()
	var f float64
	require.NoError(t, db.Raw(`SELECT total FROM meal_plans WHERE id = ?`, planID.String()).Scan(&f).Error)
	return f
}

func eventCountFor(t *testing.T, db *gorm.DB, userID uuid.UUID) int {
	t.Helper()
	var n int64
	require.NoError(t, db.Raw(
		`SELECT COUNT(*) FROM outbox_events WHERE aggregate_type = 'event' AND aggregate_id = ?`,
		userID.String()).Scan(&n).Error)
	return int(n)
}

// ── RespondMealPlan (chef) ──

// Payment-after-approval: accept-all does NOT auto-confirm — it goes to the
// customer for approval + payment. Days become accepted (not confirmed), the
// approval window is set, and there is no confirmed_at yet; charge basis = all
// accepted days.
func TestRespondMealPlan_AcceptAll_AwaitsCustomerApproval(t *testing.T) {
	escrowOff(t)
	db := setupOrchestrationDB(t)
	chefID, chefUserID := seedOrchChef(t, db)
	customerID := uuid.New()
	planID := seedOrchPlan(t, db, models.MealPlanPendingChef, customerID, chefID)
	d1 := seedOrchDay(t, db, planID, models.MealPlanDayRequested, 100)
	d2 := seedOrchDay(t, db, planID, models.MealPlanDayRequested, 150)

	w := respondReq(t, chefUserID, planID, map[string]any{"acceptAll": true})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Equal(t, string(models.MealPlanAwaitingCustomer), planField(t, db, planID, "status"))
	require.Equal(t, string(models.MealPlanDayAccepted), dayStatusOf(t, db, d1))
	require.Equal(t, string(models.MealPlanDayAccepted), dayStatusOf(t, db, d2))
	require.Equal(t, 250.0, planTotalOf(t, db, planID), "charge basis = all accepted days")
	require.Empty(t, planField(t, db, planID, "confirmed_at"), "not confirmed until the customer approves + pays")
	require.NotEmpty(t, planField(t, db, planID, "customer_approve_by"), "approval window is set")
	require.Equal(t, 1, eventCountFor(t, db, customerID), "customer notified of the response")
}

// Cherry-pick moves the plan to awaiting_customer; accepted stays accepted, the rest declined,
// charge basis excludes the declined day.
func TestRespondMealPlan_CherryPick_AwaitsCustomerAndDeclines(t *testing.T) {
	escrowOff(t)
	db := setupOrchestrationDB(t)
	chefID, chefUserID := seedOrchChef(t, db)
	customerID := uuid.New()
	planID := seedOrchPlan(t, db, models.MealPlanPendingChef, customerID, chefID)
	dAccept := seedOrchDay(t, db, planID, models.MealPlanDayRequested, 100)
	dDecline := seedOrchDay(t, db, planID, models.MealPlanDayRequested, 150)

	w := respondReq(t, chefUserID, planID, map[string]any{"acceptedDayIds": []string{dAccept.String()}})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Equal(t, string(models.MealPlanAwaitingCustomer), planField(t, db, planID, "status"))
	require.Equal(t, string(models.MealPlanDayAccepted), dayStatusOf(t, db, dAccept))
	require.Equal(t, string(models.MealPlanDayDeclined), dayStatusOf(t, db, dDecline))
	require.Equal(t, 100.0, planTotalOf(t, db, planID), "declined day excluded from the charge basis")
	require.NotEmpty(t, planField(t, db, planID, "customer_approve_by"), "approval window is set")
}

// A plan already responded to (not pending_chef) is rejected.
func TestRespondMealPlan_NotPending_Conflict(t *testing.T) {
	escrowOff(t)
	db := setupOrchestrationDB(t)
	chefID, chefUserID := seedOrchChef(t, db)
	planID := seedOrchPlan(t, db, models.MealPlanConfirmed, uuid.New(), chefID)
	seedOrchDay(t, db, planID, models.MealPlanDayConfirmed, 100)

	w := respondReq(t, chefUserID, planID, map[string]any{"acceptAll": true})
	require.Equal(t, http.StatusConflict, w.Code, w.Body.String())
}

// Neither acceptAll nor any accepted day → bad request.
func TestRespondMealPlan_NothingAccepted_BadRequest(t *testing.T) {
	escrowOff(t)
	db := setupOrchestrationDB(t)
	chefID, chefUserID := seedOrchChef(t, db)
	planID := seedOrchPlan(t, db, models.MealPlanPendingChef, uuid.New(), chefID)
	seedOrchDay(t, db, planID, models.MealPlanDayRequested, 100)

	w := respondReq(t, chefUserID, planID, map[string]any{"acceptedDayIds": []string{}})
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
}

// ── finalizeByCustomer (approve / reject) ──

// Approve confirms the accepted days, leaves declined days declined, charge basis = accepted.
func TestApproveMealPlan_ConfirmsAcceptedDays(t *testing.T) {
	escrowOff(t)
	db := setupOrchestrationDB(t)
	chefID, chefUserID := seedOrchChef(t, db)
	customerID := uuid.New()
	planID := seedOrchPlan(t, db, models.MealPlanAwaitingCustomer, customerID, chefID)
	dAccept := seedOrchDay(t, db, planID, models.MealPlanDayAccepted, 100)
	dDecline := seedOrchDay(t, db, planID, models.MealPlanDayDeclined, 150)

	w := finalizeReq(t, customerID, planID, "approve")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Equal(t, string(models.MealPlanConfirmed), planField(t, db, planID, "status"))
	require.Equal(t, string(models.MealPlanDayConfirmed), dayStatusOf(t, db, dAccept))
	require.Equal(t, string(models.MealPlanDayDeclined), dayStatusOf(t, db, dDecline), "declined stays declined")
	require.Equal(t, 100.0, planTotalOf(t, db, planID), "charge basis = accepted days only")
	require.NotEmpty(t, planField(t, db, planID, "confirmed_at"))
	require.Equal(t, 1, eventCountFor(t, db, chefUserID), "chef notified the plan was finalized")
}

// Reject cancels the plan with a reason.
func TestRejectMealPlan_CancelsPlan(t *testing.T) {
	escrowOff(t)
	db := setupOrchestrationDB(t)
	chefID, _ := seedOrchChef(t, db)
	customerID := uuid.New()
	planID := seedOrchPlan(t, db, models.MealPlanAwaitingCustomer, customerID, chefID)
	seedOrchDay(t, db, planID, models.MealPlanDayAccepted, 100)

	w := finalizeReq(t, customerID, planID, "reject")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Equal(t, string(models.MealPlanCancelled), planField(t, db, planID, "status"))
	require.Contains(t, planField(t, db, planID, "cancel_reason"), "rejected")
	require.NotEmpty(t, planField(t, db, planID, "cancelled_at"))
}

// Approving a plan that is not awaiting the customer is rejected.
func TestApproveMealPlan_NotAwaiting_Conflict(t *testing.T) {
	escrowOff(t)
	db := setupOrchestrationDB(t)
	chefID, _ := seedOrchChef(t, db)
	customerID := uuid.New()
	planID := seedOrchPlan(t, db, models.MealPlanConfirmed, customerID, chefID)

	w := finalizeReq(t, customerID, planID, "approve")
	require.Equal(t, http.StatusConflict, w.Code, w.Body.String())
}

// A plan owned by a different customer is not found for this caller (owner scoping).
func TestApproveMealPlan_WrongCustomer_NotFound(t *testing.T) {
	escrowOff(t)
	db := setupOrchestrationDB(t)
	chefID, _ := seedOrchChef(t, db)
	planID := seedOrchPlan(t, db, models.MealPlanAwaitingCustomer, uuid.New(), chefID)

	w := finalizeReq(t, uuid.New(), planID, "approve") // a different customer
	require.Equal(t, http.StatusNotFound, w.Code, w.Body.String())
}
