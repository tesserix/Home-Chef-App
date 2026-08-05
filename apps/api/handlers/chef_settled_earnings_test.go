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
		delivery_fee REAL DEFAULT 0, chef_tip REAL DEFAULT 0,
		delivery_address_state TEXT, commission_rate REAL DEFAULT 0,
		payout_hold_status TEXT, created_at DATETIME, delivered_at DATETIME,
		refunded_at DATETIME, deleted_at DATETIME)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE chef_profiles (id TEXT PRIMARY KEY, mode TEXT)`).Error)

	chef := models.ChefProfile{ID: uuid.New(), State: "Maharashtra"}
	require.NoError(t, db.Exec(`INSERT INTO chef_profiles (id, mode) VALUES (?, 'live')`, chef.ID.String()).Error)

	prev := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = prev })
	return db, chef
}

type settledOrderSeed struct {
	subtotal    float64
	taxFood     float64
	tip         float64
	deliveryFee float64
	mode        string
	createdAt   time.Time
	deliveredAt time.Time
	refundedAt  *time.Time
	deletedAt   *time.Time
	status      string
}

func seedSettledOrder(t *testing.T, db *gorm.DB, chefID uuid.UUID, s settledOrderSeed) uuid.UUID {
	t.Helper()
	if s.mode == "" {
		s.mode = "live"
	}
	if s.status == "" {
		s.status = "delivered"
	}
	if s.createdAt.IsZero() {
		s.createdAt = s.deliveredAt
	}
	id := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO orders (id, order_number, chef_id, status, mode,
		subtotal, tax, tax_food, tax_service, delivery_fee, chef_tip,
		delivery_address_state, created_at, delivered_at, refunded_at, deleted_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, 0, ?, ?, 'Maharashtra', ?, ?, ?, ?)`,
		id.String(), "HC-"+id.String()[:6], chefID.String(), s.status, s.mode,
		s.subtotal, s.taxFood, s.taxFood, s.deliveryFee, s.tip,
		s.createdAt, s.deliveredAt, s.refundedAt, s.deletedAt).Error)
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

	require.Equal(t, withFee.NetPayout, noFee.NetPayout, "the delivery fee is the driver's money")
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

func TestChefSettledEarnings_EmptyWindow(t *testing.T) {
	_, chef := setupSettledEarningsDB(t)
	now := time.Now()

	totals, orders, err := chefSettledEarnings(chef, now.AddDate(0, 0, -7), now, 0.06)
	require.NoError(t, err)
	require.Zero(t, totals.OrdersCount)
	require.Zero(t, totals.NetPayout)
	require.Empty(t, orders)
}
