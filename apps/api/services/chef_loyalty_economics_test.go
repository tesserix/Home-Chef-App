package services

// chef_loyalty_economics_test.go — what the programme costs the platform.
//
// The cost is earn × redeem, not either number alone, and neither is obvious
// from reading one line of config. It shipped at 1 point per ₹1 and ₹0.05 a
// point — 5% of subtotal against a 6% commission, so it consumed five sixths of
// the take on every delivered order and lost money once the ~2% payment MDR is
// counted. Nothing in the code said so, which is how it survived.

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// setupEmptySettingsDB gives GetChefLoyaltyConfig a settings table with no
// overrides, so it returns the shipped defaults.
func setupEmptySettingsDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"),
		&gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	require.NoError(t, err)
	require.NoError(t, db.Exec(
		`CREATE TABLE platform_settings (key TEXT PRIMARY KEY, value TEXT)`).Error)
	return db
}

// The ceiling the platform can afford. Commission is 6% of subtotal, and the
// payment gateway takes ~2% of the total before anything reaches us — so a
// supply-side giveaway above about 1% is spending margin that is not there.
const chefLoyaltyMaxAffordablePct = 1.0

func TestChefLoyaltyCost_StaysWithinCommission(t *testing.T) {
	cfg := GetChefLoyaltyConfig(setupEmptySettingsDB(t))

	// Points earned on ₹1, valued in rupees, is the giveaway per rupee of
	// subtotal — the only figure that matters.
	costPct := cfg.EarnRate * cfg.RedeemRate * 100

	require.LessOrEqual(t, costPct, chefLoyaltyMaxAffordablePct,
		"chef loyalty gives away %.2f%% of subtotal against a 6%% commission", costPct)
	require.Greater(t, costPct, 0.0, "a programme worth nothing motivates nothing")
}

// The earn rate is high on purpose: a ₹400 order showing 400 points is what
// makes the programme feel worth chasing. The cost is controlled by the point
// VALUE, so a future tweak that "makes points rarer" instead of cheaper would
// hurt the incentive without helping the margin.
func TestChefLoyaltyEarnRate_StaysGenerousEnoughToNotice(t *testing.T) {
	cfg := GetChefLoyaltyConfig(setupEmptySettingsDB(t))
	require.GreaterOrEqual(t, cfg.EarnRate, 1.0,
		"points should climb visibly with order value")
}

// A threshold the average chef never reaches is a programme that never pays,
// which reads as a broken promise rather than a saving.
func TestChefLoyaltyThreshold_IsReachable(t *testing.T) {
	cfg := GetChefLoyaltyConfig(setupEmptySettingsDB(t))
	payout := cfg.MinConvertPoints * cfg.RedeemRate
	require.GreaterOrEqual(t, payout, 25.0,
		"converting should be worth the wait, not ₹%.2f", payout)

	// At ₹400 average subtotal, how many delivered orders to first payout.
	orders := cfg.MinConvertPoints / (400 * cfg.EarnRate)
	require.LessOrEqual(t, orders, 40.0,
		"first payout takes %.0f orders — too far away to motivate", orders)
}

// The cap is the backstop the customer programme has always had and this side
// shipped without: a minimum tells a chef when they MAY convert, nothing told
// them when to stop. A pricing mistake or one very high-volume kitchen could
// convert without bound until someone noticed in the numbers.
func TestChefLoyaltyMonthlyCap_IsSetByDefault(t *testing.T) {
	cfg := GetChefLoyaltyConfig(setupEmptySettingsDB(t))
	require.Greater(t, cfg.MonthlyConvertCap, 0.0,
		"an uncapped giveaway is bounded only by order volume")

	// A cap below the conversion threshold would make the programme unusable:
	// the chef reaches the minimum and is refused every time.
	minPayout := cfg.MinConvertPoints * cfg.RedeemRate
	require.GreaterOrEqual(t, cfg.MonthlyConvertCap, minPayout,
		"cap ₹%.2f is below the ₹%.2f minimum conversion — nobody could ever convert",
		cfg.MonthlyConvertCap, minPayout)
}

// "0" must switch the cap off. A >0 guard would drop the value and leave the
// default in place while the settings screen showed the admin's 0 back to them.
func TestChefLoyaltyMonthlyCap_ZeroDisablesIt(t *testing.T) {
	db := setupEmptySettingsDB(t)
	require.NoError(t, db.Exec(
		`INSERT INTO platform_settings (key, value) VALUES ('chef_loyalty.monthly_convert_cap', '0')`).Error)
	require.Equal(t, 0.0, GetChefLoyaltyConfig(db).MonthlyConvertCap)
}
