package services

// chef_payout_estimate_test.go — the pre-delivery figure.
//
// A chef decides whether to take an order when it ARRIVES. Gating the payout on
// the delivery-time write left every pending order rendering the CUSTOMER's
// total (₹745.06 on an order the kitchen earns ₹679.16 for), which is the
// defect the payout row was introduced to remove. These pin that the estimate
// and the persisted row are the same formula, and that the row wins once it
// exists — a settled figure is never recomputed underneath a chef.

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/homechef/api/models"
)

func setupPayoutReadDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TABLE order_chef_payouts (id TEXT PRIMARY KEY, order_id TEXT UNIQUE,
		chef_id TEXT, currency TEXT, food_amount REAL, delivery_fee REAL, chef_tip REAL, penalty REAL,
		net_payout REAL, status TEXT, computed_at DATETIME, created_at DATETIME, updated_at DATETIME)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE chef_penalties (id TEXT PRIMARY KEY, chef_id TEXT, user_id TEXT,
		kind TEXT, status TEXT, source_key TEXT UNIQUE, order_id TEXT, reference TEXT, currency TEXT,
		basis_amount REAL, rate_percent REAL, amount REAL, lead_hours REAL, reason TEXT,
		deducted_statement_id TEXT, deducted_at DATETIME, waived_by TEXT, waived_at DATETIME,
		waive_reason TEXT, occurred_at DATETIME, created_at DATETIME, updated_at DATETIME)`).Error)
	return db
}

// pendingOrder is an order the chef has not yet accepted — no payout row can
// exist for it.
func pendingOrder(subtotal, deliveryFee float64, ft models.FulfillmentType) *models.Order {
	o := chefOrder(subtotal, deliveryFee, 0, ft)
	o.ID = uuid.New()
	o.ChefID = uuid.New()
	o.Status = models.OrderStatusPending
	return o
}

// The order from the report: the chef must see ₹679.16, not the customer's
// ₹745.06, while it is still sitting in the incoming queue.
func TestChefPayoutFor_EstimatesBeforeDelivery(t *testing.T) {
	db := setupPayoutReadDB(t)
	o := pendingOrder(640, 39.16, models.FulfillmentChefDelivery)

	p := ChefPayoutFor(db, o)
	require.NotNil(t, p, "a chef triaging an order must always have a figure")
	require.Equal(t, models.ChefPayoutEstimated, p.Status)
	require.Equal(t, 640.00, p.FoodAmount)
	require.Equal(t, 39.16, p.DeliveryFee)
	require.Equal(t, 679.16, p.NetPayout)
	require.Equal(t, "INR", p.Currency)
}

// The estimate is the SAME function the delivery-time write uses — one formula,
// so the number a chef accepted on is the number they are paid.
func TestChefPayoutFor_EstimateMatchesThePersistedRow(t *testing.T) {
	db := setupPayoutReadDB(t)
	o := pendingOrder(640, 39.16, models.FulfillmentChefDelivery)
	estimate := ChefPayoutFor(db, o)

	_, err := RecordChefPayout(db, o, 0)
	require.NoError(t, err)

	settled := ChefPayoutFor(db, o)
	require.Equal(t, models.ChefPayoutPending, settled.Status, "the row must win once written")
	require.Equal(t, estimate.NetPayout, settled.NetPayout)
	require.Equal(t, estimate.FoodAmount, settled.FoodAmount)
	require.Equal(t, estimate.DeliveryFee, settled.DeliveryFee)
}

// Once money is settled the stored figures stand, even if the order row is
// later edited. A released payout is history, not a projection.
func TestChefPayoutFor_SettledRowIsNotRecomputed(t *testing.T) {
	db := setupPayoutReadDB(t)
	o := pendingOrder(640, 0, models.FulfillmentDelivery)
	_, err := RecordChefPayout(db, o, 0)
	require.NoError(t, err)
	require.NoError(t, db.Exec(`UPDATE order_chef_payouts SET status = ? WHERE order_id = ?`,
		models.ChefPayoutReleased, o.ID).Error)

	o.Subtotal = 1000 // the order changed underneath; the payment did not

	p := ChefPayoutFor(db, o)
	require.Equal(t, models.ChefPayoutReleased, p.Status)
	require.Equal(t, 640.00, p.NetPayout, "a released payout must not follow the order row")
}

// A levy already raised against the order is deducted from the estimate too —
// the chef is not shown a figure the settlement will then quietly reduce.
func TestChefPayoutFor_EstimateDeductsAnOpenPenalty(t *testing.T) {
	db := setupPayoutReadDB(t)
	o := pendingOrder(640, 0, models.FulfillmentDelivery)
	require.NoError(t, db.Create(&models.ChefPenalty{
		ID: uuid.New(), ChefID: o.ChefID, SourceKey: ChefCancelPenaltySourceKey(o.ID),
		Status: models.ChefPenaltyPending, Amount: 40, OccurredAt: time.Now().UTC(),
	}).Error)

	p := ChefPayoutFor(db, o)
	require.Equal(t, 40.00, p.Penalty)
	require.Equal(t, 600.00, p.NetPayout)
}

// The list path must agree with the detail path row for row, and mix settled
// rows with estimates in one page — the vendor feed shows both at once.
func TestChefPayoutsFor_BatchMatchesTheSingleReader(t *testing.T) {
	db := setupPayoutReadDB(t)
	delivered := pendingOrder(320, 0, models.FulfillmentDelivery)
	_, err := RecordChefPayout(db, delivered, 0)
	require.NoError(t, err)
	incoming := pendingOrder(640, 39.16, models.FulfillmentChefDelivery)
	penalised := pendingOrder(500, 0, models.FulfillmentPickup)
	require.NoError(t, db.Create(&models.ChefPenalty{
		ID: uuid.New(), ChefID: penalised.ChefID, SourceKey: ChefCancelPenaltySourceKey(penalised.ID),
		Status: models.ChefPenaltyPending, Amount: 30, OccurredAt: time.Now().UTC(),
	}).Error)

	orders := []models.Order{*delivered, *incoming, *penalised}
	batch := ChefPayoutsFor(db, orders)
	require.Len(t, batch, 3)
	for i := range orders {
		one := ChefPayoutFor(db, &orders[i])
		got := batch[orders[i].ID]
		require.NotNil(t, got)
		require.Equal(t, one.Status, got.Status)
		require.Equal(t, one.NetPayout, got.NetPayout)
		require.Equal(t, one.Penalty, got.Penalty)
	}
	require.Equal(t, models.ChefPayoutPending, batch[delivered.ID].Status)
	require.Equal(t, 679.16, batch[incoming.ID].NetPayout)
	require.Equal(t, 470.00, batch[penalised.ID].NetPayout)
}

// A waived levy was cancelled by an admin, so it was never owed — it must not
// depress the estimate either.
func TestChefPayoutsFor_IgnoresWaivedPenalties(t *testing.T) {
	db := setupPayoutReadDB(t)
	o := pendingOrder(640, 0, models.FulfillmentDelivery)
	require.NoError(t, db.Create(&models.ChefPenalty{
		ID: uuid.New(), ChefID: o.ChefID, SourceKey: ChefCancelPenaltySourceKey(o.ID),
		Status: models.ChefPenaltyWaived, Amount: 40, OccurredAt: time.Now().UTC(),
	}).Error)

	batch := ChefPayoutsFor(db, []models.Order{*o})
	require.Equal(t, 0.00, batch[o.ID].Penalty)
	require.Equal(t, 640.00, batch[o.ID].NetPayout)
}

func TestChefPayoutsFor_EmptyPage(t *testing.T) {
	require.Empty(t, ChefPayoutsFor(setupPayoutReadDB(t), nil))
}
