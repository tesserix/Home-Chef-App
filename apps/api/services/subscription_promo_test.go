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

// subscription_promo_test.go — #269. Promo applied to a platform subscription:
// platform-funded only, one-time on the first invoice, budget/usage race-safe via
// the shared #39 engine.

func setupSubPromoDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	stmts := []string{
		`CREATE TABLE promo_codes (id TEXT PRIMARY KEY, code TEXT, discount_type TEXT, discount_value REAL,
			max_discount REAL DEFAULT 0, min_order_amount REAL DEFAULT 0, usage_limit INTEGER DEFAULT 0,
			usage_count INTEGER DEFAULT 0, per_user_limit INTEGER DEFAULT 0, valid_from DATETIME, valid_until DATETIME,
			is_active INTEGER DEFAULT 1, applicable_to TEXT DEFAULT 'all', funding_source TEXT DEFAULT 'platform',
			chef_id TEXT, budget_cap REAL DEFAULT 0, budget_spent REAL DEFAULT 0, updated_at DATETIME, deleted_at DATETIME)`,
		`CREATE TABLE promo_code_usages (id TEXT PRIMARY KEY, promo_code_id TEXT, user_id TEXT, order_id TEXT,
			subscription_id TEXT, discount REAL, used_at DATETIME)`,
		`CREATE TABLE subscriptions (id TEXT PRIMARY KEY, user_id TEXT, promo_code_id TEXT, updated_at DATETIME, deleted_at DATETIME)`,
		`CREATE TABLE orders (delivery_address_line1_enc text DEFAULT '', delivery_address_line2_enc text DEFAULT '', id TEXT PRIMARY KEY, customer_id TEXT, deleted_at DATETIME)`,
	}
	for _, s := range stmts {
		require.NoError(t, db.Exec(s).Error)
	}
	prev := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = prev })
	return db
}

func seedSubPromo(t *testing.T, db *gorm.DB, funding string, budgetCap, budgetSpent float64) uuid.UUID {
	t.Helper()
	id := uuid.New()
	require.NoError(t, db.Exec(
		`INSERT INTO promo_codes (id, code, discount_type, discount_value, is_active, valid_from, applicable_to, funding_source, budget_cap, budget_spent)
		 VALUES (?, 'WELCOME20', 'percentage', 20, 1, ?, 'all', ?, ?, ?)`,
		id.String(), time.Now().Add(-time.Hour), funding, budgetCap, budgetSpent).Error)
	return id
}

func TestValidateSubscriptionPromo(t *testing.T) {
	t.Run("platform promo previews the discount", func(t *testing.T) {
		db := setupSubPromoDB(t)
		seedSubPromo(t, db, models.PromoFundingPlatform, 0, 0)
		discount, promo, err := ValidateSubscriptionPromo(db, "welcome20", uuid.New(), 1000)
		require.NoError(t, err)
		assert.Equal(t, 200.0, discount) // 20% of 1000
		assert.Equal(t, "WELCOME20", promo.Code)
	})

	t.Run("chef-funded promo is rejected for subscriptions", func(t *testing.T) {
		db := setupSubPromoDB(t)
		seedSubPromo(t, db, models.PromoFundingChef, 0, 0)
		_, _, err := ValidateSubscriptionPromo(db, "WELCOME20", uuid.New(), 1000)
		assert.ErrorIs(t, err, ErrPromoNotForSubscriptions)
	})

	t.Run("unknown code is rejected", func(t *testing.T) {
		db := setupSubPromoDB(t)
		_, _, err := ValidateSubscriptionPromo(db, "NOPE", uuid.New(), 1000)
		assert.ErrorIs(t, err, ErrPromoInactive)
	})
}

func TestApplySubscriptionPromoToInvoice(t *testing.T) {
	apply := func(db *gorm.DB, sub *models.Subscription, plan float64) float64 {
		var discount float64
		require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
			discount = ApplySubscriptionPromoToInvoice(tx, sub, plan)
			return nil
		}))
		return discount
	}

	t.Run("applies discount, records subscription usage, clears the one-time code", func(t *testing.T) {
		db := setupSubPromoDB(t)
		promoID := seedSubPromo(t, db, models.PromoFundingPlatform, 0, 0)
		user, subID := uuid.New(), uuid.New()
		require.NoError(t, db.Exec(`INSERT INTO subscriptions (id, user_id, promo_code_id) VALUES (?, ?, ?)`,
			subID.String(), user.String(), promoID.String()).Error)
		sub := &models.Subscription{ID: subID, UserID: user, PromoCodeID: &promoID}

		discount := apply(db, sub, 1000)
		assert.Equal(t, 200.0, discount)

		// Usage recorded against the subscription (not an order).
		var usageCount int64
		var subUsage, orderUsage int64
		db.Raw(`SELECT count(*) FROM promo_code_usages WHERE promo_code_id = ?`, promoID.String()).Scan(&usageCount)
		db.Raw(`SELECT count(*) FROM promo_code_usages WHERE subscription_id = ?`, subID.String()).Scan(&subUsage)
		db.Raw(`SELECT count(*) FROM promo_code_usages WHERE order_id IS NOT NULL AND order_id != ''`).Scan(&orderUsage)
		assert.Equal(t, int64(1), usageCount)
		assert.Equal(t, int64(1), subUsage)
		assert.Equal(t, int64(0), orderUsage)

		// Budget + usage claimed; promo cleared from the subscription (one-time).
		var spent float64
		var clearedPromo string
		db.Raw(`SELECT budget_spent FROM promo_codes WHERE id = ?`, promoID.String()).Scan(&spent)
		db.Raw(`SELECT COALESCE(promo_code_id,'') FROM subscriptions WHERE id = ?`, subID.String()).Scan(&clearedPromo)
		assert.Equal(t, 200.0, spent)
		assert.Equal(t, "", clearedPromo)
		assert.Nil(t, sub.PromoCodeID)
	})

	t.Run("no promo → no discount", func(t *testing.T) {
		db := setupSubPromoDB(t)
		sub := &models.Subscription{ID: uuid.New(), UserID: uuid.New()}
		assert.Equal(t, 0.0, apply(db, sub, 1000))
	})

	t.Run("budget-exhausted promo bills full (discount 0, nothing claimed)", func(t *testing.T) {
		db := setupSubPromoDB(t)
		// budget cap 100, already 50 spent → a 200 discount would exceed it.
		promoID := seedSubPromo(t, db, models.PromoFundingPlatform, 100, 50)
		user, subID := uuid.New(), uuid.New()
		require.NoError(t, db.Exec(`INSERT INTO subscriptions (id, user_id, promo_code_id) VALUES (?, ?, ?)`,
			subID.String(), user.String(), promoID.String()).Error)
		sub := &models.Subscription{ID: subID, UserID: user, PromoCodeID: &promoID}

		assert.Equal(t, 0.0, apply(db, sub, 1000))
		var spent float64
		db.Raw(`SELECT budget_spent FROM promo_codes WHERE id = ?`, promoID.String()).Scan(&spent)
		assert.Equal(t, 50.0, spent) // unchanged — nothing claimed
		// Still cleared so a dead code doesn't linger on the subscription.
		assert.Nil(t, sub.PromoCodeID)
	})

	t.Run("one-time: a second application is a no-op (no double discount/budget)", func(t *testing.T) {
		db := setupSubPromoDB(t)
		promoID := seedSubPromo(t, db, models.PromoFundingPlatform, 0, 0)
		user, subID := uuid.New(), uuid.New()
		require.NoError(t, db.Exec(`INSERT INTO subscriptions (id, user_id, promo_code_id) VALUES (?, ?, ?)`,
			subID.String(), user.String(), promoID.String()).Error)
		sub := &models.Subscription{ID: subID, UserID: user, PromoCodeID: &promoID}

		// First application wins the atomic clear.
		assert.Equal(t, 200.0, apply(db, sub, 1000))
		// A second pass (e.g. a racing invoice generation) re-reads the stored code
		// but the row is already cleared → no discount, no extra claim.
		sub2 := &models.Subscription{ID: subID, UserID: user, PromoCodeID: &promoID}
		assert.Equal(t, 0.0, apply(db, sub2, 1000))

		var usageCount int64
		var spent float64
		db.Raw(`SELECT count(*) FROM promo_code_usages WHERE promo_code_id = ?`, promoID.String()).Scan(&usageCount)
		db.Raw(`SELECT budget_spent FROM promo_codes WHERE id = ?`, promoID.String()).Scan(&spent)
		assert.Equal(t, int64(1), usageCount, "claimed exactly once")
		assert.Equal(t, 200.0, spent, "budget spent exactly once")
	})

	t.Run("chef-funded stored code is not applied", func(t *testing.T) {
		db := setupSubPromoDB(t)
		promoID := seedSubPromo(t, db, models.PromoFundingChef, 0, 0)
		user, subID := uuid.New(), uuid.New()
		require.NoError(t, db.Exec(`INSERT INTO subscriptions (id, user_id, promo_code_id) VALUES (?, ?, ?)`,
			subID.String(), user.String(), promoID.String()).Error)
		sub := &models.Subscription{ID: subID, UserID: user, PromoCodeID: &promoID}
		assert.Equal(t, 0.0, apply(db, sub, 1000))
	})
}
