package handlers

// chef_counted_orders_test.go — the shared definition of "orders that count as this
// kitchen's business".
//
// The dashboard and the Analytics screen used to disagree wildly for the same week —
// Analytics reported 47 orders / ₹25,257 against a dashboard showing 19 / ₹5,348 —
// because every analytics aggregate filtered on chef_id alone and so counted abandoned
// checkouts, failed payments and sandbox orders as revenue. Both surfaces now build on
// chefCountedOrdersSQL; these tests pin what it admits, and that the builder path
// (chefVisibleOrders) and the raw-SQL path still select the same rows.

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
)

func setupCountedOrdersDB(t *testing.T) (*gorm.DB, uuid.UUID) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	for _, s := range []string{
		`CREATE TABLE orders (id TEXT PRIMARY KEY, chef_id TEXT, customer_id TEXT,
			payment_status TEXT, status TEXT, mode TEXT DEFAULT 'live', total REAL,
			refund_amount REAL DEFAULT 0, created_at DATETIME, deleted_at DATETIME)`,
		`CREATE TABLE chef_profiles (id TEXT PRIMARY KEY, mode TEXT)`,
		`CREATE TABLE meal_plan_days (id TEXT PRIMARY KEY, order_id TEXT)`,
	} {
		require.NoError(t, db.Exec(s).Error)
	}
	chefID := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO chef_profiles (id, mode) VALUES (?, 'live')`, chefID.String()).Error)

	prev := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = prev })
	return db, chefID
}

// order writes one order and returns its id.
func seedCountedOrder(t *testing.T, db *gorm.DB, chefID uuid.UUID, pay models.PaymentStatus, mode string, total, refunded float64) uuid.UUID {
	t.Helper()
	id := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO orders (id, chef_id, customer_id, payment_status, status, mode,
		total, refund_amount, created_at) VALUES (?, ?, ?, ?, 'delivered', ?, ?, ?, ?)`,
		id.String(), chefID.String(), uuid.New().String(), string(pay), mode, total, refunded, time.Now()).Error)
	return id
}

// countedRevenue runs the raw-SQL path the analytics aggregates use.
func countedRevenue(t *testing.T, db *gorm.DB, chefID uuid.UUID) (int64, float64) {
	t.Helper()
	var row struct {
		Orders  int64
		Revenue float64
	}
	require.NoError(t, db.Raw(`SELECT COUNT(*) AS orders, `+chefCountedRevenueExpr("o")+` AS revenue
		FROM orders o WHERE `+chefCountedOrdersSQL("o"), chefID, chefID).Scan(&row).Error)
	return row.Orders, row.Revenue
}

// An abandoned checkout is not revenue. This is the whole bug: ~₹20k of never-captured
// money was being reported to the chef as their week's takings.
func TestCountedOrders_ExcludesUnpaidAndFailed(t *testing.T) {
	db, chefID := setupCountedOrdersDB(t)

	seedCountedOrder(t, db, chefID, models.PaymentCompleted, "live", 500, 0)
	seedCountedOrder(t, db, chefID, models.PaymentCompleted, "live", 700, 0)
	seedCountedOrder(t, db, chefID, models.PaymentPending, "live", 5000, 0) // abandoned checkout
	seedCountedOrder(t, db, chefID, models.PaymentFailed, "live", 3000, 0)  // declined card

	orders, revenue := countedRevenue(t, db, chefID)
	require.EqualValues(t, 2, orders)
	require.EqualValues(t, 1200, revenue)
}

// Sandbox orders never mix into a live kitchen's figures.
func TestCountedOrders_ExcludesTheOtherMode(t *testing.T) {
	db, chefID := setupCountedOrdersDB(t)

	seedCountedOrder(t, db, chefID, models.PaymentCompleted, "live", 500, 0)
	seedCountedOrder(t, db, chefID, models.PaymentCompleted, "test", 9000, 0)

	orders, revenue := countedRevenue(t, db, chefID)
	require.EqualValues(t, 1, orders)
	require.EqualValues(t, 500, revenue)

	// Flip the kitchen into test: it now sees its sandbox and nothing else.
	require.NoError(t, db.Exec(`UPDATE chef_profiles SET mode = 'test' WHERE id = ?`, chefID.String()).Error)
	orders, revenue = countedRevenue(t, db, chefID)
	require.EqualValues(t, 1, orders)
	require.EqualValues(t, 9000, revenue)
}

// A refunded order still happened — it keeps its place in the count and contributes only
// the money the kitchen actually kept, so counts and revenue describe one population.
func TestCountedOrders_RefundedCountsButNetsOutTheRefund(t *testing.T) {
	db, chefID := setupCountedOrdersDB(t)

	seedCountedOrder(t, db, chefID, models.PaymentCompleted, "live", 1000, 0)
	seedCountedOrder(t, db, chefID, models.PaymentRefunded, "live", 800, 800)  // fully refunded
	seedCountedOrder(t, db, chefID, models.PaymentCompleted, "live", 600, 200) // partial refund

	orders, revenue := countedRevenue(t, db, chefID)
	require.EqualValues(t, 3, orders)
	require.EqualValues(t, 1400, revenue, "1000 + 0 + 400")
}

// A confirmed meal-plan day reaches the chef even while payment is pending (escrow off),
// so it must count as real work.
func TestCountedOrders_AdmitsMealPlanDays(t *testing.T) {
	db, chefID := setupCountedOrdersDB(t)

	dayOrder := seedCountedOrder(t, db, chefID, models.PaymentPending, "live", 350, 0)
	require.NoError(t, db.Exec(`INSERT INTO meal_plan_days (id, order_id) VALUES (?, ?)`,
		uuid.New().String(), dayOrder.String()).Error)
	seedCountedOrder(t, db, chefID, models.PaymentPending, "live", 5000, 0) // plain abandoned

	orders, revenue := countedRevenue(t, db, chefID)
	require.EqualValues(t, 1, orders)
	require.EqualValues(t, 350, revenue)
}

// The anti-drift guarantee: the dashboard's builder path and the analytics raw-SQL path
// must select the SAME rows. They diverging silently is what produced two different
// numbers for one week on two screens.
func TestCountedOrders_BuilderAndRawPathsAgree(t *testing.T) {
	db, chefID := setupCountedOrdersDB(t)

	seedCountedOrder(t, db, chefID, models.PaymentCompleted, "live", 500, 0)
	seedCountedOrder(t, db, chefID, models.PaymentRefunded, "live", 800, 300)
	seedCountedOrder(t, db, chefID, models.PaymentPending, "live", 5000, 0)
	seedCountedOrder(t, db, chefID, models.PaymentFailed, "live", 3000, 0)
	seedCountedOrder(t, db, chefID, models.PaymentCompleted, "test", 9000, 0)
	seedCountedOrder(t, db, chefID, models.PaymentCompleted, "live", 250, 0)

	var builderOrders int64
	chefVisibleOrders(chefID).Count(&builderOrders)
	var builderRevenue float64
	chefVisibleOrders(chefID).Select(chefCountedRevenueExpr("orders")).Scan(&builderRevenue)

	rawOrders, rawRevenue := countedRevenue(t, db, chefID)
	require.Equal(t, rawOrders, builderOrders)
	require.Equal(t, rawRevenue, builderRevenue)
	require.EqualValues(t, 3, builderOrders)
	require.EqualValues(t, 1250, builderRevenue, "500 + 500 + 250")
}
