package handlers

// chef_orders_payout_test.go — the chef's LIST surfaces must carry what the
// kitchen earns, not the customer's bill.
//
// The reported bug: the incoming-order card read ₹745.06 on an order the chef
// earns ₹679.16 for. The payout row is written at delivery, so a pending order
// had none and every list fell back to `total`. A chef decides whether to take
// an order when it ARRIVES — so the figure has to be served from the moment they
// first see it, estimated by the same formula the delivery-time row freezes.
//
// Route-level on purpose: the arithmetic is pinned in services, what these hold
// is that the JSON a vendor app actually receives carries it.

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
)

func payoutTables(t *testing.T, db *gorm.DB) {
	t.Helper()
	require.NoError(t, db.Exec(`CREATE TABLE order_chef_payouts (id TEXT PRIMARY KEY, order_id TEXT UNIQUE,
		chef_id TEXT, currency TEXT, food_amount REAL, delivery_fee REAL, chef_tip REAL, penalty REAL,
		net_payout REAL, status TEXT, computed_at DATETIME, created_at DATETIME, updated_at DATETIME)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE chef_penalties (id TEXT PRIMARY KEY, chef_id TEXT, user_id TEXT,
		kind TEXT, status TEXT, source_key TEXT UNIQUE, order_id TEXT, reference TEXT, currency TEXT,
		basis_amount REAL, rate_percent REAL, amount REAL, lead_hours REAL, reason TEXT,
		deducted_statement_id TEXT, deducted_at DATETIME, waived_by TEXT, waived_at DATETIME,
		waive_reason TEXT, occurred_at DATETIME, created_at DATETIME, updated_at DATETIME)`).Error)
}

// seedPayoutOrder is the order from the report: ₹640 of food, a ₹39.16 delivery
// leg, billed to the customer at ₹745.06 with the platform fee and GST on top.
func seedPayoutOrder(t *testing.T, db *gorm.DB, chefID uuid.UUID, status string, ft models.FulfillmentType) uuid.UUID {
	t.Helper()
	id := seedVisOrderID(t, db, chefID, status, "completed")
	require.NoError(t, db.Exec(`UPDATE orders
		SET subtotal = 640, delivery_fee = 39.16, total = 745.06, fulfillment_type = ? WHERE id = ?`,
		string(ft), id.String()).Error)
	return id
}

// payoutsByID pulls each order's `chefPayout` object out of a /chef/orders
// response, keyed by order id.
func payoutsByID(t *testing.T, body map[string]any) map[string]map[string]any {
	t.Helper()
	out := map[string]map[string]any{}
	orders, _ := body["orders"].([]any)
	for _, o := range orders {
		m, _ := o.(map[string]any)
		id, _ := m["id"].(string)
		p, _ := m["chefPayout"].(map[string]any)
		out[id] = p
	}
	return out
}

// The card the chef triages on. It must carry the payout while the order is
// still pending — that is the whole point of the estimate.
func TestChefOrders_PendingOrderCarriesTheEstimatedPayout(t *testing.T) {
	db := setupChefVisDB(t)
	payoutTables(t, db)
	userID, chefID := seedVisChef(t, db)
	id := seedPayoutOrder(t, db, chefID, "pending", models.FulfillmentChefDelivery)

	body := chefVisGET(t, chefVisRouter(userID), "/chef/orders?status=pending")
	p := payoutsByID(t, body)[id.String()]
	require.NotNil(t, p, "an incoming order must carry chefPayout, not just total")
	require.Equal(t, models.ChefPayoutEstimated, p["status"])
	require.Equal(t, 679.16, p["netPayout"], "chef earns food + the leg they carry")
	require.Equal(t, 640.00, p["foodAmount"])
	require.Equal(t, 39.16, p["deliveryFee"])
}

// The customer's ₹745.06 carries the platform fee and the GST the platform
// accounts for. It must never be the figure a chef reads.
func TestChefOrders_PayoutIsNotTheCustomerTotal(t *testing.T) {
	db := setupChefVisDB(t)
	payoutTables(t, db)
	userID, chefID := seedVisChef(t, db)
	id := seedPayoutOrder(t, db, chefID, "pending", models.FulfillmentChefDelivery)

	body := chefVisGET(t, chefVisRouter(userID), "/chef/orders?status=pending")
	p := payoutsByID(t, body)[id.String()]
	require.NotEqual(t, 745.06, p["netPayout"])
}

// A platform-carried leg is not the kitchen's, so the fee is excluded — the
// same rule the list and the detail screen both render.
func TestChefOrders_PlatformCarriedLegExcludesTheDeliveryFee(t *testing.T) {
	db := setupChefVisDB(t)
	payoutTables(t, db)
	userID, chefID := seedVisChef(t, db)
	id := seedPayoutOrder(t, db, chefID, "pending", models.FulfillmentDelivery)

	body := chefVisGET(t, chefVisRouter(userID), "/chef/orders?status=pending")
	p := payoutsByID(t, body)[id.String()]
	require.Equal(t, 0.00, p["deliveryFee"])
	require.Equal(t, 640.00, p["netPayout"])
}

// Once the order is delivered the persisted row is served instead — the list
// must show the settled figure, not recompute one from a row that may have
// moved since.
func TestChefOrders_DeliveredOrderServesThePersistedRow(t *testing.T) {
	db := setupChefVisDB(t)
	payoutTables(t, db)
	userID, chefID := seedVisChef(t, db)
	id := seedPayoutOrder(t, db, chefID, "delivered", models.FulfillmentChefDelivery)
	require.NoError(t, db.Exec(`INSERT INTO order_chef_payouts
		(id, order_id, chef_id, currency, food_amount, delivery_fee, chef_tip, penalty, net_payout, status, computed_at)
		VALUES (?, ?, ?, 'INR', 640, 39.16, 0, 0, 679.16, ?, CURRENT_TIMESTAMP)`,
		uuid.NewString(), id.String(), chefID.String(), models.ChefPayoutPending).Error)

	body := chefVisGET(t, chefVisRouter(userID), "/chef/orders")
	p := payoutsByID(t, body)[id.String()]
	require.Equal(t, models.ChefPayoutPending, p["status"], "a written row is not an estimate")
	require.Equal(t, 679.16, p["netPayout"])
}

// A levy raised against the order is deducted from the list figure too, so the
// chef is never shown a number the settlement will quietly reduce.
func TestChefOrders_EstimateDeductsAnOpenPenalty(t *testing.T) {
	db := setupChefVisDB(t)
	payoutTables(t, db)
	userID, chefID := seedVisChef(t, db)
	id := seedPayoutOrder(t, db, chefID, "pending", models.FulfillmentDelivery)
	require.NoError(t, db.Exec(`INSERT INTO chef_penalties (id, chef_id, source_key, status, amount)
		VALUES (?, ?, ?, ?, 40)`,
		uuid.NewString(), chefID.String(), services.ChefCancelPenaltySourceKey(id), models.ChefPenaltyPending).Error)

	body := chefVisGET(t, chefVisRouter(userID), "/chef/orders?status=pending")
	p := payoutsByID(t, body)[id.String()]
	require.Equal(t, 40.00, p["penalty"])
	require.Equal(t, 600.00, p["netPayout"])
}

// The dashboard's in-flight cards render the payout too — the second surface
// that was showing `total`.
func TestChefDashboard_ActiveOrdersCarryThePayout(t *testing.T) {
	db := setupChefVisDB(t)
	payoutTables(t, db)
	userID, chefID := seedVisChef(t, db)
	id := seedPayoutOrder(t, db, chefID, "preparing", models.FulfillmentChefDelivery)

	body := chefVisGET(t, chefVisRouter(userID), "/chef/dashboard")
	active, _ := body["activeOrders"].([]any)
	require.Len(t, active, 1)
	row, _ := active[0].(map[string]any)
	require.Equal(t, id.String(), row["id"])
	p, _ := row["chefPayout"].(map[string]any)
	require.NotNil(t, p, "the in-flight card must carry chefPayout")
	require.Equal(t, 679.16, p["netPayout"])
	// `total` stays on the row so an app older than this change still renders
	// something; new builds ignore it.
	require.Equal(t, 745.06, row["total"])
}

// The reader must not fall over on a database that predates the payout table —
// a list of orders is more important than the payout line on it.
func TestChefOrders_SurvivesAMissingPayoutTable(t *testing.T) {
	db := setupChefVisDB(t) // deliberately WITHOUT the payout/penalty tables
	userID, chefID := seedVisChef(t, db)
	seedPayoutOrder(t, db, chefID, "pending", models.FulfillmentChefDelivery)

	body := chefVisGET(t, chefVisRouter(userID), "/chef/orders?status=pending")
	orders, _ := body["orders"].([]any)
	require.Len(t, orders, 1)
}
