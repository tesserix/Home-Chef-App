package handlers

// meal_plan_skip_test.go — #422 policy change. SkipMealPlanDay no longer auto-credits the
// customer: it RAISES an admin-reviewed request. HTTP-level coverage of the new handler +
// the 12h-before-START gate helper (mealPlanDayStartIST). Escrow is OFF, so the only money
// touch would be a (now-removed) RefundDay — proving the absence of a refund event proves
// no auto-credit. Reuses setupOrchestrationDB, augmented with the chef_schedules table and
// a slot column the skip gate reads.

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
)

// ── mealPlanDayStartIST (pure) ───────────────────────────────────────────────

func TestMealPlanDayStartIST(t *testing.T) {
	date := time.Date(2026, 7, 29, 0, 0, 0, 0, istLoc)
	require.Equal(t, time.Wednesday, date.Weekday(), "fixture sanity")
	wed := int(time.Wednesday) // 3

	// (1) an open schedule row for that weekday → its OpenTime wins over the slot default.
	sched := []models.ChefSchedule{{DayOfWeek: wed, OpenTime: "10:30"}}
	got := mealPlanDayStartIST(sched, &models.MealPlanDay{Date: date, Slot: models.MealSlotDinner})
	require.Equal(t, time.Date(2026, 7, 29, 10, 30, 0, 0, istLoc), got)

	// (2) no schedule row → lunch slot default 12:00 IST.
	got = mealPlanDayStartIST(nil, &models.MealPlanDay{Date: date, Slot: models.MealSlotLunch})
	require.Equal(t, time.Date(2026, 7, 29, 12, 0, 0, 0, istLoc), got)

	// (3) no schedule row → dinner slot default 19:00 IST.
	got = mealPlanDayStartIST(nil, &models.MealPlanDay{Date: date, Slot: models.MealSlotDinner})
	require.Equal(t, time.Date(2026, 7, 29, 19, 0, 0, 0, istLoc), got)

	// (4) a CLOSED schedule row → falls back to the slot default.
	closed := []models.ChefSchedule{{DayOfWeek: wed, OpenTime: "10:30", IsClosed: true}}
	got = mealPlanDayStartIST(closed, &models.MealPlanDay{Date: date, Slot: models.MealSlotLunch})
	require.Equal(t, time.Date(2026, 7, 29, 12, 0, 0, 0, istLoc), got)

	// (5) a row for a DIFFERENT weekday → ignored, slot default.
	other := []models.ChefSchedule{{DayOfWeek: (wed + 1) % 7, OpenTime: "08:00"}}
	got = mealPlanDayStartIST(other, &models.MealPlanDay{Date: date, Slot: models.MealSlotDinner})
	require.Equal(t, time.Date(2026, 7, 29, 19, 0, 0, 0, istLoc), got)
}

// ── SkipMealPlanDay (HTTP) ───────────────────────────────────────────────────

func setupSkipDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := setupOrchestrationDB(t)
	require.NoError(t, db.Exec(`ALTER TABLE meal_plan_days ADD COLUMN slot text DEFAULT ''`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE chef_schedules (id text PRIMARY KEY, chef_id text,
		day_of_week int, open_time text, close_time text, is_closed integer DEFAULT 0,
		created_at datetime, updated_at datetime)`).Error)
	return db
}

func seedSkipDay(t *testing.T, db *gorm.DB, planID uuid.UUID, date time.Time, slot models.MealSlot, orderID *uuid.UUID) uuid.UUID {
	t.Helper()
	dayID := uuid.New()
	var ord any
	if orderID != nil {
		ord = orderID.String()
	}
	require.NoError(t, db.Exec(`INSERT INTO meal_plan_days
		(id, meal_plan_id, status, price, date, slot, payout_hold_status, order_id)
		VALUES (?,?,?,?,?,?,?,?)`,
		dayID.String(), planID.String(), string(models.MealPlanDayConfirmed), 200.0, date,
		string(slot), "", ord).Error)
	return dayID
}

func dayFieldH(t *testing.T, db *gorm.DB, dayID uuid.UUID, col string) string {
	t.Helper()
	var s string
	require.NoError(t, db.Raw(`SELECT COALESCE(CAST(`+col+` AS TEXT),'') FROM meal_plan_days WHERE id = ?`, dayID.String()).Scan(&s).Error)
	return s
}

func countOutboxH(t *testing.T, db *gorm.DB, subject string) int {
	t.Helper()
	var n int64
	require.NoError(t, db.Raw(`SELECT COUNT(*) FROM outbox_events WHERE subject = ?`, subject).Scan(&n).Error)
	return int(n)
}

func skipDayReq(t *testing.T, customerID, planID, dayID uuid.UUID) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("userID", customerID); c.Next() })
	r.PUT("/meal-plans/:id/days/:dayId/skip", (&MealPlanHandler{}).SkipMealPlanDay)
	req := httptest.NewRequest(http.MethodPut, "/meal-plans/"+planID.String()+"/days/"+dayID.String()+"/skip", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestSkipMealPlanDay_RaisesRequestFreezesHoldNoCredit(t *testing.T) {
	escrowOff(t)
	db := setupSkipDB(t)
	chefID, _ := seedOrchChef(t, db)
	cust := uuid.New()
	planID := seedOrchPlan(t, db, models.MealPlanConfirmed, cust, chefID)
	// A confirmed day 3 days out (well before the 12h-before-start cutoff), no order yet.
	future := time.Now().In(istLoc).AddDate(0, 0, 3)
	futureDay := time.Date(future.Year(), future.Month(), future.Day(), 0, 0, 0, 0, istLoc)
	dayID := seedSkipDay(t, db, planID, futureDay, models.MealSlotLunch, nil)

	w := skipDayReq(t, cust, planID, dayID)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.True(t, strings.Contains(w.Body.String(), "skip_requested"), w.Body.String())
	require.Equal(t, string(models.MealPlanDaySkipRequested), dayFieldH(t, db, dayID, "status"), "day → skip_req, not skipped")
	require.Equal(t, string(models.PayoutHoldDisputed), dayFieldH(t, db, dayID, "payout_hold_status"), "chef payout frozen")
	require.Equal(t, "", dayFieldH(t, db, dayID, "refund_txn_id"), "no auto-credit")
	require.Equal(t, 1, countOutboxH(t, db, services.SubjectMealPlanDaySkipRequested), "chef told a skip is pending")
	require.Equal(t, 0, countOutboxH(t, db, services.SubjectMealPlanDayRefunded), "no refund event — not credited")
}

func TestSkipMealPlanDay_TooLatePastStartGate(t *testing.T) {
	escrowOff(t)
	db := setupSkipDB(t)
	chefID, _ := seedOrchChef(t, db)
	cust := uuid.New()
	planID := seedOrchPlan(t, db, models.MealPlanConfirmed, cust, chefID)
	// A day whose start is already in the past → strictly past the 12h-before-start cutoff.
	past := time.Now().In(istLoc).AddDate(0, 0, -1)
	pastDay := time.Date(past.Year(), past.Month(), past.Day(), 0, 0, 0, 0, istLoc)
	dayID := seedSkipDay(t, db, planID, pastDay, models.MealSlotLunch, nil)

	w := skipDayReq(t, cust, planID, dayID)

	require.Equal(t, http.StatusConflict, w.Code, w.Body.String())
	require.Equal(t, string(models.MealPlanDayConfirmed), dayFieldH(t, db, dayID, "status"), "day unchanged past the cutoff")
}

func TestSkipMealPlanDay_OrderGeneratedConflict(t *testing.T) {
	escrowOff(t)
	db := setupSkipDB(t)
	chefID, _ := seedOrchChef(t, db)
	cust := uuid.New()
	planID := seedOrchPlan(t, db, models.MealPlanConfirmed, cust, chefID)
	future := time.Now().In(istLoc).AddDate(0, 0, 3)
	futureDay := time.Date(future.Year(), future.Month(), future.Day(), 0, 0, 0, 0, istLoc)
	ord := uuid.New()
	dayID := seedSkipDay(t, db, planID, futureDay, models.MealSlotLunch, &ord) // order already generated

	w := skipDayReq(t, cust, planID, dayID)

	require.Equal(t, http.StatusConflict, w.Code, w.Body.String())
	require.Equal(t, string(models.MealPlanDayConfirmed), dayFieldH(t, db, dayID, "status"), "a generated day can't be skipped")
}
