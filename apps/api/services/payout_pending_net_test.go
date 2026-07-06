package services

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

// #462 (item 3): the admin payout-approval queue must surface the chef's actual
// NET payout (what the held Route transfer carries), not just the gross customer
// amount. Amount stays the gross customer value (Order.Total / day gross /
// group slice) for context; NetPayout is what the admin is really releasing.

// seedOrderHoldNet inserts a delivered order with the money columns the net
// computation reads, in the given hold state.
func seedOrderHoldNet(t *testing.T, db *gorm.DB, hold models.PayoutHoldStatus, delivered time.Time,
	total, subtotal, tax, tip, rate float64) uuid.UUID {
	t.Helper()
	id, chef := uuid.New(), uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO orders
		(id, order_number, customer_id, chef_id, status, razorpay_order_id, total, subtotal, tax,
		 chef_tip, commission_rate, payout_hold_status, delivered_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		id.String(), "ORD-"+id.String()[:8], uuid.NewString(), chef.String(),
		"delivered", "order_rzp_net", total, subtotal, tax, tip, rate, string(hold), delivered).Error)
	return id
}

// seedDayHoldNet inserts a delivered meal-plan day + parent plan carrying the
// plan-level subtotal/tax the per-day net basis needs.
func seedDayHoldNet(t *testing.T, db *gorm.DB, hold models.PayoutHoldStatus, delivered time.Time,
	dayPrice, planSubtotal, planTax float64) uuid.UUID {
	t.Helper()
	dayID, planID, chef := uuid.New(), uuid.New(), uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO meal_plans (id, meal_plan_number, customer_id, chef_id, status, subtotal, tax)
		VALUES (?,?,?,?,?,?,?)`, planID.String(), "MP-"+planID.String()[:8], uuid.NewString(), chef.String(),
		"active", planSubtotal, planTax).Error)
	require.NoError(t, db.Exec(`INSERT INTO meal_plan_days
		(id, meal_plan_id, status, payout_transfer_id, price, payout_hold_status, delivered_at)
		VALUES (?,?,?,?,?,?,?)`,
		dayID.String(), planID.String(), "delivered", "trf_net", dayPrice, string(hold), delivered).Error)
	return dayID
}

// seedGroupHoldNet inserts a delivered group order with a chef slice.
func seedGroupHoldNet(t *testing.T, db *gorm.DB, hold models.PayoutHoldStatus, delivered time.Time,
	subtotal, tax float64) uuid.UUID {
	t.Helper()
	id, chef := uuid.New(), uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO group_orders
		(id, host_id, chef_id, status, payout_transfer_id, payout_hold_status, delivered_at, subtotal, tax)
		VALUES (?,?,?,?,?,?,?,?,?)`,
		id.String(), uuid.NewString(), chef.String(), "delivered", "trf_grp", string(hold), delivered, subtotal, tax).Error)
	return id
}

func TestListPendingPayouts_SurfacesChefNetPayout(t *testing.T) {
	db := setupReleaseDB(t)
	now := time.Now()

	// Order: gross customer total 260, chef income basis food 200 + tax 20 + tip 10.
	ordID := seedOrderHoldNet(t, db, models.PayoutHoldReleaseEligible, now.Add(-30*time.Hour),
		260.0, 200.0, 20.0, 10.0, 0.06)
	// Day: price 120, plan subtotal 240 / tax 24 → proportional food GST 12.
	dayID := seedDayHoldNet(t, db, models.PayoutHoldReleaseEligible, now.Add(-20*time.Hour),
		120.0, 240.0, 24.0)
	// Group: chef slice subtotal 300 + tax 30.
	grpID := seedGroupHoldNet(t, db, models.PayoutHoldReleaseEligible, now.Add(-10*time.Hour),
		300.0, 30.0)

	rows, err := ListPendingPayouts(db, PendingFilter{})
	require.NoError(t, err)

	got := map[uuid.UUID]PendingPayout{}
	for _, r := range rows {
		got[r.ID] = r
	}

	// Expected nets computed via the SAME functions that build the held transfers,
	// so the queue can never drift from what actually moves.
	wantOrderNet := ComputeOrderEarnings(EarningsInput{
		ItemRevenue: 200.0, Tax: 20.0, ChefTip: 10.0, CommissionRate: 0.06,
	}, "").NetPayout
	rate := GetCommissionRate(db)
	wantDayNet := perDayNetPayout(&models.MealPlan{Subtotal: 240.0, Tax: 24.0},
		&models.MealPlanDay{Price: 120.0}, rate)
	// #546: group net is now commission+TDS-deducted (was the gross subtotal+tax slice).
	wantGroupNet := groupNetPayout(&models.GroupOrder{Subtotal: 300.0, Tax: 30.0}, rate)

	require.InDelta(t, wantOrderNet, got[ordID].NetPayout, 0.001, "order net = ComputeOrderEarnings.NetPayout")
	require.Less(t, got[ordID].NetPayout, got[ordID].Amount, "order net is below the gross customer total")
	require.Equal(t, 260.0, got[ordID].Amount, "order Amount stays gross Total")

	require.InDelta(t, wantDayNet, got[dayID].NetPayout, 0.001, "day net = perDayNetPayout")

	require.InDelta(t, wantGroupNet, got[grpID].NetPayout, 0.001, "group net = groupNetPayout (net of commission+TDS)")
	require.Less(t, got[grpID].NetPayout, got[grpID].Amount, "group net is below the gross chef slice")
}

// #547: the queue net must use the rate FROZEN on the aggregate when the transfer was held
// (meal_plan_days.commission_rate / group_orders.commission_rate), not the current platform
// rate — so the display doesn't drift if the flat rate changed while the hold was pending.
func TestListPendingPayouts_UsesFrozenCommissionRate(t *testing.T) {
	db := setupReleaseDB(t)
	now := time.Now()
	current := GetCommissionRate(db) // default flat rate (≈0.06)
	const frozen = 0.20              // deliberately far from the current rate
	require.Greater(t, frozen-current, 0.05, "precondition: frozen rate differs from current")

	dayID, planID := uuid.New(), uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO meal_plans (id, meal_plan_number, customer_id, chef_id, status, subtotal, tax)
		VALUES (?,?,?,?,?,?,?)`, planID.String(), "MP-frz", uuid.NewString(), uuid.NewString(), "active", 240.0, 24.0).Error)
	require.NoError(t, db.Exec(`INSERT INTO meal_plan_days
		(id, meal_plan_id, status, payout_transfer_id, price, payout_hold_status, delivered_at, commission_rate)
		VALUES (?,?,?,?,?,?,?,?)`,
		dayID.String(), planID.String(), "delivered", "trf_frz", 120.0,
		string(models.PayoutHoldReleaseEligible), now.Add(-5*time.Hour), frozen).Error)

	grpID := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO group_orders
		(id, host_id, chef_id, status, payout_transfer_id, payout_hold_status, delivered_at, subtotal, tax, commission_rate)
		VALUES (?,?,?,?,?,?,?,?,?,?)`,
		grpID.String(), uuid.NewString(), uuid.NewString(), "delivered", "trf_grpfrz",
		string(models.PayoutHoldReleaseEligible), now.Add(-5*time.Hour), 300.0, 30.0, frozen).Error)

	rows, err := ListPendingPayouts(db, PendingFilter{})
	require.NoError(t, err)
	got := map[uuid.UUID]PendingPayout{}
	for _, r := range rows {
		got[r.ID] = r
	}

	wantDay := perDayNetPayout(&models.MealPlan{Subtotal: 240.0, Tax: 24.0}, &models.MealPlanDay{Price: 120.0}, frozen)
	currentDay := perDayNetPayout(&models.MealPlan{Subtotal: 240.0, Tax: 24.0}, &models.MealPlanDay{Price: 120.0}, current)
	require.InDelta(t, wantDay, got[dayID].NetPayout, 0.001, "day net uses the FROZEN rate")
	require.Greater(t, currentDay-wantDay, 0.5, "sanity: the current-rate net differs materially (a wrong impl would show this)")

	wantGrp := groupNetPayout(&models.GroupOrder{Subtotal: 300.0, Tax: 30.0}, frozen)
	require.InDelta(t, wantGrp, got[grpID].NetPayout, 0.001, "group net uses the FROZEN rate")
}
