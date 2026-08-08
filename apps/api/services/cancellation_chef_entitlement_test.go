package services

// cancellation_chef_entitlement_test.go — #947. The chef's retained share of a
// cancelled order moves real money, so every branch is pinned, and the
// conservation invariant (capture == refund + vendorKept + platformKept) is
// re-proved end-to-end rather than assumed from the snapshot.
//
// The table in TestCancellationEntitlement_ProductionRows is the SIX real
// production rows this issue is about, verbatim. One of them (a 100%-refunded
// order whose snapshot still claims a vendor share) is the row that would have
// paid a chef out of a refunded capture; it is here so that can never regress.

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

func setupEntitlementDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	// Hand-rolled tables (project convention: AutoMigrate can't render the
	// Postgres gen_random_uuid() default on sqlite). These mirror the REAL
	// columns the code reads — a fixture missing refund_amount/refunded_at would
	// silently confirm the bug instead of catching it.
	require.NoError(t, db.Exec(`
		CREATE TABLE orders (
			id TEXT PRIMARY KEY, order_number TEXT, chef_id TEXT, customer_id TEXT,
			status TEXT, payment_status TEXT,
			total REAL DEFAULT 0, subtotal REAL DEFAULT 0, tax REAL DEFAULT 0,
			discount REAL DEFAULT 0, service_fee REAL DEFAULT 0, delivery_fee REAL DEFAULT 0,
			refund_amount REAL DEFAULT 0, refunded_at DATETIME, cancelled_at DATETIME,
			created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
		)`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE cancellation_requests (
			id TEXT PRIMARY KEY, order_id TEXT NOT NULL UNIQUE, customer_id TEXT, chef_id TEXT,
			status TEXT, vendor_reason TEXT, refund_destination TEXT,
			food_refund_paise INTEGER DEFAULT 0, delivery_refund_paise INTEGER DEFAULT 0,
			tax_refund_paise INTEGER DEFAULT 0, refund_total_paise INTEGER DEFAULT 0,
			vendor_kept_paise INTEGER DEFAULT 0, platform_kept_paise INTEGER DEFAULT 0,
			refund_executed NUMERIC DEFAULT 0, refund_ref TEXT,
			resolved_at DATETIME, created_at DATETIME, updated_at DATETIME
		)`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE chef_bonuses (
			id TEXT PRIMARY KEY, chef_id TEXT NOT NULL, user_id TEXT NOT NULL,
			kind TEXT NOT NULL, status TEXT NOT NULL DEFAULT 'pending',
			source_key TEXT NOT NULL UNIQUE, referral_id TEXT,
			currency TEXT DEFAULT 'INR', amount REAL DEFAULT 0, reason TEXT,
			credited_statement_id TEXT, credited_at DATETIME,
			voided_by TEXT, voided_at DATETIME, void_reason TEXT,
			created_at DATETIME, updated_at DATETIME
		)`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE chef_profiles (
			id TEXT PRIMARY KEY, user_id TEXT, kitchen_name TEXT, state TEXT,
			created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
		)`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE weekly_statements (
			id TEXT PRIMARY KEY, chef_id TEXT, user_id TEXT,
			week_start DATETIME, week_end DATETIME, currency TEXT,
			orders_count INTEGER DEFAULT 0, gross_revenue REAL DEFAULT 0,
			platform_commission REAL DEFAULT 0, cgst REAL DEFAULT 0, sgst REAL DEFAULT 0,
			igst REAL DEFAULT 0, tds REAL DEFAULT 0, penalty_deductions REAL DEFAULT 0,
			bonus_additions REAL DEFAULT 0, recovery_deductions REAL DEFAULT 0, net_payout REAL DEFAULT 0,
			status TEXT, paid_at DATETIME, payout_ref TEXT, created_at DATETIME
		)`).Error)
	return db
}

type entRow struct {
	name     string
	status   models.CancellationRequestStatus
	executed bool
	kept     int
	refund   int
	platform int
	total    int // capture, paise
	refunded int // actually refunded, paise
	wantPay  bool
	wantPais int
}

// The six real production rows (#947), plus the states the code must refuse.
func productionRows() []entRow {
	return []entRow{
		{"HC26073114065439 approved/not_started", models.CancelReqApproved, true,
			2400, 26787, 1378, 30565, 26787, true, 2400},
		{"HC26073114120981 DISPUTED — held, not paid", models.CancelReqDisputed, true,
			25200, 21747, 3461, 50408, 21747, false, 0},
		{"HC26080212501971 auto_refunded, capture FULLY refunded", models.CancelReqAutoRefunded, true,
			4053, 32664, 2347, 32664, 32664, false, 0},
		{"HC26080213040996 approved/not_started", models.CancelReqApproved, true,
			4200, 43797, 2411, 50408, 43797, true, 4200},
		{"HC26080214481790 approved/not_started", models.CancelReqApproved, true,
			3200, 34347, 1837, 39384, 34347, true, 3200},
		{"HC26080214576700 approved/in_preparation", models.CancelReqApproved, true,
			32000, 4107, 3277, 39384, 4107, true, 32000},
	}
}

func TestCancellationEntitlement_ProductionRows(t *testing.T) {
	for _, r := range productionRows() {
		t.Run(r.name, func(t *testing.T) {
			e := ComputeCancellationEntitlement(r.status, r.executed, r.kept, true, r.total, r.refunded)
			assert.Equal(t, r.wantPay, e.Payable, "payable: %s", e.Reason)
			assert.Equal(t, r.wantPais, e.Paise)
			if !e.Payable {
				assert.NotEmpty(t, e.Reason, "a skip must always be explainable")
			}
			// THE safety property: a chef is never credited more than the platform
			// still holds for the order, whatever the snapshot claims.
			assert.LessOrEqual(t, e.Paise, r.total-r.refunded,
				"entitlement must never exceed the unrefunded capture")
		})
	}
}

// The total actually owed across the production set — the number that gets
// backfilled. ₹710.53 was claimed; ₹40.53 of it is phantom (a fully-refunded
// capture) and ₹252.00 is held pending a dispute.
func TestCancellationEntitlement_ProductionTotals(t *testing.T) {
	var payable, claimed int
	for _, r := range productionRows() {
		claimed += r.kept
		e := ComputeCancellationEntitlement(r.status, r.executed, r.kept, true, r.total, r.refunded)
		payable += e.Paise
	}
	assert.Equal(t, 71053, claimed, "the figure the issue reports as owed")
	assert.Equal(t, 41800, payable, "₹418.00 payable now; ₹252 held on dispute, ₹40.53 never owed")
}

func TestCancellationEntitlement_Refusals(t *testing.T) {
	base := func() (models.CancellationRequestStatus, bool, int, bool, int, int) {
		return models.CancelReqApproved, true, 2400, true, 30565, 26787
	}
	t.Run("refund not executed", func(t *testing.T) {
		_, _, k, c, tot, ref := base()
		e := ComputeCancellationEntitlement(models.CancelReqApproved, false, k, c, tot, ref)
		assert.False(t, e.Payable)
	})
	t.Run("still pending vendor", func(t *testing.T) {
		_, ex, k, c, tot, ref := base()
		e := ComputeCancellationEntitlement(models.CancelReqPendingVendor, ex, k, c, tot, ref)
		assert.False(t, e.Payable)
	})
	t.Run("routed to admin review", func(t *testing.T) {
		_, ex, k, c, tot, ref := base()
		e := ComputeCancellationEntitlement(models.CancelReqAdminReview, ex, k, c, tot, ref)
		assert.False(t, e.Payable)
	})
	t.Run("order not cancelled", func(t *testing.T) {
		s, ex, k, _, tot, ref := base()
		e := ComputeCancellationEntitlement(s, ex, k, false, tot, ref)
		assert.False(t, e.Payable)
	})
	t.Run("nothing retained", func(t *testing.T) {
		s, ex, _, c, tot, ref := base()
		e := ComputeCancellationEntitlement(s, ex, 0, c, tot, ref)
		assert.False(t, e.Payable)
	})
	t.Run("over-refunded capture", func(t *testing.T) {
		s, ex, k, c, tot, _ := base()
		e := ComputeCancellationEntitlement(s, ex, k, c, tot, tot+500)
		assert.False(t, e.Payable)
		assert.Zero(t, e.Paise)
	})
	t.Run("admin resolve makes a disputed row payable", func(t *testing.T) {
		e := ComputeCancellationEntitlement(models.CancelReqResolved, true, 25200, true, 50408, 21747)
		assert.True(t, e.Payable)
		assert.Equal(t, 25200, e.Paise)
	})
}

// seedEntitlementCase writes one order + cancellation request + chef.
func seedEntitlementCase(t *testing.T, db *gorm.DB, r entRow) (models.Order, models.CancellationRequest, uuid.UUID) {
	t.Helper()
	chefID, userID, orderID := uuid.New(), uuid.New(), uuid.New()
	require.NoError(t, db.Exec(
		`INSERT INTO chef_profiles (id, user_id, kitchen_name) VALUES (?,?,?)`,
		chefID.String(), userID.String(), "Saffron Home Kitchen").Error)
	require.NoError(t, db.Exec(
		`INSERT INTO orders (id, order_number, chef_id, status, payment_status, total, refund_amount, refunded_at)
		 VALUES (?,?,?,?,?,?,?,CURRENT_TIMESTAMP)`,
		orderID.String(), "HC-"+orderID.String()[:8], chefID.String(),
		string(models.OrderStatusCancelled), string(models.PaymentRefunded),
		float64(r.total)/100, float64(r.refunded)/100).Error)
	require.NoError(t, db.Exec(
		`INSERT INTO cancellation_requests (id, order_id, chef_id, status, vendor_reason,
		   refund_total_paise, vendor_kept_paise, platform_kept_paise, refund_executed)
		 VALUES (?,?,?,?,?,?,?,?,?)`,
		uuid.New().String(), orderID.String(), chefID.String(), string(r.status), "not_started",
		r.refund, r.kept, r.platform, r.executed).Error)

	var order models.Order
	require.NoError(t, db.First(&order, "id = ?", orderID.String()).Error)
	var cr models.CancellationRequest
	require.NoError(t, db.First(&cr, "order_id = ?", orderID.String()).Error)
	return order, cr, userID
}

func TestRaiseCancellationRetainedBonus_CreditsAndIsIdempotent(t *testing.T) {
	db := setupEntitlementDB(t)
	row := productionRows()[5] // in_preparation, ₹320 retained
	order, cr, userID := seedEntitlementCase(t, db, row)

	require.NoError(t, RaiseCancellationRetainedBonus(db, &order, &cr))

	var bonuses []models.ChefBonus
	require.NoError(t, db.Find(&bonuses).Error)
	require.Len(t, bonuses, 1)
	assert.Equal(t, models.ChefBonusCancellationRetained, bonuses[0].Kind)
	assert.Equal(t, models.ChefBonusPending, bonuses[0].Status)
	assert.Equal(t, 320.0, bonuses[0].Amount)
	assert.Equal(t, userID, bonuses[0].UserID)
	assert.Equal(t, CancellationEntitlementSourceKey(order.ID), bonuses[0].SourceKey)

	// Re-driving (inline retry + sweep) credits exactly once.
	require.NoError(t, RaiseCancellationRetainedBonus(db, &order, &cr))
	require.NoError(t, RaiseCancellationRetainedBonus(db, &order, &cr))
	var count int64
	db.Model(&models.ChefBonus{}).Count(&count)
	assert.EqualValues(t, 1, count, "a chef is credited once, however many times the path re-drives")
}

func TestRaiseCancellationRetainedBonus_RefusesRefundedCapture(t *testing.T) {
	db := setupEntitlementDB(t)
	// The real production row: 100% refunded, snapshot still claims ₹40.53.
	order, cr, _ := seedEntitlementCase(t, db, productionRows()[2])

	require.NoError(t, RaiseCancellationRetainedBonus(db, &order, &cr),
		"a non-payable row is a normal outcome, not an error")

	var count int64
	db.Model(&models.ChefBonus{}).Count(&count)
	assert.EqualValues(t, 0, count, "must never pay a chef out of a fully-refunded capture")
}

func TestRaiseCancellationRetainedBonus_HoldsDisputed(t *testing.T) {
	db := setupEntitlementDB(t)
	order, cr, _ := seedEntitlementCase(t, db, productionRows()[1])

	require.NoError(t, RaiseCancellationRetainedBonus(db, &order, &cr))
	var count int64
	db.Model(&models.ChefBonus{}).Count(&count)
	assert.EqualValues(t, 0, count, "a contested tier can still move money back to the customer")

	// Once an admin settles it, the same call pays.
	require.NoError(t, db.Model(&models.CancellationRequest{}).Where("order_id = ?", order.ID).
		Update("status", string(models.CancelReqResolved)).Error)
	require.NoError(t, db.First(&cr, "order_id = ?", order.ID).Error)
	require.NoError(t, RaiseCancellationRetainedBonus(db, &order, &cr))

	var bonus models.ChefBonus
	require.NoError(t, db.First(&bonus).Error)
	assert.Equal(t, 252.0, bonus.Amount)
}

// Conservation, end to end: what the customer got back plus what the chef is
// credited can never exceed what was captured — for every production row.
func TestCancellationEntitlement_ConservationHolds(t *testing.T) {
	db := setupEntitlementDB(t)
	for _, r := range productionRows() {
		order, cr, _ := seedEntitlementCase(t, db, r)
		require.NoError(t, RaiseCancellationRetainedBonus(db, &order, &cr))

		var bonus models.ChefBonus
		chefPaise := 0
		if err := db.First(&bonus, "source_key = ?", CancellationEntitlementSourceKey(order.ID)).Error; err == nil {
			chefPaise = ToPaise(bonus.Amount)
		}
		assert.LessOrEqual(t, r.refunded+chefPaise, r.total,
			"%s: refunded + chef credit must not exceed the capture", r.name)
	}
}

// The sweep is the backfill: it credits every settled cancellation that has no
// credit yet, skips the ones that must not be paid, and is safe to run forever.
func TestSweepCancellationChefEntitlements_BackfillsExactlyWhatIsOwed(t *testing.T) {
	db := setupEntitlementDB(t)
	orig := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = orig })

	for _, r := range productionRows() {
		seedEntitlementCase(t, db, r)
	}

	SweepCancellationChefEntitlements()

	var bonuses []models.ChefBonus
	require.NoError(t, db.Where("kind = ?", models.ChefBonusCancellationRetained).Find(&bonuses).Error)
	assert.Len(t, bonuses, 4, "4 payable; the disputed and fully-refunded rows are skipped")

	total := 0.0
	for _, b := range bonuses {
		total += b.Amount
	}
	assert.Equal(t, 418.0, total, "₹418.00 — not the ₹710.53 the snapshots claim")

	// Re-running the sweep (every 2 minutes, forever) credits nothing more.
	SweepCancellationChefEntitlements()
	SweepCancellationChefEntitlements()
	var count int64
	db.Model(&models.ChefBonus{}).Count(&count)
	assert.EqualValues(t, 4, count, "the sweep is idempotent across runs")
}

// The credit reaches the chef through the statement the same way every other
// bonus does — the half of #947 that was missing entirely.
func TestCancellationEntitlement_RidesTheWeeklyStatement(t *testing.T) {
	db := setupEntitlementDB(t)
	order, cr, userID := seedEntitlementCase(t, db, productionRows()[5])
	require.NoError(t, RaiseCancellationRetainedBonus(db, &order, &cr))

	stmt := models.WeeklyStatement{ChefID: cr.ChefID, UserID: userID, NetPayout: 1000}
	stmt.ID = uuid.New()
	require.NoError(t, db.Create(&stmt).Error)

	credited, err := ApplyChefBonusesToStatement(db, &stmt)
	require.NoError(t, err)
	assert.Equal(t, 320.0, credited)
	assert.Equal(t, 1320.0, stmt.NetPayout, "the retained share reaches the chef's payout")

	var bonus models.ChefBonus
	require.NoError(t, db.First(&bonus).Error)
	assert.Equal(t, models.ChefBonusCredited, bonus.Status)
	require.NotNil(t, bonus.CreditedStatementID)
	assert.Equal(t, stmt.ID, *bonus.CreditedStatementID)
}
