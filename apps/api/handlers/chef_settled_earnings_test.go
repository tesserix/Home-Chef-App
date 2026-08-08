package handlers

// chef_settled_earnings_test.go — one definition of "money this kitchen earned".
//
// The Earnings screen and the Analytics screen reported different figures for the
// same week and neither matched the payout: Analytics summed anything captured,
// bucketed on created_at, while Earnings summed delivered, un-refunded orders on
// delivered_at (#1030). Both now call chefSettledEarnings; these tests pin what it
// admits and that its totals are the sum of the per-order rows it hands back.

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
)

func setupSettledEarningsDB(t *testing.T) (*gorm.DB, models.ChefProfile) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TABLE orders (id TEXT PRIMARY KEY, order_number TEXT,
		chef_id TEXT, status TEXT, mode TEXT DEFAULT 'live',
		subtotal REAL DEFAULT 0, tax REAL DEFAULT 0, tax_food REAL DEFAULT 0,
		tax_service REAL DEFAULT 0, chef_funded_discount REAL DEFAULT 0,
		delivery_fee REAL DEFAULT 0, delivery_fee_final REAL, chef_tip REAL DEFAULT 0,
		fulfillment_type TEXT DEFAULT 'delivery',
		delivery_address_state TEXT, commission_rate REAL DEFAULT 0,
		payout_hold_status TEXT, gateway_split_paise INTEGER DEFAULT 0,
		created_at DATETIME, delivered_at DATETIME,
		refunded_at DATETIME, deleted_at DATETIME)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE chef_profiles (id TEXT PRIMARY KEY, mode TEXT)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE chef_penalties (id TEXT PRIMARY KEY, chef_id TEXT,
		user_id TEXT, kind TEXT, status TEXT, source_key TEXT, order_id TEXT, reference TEXT,
		currency TEXT, basis_amount REAL, rate_percent REAL, amount REAL,
		created_at DATETIME, updated_at DATETIME)`).Error)

	chef := models.ChefProfile{ID: uuid.New(), State: "Maharashtra"}
	require.NoError(t, db.Exec(`INSERT INTO chef_profiles (id, mode) VALUES (?, 'live')`, chef.ID.String()).Error)

	prev := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = prev })
	return db, chef
}

type settledOrderSeed struct {
	subtotal        float64
	taxFood         float64
	tip             float64
	deliveryFee     float64
	fulfillmentType string
	mode            string
	createdAt       time.Time
	deliveredAt     time.Time
	refundedAt      *time.Time
	deletedAt       *time.Time
	status          string
	// splitPaise is what Easy Split actually settled to the chef's vendor account
	// at capture; 0 means the order settles on the weekly-statement rail instead.
	splitPaise int
}

func seedSettledOrder(t *testing.T, db *gorm.DB, chefID uuid.UUID, s settledOrderSeed) uuid.UUID {
	t.Helper()
	if s.mode == "" {
		s.mode = "live"
	}
	if s.status == "" {
		s.status = "delivered"
	}
	if s.fulfillmentType == "" {
		s.fulfillmentType = "delivery"
	}
	if s.createdAt.IsZero() {
		s.createdAt = s.deliveredAt
	}
	id := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO orders (id, order_number, chef_id, status, mode,
		subtotal, tax, tax_food, tax_service, delivery_fee, chef_tip, fulfillment_type,
		delivery_address_state, created_at, delivered_at, refunded_at, deleted_at,
		gateway_split_paise)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, 0, ?, ?, ?, 'Maharashtra', ?, ?, ?, ?, ?)`,
		id.String(), "HC-"+id.String()[:6], chefID.String(), s.status, s.mode,
		s.subtotal, s.taxFood, s.taxFood, s.deliveryFee, s.tip, s.fulfillmentType,
		s.createdAt, s.deliveredAt, s.refundedAt, s.deletedAt, s.splitPaise).Error)
	return id
}

func TestChefSettledEarnings_TotalsAreTheSumOfTheRows(t *testing.T) {
	db, chef := setupSettledEarningsDB(t)
	now := time.Now()
	for _, sub := range []float64{250, 333.33, 1000} {
		seedSettledOrder(t, db, chef.ID, settledOrderSeed{
			subtotal: sub, taxFood: sub * 0.05, deliveredAt: now.Add(-time.Hour),
		})
	}

	totals, orders, err := chefSettledEarnings(chef, now.AddDate(0, 0, -7), now, 0.06)
	require.NoError(t, err)
	require.Len(t, orders, 3)

	var netFromRows float64
	for _, o := range orders {
		netFromRows += o.NetPayout
	}
	require.Equal(t, 3, totals.OrdersCount)
	require.InDelta(t, netFromRows, totals.NetPayout, 0.005,
		"the headline must be the sum of the rows the chef can see")
}

func TestChefSettledEarnings_ExcludesRefundedSandboxAndDeleted(t *testing.T) {
	db, chef := setupSettledEarningsDB(t)
	now := time.Now()
	at := now.Add(-time.Hour)
	refunded, deleted := at, at

	seedSettledOrder(t, db, chef.ID, settledOrderSeed{subtotal: 500, deliveredAt: at})
	seedSettledOrder(t, db, chef.ID, settledOrderSeed{subtotal: 500, deliveredAt: at, refundedAt: &refunded})
	seedSettledOrder(t, db, chef.ID, settledOrderSeed{subtotal: 500, deliveredAt: at, deletedAt: &deleted})
	seedSettledOrder(t, db, chef.ID, settledOrderSeed{subtotal: 500, deliveredAt: at, mode: "sandbox"})
	seedSettledOrder(t, db, chef.ID, settledOrderSeed{subtotal: 500, deliveredAt: at, status: "preparing"})

	totals, _, err := chefSettledEarnings(chef, now.AddDate(0, 0, -7), now, 0.06)
	require.NoError(t, err)
	require.Equal(t, 1, totals.OrdersCount, "only the delivered, un-refunded live order settles")
}

func TestChefSettledEarnings_WindowsOnDeliveryNotPlacement(t *testing.T) {
	// The defect behind #1030: an order placed last week and delivered this week is
	// this week's money. Analytics used to bucket it on created_at, so the same order
	// landed in a different week than the Earnings screen and the payout.
	db, chef := setupSettledEarningsDB(t)
	now := time.Now()
	from := now.AddDate(0, 0, -1)

	seedSettledOrder(t, db, chef.ID, settledOrderSeed{
		subtotal: 400, createdAt: now.AddDate(0, 0, -5), deliveredAt: now.Add(-time.Hour),
	})
	seedSettledOrder(t, db, chef.ID, settledOrderSeed{
		subtotal: 400, createdAt: now.Add(-time.Hour), deliveredAt: now.AddDate(0, 0, -5),
	})

	totals, orders, err := chefSettledEarnings(chef, from, now, 0.06)
	require.NoError(t, err)
	require.Equal(t, 1, totals.OrdersCount)
	require.True(t, orders[0].CompletedAt.After(from), "the row is dated by delivery")
}

func TestChefSettledEarnings_DeliveryFeeIsNotChefMoney(t *testing.T) {
	db, chef := setupSettledEarningsDB(t)
	now := time.Now()

	seedSettledOrder(t, db, chef.ID, settledOrderSeed{
		subtotal: 500, taxFood: 25, deliveredAt: now.Add(-time.Hour),
	})
	withFee, _, err := chefSettledEarnings(chef, now.AddDate(0, 0, -7), now, 0.06)
	require.NoError(t, err)

	db2, chef2 := setupSettledEarningsDB(t)
	seedSettledOrder(t, db2, chef2.ID, settledOrderSeed{
		subtotal: 500, taxFood: 25, deliveryFee: 60, deliveredAt: now.Add(-time.Hour),
	})
	noFee, _, err := chefSettledEarnings(chef2, now.AddDate(0, 0, -7), now, 0.06)
	require.NoError(t, err)

	require.Equal(t, withFee.NetPayout, noFee.NetPayout, "a platform-carried leg's fee is the driver's money")
}

// The other half: the chef priced this leg from their own published rates, was
// paid for it by the customer, and drove it. It is their income.
func TestChefSettledEarnings_ChefCarriedLegKeepsTheFee(t *testing.T) {
	db, chef := setupSettledEarningsDB(t)
	now := time.Now()

	seedSettledOrder(t, db, chef.ID, settledOrderSeed{
		subtotal: 500, taxFood: 25, deliveryFee: 60,
		fulfillmentType: "chef_delivery", deliveredAt: now.Add(-time.Hour),
	})

	totals, orders, err := chefSettledEarnings(chef, now.AddDate(0, 0, -7), now, 0.06)
	require.NoError(t, err)
	require.Len(t, orders, 1)

	// gross 500 + 25 + 60 = 585; commission 6% of FOOD only = 30; TDS 1% = 5.85.
	require.InDelta(t, 585.0, totals.GrossRevenue, 0.005)
	require.InDelta(t, 30.0, totals.PlatformCommission, 0.005)
	require.InDelta(t, 549.15, totals.NetPayout, 0.005)
}

// The dashboard hero labelled "Total earnings" taps straight through to the
// Earnings screen, so the two must report the same money. It used to sum every
// captured order on created_at while Earnings summed delivered, un-refunded
// orders on delivered_at, and the chef read two different lifetime totals (#1030).
func TestChefDashboardTotalEarnings_MatchesEarningsScreen(t *testing.T) {
	db := setupChefVisDB(t)
	userID, chefID := seedVisChef(t, db)

	deliveredAt := services.CapacityDay(time.Now()).Add(13 * time.Hour)
	require.NoError(t, db.Exec(`INSERT INTO orders (id, order_number, customer_id, chef_id,
		status, payment_status, subtotal, tax, tax_food, chef_tip, total,
		created_at, delivered_at) VALUES (?, 'ORD-SET', ?, ?, 'delivered', 'completed',
		500, 25, 25, 20, 545, ?, ?)`,
		uuid.NewString(), uuid.NewString(), chefID.String(), deliveredAt, deliveredAt).Error)
	// Paid but still cooking: real money to the customer, not yet earnings.
	seedVisOrder(t, db, chefID, "preparing", "completed")

	dash := chefVisGET(t, chefVisRouter(userID), "/chef/dashboard")

	chef := models.ChefProfile{ID: chefID}
	totals, _, err := chefSettledEarnings(chef, time.Unix(0, 0),
		services.BusinessDayEnd(time.Now()), services.GetCommissionRate(database.DB))
	require.NoError(t, err)

	require.InDelta(t, totals.GrossRevenue, dash["totalEarnings"], 0.005)
	require.InDelta(t, 545.0, dash["totalEarnings"], 0.005, "food + food GST + tip, never the in-flight order")
	require.InDelta(t, totals.GrossRevenue, dash["weekRevenue"], 0.005)
}

// The week line pairs money with a count in one sentence. The money settles on
// delivery, so the count beside it has to be the delivered count — otherwise a
// chef with 5 placed and 3 delivered reads "₹1,200 · 5 orders" and the arithmetic
// does not work. weekOrders stays the placed count for the tab counters.
func TestChefDashboard_SeparatesPlacedFromSettledWeekCounts(t *testing.T) {
	db := setupChefVisDB(t)
	userID, chefID := seedVisChef(t, db)

	deliveredAt := services.CapacityDay(time.Now()).Add(13 * time.Hour)
	for range 3 {
		require.NoError(t, db.Exec(`INSERT INTO orders (id, order_number, customer_id, chef_id,
			status, payment_status, subtotal, tax, tax_food, total, created_at, delivered_at)
			VALUES (?, 'ORD-D', ?, ?, 'delivered', 'completed', 400, 20, 20, 420, ?, ?)`,
			uuid.NewString(), uuid.NewString(), chefID.String(), deliveredAt, deliveredAt).Error)
	}
	// Two more placed and paid this week, still in the kitchen.
	seedVisOrder(t, db, chefID, "preparing", "completed")
	seedVisOrder(t, db, chefID, "accepted", "completed")

	dash := chefVisGET(t, chefVisRouter(userID), "/chef/dashboard")

	require.EqualValues(t, 5, dash["weekOrders"], "the tab counter counts every paid order")
	require.EqualValues(t, 3, dash["weekSettledOrders"], "the money line counts the ones that paid out")
	require.InDelta(t, 1260.0, dash["weekRevenue"], 0.005, "3 × (400 food + 20 food GST)")
}

func TestChefSettledEarnings_EmptyWindow(t *testing.T) {
	_, chef := setupSettledEarningsDB(t)
	now := time.Now()

	totals, orders, err := chefSettledEarnings(chef, now.AddDate(0, 0, -7), now, 0.06)
	require.NoError(t, err)
	require.Zero(t, totals.OrdersCount)
	require.Zero(t, totals.NetPayout)
	require.Empty(t, orders)
}

// A cancellation levy is money the chef does not keep. The Earnings screen showed
// the payout before it, so it disagreed with both the order's payout card and the
// settlement that actually nets the levy off.
func TestChefSettledEarnings_SubtractsTheOrdersPenalty(t *testing.T) {
	db, chef := setupSettledEarningsDB(t)
	now := time.Now()

	orderID := seedSettledOrder(t, db, chef.ID, settledOrderSeed{
		subtotal: 500, taxFood: 25, deliveredAt: now.Add(-time.Hour),
	})
	require.NoError(t, db.Exec(`INSERT INTO chef_penalties (id, chef_id, kind, status, source_key, order_id, amount)
		VALUES (?, ?, 'chef_cancel', 'pending', ?, ?, ?)`,
		uuid.New().String(), chef.ID.String(),
		services.ChefCancelPenaltySourceKey(orderID), orderID.String(), 40.0).Error)

	totals, orders, err := chefSettledEarnings(chef, now.AddDate(0, 0, -7), now, 0.06)
	require.NoError(t, err)
	require.Len(t, orders, 1)

	// gross 525; commission 30; TDS 5.25 → 489.75 before the levy, 449.75 after.
	require.InDelta(t, 40.0, orders[0].Penalty, 0.005)
	require.InDelta(t, 449.75, orders[0].NetPayout, 0.005)
	require.InDelta(t, 40.0, totals.Penalties, 0.005)
	require.InDelta(t, 449.75, totals.NetPayout, 0.005)
}

// A waived levy was cancelled by an admin — it was never owed and must not be shown.
func TestChefSettledEarnings_IgnoresWaivedPenalties(t *testing.T) {
	db, chef := setupSettledEarningsDB(t)
	now := time.Now()

	orderID := seedSettledOrder(t, db, chef.ID, settledOrderSeed{
		subtotal: 500, taxFood: 25, deliveredAt: now.Add(-time.Hour),
	})
	require.NoError(t, db.Exec(`INSERT INTO chef_penalties (id, chef_id, kind, status, source_key, order_id, amount)
		VALUES (?, ?, 'chef_cancel', 'waived', ?, ?, ?)`,
		uuid.New().String(), chef.ID.String(),
		services.ChefCancelPenaltySourceKey(orderID), orderID.String(), 40.0).Error)

	totals, orders, err := chefSettledEarnings(chef, now.AddDate(0, 0, -7), now, 0.06)
	require.NoError(t, err)
	require.Zero(t, orders[0].Penalty)
	require.InDelta(t, 489.75, totals.NetPayout, 0.005)
}

// A levy larger than the order cannot make the chef owe money on it; the
// remainder stays outstanding in the ledger.
func TestChefSettledEarnings_PenaltyNeverDrivesAnOrderNegative(t *testing.T) {
	db, chef := setupSettledEarningsDB(t)
	now := time.Now()

	orderID := seedSettledOrder(t, db, chef.ID, settledOrderSeed{
		subtotal: 100, deliveredAt: now.Add(-time.Hour),
	})
	require.NoError(t, db.Exec(`INSERT INTO chef_penalties (id, chef_id, kind, status, source_key, order_id, amount)
		VALUES (?, ?, 'chef_cancel', 'pending', ?, ?, ?)`,
		uuid.New().String(), chef.ID.String(),
		services.ChefCancelPenaltySourceKey(orderID), orderID.String(), 5000.0).Error)

	totals, orders, err := chefSettledEarnings(chef, now.AddDate(0, 0, -7), now, 0.06)
	require.NoError(t, err)
	require.Zero(t, orders[0].NetPayout)
	require.Zero(t, totals.NetPayout)
}
