package services

// chef_tip_backfill_test.go — #964. Customer tips were charged and reached nobody.
//
// The trap this pins is the DOUBLE-PAY one: the catch-up credit is only correct
// for an order a frozen statement has ALREADY billed. An order not yet on a
// statement will pick the backfilled chef_tip up naturally on its next one, so
// crediting it as well would pay the same tip twice.

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

func setupTipBackfillDB(t *testing.T) (*gorm.DB, uuid.UUID) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	require.NoError(t, err)
	createTablesFor(t, db,
		&models.Order{}, &models.ChefProfile{}, &models.ChefBonus{}, &models.WeeklyStatement{})
	orig := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = orig })

	chefID := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO chef_profiles (id, user_id, business_name) VALUES (?,?,?)`,
		chefID.String(), uuid.New().String(), "Saffron Home Kitchen").Error)
	return db, chefID
}

func addTippedOrder(t *testing.T, db *gorm.DB, chefID uuid.UUID, num string, tip, chefTip float64, delivered time.Time) uuid.UUID {
	t.Helper()
	id := uuid.New()
	require.NoError(t, db.Exec(
		`INSERT INTO orders (id, order_number, chef_id, status, delivered_at, subtotal, tax, tip, chef_tip, total)
		 VALUES (?,?,?,?,?,?,?,?,?,?)`,
		id.String(), num, chefID.String(), "delivered", delivered, 500.0, 25.0, tip, chefTip, 630.0).Error)
	return id
}

func addStatement(t *testing.T, db *gorm.DB, chefID uuid.UUID, weekStart time.Time) {
	t.Helper()
	require.NoError(t, db.Exec(
		`INSERT INTO weekly_statements (id, chef_id, user_id, week_start, week_end, status, net_payout)
		 VALUES (?,?,?,?,?,?,?)`,
		uuid.NewString(), chefID.String(), uuid.NewString(),
		weekStart, weekStart.AddDate(0, 0, 7), "pending", 1000.0).Error)
}

func TestBackfillChefTips_CreditsOnlyOrdersAlreadyBilled(t *testing.T) {
	db, chefID := setupTipBackfillDB(t)

	weekStart := time.Date(2026, 7, 19, 0, 0, 0, 0, time.UTC)
	addStatement(t, db, chefID, weekStart)

	// Billed on the frozen statement — the tip can never reach it.
	billed := addTippedOrder(t, db, chefID, "BILLED-30", 30, 0, weekStart.Add(48*time.Hour))
	// Delivered AFTER the statement window — no statement covers it yet.
	unbilled := addTippedOrder(t, db, chefID, "UNBILLED-25", 25, 0, weekStart.AddDate(0, 0, 20))

	BackfillChefTips()

	// Both orders get chef_tip repaired, so every re-derived surface is correct.
	var got []models.Order
	require.NoError(t, db.Order("order_number").Find(&got).Error)
	byNum := map[string]models.Order{}
	for _, o := range got {
		byNum[o.OrderNumber] = o
	}
	assert.Equal(t, 30.0, byNum["BILLED-30"].ChefTip)
	assert.Equal(t, 25.0, byNum["UNBILLED-25"].ChefTip)

	// Only the already-billed one gets a catch-up credit.
	var bonuses []models.ChefBonus
	require.NoError(t, db.Find(&bonuses).Error)
	require.Len(t, bonuses, 1, "an unbilled order must NOT be credited — its next statement pays the tip")
	assert.Equal(t, models.ChefBonusTipCatchup, bonuses[0].Kind)
	assert.Equal(t, 30.0, bonuses[0].Amount)
	assert.Equal(t, ChefTipCatchupSourceKey(billed), bonuses[0].SourceKey)
	assert.Contains(t, bonuses[0].Reason, "BILLED-30")

	_ = unbilled
}

// The cron runs every 10 minutes forever — it must never re-credit.
func TestBackfillChefTips_Idempotent(t *testing.T) {
	db, chefID := setupTipBackfillDB(t)
	weekStart := time.Date(2026, 7, 19, 0, 0, 0, 0, time.UTC)
	addStatement(t, db, chefID, weekStart)
	addTippedOrder(t, db, chefID, "BILLED-30", 30, 0, weekStart.Add(48*time.Hour))

	BackfillChefTips()
	BackfillChefTips()
	BackfillChefTips()

	var count int64
	db.Model(&models.ChefBonus{}).Count(&count)
	assert.EqualValues(t, 1, count, "one tip, one credit, however often the cron runs")

	var order models.Order
	require.NoError(t, db.First(&order).Error)
	assert.Equal(t, 30.0, order.ChefTip, "the repair is not re-applied on top of itself")
}

// An order whose tip already reached the chef must be left completely alone.
func TestBackfillChefTips_SkipsAlreadyPaidTips(t *testing.T) {
	db, chefID := setupTipBackfillDB(t)
	weekStart := time.Date(2026, 7, 19, 0, 0, 0, 0, time.UTC)
	addStatement(t, db, chefID, weekStart)
	addTippedOrder(t, db, chefID, "POST-FIX", 40, 40, weekStart.Add(48*time.Hour))

	BackfillChefTips()

	var count int64
	db.Model(&models.ChefBonus{}).Count(&count)
	assert.EqualValues(t, 0, count, "chef_tip is already set — checkout paid it, nothing is owed")
}

// The chef's payout must actually include the tip once chef_tip is set — the
// whole point of the column.
func TestChefTip_ReachesEarningsAndPayout(t *testing.T) {
	withTip := ComputeOrderEarnings(EarningsInput{
		ItemRevenue: 500, Tax: 25, ChefTip: 30, CommissionRate: 0.06,
	}, "")
	without := ComputeOrderEarnings(EarningsInput{
		ItemRevenue: 500, Tax: 25, ChefTip: 0, CommissionRate: 0.06,
	}, "")

	assert.Equal(t, 30.0, Round2(withTip.Gross-without.Gross), "the tip is the chef's income")
	assert.Equal(t, withTip.PlatformCommission, without.PlatformCommission,
		"no commission on a tip — the customer is promised 100% goes to the chef")
	// Net rises by the tip less the 1% statutory TDS on it, which is remitted to
	// the chef's PAN rather than kept by the platform.
	assert.InDelta(t, 29.70, withTip.NetPayout-without.NetPayout, 0.01)
}
