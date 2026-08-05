package services

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/homechef/api/models"
)

// chef_referral_test.go — the chef-refers-chef program: accept guards, the
// delivered-orders milestone grant (idempotency + monthly cap), loyalty earn /
// conversion, and the settlement bonus credit. These move real payout money,
// so every path is pinned.

func setupChefReferralDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	// Hand-rolled tables — AutoMigrate can't render the Postgres
	// gen_random_uuid() column default on sqlite (project test convention).
	require.NoError(t, db.Exec(`
		CREATE TABLE referral_codes (
			id TEXT PRIMARY KEY, user_id TEXT NOT NULL UNIQUE, code TEXT NOT NULL UNIQUE, created_at DATETIME
		)`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE chef_referrals (
			id TEXT PRIMARY KEY,
			referrer_chef_id TEXT NOT NULL, referrer_user_id TEXT NOT NULL,
			referee_chef_id TEXT NOT NULL UNIQUE, referee_user_id TEXT NOT NULL,
			code TEXT NOT NULL, status TEXT NOT NULL DEFAULT 'pending',
			milestone_orders INTEGER NOT NULL DEFAULT 10,
			referrer_amount REAL DEFAULT 0, referee_amount REAL DEFAULT 0,
			rewarded_at DATETIME, created_at DATETIME, updated_at DATETIME
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
		CREATE TABLE chef_loyalty_accounts (
			id TEXT PRIMARY KEY, chef_id TEXT NOT NULL UNIQUE, user_id TEXT NOT NULL,
			points REAL DEFAULT 0, lifetime_points REAL DEFAULT 0,
			created_at DATETIME, updated_at DATETIME
		)`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE chef_loyalty_txns (
			id TEXT PRIMARY KEY, chef_id TEXT NOT NULL, user_id TEXT NOT NULL,
			kind TEXT NOT NULL, points REAL NOT NULL,
			source_key TEXT NOT NULL UNIQUE, order_id TEXT, bonus_id TEXT,
			description TEXT, created_at DATETIME
		)`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE platform_settings (
			id TEXT PRIMARY KEY, key TEXT NOT NULL UNIQUE, value TEXT,
			type TEXT DEFAULT 'string', updated_by TEXT, updated_at DATETIME
		)`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE weekly_statements (
			id TEXT PRIMARY KEY, chef_id TEXT NOT NULL, user_id TEXT NOT NULL,
			week_start DATETIME, week_end DATETIME, currency TEXT DEFAULT 'INR',
			orders_count INTEGER DEFAULT 0, gross_revenue REAL DEFAULT 0,
			platform_commission REAL DEFAULT 0, cgst REAL DEFAULT 0, sgst REAL DEFAULT 0,
			igst REAL DEFAULT 0, tds REAL DEFAULT 0, penalty_deductions REAL DEFAULT 0,
			bonus_additions REAL DEFAULT 0, net_payout REAL DEFAULT 0,
			status TEXT DEFAULT 'pending', paid_at DATETIME, payout_ref TEXT, created_at DATETIME
		)`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE chef_profiles (
			id TEXT PRIMARY KEY, user_id TEXT NOT NULL, business_name TEXT,
			is_verified INTEGER DEFAULT 0, is_active INTEGER DEFAULT 1
		)`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE orders (
			id TEXT PRIMARY KEY, chef_id TEXT, customer_id TEXT, status TEXT,
			subtotal REAL DEFAULT 0, payment_status TEXT, deleted_at DATETIME
		)`).Error)
	return db
}

type chefFixture struct {
	chefID uuid.UUID
	userID uuid.UUID
	code   string
}

// seedChef inserts a chef profile; verified+active unless flags say otherwise.
func seedRefChef(t *testing.T, db *gorm.DB, verified bool) chefFixture {
	t.Helper()
	f := chefFixture{chefID: uuid.New(), userID: uuid.New()}
	require.NoError(t, db.Exec(
		`INSERT INTO chef_profiles (id, user_id, business_name, is_verified, is_active) VALUES (?, ?, ?, ?, 1)`,
		f.chefID, f.userID, "Kitchen "+f.chefID.String()[:8], verified,
	).Error)
	code, err := GetOrCreateReferralCode(db, f.userID)
	require.NoError(t, err)
	f.code = code
	return f
}

func seedDeliveredOrders(t *testing.T, db *gorm.DB, chefID uuid.UUID, n int, subtotal float64) {
	t.Helper()
	for i := 0; i < n; i++ {
		require.NoError(t, db.Exec(
			`INSERT INTO orders (id, chef_id, customer_id, status, subtotal) VALUES (?, ?, ?, 'delivered', ?)`,
			uuid.New(), chefID, uuid.New(), subtotal,
		).Error)
	}
}

func TestAcceptChefReferralGuards(t *testing.T) {
	db := setupChefReferralDB(t)
	referrer := seedRefChef(t, db, true)
	referee := seedRefChef(t, db, false)

	// Happy path — freezes the configured milestone.
	ref, err := AcceptChefReferral(db, AcceptChefReferralInput{
		RefereeChefID: referee.chefID, RefereeUserID: referee.userID, Code: referrer.code,
	})
	require.NoError(t, err)
	assert.Equal(t, models.ChefReferralPending, ref.Status)
	assert.Equal(t, 10, ref.MilestoneOrders)
	assert.Equal(t, referrer.chefID, ref.ReferrerChefID)

	// Idempotent same-code re-accept; different code rejected.
	again, err := AcceptChefReferral(db, AcceptChefReferralInput{
		RefereeChefID: referee.chefID, RefereeUserID: referee.userID, Code: referrer.code,
	})
	require.NoError(t, err)
	assert.Equal(t, ref.ID, again.ID)
	other := seedRefChef(t, db, true)
	_, err = AcceptChefReferral(db, AcceptChefReferralInput{
		RefereeChefID: referee.chefID, RefereeUserID: referee.userID, Code: other.code,
	})
	assert.ErrorIs(t, err, ErrChefReferralAlreadyUsed)

	// Unknown code.
	newKitchen := seedRefChef(t, db, false)
	_, err = AcceptChefReferral(db, AcceptChefReferralInput{
		RefereeChefID: newKitchen.chefID, RefereeUserID: newKitchen.userID, Code: "NOPE1234",
	})
	assert.ErrorIs(t, err, ErrReferralCodeInvalid)

	// A plain customer's code can't recruit kitchens.
	customerID := uuid.New()
	customerCode, err := GetOrCreateReferralCode(db, customerID)
	require.NoError(t, err)
	_, err = AcceptChefReferral(db, AcceptChefReferralInput{
		RefereeChefID: newKitchen.chefID, RefereeUserID: newKitchen.userID, Code: customerCode,
	})
	assert.ErrorIs(t, err, ErrChefReferralNotAChef)

	// An unverified kitchen's code can't refer either.
	unverified := seedRefChef(t, db, false)
	_, err = AcceptChefReferral(db, AcceptChefReferralInput{
		RefereeChefID: newKitchen.chefID, RefereeUserID: newKitchen.userID, Code: unverified.code,
	})
	assert.ErrorIs(t, err, ErrChefReferralNotAChef)

	// Self-referral.
	_, err = AcceptChefReferral(db, AcceptChefReferralInput{
		RefereeChefID: referrer.chefID, RefereeUserID: referrer.userID, Code: referrer.code,
	})
	assert.ErrorIs(t, err, ErrChefReferralSelf)

	// A kitchen with delivered orders isn't "new".
	veteran := seedRefChef(t, db, false)
	seedDeliveredOrders(t, db, veteran.chefID, 1, 500)
	_, err = AcceptChefReferral(db, AcceptChefReferralInput{
		RefereeChefID: veteran.chefID, RefereeUserID: veteran.userID, Code: referrer.code,
	})
	assert.ErrorIs(t, err, ErrChefReferralNotNewKitchen)
}

func TestChefReferralMilestoneGrant(t *testing.T) {
	db := setupChefReferralDB(t)
	referrer := seedRefChef(t, db, true)
	referee := seedRefChef(t, db, false)
	_, err := AcceptChefReferral(db, AcceptChefReferralInput{
		RefereeChefID: referee.chefID, RefereeUserID: referee.userID, Code: referrer.code,
	})
	require.NoError(t, err)

	// Below milestone → nothing vests.
	seedDeliveredOrders(t, db, referee.chefID, 9, 300)
	MaybeGrantChefReferralReward(db, referee.chefID)
	var bonusCount int64
	db.Model(&models.ChefBonus{}).Count(&bonusCount)
	assert.EqualValues(t, 0, bonusCount)

	// 10th delivered order → both bonuses raised, referral rewarded.
	seedDeliveredOrders(t, db, referee.chefID, 1, 300)
	MaybeGrantChefReferralReward(db, referee.chefID)

	var ref models.ChefReferral
	require.NoError(t, db.First(&ref, "referee_chef_id = ?", referee.chefID).Error)
	assert.Equal(t, models.ChefReferralRewarded, ref.Status)
	assert.Equal(t, 500.0, ref.ReferrerAmount)
	assert.Equal(t, 250.0, ref.RefereeAmount)
	require.NotNil(t, ref.RewardedAt)

	var referrerBonus, refereeBonus models.ChefBonus
	require.NoError(t, db.First(&referrerBonus, "chef_id = ? AND kind = ?", referrer.chefID, models.ChefBonusReferralReferrer).Error)
	require.NoError(t, db.First(&refereeBonus, "chef_id = ? AND kind = ?", referee.chefID, models.ChefBonusReferralReferee).Error)
	assert.Equal(t, 500.0, referrerBonus.Amount)
	assert.Equal(t, 250.0, refereeBonus.Amount)
	assert.Equal(t, models.ChefBonusPending, referrerBonus.Status)

	// Re-running (a later delivered order) never double-pays.
	seedDeliveredOrders(t, db, referee.chefID, 1, 300)
	MaybeGrantChefReferralReward(db, referee.chefID)
	db.Model(&models.ChefBonus{}).Count(&bonusCount)
	assert.EqualValues(t, 2, bonusCount)
}

func TestChefReferralMonthlyCap(t *testing.T) {
	db := setupChefReferralDB(t)
	require.NoError(t, db.Create(&models.PlatformSettings{ID: uuid.New(), Key: "chef_referral.monthly_spend_cap", Value: "700"}).Error)

	referrer := seedRefChef(t, db, true)
	refereeA := seedRefChef(t, db, false)
	refereeB := seedRefChef(t, db, false)
	for _, r := range []chefFixture{refereeA, refereeB} {
		_, err := AcceptChefReferral(db, AcceptChefReferralInput{
			RefereeChefID: r.chefID, RefereeUserID: r.userID, Code: referrer.code,
		})
		require.NoError(t, err)
		seedDeliveredOrders(t, db, r.chefID, 10, 200)
	}

	// First vest (₹750) fits the ₹700 cap? No — 500+250 > 700, so nothing vests.
	MaybeGrantChefReferralReward(db, refereeA.chefID)
	var count int64
	db.Model(&models.ChefBonus{}).Count(&count)
	assert.EqualValues(t, 0, count)

	// Raise the cap; A vests, then B is blocked by the spent budget.
	require.NoError(t, db.Model(&models.PlatformSettings{}).
		Where("key = ?", "chef_referral.monthly_spend_cap").Update("value", "1000").Error)
	MaybeGrantChefReferralReward(db, refereeA.chefID)
	db.Model(&models.ChefBonus{}).Count(&count)
	assert.EqualValues(t, 2, count)
	MaybeGrantChefReferralReward(db, refereeB.chefID)
	db.Model(&models.ChefBonus{}).Count(&count)
	assert.EqualValues(t, 2, count, "second referral must wait for next month's budget")

	var refB models.ChefReferral
	require.NoError(t, db.First(&refB, "referee_chef_id = ?", refereeB.chefID).Error)
	assert.Equal(t, models.ChefReferralPending, refB.Status, "capped referral stays pending, not lost")
}

func TestEarnChefLoyaltyForOrder(t *testing.T) {
	db := setupChefReferralDB(t)
	chef := seedRefChef(t, db, true)

	orderID := uuid.New()
	require.NoError(t, db.Exec(
		`INSERT INTO orders (id, chef_id, customer_id, status, subtotal) VALUES (?, ?, ?, 'delivered', 450)`,
		orderID, chef.chefID, uuid.New(),
	).Error)

	EarnChefLoyaltyForOrder(db, orderID)
	var acct models.ChefLoyaltyAccount
	require.NoError(t, db.First(&acct, "chef_id = ?", chef.chefID).Error)
	assert.Equal(t, 450.0, acct.Points) // default 1 pt / ₹1
	assert.Equal(t, 450.0, acct.LifetimePoints)

	// Re-stamped delivered status never double-earns.
	EarnChefLoyaltyForOrder(db, orderID)
	require.NoError(t, db.First(&acct, "chef_id = ?", chef.chefID).Error)
	assert.Equal(t, 450.0, acct.Points)

	// A non-delivered order earns nothing.
	pendingID := uuid.New()
	require.NoError(t, db.Exec(
		`INSERT INTO orders (id, chef_id, customer_id, status, subtotal) VALUES (?, ?, ?, 'pending', 900)`,
		pendingID, chef.chefID, uuid.New(),
	).Error)
	EarnChefLoyaltyForOrder(db, pendingID)
	require.NoError(t, db.First(&acct, "chef_id = ?", chef.chefID).Error)
	assert.Equal(t, 450.0, acct.Points)
}

// The cap has to bind on the real conversion path, not just parse from config:
// a chef at the ceiling is refused, and their points are kept rather than burnt.
func TestConvertChefLoyalty_MonthlyCapBlocksAndKeepsThePoints(t *testing.T) {
	db := setupChefReferralDB(t)
	chef := seedRefChef(t, db, true)
	cfg := GetChefLoyaltyConfig(db)

	// Already at the ceiling for this window.
	require.NoError(t, db.Create(&models.ChefBonus{
		ChefID: chef.chefID, UserID: chef.userID,
		Kind:   models.ChefBonusLoyaltyCashback,
		Status: models.ChefBonusPending,
		Amount: cfg.MonthlyConvertCap,
	}).Error)
	require.NoError(t, db.Create(&models.ChefLoyaltyAccount{
		ChefID: chef.chefID, UserID: chef.userID,
		Points: 20000, LifetimePoints: 20000,
	}).Error)

	_, _, err := ConvertChefLoyalty(db, chef.chefID, chef.userID)
	assert.ErrorIs(t, err, ErrChefLoyaltyMonthlyCap)

	// Refused, not spent — the chef converts once the window moves on.
	var acct models.ChefLoyaltyAccount
	require.NoError(t, db.Where("chef_id = ?", chef.chefID).First(&acct).Error)
	assert.Equal(t, 20000.0, acct.Points, "a refused conversion must not burn points")
}

func TestConvertChefLoyalty(t *testing.T) {
	db := setupChefReferralDB(t)
	chef := seedRefChef(t, db, true)

	// Below the 10,000-point threshold.
	require.NoError(t, db.Create(&models.ChefLoyaltyAccount{
		ChefID: chef.chefID, UserID: chef.userID, Points: 9999, LifetimePoints: 9999,
	}).Error)
	_, _, err := ConvertChefLoyalty(db, chef.chefID, chef.userID)
	assert.ErrorIs(t, err, ErrChefLoyaltyBelowMinimum)

	// Crossing it converts the FULL balance at the configured redeem rate.
	// Asserted against the config rather than a literal, so a deliberate rate
	// change does not read as a broken conversion.
	require.NoError(t, db.Model(&models.ChefLoyaltyAccount{}).
		Where("chef_id = ?", chef.chefID).
		Updates(map[string]any{"points": 12000, "lifetime_points": 12000}).Error)
	bonus, points, err := ConvertChefLoyalty(db, chef.chefID, chef.userID)
	require.NoError(t, err)
	assert.Equal(t, 12000.0, points)
	assert.Equal(t, Round2(12000.0*GetChefLoyaltyConfig(db).RedeemRate), bonus.Amount)
	assert.Equal(t, models.ChefBonusLoyaltyCashback, bonus.Kind)
	assert.Equal(t, models.ChefBonusPending, bonus.Status)

	var acct models.ChefLoyaltyAccount
	require.NoError(t, db.First(&acct, "chef_id = ?", chef.chefID).Error)
	assert.Equal(t, 0.0, acct.Points)
	assert.Equal(t, 12000.0, acct.LifetimePoints, "lifetime total survives conversion")

	// Emptied balance can't convert again.
	_, _, err = ConvertChefLoyalty(db, chef.chefID, chef.userID)
	assert.ErrorIs(t, err, ErrChefLoyaltyBelowMinimum)

	// The conversion txn is on the audit trail.
	var txn models.ChefLoyaltyTxn
	require.NoError(t, db.First(&txn, "chef_id = ? AND kind = ?", chef.chefID, models.ChefLoyaltyConvert).Error)
	assert.Equal(t, -12000.0, txn.Points)
	require.NotNil(t, txn.BonusID)
	assert.Equal(t, bonus.ID, *txn.BonusID)
}

func TestApplyChefBonusesToStatement(t *testing.T) {
	db := setupChefReferralDB(t)
	chef := seedRefChef(t, db, true)

	stmt := models.WeeklyStatement{ChefID: chef.chefID, UserID: chef.userID, NetPayout: 1000}
	stmt.ID = uuid.New()
	require.NoError(t, db.Create(&stmt).Error)

	for i, amt := range []float64{500, 250} {
		require.NoError(t, db.Create(&models.ChefBonus{
			ChefID: chef.chefID, UserID: chef.userID,
			Kind: models.ChefBonusReferralReferrer, Status: models.ChefBonusPending,
			SourceKey: uuid.New().String(), Amount: amt, Currency: "INR",
			Reason: map[int]string{0: "a", 1: "b"}[i],
		}).Error)
	}

	credited, err := ApplyChefBonusesToStatement(db, &stmt)
	require.NoError(t, err)
	assert.Equal(t, 750.0, credited)
	assert.Equal(t, 750.0, stmt.BonusAdditions)
	assert.Equal(t, 1750.0, stmt.NetPayout)

	var persisted models.WeeklyStatement
	require.NoError(t, db.First(&persisted, "id = ?", stmt.ID).Error)
	assert.Equal(t, 1750.0, persisted.NetPayout)

	// All claimed — a second statement run credits nothing more.
	credited, err = ApplyChefBonusesToStatement(db, &stmt)
	require.NoError(t, err)
	assert.Equal(t, 0.0, credited)

	var pending int64
	db.Model(&models.ChefBonus{}).Where("status = ?", models.ChefBonusPending).Count(&pending)
	assert.EqualValues(t, 0, pending)
}
