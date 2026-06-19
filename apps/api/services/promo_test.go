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

	"github.com/homechef/api/models"
)

// promo_test.go — #39. The promo discount math + eligibility rules. Money
// correctness: the discount never exceeds the base, percentage caps hold, and a
// code is refused once it's expired, over a usage/budget cap, off-target, or for
// the wrong (chef-funded) kitchen.

func TestComputePromoDiscount(t *testing.T) {
	pct := func(v, max float64) *models.PromoCode {
		return &models.PromoCode{DiscountType: PromoDiscountPercentage, DiscountValue: v, MaxDiscount: max}
	}
	fixed := func(v float64) *models.PromoCode {
		return &models.PromoCode{DiscountType: PromoDiscountFixed, DiscountValue: v}
	}
	assert.Equal(t, 100.0, ComputePromoDiscount(pct(20, 0), 500))   // 20% of 500
	assert.Equal(t, 150.0, ComputePromoDiscount(pct(20, 150), 1000)) // capped at MaxDiscount
	assert.Equal(t, 75.0, ComputePromoDiscount(fixed(75), 500))      // flat
	assert.Equal(t, 500.0, ComputePromoDiscount(fixed(800), 500))    // never exceeds the base
	assert.Equal(t, 0.0, ComputePromoDiscount(fixed(0), 500))
}

func activePromo() *models.PromoCode {
	return &models.PromoCode{
		IsActive:     true,
		DiscountType: PromoDiscountPercentage,
		DiscountValue: 10,
		ValidFrom:    time.Now().Add(-time.Hour),
		FundingSource: models.PromoFundingPlatform,
		ApplicableTo: "all",
	}
}

func TestCheckPromoEligibility(t *testing.T) {
	now := time.Now()
	base := PromoContext{Now: now, OrderSubtotal: 500}

	t.Run("happy path", func(t *testing.T) {
		assert.NoError(t, CheckPromoEligibility(activePromo(), base, 50))
	})

	t.Run("inactive", func(t *testing.T) {
		p := activePromo()
		p.IsActive = false
		assert.ErrorIs(t, CheckPromoEligibility(p, base, 50), ErrPromoInactive)
	})

	t.Run("not yet started / expired", func(t *testing.T) {
		p := activePromo()
		p.ValidFrom = now.Add(time.Hour)
		assert.ErrorIs(t, CheckPromoEligibility(p, base, 50), ErrPromoNotStarted)
		p2 := activePromo()
		until := now.Add(-time.Minute)
		p2.ValidUntil = &until
		assert.ErrorIs(t, CheckPromoEligibility(p2, base, 50), ErrPromoExpired)
	})

	t.Run("global + per-user usage caps", func(t *testing.T) {
		p := activePromo()
		p.UsageLimit, p.UsageCount = 5, 5
		assert.ErrorIs(t, CheckPromoEligibility(p, base, 50), ErrPromoUsageLimit)
		p2 := activePromo()
		p2.PerUserLimit = 1
		assert.ErrorIs(t, CheckPromoEligibility(p2, PromoContext{Now: now, OrderSubtotal: 500, UserUsageCount: 1}, 50), ErrPromoPerUserLimit)
	})

	t.Run("minimum order not met", func(t *testing.T) {
		p := activePromo()
		p.MinOrderAmount = 600
		err := CheckPromoEligibility(p, base, 50)
		var minErr ErrPromoMinOrder
		assert.ErrorAs(t, err, &minErr)
		assert.Equal(t, 600.0, minErr.MinOrderAmount)
	})

	t.Run("new / returning targeting", func(t *testing.T) {
		p := activePromo()
		p.ApplicableTo = "new_users"
		assert.ErrorIs(t, CheckPromoEligibility(p, PromoContext{Now: now, OrderSubtotal: 500, UserOrderCount: 2}, 50), ErrPromoNewUsersOnly)
		assert.NoError(t, CheckPromoEligibility(p, PromoContext{Now: now, OrderSubtotal: 500, UserOrderCount: 0}, 50))
		p2 := activePromo()
		p2.ApplicableTo = "returning_users"
		assert.ErrorIs(t, CheckPromoEligibility(p2, PromoContext{Now: now, OrderSubtotal: 500, UserOrderCount: 0}, 50), ErrPromoReturningOnly)
	})

	t.Run("chef-funded promo only valid for its chef", func(t *testing.T) {
		chef, other := uuid.New(), uuid.New()
		p := activePromo()
		p.FundingSource = models.PromoFundingChef
		p.ChefID = &chef
		// Wrong chef → rejected.
		assert.ErrorIs(t, CheckPromoEligibility(p, PromoContext{Now: now, OrderSubtotal: 500, ChefID: other}, 50), ErrPromoWrongChef)
		// Right chef → ok.
		assert.NoError(t, CheckPromoEligibility(p, PromoContext{Now: now, OrderSubtotal: 500, ChefID: chef}, 50))
		// Preview (ChefID unknown) → not blocked.
		assert.NoError(t, CheckPromoEligibility(p, PromoContext{Now: now, OrderSubtotal: 500}, 50))
	})

	t.Run("budget cap blocks the redemption that would exceed it", func(t *testing.T) {
		p := activePromo()
		p.BudgetCap, p.BudgetSpent = 1000, 970
		assert.NoError(t, CheckPromoEligibility(p, base, 30))                                  // 970+30 = 1000, exactly at cap
		assert.ErrorIs(t, CheckPromoEligibility(p, base, 31), ErrPromoBudgetExhausted)         // 970+31 > 1000
	})
}

func TestChefFundedPortion(t *testing.T) {
	platform := &models.PromoCode{FundingSource: models.PromoFundingPlatform}
	chef := &models.PromoCode{FundingSource: models.PromoFundingChef}
	assert.Equal(t, 0.0, ChefFundedPortion(platform, 120))
	assert.Equal(t, 120.0, ChefFundedPortion(chef, 120))
	assert.Equal(t, 0.0, ChefFundedPortion(nil, 120))
}

func setupPromoDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TABLE promo_codes (
		id TEXT PRIMARY KEY, code TEXT, usage_limit INTEGER DEFAULT 0, usage_count INTEGER DEFAULT 0,
		budget_cap REAL DEFAULT 0, budget_spent REAL DEFAULT 0, updated_at DATETIME, deleted_at DATETIME)`).Error)
	return db
}

func TestClaimPromoRedemption(t *testing.T) {
	t.Run("claims and increments usage + budget atomically", func(t *testing.T) {
		db := setupPromoDB(t)
		id := uuid.New()
		require.NoError(t, db.Exec(`INSERT INTO promo_codes (id, code, usage_limit, usage_count, budget_cap, budget_spent)
			VALUES (?, 'SAVE', 3, 0, 1000, 0)`, id.String()).Error)

		ok, err := ClaimPromoRedemption(db, id, 120)
		require.NoError(t, err)
		assert.True(t, ok)

		var count int
		var spent float64
		db.Raw(`SELECT usage_count FROM promo_codes WHERE id = ?`, id.String()).Scan(&count)
		db.Raw(`SELECT budget_spent FROM promo_codes WHERE id = ?`, id.String()).Scan(&spent)
		assert.Equal(t, 1, count)
		assert.Equal(t, 120.0, spent)
	})

	t.Run("refuses once the global usage limit is reached", func(t *testing.T) {
		db := setupPromoDB(t)
		id := uuid.New()
		require.NoError(t, db.Exec(`INSERT INTO promo_codes (id, code, usage_limit, usage_count) VALUES (?, 'ONCE', 1, 1)`, id.String()).Error)
		ok, err := ClaimPromoRedemption(db, id, 10)
		require.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("refuses the redemption that would exceed the budget cap", func(t *testing.T) {
		db := setupPromoDB(t)
		id := uuid.New()
		require.NoError(t, db.Exec(`INSERT INTO promo_codes (id, code, budget_cap, budget_spent) VALUES (?, 'BUDGET', 1000, 970)`, id.String()).Error)
		// 970 + 31 > 1000 → refused, nothing changes.
		ok, err := ClaimPromoRedemption(db, id, 31)
		require.NoError(t, err)
		assert.False(t, ok)
		var spent float64
		db.Raw(`SELECT budget_spent FROM promo_codes WHERE id = ?`, id.String()).Scan(&spent)
		assert.Equal(t, 970.0, spent)
		// 970 + 30 == 1000 → allowed.
		ok, err = ClaimPromoRedemption(db, id, 30)
		require.NoError(t, err)
		assert.True(t, ok)
	})
}
