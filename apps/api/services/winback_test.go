package services

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/homechef/api/models"
)

// winback_test.go — #42. The win-back engine: config, offer minting (unique
// single-use promo + offer + notification event), cooldown dedup, and the
// reactivation/expiry reconciliation.

func setupWinbackDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	// Hand-rolled tables — AutoMigrate of these models chokes on sqlite (Postgres
	// defaults / composite indexes), the established pattern for the promo tests.
	stmts := []string{
		`CREATE TABLE winback_offers (id TEXT PRIMARY KEY, user_id TEXT, audience_type TEXT, trigger TEXT,
			promo_code_id TEXT, code TEXT, discount_percent REAL DEFAULT 0, status TEXT DEFAULT 'offered',
			subscription_id TEXT, offered_at DATETIME, expires_at DATETIME, reactivated_at DATETIME,
			created_at DATETIME, updated_at DATETIME, deleted_at DATETIME)`,
		`CREATE TABLE promo_codes (id TEXT PRIMARY KEY, code TEXT, description TEXT, discount_type TEXT,
			discount_value REAL, max_discount REAL DEFAULT 0, min_order_amount REAL DEFAULT 0,
			usage_limit INTEGER DEFAULT 0, usage_count INTEGER DEFAULT 0, per_user_limit INTEGER DEFAULT 0,
			valid_from DATETIME, valid_until DATETIME, is_active INTEGER DEFAULT 1, applicable_to TEXT DEFAULT 'all',
			funding_source TEXT DEFAULT 'platform', chef_id TEXT, budget_cap REAL DEFAULT 0, budget_spent REAL DEFAULT 0,
			created_by_id TEXT, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME)`,
		`CREATE TABLE promo_code_usages (id TEXT PRIMARY KEY, promo_code_id TEXT, user_id TEXT, order_id TEXT,
			subscription_id TEXT, discount REAL, used_at DATETIME)`,
		`CREATE TABLE platform_settings (id TEXT PRIMARY KEY, key TEXT UNIQUE, value TEXT, type TEXT, updated_by TEXT, updated_at DATETIME)`,
		`CREATE TABLE outbox_events (id TEXT PRIMARY KEY, subject TEXT, msg_id TEXT, aggregate_type TEXT,
			aggregate_id TEXT, payload TEXT, status TEXT DEFAULT 'pending', attempts INTEGER DEFAULT 0,
			last_error TEXT, next_retry_at DATETIME, created_at DATETIME, updated_at DATETIME, published_at DATETIME)`,
		`CREATE TABLE orders (delivery_address_line1_enc text DEFAULT '', delivery_address_line2_enc text DEFAULT '', id TEXT PRIMARY KEY, customer_id TEXT, status TEXT, created_at DATETIME, deleted_at DATETIME)`,
		// Mirror the prod backstop: at most one open offer per user (#42).
		`CREATE UNIQUE INDEX idx_winback_one_open ON winback_offers (user_id) WHERE status = 'offered'`,
	}
	for _, s := range stmts {
		require.NoError(t, db.Exec(s).Error)
	}
	return db
}

func TestGetWinbackConfig(t *testing.T) {
	db := setupWinbackDB(t)
	// Defaults when nothing set.
	cfg := GetWinbackConfig(db)
	assert.True(t, cfg.Enabled)
	assert.Equal(t, 20.0, cfg.DiscountPercent)
	assert.Equal(t, 14, cfg.ValidityDays)
	assert.Equal(t, 30, cfg.LapseThresholdDays)

	// Admin overrides via PlatformSettings.
	for k, v := range map[string]string{
		"winback.enabled": "false", "winback.discount_percent": "35",
		"winback.validity_days": "7", "winback.lapse_threshold_days": "45", "winback.cooldown_days": "60",
	} {
		require.NoError(t, db.Create(&models.PlatformSettings{ID: uuid.New(), Key: k, Value: v}).Error)
	}
	cfg = GetWinbackConfig(db)
	assert.False(t, cfg.Enabled)
	assert.Equal(t, 35.0, cfg.DiscountPercent)
	assert.Equal(t, 7, cfg.ValidityDays)
	assert.Equal(t, 45, cfg.LapseThresholdDays)
	assert.Equal(t, 60, cfg.CooldownDays)
}

func TestGenerateWinbackCode(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 200; i++ {
		c := generateWinbackCode()
		assert.True(t, strings.HasPrefix(c, "WB-"), "code has WB- prefix: %s", c)
		assert.Len(t, c, 9) // "WB-" + 6
		seen[c] = true
	}
	assert.Greater(t, len(seen), 190, "codes are unique enough")
}

func TestOfferWinback(t *testing.T) {
	t.Run("mints a unique single-use promo + offer + notification event", func(t *testing.T) {
		db := setupWinbackDB(t)
		user := uuid.New()

		offer, err := OfferWinback(db, user, models.WinbackAudienceCustomer, models.WinbackTriggerLapsed, nil)
		require.NoError(t, err)
		require.NotNil(t, offer)
		assert.Equal(t, models.WinbackStatusOffered, offer.Status)
		assert.Equal(t, 20.0, offer.DiscountPercent)
		assert.True(t, strings.HasPrefix(offer.Code, "WB-"))

		// The promo is platform-funded, single-use, percentage, valid in the future.
		var promo models.PromoCode
		require.NoError(t, db.First(&promo, "id = ?", offer.PromoCodeID).Error)
		assert.Equal(t, models.PromoFundingPlatform, promo.FundingSource)
		assert.Equal(t, 1, promo.PerUserLimit)
		assert.Equal(t, 1, promo.UsageLimit)
		assert.Equal(t, PromoDiscountPercentage, promo.DiscountType)
		assert.True(t, promo.ValidUntil.After(time.Now()))

		// A notification event was enqueued to the outbox.
		var events int64
		db.Model(&models.OutboxEvent{}).Where("subject = ?", SubjectSubscriptionWinbackOffered).Count(&events)
		assert.Equal(t, int64(1), events)
	})

	t.Run("respects the cooldown — no duplicate offer within the window", func(t *testing.T) {
		db := setupWinbackDB(t)
		user := uuid.New()
		first, err := OfferWinback(db, user, models.WinbackAudienceCustomer, models.WinbackTriggerLapsed, nil)
		require.NoError(t, err)
		require.NotNil(t, first)
		// Second call within cooldown → no new offer.
		second, err := OfferWinback(db, user, models.WinbackAudienceCustomer, models.WinbackTriggerLapsed, nil)
		require.NoError(t, err)
		assert.Nil(t, second)

		var offers int64
		db.Model(&models.WinbackOffer{}).Where("user_id = ?", user).Count(&offers)
		assert.Equal(t, int64(1), offers)
	})

	t.Run("disabled program issues nothing", func(t *testing.T) {
		db := setupWinbackDB(t)
		require.NoError(t, db.Create(&models.PlatformSettings{ID: uuid.New(), Key: "winback.enabled", Value: "false"}).Error)
		offer, err := OfferWinback(db, uuid.New(), models.WinbackAudienceCustomer, models.WinbackTriggerLapsed, nil)
		require.NoError(t, err)
		assert.Nil(t, offer)
	})
}

func TestReconcileWinbackOffers(t *testing.T) {
	t.Run("redeemed offer → reactivated; lapsed-window offer → expired", func(t *testing.T) {
		db := setupWinbackDB(t)
		user := uuid.New()
		offer, err := OfferWinback(db, user, models.WinbackAudienceCustomer, models.WinbackTriggerLapsed, nil)
		require.NoError(t, err)

		// Simulate the user redeeming the offer's code.
		require.NoError(t, db.Create(&models.PromoCodeUsage{ID: uuid.New(),
			PromoCodeID: offer.PromoCodeID, UserID: user, Discount: 50,
		}).Error)

		reactivated, expired := ReconcileWinbackOffers(db)
		assert.Equal(t, 1, reactivated)
		assert.Equal(t, 0, expired)

		var got models.WinbackOffer
		require.NoError(t, db.First(&got, "id = ?", offer.ID).Error)
		assert.Equal(t, models.WinbackStatusReactivated, got.Status)
		assert.NotNil(t, got.ReactivatedAt)
	})

	t.Run("unredeemed past-expiry offer → expired", func(t *testing.T) {
		db := setupWinbackDB(t)
		user := uuid.New()
		offer, err := OfferWinback(db, user, models.WinbackAudienceCustomer, models.WinbackTriggerLapsed, nil)
		require.NoError(t, err)
		// Force it past expiry.
		require.NoError(t, db.Model(&models.WinbackOffer{}).Where("id = ?", offer.ID).
			Update("expires_at", time.Now().Add(-time.Hour)).Error)

		reactivated, expired := ReconcileWinbackOffers(db)
		assert.Equal(t, 0, reactivated)
		assert.Equal(t, 1, expired)
	})
}

func TestFindLapsedCustomers(t *testing.T) {
	db := setupWinbackDB(t)
	order := func(customer uuid.UUID, daysAgo int) {
		require.NoError(t, db.Exec(`INSERT INTO orders (id, customer_id, status, created_at) VALUES (?, ?, 'delivered', ?)`,
			uuid.New().String(), customer.String(), time.Now().AddDate(0, 0, -daysAgo)).Error)
	}
	recent := uuid.New()   // ordered 5 days ago → active, not lapsed
	lapsed := uuid.New()   // last order 40 days ago → lapsed
	offered := uuid.New()  // lapsed but already offered → excluded
	order(recent, 5)
	order(lapsed, 40)
	order(lapsed, 90) // older order too; MAX(created_at) is the 40-day one
	order(offered, 50)
	// `offered` already has a recent win-back offer.
	require.NoError(t, db.Create(&models.WinbackOffer{
		UserID: offered, AudienceType: models.WinbackAudienceCustomer, Trigger: models.WinbackTriggerLapsed,
		PromoCodeID: uuid.New(), Code: "WB-XXXXXX", Status: models.WinbackStatusOffered,
		OfferedAt: time.Now().AddDate(0, 0, -2), ExpiresAt: time.Now().AddDate(0, 0, 12),
	}).Error)

	ids := FindLapsedCustomers(db, 30, 30, 100) // threshold 30d, cooldown 30d
	assert.Contains(t, ids, lapsed)
	assert.NotContains(t, ids, recent, "recent orderer is not lapsed")
	assert.NotContains(t, ids, offered, "already-offered is excluded by cooldown")
}

func TestGetActiveWinbackOffer(t *testing.T) {
	db := setupWinbackDB(t)
	user := uuid.New()
	assert.Nil(t, GetActiveWinbackOffer(db, user))

	offer, err := OfferWinback(db, user, models.WinbackAudienceCustomer, models.WinbackTriggerLapsed, nil)
	require.NoError(t, err)
	active := GetActiveWinbackOffer(db, user)
	require.NotNil(t, active)
	assert.Equal(t, offer.ID, active.ID)

	// Once expired, it's no longer "active".
	require.NoError(t, db.Model(&models.WinbackOffer{}).Where("id = ?", offer.ID).
		Update("status", models.WinbackStatusExpired).Error)
	assert.Nil(t, GetActiveWinbackOffer(db, user))
}
