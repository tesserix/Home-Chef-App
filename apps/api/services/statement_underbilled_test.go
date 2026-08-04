package services

// statement_underbilled_test.go — D-15. A weekly statement is frozen and each
// (chef, week) is generated exactly once, so an order that gains value AFTER its
// week closes is owed money nothing will ever pay: the catch-up selects on an
// unset billed stamp and therefore sees orders that were never billed, not an
// order billed for LESS than it is now worth.
//
// Found as a ₹25 tip on a statement that reconciled exactly on population (10
// orders = 10 orders) and was ₹25.02 short on money.

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

// billOrder stamps an order as settled for a recorded net payout — the shape every
// statement written since the column existed leaves behind.
func billOrder(t *testing.T, db *gorm.DB, orderID, stmtID uuid.UUID, net float64) {
	t.Helper()
	require.NoError(t, db.Exec(`UPDATE orders SET billed_statement_id = ?, settled_net_payout = ? WHERE id = ?`,
		stmtID.String(), net, orderID.String()).Error)
}

func addTip(t *testing.T, db *gorm.DB, orderID uuid.UUID, tip float64) {
	t.Helper()
	require.NoError(t, db.Exec(`UPDATE orders SET chef_tip = ? WHERE id = ?`, tip, orderID.String()).Error)
}

// The order this defect was found on: billed at its pre-tip value, then tipped.
func TestUnderbilled_CreditsValueAddedAfterTheWeekClosed(t *testing.T) {
	db, chefID := setupCatchupDB(t)
	stmt := addCatchupStatement(t, db, chefID)
	order := addCatchupOrder(t, db, chefID, "TIPPED-LATE", "release_eligible", nil)
	billOrder(t, db, order, stmt, 489.75) // gross 525 − 6% commission − 1% TDS

	addTip(t, db, order, 25.0)
	reconcileStatementCatchup()

	var bonuses []models.ChefBonus
	require.NoError(t, db.Find(&bonuses).Error)
	require.Len(t, bonuses, 1)
	assert.Equal(t, models.ChefBonusStatementAdjust, bonuses[0].Kind)
	assert.Contains(t, bonuses[0].Reason, "TIPPED-LATE")
	// The tip enters gross, so TDS applies to it: 25.00 − 1% = 24.75. The chef is
	// paid the DIFFERENCE, never the order over again.
	assert.InDelta(t, 24.75, bonuses[0].Amount, 0.01)

	var got models.Order
	require.NoError(t, db.First(&got, "id = ?", order.String()).Error)
	require.NotNil(t, got.SettledNetPayout)
	assert.InDelta(t, 514.50, *got.SettledNetPayout, 0.01, "settled figure advances to what is now paid")
}

// The cron runs hourly forever. Once the shortfall is credited the order must
// stop matching — this is the half that would pay a chef the same tip every hour.
func TestUnderbilled_Idempotent(t *testing.T) {
	db, chefID := setupCatchupDB(t)
	stmt := addCatchupStatement(t, db, chefID)
	order := addCatchupOrder(t, db, chefID, "TIPPED-LATE", "release_eligible", nil)
	billOrder(t, db, order, stmt, 489.75)
	addTip(t, db, order, 25.0)

	reconcileStatementCatchup()
	reconcileStatementCatchup()
	reconcileStatementCatchup()

	var count int64
	db.Model(&models.ChefBonus{}).Count(&count)
	assert.EqualValues(t, 1, count, "one shortfall, one credit, however often the cron runs")
}

// A second late addition after the first was settled is a second real shortfall.
func TestUnderbilled_CreditsAFurtherIncrease(t *testing.T) {
	db, chefID := setupCatchupDB(t)
	stmt := addCatchupStatement(t, db, chefID)
	order := addCatchupOrder(t, db, chefID, "TIPPED-TWICE", "release_eligible", nil)
	billOrder(t, db, order, stmt, 489.75)

	addTip(t, db, order, 25.0)
	reconcileStatementCatchup()
	addTip(t, db, order, 60.0)
	reconcileStatementCatchup()

	var bonuses []models.ChefBonus
	require.NoError(t, db.Order("amount ASC").Find(&bonuses).Error)
	require.Len(t, bonuses, 2)
	assert.InDelta(t, 24.75, bonuses[0].Amount, 0.01)
	assert.InDelta(t, 34.65, bonuses[1].Amount, 0.01, "only the further 35 of tip, less TDS")
}

// An order billed for what it is still worth owes nothing, and per-row rounding
// against the statement total is not a shortfall.
func TestUnderbilled_IgnoresAnUnchangedOrderAndRounding(t *testing.T) {
	db, chefID := setupCatchupDB(t)
	stmt := addCatchupStatement(t, db, chefID)
	exact := addCatchupOrder(t, db, chefID, "EXACT", "release_eligible", nil)
	billOrder(t, db, exact, stmt, 489.75)
	rounded := addCatchupOrder(t, db, chefID, "ROUNDED", "release_eligible", nil)
	billOrder(t, db, rounded, stmt, 489.73) // 2 paise of per-row vs total rounding

	reconcileStatementCatchup()

	var count int64
	db.Model(&models.ChefBonus{}).Count(&count)
	assert.EqualValues(t, 0, count)
}

// An order whose value FELL is a refund or an adjustment, and those paths own it.
// This sweep only ever pays a shortfall — it must never claw a settlement back.
func TestUnderbilled_NeverClawsBack(t *testing.T) {
	db, chefID := setupCatchupDB(t)
	stmt := addCatchupStatement(t, db, chefID)
	order := addCatchupOrder(t, db, chefID, "OVERBILLED", "release_eligible", nil)
	billOrder(t, db, order, stmt, 900.0) // more than the order is worth

	reconcileStatementCatchup()

	var count int64
	db.Model(&models.ChefBonus{}).Count(&count)
	assert.EqualValues(t, 0, count)

	var got models.Order
	require.NoError(t, db.First(&got, "id = ?", order.String()).Error)
	assert.InDelta(t, 900.0, *got.SettledNetPayout, 0.01, "the recorded settlement is left alone")
}

// Orders stamped by BackfillBilledStatementIDs carry no settled figure, so what is
// still owed on them is unknowable — and guessing pays the chef twice.
func TestUnderbilled_SkipsOrdersWithNoRecordedSettlement(t *testing.T) {
	db, chefID := setupCatchupDB(t)
	stmt := addCatchupStatement(t, db, chefID)
	order := addCatchupOrder(t, db, chefID, "LEGACY-STAMP", "release_eligible", &stmt)
	addTip(t, db, order, 25.0)

	reconcileStatementCatchup()

	var count int64
	db.Model(&models.ChefBonus{}).Count(&count)
	assert.EqualValues(t, 0, count)
}

// A refunded order is never owed anything, however its value moved.
func TestUnderbilled_SkipsRefundedOrder(t *testing.T) {
	db, chefID := setupCatchupDB(t)
	stmt := addCatchupStatement(t, db, chefID)
	order := addCatchupOrder(t, db, chefID, "REFUNDED", "release_eligible", nil)
	billOrder(t, db, order, stmt, 489.75)
	addTip(t, db, order, 25.0)
	require.NoError(t, db.Exec(`UPDATE orders SET refunded_at = ? WHERE id = ?`,
		catchupWeek, order.String()).Error)

	reconcileStatementCatchup()

	var count int64
	db.Model(&models.ChefBonus{}).Count(&count)
	assert.EqualValues(t, 0, count)
}

// The legacy tip backfill credits the same money on its own key. The two paths
// must stay disjoint or a pre-#964 tip is paid twice.
func TestUnderbilled_SkipsOrderTheTipBackfillAlreadySettled(t *testing.T) {
	db, chefID := setupCatchupDB(t)
	stmt := addCatchupStatement(t, db, chefID)
	order := addCatchupOrder(t, db, chefID, "TIP-BACKFILLED", "release_eligible", nil)
	billOrder(t, db, order, stmt, 489.75)
	addTip(t, db, order, 25.0)
	require.NoError(t, db.Create(&models.ChefBonus{
		ChefID: chefID, UserID: uuid.New(), Kind: models.ChefBonusTipCatchup,
		Status: models.ChefBonusPending, SourceKey: ChefTipCatchupSourceKey(order),
		Currency: EarningsCurrency, Amount: 25.0,
	}).Error)

	reconcileStatementCatchup()

	var adjustments int64
	db.Model(&models.ChefBonus{}).Where("kind = ?", models.ChefBonusStatementAdjust).Count(&adjustments)
	assert.EqualValues(t, 0, adjustments)
}
