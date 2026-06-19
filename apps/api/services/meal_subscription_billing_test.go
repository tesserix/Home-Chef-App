package services

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

// meal_subscription_billing_test.go — #281. The credit + cycle-charge math and
// next-cycle invoice generation with carried skip/missed credits.

func TestPerDayMealCost(t *testing.T) {
	assert.Equal(t, 240.0, PerDayMealCost(120, 2)) // lunch + dinner @ 120
	assert.Equal(t, 120.0, PerDayMealCost(120, 1))
	assert.Equal(t, 0.0, PerDayMealCost(120, 0))
	assert.Equal(t, 0.0, PerDayMealCost(-5, 2))
}

func TestComputeMealCycleCharge(t *testing.T) {
	t.Run("no credit → full charge", func(t *testing.T) {
		charge, rem := ComputeMealCycleCharge(1200, 0)
		assert.Equal(t, 1200.0, charge)
		assert.Equal(t, 0.0, rem)
	})
	t.Run("partial credit reduces the charge", func(t *testing.T) {
		charge, rem := ComputeMealCycleCharge(1200, 240) // 1 missed day credited
		assert.Equal(t, 960.0, charge)
		assert.Equal(t, 0.0, rem)
	})
	t.Run("credit exceeding the cycle carries forward", func(t *testing.T) {
		charge, rem := ComputeMealCycleCharge(1000, 1500)
		assert.Equal(t, 0.0, charge)
		assert.Equal(t, 500.0, rem)
	})
}

func setupMealBillingDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	stmts := []string{
		`CREATE TABLE meal_subscriptions (id TEXT PRIMARY KEY, customer_id TEXT, chef_id TEXT, cycle_amount REAL,
			credit_balance REAL DEFAULT 0, currency TEXT DEFAULT 'INR', status TEXT, cadence TEXT DEFAULT 'weekly',
			gateway_sub_id TEXT, current_period_start DATETIME, current_period_end DATETIME, updated_at DATETIME, deleted_at DATETIME)`,
		`CREATE TABLE meal_subscription_invoices (id TEXT PRIMARY KEY, meal_subscription_id TEXT, invoice_number TEXT,
			status TEXT, cycle_amount REAL, credit_applied REAL, amount REAL, tax_amount REAL, total_amount REAL,
			currency TEXT, period_start DATETIME, period_end DATETIME, gateway_payment_id TEXT, paid_at DATETIME,
			created_at DATETIME, updated_at DATETIME, deleted_at DATETIME)`,
		// GetTaxConfig (for the invoice tax) reads platform_settings off the global DB.
		`CREATE TABLE platform_settings (id TEXT PRIMARY KEY, key TEXT UNIQUE, value TEXT, type TEXT, updated_by TEXT, updated_at DATETIME)`,
	}
	for _, s := range stmts {
		require.NoError(t, db.Exec(s).Error)
	}
	prev := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = prev })
	return db
}

func TestGenerateMealCycleInvoice(t *testing.T) {
	db := setupMealBillingDB(t)
	subID := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO meal_subscriptions (id, cycle_amount, credit_balance, currency, status)
		VALUES (?, 1200, 240, 'INR', 'active')`, subID.String()).Error)
	sub := &models.MealSubscription{ID: subID, CycleAmount: 1200, CreditBalance: 240, Currency: "INR"}

	start := time.Now()
	inv, err := GenerateMealCycleInvoice(db, sub, start, start.AddDate(0, 0, 7))
	require.NoError(t, err)

	// 1200 gross − 240 credit = 960 net; GST 18% of 960 = 172.8; total 1132.8.
	assert.Equal(t, 1200.0, inv.CycleAmount)
	assert.Equal(t, 240.0, inv.CreditApplied)
	assert.Equal(t, 960.0, inv.Amount)
	assert.Equal(t, 172.8, inv.TaxAmount)
	assert.Equal(t, 1132.8, inv.TotalAmount)
	assert.Equal(t, models.MealInvoiceStatusPending, inv.Status)

	// The applied credit was consumed on the subscription.
	var credit float64
	db.Raw(`SELECT credit_balance FROM meal_subscriptions WHERE id = ?`, subID.String()).Scan(&credit)
	assert.Equal(t, 0.0, credit)
	assert.Equal(t, 0.0, sub.CreditBalance)
}

func TestActivateMealSubscriptionOnCharge(t *testing.T) {
	db := setupMealBillingDB(t)
	subID := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO meal_subscriptions (id, cycle_amount, credit_balance, currency, status, cadence, gateway_sub_id)
		VALUES (?, 1000, 0, 'INR', 'trialing', 'weekly', 'sub_RZP123')`, subID.String()).Error)

	inv, err := ActivateMealSubscriptionOnCharge(db, "sub_RZP123", "pay_ABC")
	require.NoError(t, err)
	require.NotNil(t, inv)
	assert.Equal(t, models.MealInvoiceStatusPaid, statusOf(t, db, inv.ID))

	// Subscription is now active with a forward period.
	var status string
	db.Raw(`SELECT status FROM meal_subscriptions WHERE id = ?`, subID.String()).Scan(&status)
	assert.Equal(t, models.MealSubStatusActive, status)

	// A duplicate charge for the same cycle is a no-op (idempotent).
	dup, err := ActivateMealSubscriptionOnCharge(db, "sub_RZP123", "pay_ABC")
	require.NoError(t, err)
	assert.Nil(t, dup)
	var invoiceCount int64
	db.Model(&models.MealSubscriptionInvoice{}).Where("meal_subscription_id = ?", subID.String()).Count(&invoiceCount)
	assert.Equal(t, int64(1), invoiceCount)

	// Unknown gateway id → no match, no error (caller falls through to other tables).
	none, err := ActivateMealSubscriptionOnCharge(db, "sub_UNKNOWN", "pay_X")
	require.NoError(t, err)
	assert.Nil(t, none)
}

func TestHaltMealSubscriptionOnFailure(t *testing.T) {
	db := setupMealBillingDB(t)
	subID := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO meal_subscriptions (id, cycle_amount, status, gateway_sub_id)
		VALUES (?, 1000, 'active', 'sub_HALT')`, subID.String()).Error)
	assert.True(t, HaltMealSubscriptionOnFailure(db, "sub_HALT"))
	var status string
	db.Raw(`SELECT status FROM meal_subscriptions WHERE id = ?`, subID.String()).Scan(&status)
	assert.Equal(t, models.MealSubStatusPastDue, status)
	assert.False(t, HaltMealSubscriptionOnFailure(db, "sub_NONE"))
}

func statusOf(t *testing.T, db *gorm.DB, id uuid.UUID) string {
	t.Helper()
	var s string
	db.Raw(`SELECT status FROM meal_subscription_invoices WHERE id = ?`, id.String()).Scan(&s)
	return s
}

func TestCreditMealSubscription(t *testing.T) {
	db := setupMealBillingDB(t)
	subID := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO meal_subscriptions (id, cycle_amount, credit_balance, status)
		VALUES (?, 1200, 0, 'active')`, subID.String()).Error)

	require.NoError(t, CreditMealSubscription(db, subID, 240)) // a missed day
	require.NoError(t, CreditMealSubscription(db, subID, 120)) // a skipped lunch
	var credit float64
	db.Raw(`SELECT credit_balance FROM meal_subscriptions WHERE id = ?`, subID.String()).Scan(&credit)
	assert.Equal(t, 360.0, credit)
}
