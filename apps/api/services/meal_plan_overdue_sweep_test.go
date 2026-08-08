package services

// meal_plan_overdue_sweep_test.go — #1034. A meal-plan day whose order the chef never
// acted on (never accepted/prepped/dispatched) sat `confirmed` — the customer app's
// "Scheduled" pill — forever once its date passed, and blocked completeFinishedPlans from
// ever closing the plan. sweepOverdueDayOrders freezes such a day into the existing #393
// delivery-failure review queue (never guesses refund vs release), and
// MarkMealPlanDayDelivered now completes the plan the instant its last day delivers
// instead of waiting for the next completeFinishedPlans tick.

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

// insertOverdueOrder inserts a bare order row (no gateway_order_id — a meal-plan-day
// shell, mirroring generateDayOrder) in the given status.
func insertOverdueOrder(t *testing.T, db *gorm.DB, status models.OrderStatus) uuid.UUID {
	t.Helper()
	id := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO orders (id, order_number, customer_id, chef_id, status)
		VALUES (?,?,?,?,?)`, id.String(), "ORD-"+id.String()[:8], uuid.NewString(), uuid.NewString(), string(status)).Error)
	return id
}

// insertOverduePlanDay inserts a plan (in planStatus) with one day (in dayStatus, dated
// dayDate, optionally linked to orderID) and returns both ids.
func insertOverduePlanDay(t *testing.T, db *gorm.DB, planStatus models.MealPlanStatus,
	dayStatus models.MealPlanDayStatus, dayDate time.Time, orderID *uuid.UUID) (planID, dayID uuid.UUID) {
	t.Helper()
	planID, dayID = uuid.New(), uuid.New()
	chefID := uuid.NewString()
	seedLiveChefRow(t, db, chefID)
	require.NoError(t, db.Exec(`INSERT INTO meal_plans (id, meal_plan_number, customer_id, chef_id, status)
		VALUES (?,?,?,?,?)`, planID.String(), "MP-"+planID.String()[:8], uuid.NewString(), chefID, string(planStatus)).Error)
	var ord any
	if orderID != nil {
		ord = orderID.String()
	}
	require.NoError(t, db.Exec(`INSERT INTO meal_plan_days (id, meal_plan_id, order_id, status, price, date)
		VALUES (?,?,?,?,?,?)`, dayID.String(), planID.String(), ord, string(dayStatus), 120.0, dayDate).Error)
	return planID, dayID
}

func planStatusOf(t *testing.T, db *gorm.DB, id uuid.UUID) models.MealPlanStatus {
	t.Helper()
	var s string
	require.NoError(t, db.Raw(`SELECT status FROM meal_plans WHERE id = ?`, id.String()).Scan(&s).Error)
	return models.MealPlanStatus(s)
}

// ── sweepOverdueDayOrders ─────────────────────────────────────────────────────

func TestSweepOverdueDayOrders_FreezesChefNeverActedDay(t *testing.T) {
	db := setupCrossguardDB(t)
	orderID := insertOverdueOrder(t, db, models.OrderStatusPending)
	_, dayID := insertOverduePlanDay(t, db, models.MealPlanActive, models.MealPlanDayConfirmed,
		time.Now().Add(-48*time.Hour), &orderID)

	sweepOverdueDayOrders()

	require.Equal(t, models.MealPlanDayFailed, loadDayStatus(t, db, dayID), "overdue day frozen into review, not silently left Scheduled")
	require.Equal(t, models.PayoutHoldDisputed, loadDayHold(t, db, dayID), "hold frozen, no money moved")
	require.Equal(t, 1, countOutbox(t, db, SubjectMealPlanDayFailed))
	require.Equal(t, 1, countOutbox(t, db, SubjectDeliveryFailed))

	// Idempotent: a second run must not re-freeze or double-notify.
	sweepOverdueDayOrders()
	require.Equal(t, models.MealPlanDayFailed, loadDayStatus(t, db, dayID))
	require.Equal(t, 1, countOutbox(t, db, SubjectMealPlanDayFailed), "no duplicate freeze event on re-run")
}

func TestSweepOverdueDayOrders_SkipsWithinGrace(t *testing.T) {
	db := setupCrossguardDB(t)
	orderID := insertOverdueOrder(t, db, models.OrderStatusPending)
	_, dayID := insertOverduePlanDay(t, db, models.MealPlanActive, models.MealPlanDayConfirmed,
		time.Now().Add(-1*time.Hour), &orderID)

	sweepOverdueDayOrders()

	require.Equal(t, models.MealPlanDayConfirmed, dayStatusOf(t, db, dayID), "date only just passed — still within grace")
}

func TestSweepOverdueDayOrders_SkipsNoOrderDay(t *testing.T) {
	// order_id IS NULL is sweepStuckDays' job, not this sweep's — must not double-handle.
	db := setupCrossguardDB(t)
	_, dayID := insertOverduePlanDay(t, db, models.MealPlanActive, models.MealPlanDayConfirmed,
		time.Now().Add(-48*time.Hour), nil)

	sweepOverdueDayOrders()

	require.Equal(t, models.MealPlanDayConfirmed, dayStatusOf(t, db, dayID))
}

func TestSweepOverdueDayOrders_SkipsAlreadyTerminalDay(t *testing.T) {
	db := setupCrossguardDB(t)
	orderID := insertOverdueOrder(t, db, models.OrderStatusPending)
	for _, term := range []models.MealPlanDayStatus{
		models.MealPlanDayDelivered, models.MealPlanDayRefunded, models.MealPlanDayCancelled,
		models.MealPlanDaySkipped, models.MealPlanDayDeclined, models.MealPlanDayFailed,
	} {
		t.Run(string(term), func(t *testing.T) {
			_, dayID := insertOverduePlanDay(t, db, models.MealPlanActive, term,
				time.Now().Add(-48*time.Hour), &orderID)

			sweepOverdueDayOrders()

			require.Equal(t, term, dayStatusOf(t, db, dayID), "terminal/failed day left alone")
		})
	}
}

// A day whose order already entered the delivery hand-off (ready/picked_up/delivering)
// must NOT be frozen here — that would race a genuine in-flight completion. The
// deliveries-table sweeps own that case, on their own tighter grace.
func TestSweepOverdueDayOrders_SkipsDispatchedOrder(t *testing.T) {
	db := setupCrossguardDB(t)
	for _, st := range []models.OrderStatus{
		models.OrderStatusDelivering, models.OrderStatusPickedUp,
		models.OrderStatusDelivered, models.OrderStatusCancelled,
	} {
		t.Run(string(st), func(t *testing.T) {
			orderID := insertOverdueOrder(t, db, st)
			_, dayID := insertOverduePlanDay(t, db, models.MealPlanActive, models.MealPlanDayConfirmed,
				time.Now().Add(-48*time.Hour), &orderID)

			sweepOverdueDayOrders()

			require.Equal(t, models.MealPlanDayConfirmed, dayStatusOf(t, db, dayID), "order already in/past the delivery hand-off — not this sweep's job")
		})
	}
}

// A `prepared` day (cooked ahead of the lock) is just as overdue-stranded as a confirmed
// one if the chef never hands it to delivery — dayAwaitingOrder's sibling case for the
// stuck-day sweep, mirrored here.
func TestSweepOverdueDayOrders_FreezesPreparedDay(t *testing.T) {
	db := setupCrossguardDB(t)
	orderID := insertOverdueOrder(t, db, models.OrderStatusPreparing)
	_, dayID := insertOverduePlanDay(t, db, models.MealPlanActive, models.MealPlanDayPrepared,
		time.Now().Add(-48*time.Hour), &orderID)

	sweepOverdueDayOrders()

	require.Equal(t, models.MealPlanDayFailed, loadDayStatus(t, db, dayID))
}

// ── MarkMealPlanDayDelivered now completes the plan immediately (#1034 gap 2) ──

func TestMarkMealPlanDayDelivered_CompletesPlanWhenLastDay(t *testing.T) {
	db := setupCrossguardDB(t)
	orderID := insertOverdueOrder(t, db, models.OrderStatusReady)
	planID, dayID := insertOverduePlanDay(t, db, models.MealPlanActive, models.MealPlanDayConfirmed,
		time.Now(), &orderID)

	MarkMealPlanDayDelivered(orderID)

	require.Equal(t, models.MealPlanDayDelivered, loadDayStatus(t, db, dayID))
	require.Equal(t, models.MealPlanCompleted, planStatusOf(t, db, planID),
		"plan completes the SAME call, not on the next completeFinishedPlans tick")
	require.Equal(t, 1, countOutbox(t, db, SubjectMealPlanCompleted))
}

func TestMarkMealPlanDayDelivered_DoesNotCompletePlanWithOtherOpenDays(t *testing.T) {
	db := setupCrossguardDB(t)
	orderID := insertOverdueOrder(t, db, models.OrderStatusReady)
	planID, dayID := insertOverduePlanDay(t, db, models.MealPlanActive, models.MealPlanDayConfirmed,
		time.Now(), &orderID)
	// A sibling day on the same plan, still open.
	require.NoError(t, db.Exec(`INSERT INTO meal_plan_days (id, meal_plan_id, order_id, status, price, date)
		VALUES (?,?,?,?,?,?)`, uuid.New().String(), planID.String(), nil, string(models.MealPlanDayConfirmed), 120.0, time.Now().Add(24*time.Hour)).Error)

	MarkMealPlanDayDelivered(orderID)

	require.Equal(t, models.MealPlanDayDelivered, loadDayStatus(t, db, dayID))
	require.Equal(t, models.MealPlanActive, planStatusOf(t, db, planID), "sibling day still open — plan must not complete")
	require.Equal(t, 0, countOutbox(t, db, SubjectMealPlanCompleted))
}
