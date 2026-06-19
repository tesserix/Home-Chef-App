package services

// loyalty_test.go — points ledger correctness for the loyalty program (#40).
// The ledger mirrors the wallet, so the invariants worth pinning are the same:
// earns/redeems move the balance the right way, a redeem can't overdraw,
// PointsAfter is an accurate running snapshot, lifetime + tier track earns, and
// a retried event (same idempotency key) never double-earns. Redeem is the
// money-touching path — it must debit points and credit the wallet atomically.

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

func setupLoyaltyDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	stmts := []string{
		`CREATE TABLE loyalty_accounts (
			id text PRIMARY KEY, user_id text UNIQUE, balance real DEFAULT 0,
			lifetime_points real DEFAULT 0, tier text DEFAULT 'bronze',
			current_streak integer DEFAULT 0, longest_streak integer DEFAULT 0,
			last_streak_day datetime, created_at datetime, updated_at datetime
		)`,
		`CREATE TABLE loyalty_transactions (
			id text PRIMARY KEY, loyalty_account_id text, user_id text, type text, source text,
			points real, points_after real, order_id text, reason text,
			created_by text, idempotency_key text UNIQUE, created_at datetime
		)`,
		`CREATE TABLE wallets (
			id text PRIMARY KEY, user_id text UNIQUE, balance real DEFAULT 0,
			currency text DEFAULT 'INR', created_at datetime, updated_at datetime
		)`,
		`CREATE TABLE wallet_txns (
			id text PRIMARY KEY, wallet_id text, user_id text, type text, source text,
			amount real, balance_after real, currency text, order_id text, reason text,
			created_by text, idempotency_key text UNIQUE, created_at datetime
		)`,
		`CREATE TABLE platform_settings (
			id text PRIMARY KEY, key text UNIQUE, value text, type text, updated_by text, updated_at datetime
		)`,
		`CREATE TABLE outbox_events (id TEXT PRIMARY KEY, subject TEXT, msg_id TEXT, aggregate_type TEXT, aggregate_id TEXT,
			payload TEXT, status TEXT, attempts INT, last_error TEXT, next_retry_at DATETIME, created_at DATETIME, updated_at DATETIME, published_at DATETIME)`,
	}
	for _, s := range stmts {
		require.NoError(t, db.Exec(s).Error)
	}
	orig := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = orig })
	return db
}

func setLoyaltySetting(t *testing.T, db *gorm.DB, key, value string) {
	t.Helper()
	require.NoError(t, db.Create(&models.PlatformSettings{ID: uuid.New(), Key: key, Value: value}).Error)
}

func TestGetLoyaltyConfig_DefaultsAndOverrides(t *testing.T) {
	db := setupLoyaltyDB(t)

	// Before any settings: sane defaults, program enabled.
	cfg := GetLoyaltyConfig(db)
	require.True(t, cfg.Enabled)
	require.Equal(t, 0.1, cfg.PointsPerRupee)
	require.Equal(t, 100.0, cfg.MinRedeem)
	require.Equal(t, 7, cfg.StreakThreshold)

	// Admin overrides via loyalty.* keys.
	setLoyaltySetting(t, db, "loyalty.enabled", "false")
	setLoyaltySetting(t, db, "loyalty.points_per_rupee", "0.25")
	setLoyaltySetting(t, db, "loyalty.redeem_rate", "0.2")
	setLoyaltySetting(t, db, "loyalty.min_redeem", "200")
	setLoyaltySetting(t, db, "loyalty.streak_threshold", "5")
	setLoyaltySetting(t, db, "loyalty.streak_bonus", "75")
	setLoyaltySetting(t, db, "loyalty.tier_silver_at", "500")
	setLoyaltySetting(t, db, "loyalty.tier_gold_at", "2500")

	cfg = GetLoyaltyConfig(db)
	require.False(t, cfg.Enabled)
	require.Equal(t, 0.25, cfg.PointsPerRupee)
	require.Equal(t, 0.2, cfg.RedeemRate)
	require.Equal(t, 200.0, cfg.MinRedeem)
	require.Equal(t, 5, cfg.StreakThreshold)
	require.Equal(t, 75.0, cfg.StreakBonus)
	require.Equal(t, 500.0, cfg.TierSilverAt)
	require.Equal(t, 2500.0, cfg.TierGoldAt)
}

func TestPointsForOrder_FloorsToWholePoints(t *testing.T) {
	cfg := LoyaltyConfig{PointsPerRupee: 0.1}
	require.Equal(t, 0.0, PointsForOrder(cfg, 0))
	require.Equal(t, 10.0, PointsForOrder(cfg, 100))
	// ₹105 × 0.1 = 10.5 → floored to 10 whole points.
	require.Equal(t, 10.0, PointsForOrder(cfg, 105))
	require.Equal(t, 25.0, PointsForOrder(cfg, 259))
}

func TestComputeTier_Thresholds(t *testing.T) {
	cfg := LoyaltyConfig{TierSilverAt: 1000, TierGoldAt: 5000}
	require.Equal(t, models.LoyaltyTierBronze, ComputeTier(0, cfg))
	require.Equal(t, models.LoyaltyTierBronze, ComputeTier(999, cfg))
	require.Equal(t, models.LoyaltyTierSilver, ComputeTier(1000, cfg))
	require.Equal(t, models.LoyaltyTierSilver, ComputeTier(4999, cfg))
	require.Equal(t, models.LoyaltyTierGold, ComputeTier(5000, cfg))
}

func TestEarnLoyalty_MovesBalanceLifetimeTier(t *testing.T) {
	db := setupLoyaltyDB(t)
	uid := uuid.New()

	txn, err := EarnLoyalty(db, uid, 600, models.LoyaltySourceOrder, nil, "order #1", "loyalty:order:1")
	require.NoError(t, err)
	require.Equal(t, 600.0, txn.PointsAfter)

	// A second earn crosses the silver tier (1000 lifetime by default).
	_, err = EarnLoyalty(db, uid, 500, models.LoyaltySourceOrder, nil, "order #2", "loyalty:order:2")
	require.NoError(t, err)

	acct, err := LoyaltyBalance(db, uid)
	require.NoError(t, err)
	require.Equal(t, 1100.0, acct.Balance)
	require.Equal(t, 1100.0, acct.LifetimePoints)
	require.Equal(t, models.LoyaltyTierSilver, acct.Tier)
}

func TestEarnLoyalty_Idempotent(t *testing.T) {
	db := setupLoyaltyDB(t)
	uid := uuid.New()

	_, err := EarnLoyalty(db, uid, 50, models.LoyaltySourceOrder, nil, "order", "loyalty:order:dup")
	require.NoError(t, err)
	// Same idempotency key (redelivered event) → no double-earn.
	_, err = EarnLoyalty(db, uid, 50, models.LoyaltySourceOrder, nil, "order", "loyalty:order:dup")
	require.NoError(t, err)

	acct, _ := LoyaltyBalance(db, uid)
	require.Equal(t, 50.0, acct.Balance)

	var n int64
	db.Model(&models.LoyaltyTransaction{}).Where("user_id = ?", uid).Count(&n)
	require.Equal(t, int64(1), n)
}

func TestRedeemLoyalty_DebitsPointsCreditsWallet(t *testing.T) {
	db := setupLoyaltyDB(t)
	uid := uuid.New()

	_, err := EarnLoyalty(db, uid, 500, models.LoyaltySourceOrder, nil, "order", "loyalty:order:r1")
	require.NoError(t, err)

	// Redeem 300 points → at the default 0.1 ₹/point that's ₹30 store credit.
	lt, wt, err := RedeemLoyalty(db, uid, 300)
	require.NoError(t, err)
	require.Equal(t, models.LoyaltyDebit, lt.Type)
	require.Equal(t, 200.0, lt.PointsAfter)
	require.Equal(t, models.WalletSourceLoyalty, wt.Source)
	require.Equal(t, 30.0, wt.Amount)
	require.Equal(t, 30.0, wt.BalanceAfter)

	acct, _ := LoyaltyBalance(db, uid)
	require.Equal(t, 200.0, acct.Balance)
	// Lifetime is unaffected by redemptions — it only tracks earns.
	require.Equal(t, 500.0, acct.LifetimePoints)

	w, _ := WalletBalance(db, uid)
	require.Equal(t, 30.0, w.Balance)
}

func TestRedeemLoyalty_BelowMinimum(t *testing.T) {
	db := setupLoyaltyDB(t)
	uid := uuid.New()
	_, err := EarnLoyalty(db, uid, 500, models.LoyaltySourceOrder, nil, "order", "loyalty:order:m1")
	require.NoError(t, err)

	// Default min redeem is 100 points.
	_, _, err = RedeemLoyalty(db, uid, 50)
	require.ErrorIs(t, err, ErrLoyaltyBelowMinRedeem)
}

func TestRedeemLoyalty_InsufficientPoints(t *testing.T) {
	db := setupLoyaltyDB(t)
	uid := uuid.New()
	_, err := EarnLoyalty(db, uid, 120, models.LoyaltySourceOrder, nil, "order", "loyalty:order:i1")
	require.NoError(t, err)

	_, _, err = RedeemLoyalty(db, uid, 200)
	require.ErrorIs(t, err, ErrInsufficientLoyaltyPoints)

	// Nothing partially applied — points + wallet untouched.
	acct, _ := LoyaltyBalance(db, uid)
	require.Equal(t, 120.0, acct.Balance)
	w, _ := WalletBalance(db, uid)
	require.Equal(t, 0.0, w.Balance)
}

func TestAwardOrderLoyalty_EarnsAndEnqueuesOnce(t *testing.T) {
	db := setupLoyaltyDB(t)
	uid := uuid.New()
	orderID := uuid.New()

	pts, err := AwardOrderLoyalty(db, uid, orderID, 250)
	require.NoError(t, err)
	require.Equal(t, 25.0, pts) // ₹250 × 0.1

	// Redelivered order-delivered event → idempotent no-op (no second earn,
	// no second notification event).
	pts, err = AwardOrderLoyalty(db, uid, orderID, 250)
	require.NoError(t, err)
	require.Equal(t, 0.0, pts)

	acct, _ := LoyaltyBalance(db, uid)
	require.Equal(t, 25.0, acct.Balance)

	var n int64
	db.Raw(`SELECT count(*) FROM outbox_events WHERE subject = ?`, SubjectLoyaltyEarned).Scan(&n)
	require.Equal(t, int64(1), n)
}

func TestGetLoyaltyAnalytics_Aggregates(t *testing.T) {
	db := setupLoyaltyDB(t)
	u1, u2 := uuid.New(), uuid.New()

	_, err := EarnLoyalty(db, u1, 500, models.LoyaltySourceOrder, nil, "o", "loyalty:order:a1")
	require.NoError(t, err)
	_, err = EarnLoyalty(db, u2, 300, models.LoyaltySourceOrder, nil, "o", "loyalty:order:a2")
	require.NoError(t, err)
	// u1 redeems 200 → 300 left; wallet credit not relevant here.
	_, _, err = RedeemLoyalty(db, u1, 200)
	require.NoError(t, err)
	// Give u2 a live streak.
	_, _, err = AdvanceLoyaltyStreak(db, u2, day(1))
	require.NoError(t, err)

	a := GetLoyaltyAnalytics(db)
	require.Equal(t, int64(2), a.Members)
	require.Equal(t, 800.0, a.PointsEarned)   // 500 + 300
	require.Equal(t, 200.0, a.PointsRedeemed) // u1's redeem
	require.Equal(t, 600.0, a.OutstandingPts) // (500-200) + 300
	require.Equal(t, int64(1), a.ActiveStreaks)
	require.Equal(t, int64(1), a.LongestStreak)
}

func TestAwardOrderLoyalty_DisabledOrZero(t *testing.T) {
	db := setupLoyaltyDB(t)
	uid := uuid.New()

	setLoyaltySetting(t, db, "loyalty.enabled", "false")
	pts, err := AwardOrderLoyalty(db, uid, uuid.New(), 250)
	require.NoError(t, err)
	require.Equal(t, 0.0, pts)

	// Re-enable; a zero-value order earns nothing and creates no account.
	db.Exec(`UPDATE platform_settings SET value = 'true' WHERE key = 'loyalty.enabled'`)
	pts, err = AwardOrderLoyalty(db, uid, uuid.New(), 0)
	require.NoError(t, err)
	require.Equal(t, 0.0, pts)
}
