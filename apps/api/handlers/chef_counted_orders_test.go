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
			subtotal REAL DEFAULT 0, delivery_fee REAL DEFAULT 0, tax REAL DEFAULT 0,
			tax_food REAL DEFAULT 0, tax_service REAL DEFAULT 0, tax_inclusive BOOLEAN DEFAULT FALSE,
			currency TEXT DEFAULT 'INR',
			chef_funded_discount REAL DEFAULT 0, chef_tip REAL DEFAULT 0,
			refund_amount REAL DEFAULT 0, created_at DATETIME, deleted_at DATETIME)`,
		`CREATE TABLE chef_profiles (id TEXT PRIMARY KEY, mode TEXT)`,
		`CREATE TABLE meal_plan_days (id TEXT PRIMARY KEY, order_id TEXT)`,
		`CREATE TABLE cancellation_requests (id TEXT PRIMARY KEY, order_id TEXT, vendor_kept_paise INTEGER)`,
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

// seedCountedOrder writes one all-food order — no delivery fee, no platform fee, no tax —
// so the chef's gross equals the order total and these population tests stay about WHICH
// rows count. seedPricedOrder is the one to reach for when the split matters.
func seedCountedOrder(t *testing.T, db *gorm.DB, chefID uuid.UUID, pay models.PaymentStatus, mode string, total, refunded float64) uuid.UUID {
	t.Helper()
	id := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO orders (id, chef_id, customer_id, payment_status, status, mode,
		total, subtotal, refund_amount, created_at) VALUES (?, ?, ?, ?, 'delivered', ?, ?, ?, ?, ?)`,
		id.String(), chefID.String(), uuid.New().String(), string(pay), mode, total, total, refunded, time.Now()).Error)
	return id
}

// seedPricedOrder writes an order with the per-supply price split populated, the way
// checkout stamps it.
func seedPricedOrder(t *testing.T, db *gorm.DB, chefID uuid.UUID,
	subtotal, deliveryFee, taxFood, taxService, taxDelivery, platformFee, tip, refunded float64) uuid.UUID {
	t.Helper()
	id := uuid.New()
	total := subtotal + deliveryFee + platformFee + taxFood + taxService + taxDelivery + tip
	require.NoError(t, db.Exec(`INSERT INTO orders (id, chef_id, customer_id, payment_status, status, mode,
		total, subtotal, delivery_fee, tax, tax_food, tax_service, chef_tip, refund_amount, created_at)
		VALUES (?, ?, ?, 'completed', 'delivered', 'live', ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id.String(), chefID.String(), uuid.New().String(), total, subtotal, deliveryFee,
		taxFood+taxService+taxDelivery, taxFood, taxService, tip, refunded, time.Now()).Error)
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

// ── What the money expression is worth, not just which rows it covers ────────────────
//
// The dashboard hero is labelled "Total earnings" and taps through to payouts. It used
// to sum the CUSTOMER's order value, so it carried the driver's delivery fee, the
// platform's fee, and the platform's GST on both — and disagreed with the Earnings
// screen, the weekly statement and the payout for the same week.

// The observed order HC26080323440359, to the paisa: ₹393.05 charged, ₹336.00 earned.
func TestCountedRevenue_IsChefGrossNotOrderValue(t *testing.T) {
	db, chefID := setupCountedOrdersDB(t)

	seedPricedOrder(t, db, chefID, 320, 39.12, 16.00, 2.44, 1.96, 13.53, 0, 0)

	orders, revenue := countedRevenue(t, db, chefID)
	require.EqualValues(t, 1, orders)
	require.InDelta(t, 336.00, revenue, 0.005, "subtotal 320 + food GST 16 — not the ₹393.05 total")
}

// The tip is the chef's; the delivery fee is the driver's even when the chef delivers.
func TestCountedRevenue_IncludesTipExcludesDelivery(t *testing.T) {
	db, chefID := setupCountedOrdersDB(t)

	seedPricedOrder(t, db, chefID, 320, 39.12, 16.00, 2.44, 1.96, 13.53, 30, 0)

	_, revenue := countedRevenue(t, db, chefID)
	require.InDelta(t, 366.00, revenue, 0.005, "320 + 16 + tip 30")
}

func TestCountedRevenue_InclusiveMenuPriceUsesSavedNetSubtotal(t *testing.T) {
	for _, currency := range []string{"AUD", "NZD"} {
		t.Run(currency, func(t *testing.T) {
			db, chefID := setupCountedOrdersDB(t)
			id := seedPricedOrder(t, db, chefID, 100, 10, 10, 1, 1, 5, 5, 0)
			require.NoError(t, db.Exec(`UPDATE orders SET tax_inclusive = TRUE, currency = ?, total = 120 WHERE id = ?`, currency, id).Error)
			orders, revenue := countedRevenue(t, db, chefID)
			require.EqualValues(t, 1, orders)
			require.InDelta(t, 115, revenue, 0.005)
		})
	}
}

// Rows placed before tax was split per supply have no snapshot, so they keep the whole
// order tax — the figure they were settled against. ChefTaxOf's fallback, in SQL.
func TestCountedRevenue_LegacyRowKeepsWholeOrderTax(t *testing.T) {
	db, chefID := setupCountedOrdersDB(t)

	require.NoError(t, db.Exec(`INSERT INTO orders (id, chef_id, customer_id, payment_status, status,
		mode, total, subtotal, delivery_fee, tax, created_at)
		VALUES (?, ?, ?, 'completed', 'delivered', 'live', 393.84, 320, 39.12, 18.75, ?)`,
		uuid.New().String(), chefID.String(), uuid.New().String(), time.Now()).Error)

	_, revenue := countedRevenue(t, db, chefID)
	require.InDelta(t, 338.75, revenue, 0.005, "320 + the whole 18.75, un-split")
}

// A cancelled order earns the chef the share epic #475 retained for them — NOT their
// full gross, and NOT `total − refund`, which also hands them the platform's kept fee.
// The observed 40%-tier cancellation: customer got ₹175.48 back, the chef is owed ₹192,
// and the platform kept ₹25.57.
func TestCountedRevenue_CancelledOrderEarnsTheRetainedShare(t *testing.T) {
	db, chefID := setupCountedOrdersDB(t)

	id := seedPricedOrder(t, db, chefID, 320, 39.12, 16.00, 2.44, 1.96, 13.53, 0, 175.48)
	require.NoError(t, db.Exec(`UPDATE orders SET payment_status = 'refunded', status = 'cancelled' WHERE id = ?`, id.String()).Error)
	require.NoError(t, db.Exec(`INSERT INTO cancellation_requests (id, order_id, vendor_kept_paise) VALUES (?, ?, 19200)`,
		uuid.New().String(), id.String()).Error)

	orders, revenue := countedRevenue(t, db, chefID)
	require.EqualValues(t, 1, orders, "the cancellation happened — it keeps its place in the count")
	require.InDelta(t, 192.00, revenue, 0.005, "not 336 (full gross) and not 217.57 (total − refund)")
}

// A fully-refunded cancellation earns the chef nothing.
func TestCountedRevenue_FullyRefundedCancellationEarnsNothing(t *testing.T) {
	db, chefID := setupCountedOrdersDB(t)

	id := seedPricedOrder(t, db, chefID, 1200, 39.12, 60.00, 9.13, 1.96, 50.75, 0, 1301.08)
	require.NoError(t, db.Exec(`UPDATE orders SET payment_status = 'refunded', status = 'cancelled' WHERE id = ?`, id.String()).Error)
	require.NoError(t, db.Exec(`INSERT INTO cancellation_requests (id, order_id, vendor_kept_paise) VALUES (?, ?, 0)`,
		uuid.New().String(), id.String()).Error)

	_, revenue := countedRevenue(t, db, chefID)
	require.InDelta(t, 0, revenue, 0.005)
}

// The solvency cap from ComputeCancellationEntitlement, applied to the same rows.
// Production carries a fully-refunded order whose snapshot still claims ₹40.53 for the
// vendor; reporting it as revenue would credit a chef out of money that went back to the
// customer.
func TestCountedRevenue_RetainedShareCappedAtWhatIsStillHeld(t *testing.T) {
	db, chefID := setupCountedOrdersDB(t)

	id := seedPricedOrder(t, db, chefID, 320, 39.12, 15.55, 0, 0, 0, 0, 374.67)
	require.NoError(t, db.Exec(`UPDATE orders SET payment_status = 'refunded', status = 'cancelled' WHERE id = ?`, id.String()).Error)
	require.NoError(t, db.Exec(`INSERT INTO cancellation_requests (id, order_id, vendor_kept_paise) VALUES (?, ?, 4053)`,
		uuid.New().String(), id.String()).Error)

	_, revenue := countedRevenue(t, db, chefID)
	require.InDelta(t, 0, revenue, 0.005, "the whole capture went back — nothing is held to pay ₹40.53 from")
}

// A refund that only gives back the delivery estimate's overshoot (#703) costs the chef
// nothing: the ceiling still clears their gross.
func TestCountedRevenue_DeliveryOnlyRefundDoesNotTouchChefGross(t *testing.T) {
	db, chefID := setupCountedOrdersDB(t)

	seedPricedOrder(t, db, chefID, 320, 39.12, 16.00, 2.44, 1.96, 13.53, 0, 10)

	_, revenue := countedRevenue(t, db, chefID)
	require.InDelta(t, 336.00, revenue, 0.005)
}

// Revenue can never be negative, whatever a row claims.
func TestCountedRevenue_NeverNegative(t *testing.T) {
	db, chefID := setupCountedOrdersDB(t)

	seedCountedOrder(t, db, chefID, models.PaymentRefunded, "live", 500, 900)

	_, revenue := countedRevenue(t, db, chefID)
	require.EqualValues(t, 0, revenue)
}
