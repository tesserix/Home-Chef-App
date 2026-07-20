package handlers

// chef_delivered_gate_test.go — GH #391: a chef must not be able to mark a
// courier (`delivery`) order Delivered — doing so releases the held payout to
// themselves without ever delivering it. These tests cover the pure gate helper
// `chefMayMarkDelivered` and the DB-backed `UpdateOrderStatus` handler at the
// single choke point (before `order.Status = newStatus`). Legit self-delivery
// (`chef_delivery`) and pickup (`pickup`) handoffs stay allowed and are audited.
//
// SQLite harness mirrors meal_plan_booking_test.go: in-memory DB, hand-DDL'd
// tables (the models' gen_random_uuid() default can't create on SQLite), swap
// database.DB, restore in t.Cleanup. Delivered side-effects
// (MarkMealPlanDayDelivered / MarkGroupOrderDelivered) no-op when their tables
// are absent (they return early on the lookup error), so a blocked order never
// reaches them and an allowed order releases nothing here.

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	glogger "gorm.io/gorm/logger"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

// TestChefMayMarkDelivered is the pure decision table — no DB. The gate only
// bites the Delivered transition; every other transition is unaffected.
func TestChefMayMarkDelivered(t *testing.T) {
	cases := []struct {
		name string
		to   models.OrderStatus
		ft   models.FulfillmentType
		want bool
	}{
		{"delivery order blocked", models.OrderStatusDelivered, models.FulfillmentDelivery, false},
		{"unset fulfillment blocked", models.OrderStatusDelivered, models.FulfillmentType(""), false},
		{"chef_delivery allowed", models.OrderStatusDelivered, models.FulfillmentChefDelivery, true},
		{"pickup allowed", models.OrderStatusDelivered, models.FulfillmentPickup, true},
		{"non-delivered transition unaffected", models.OrderStatusReady, models.FulfillmentDelivery, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, chefMayMarkDelivered(tc.to, tc.ft))
		})
	}
}

// ── DB-backed harness ────────────────────────────────────────────────────────

// ordersDDL lists every column GORM's tx.Save(&order) writes; a missing column
// fails the UPDATE. id defaults to a valid UUID literal so RETURNING-style reads
// never scan NULL into a uuid field.
const ordersDDL = `CREATE TABLE orders (delivery_address_line1_enc text DEFAULT '', delivery_address_line2_enc text DEFAULT '', 
	id text PRIMARY KEY,
	order_number text, customer_id text, chef_id text, delivery_id text,
	status text, payment_status text, payment_method text, fulfillment_type text,
	subtotal real DEFAULT 0, delivery_fee real DEFAULT 0, service_fee real DEFAULT 0,
	tax real DEFAULT 0, tax_rate real DEFAULT 0, tax_name text, commission_rate real DEFAULT 0,
	tip real DEFAULT 0,
	chef_tip real DEFAULT 0, driver_tip real DEFAULT 0, discount real DEFAULT 0,
	chef_funded_discount real DEFAULT 0, total real DEFAULT 0, wallet_applied real DEFAULT 0,
	promo_code text, currency text,
	delivery_address_line1 text, delivery_address_line2 text, delivery_address_city text,
	delivery_address_state text, delivery_address_postal_code text, delivery_address_country text,
	delivery_latitude real DEFAULT 0, delivery_longitude real DEFAULT 0, delivery_instructions text,
	estimated_prep_time integer DEFAULT 0, estimated_delivery_time integer DEFAULT 0,
	scheduled_for datetime, delivery_slot text,
	accepted_at datetime, prepared_at datetime, picked_up_at datetime, delivered_at datetime,
	cancelled_at datetime, cancel_reason text, special_instructions text,
	ready_photo_url text, handover_photo_url text,
	payment_provider text, stripe_payment_intent_id text, razorpay_order_id text,
	razorpay_payment_id text, refund_id text, refunded_at datetime, refund_amount real DEFAULT 0,
	refund_reason text, refund_initiated_by text,
	payout_hold_status text DEFAULT '', customer_confirmed_at datetime,
	payout_settled_at datetime, payout_settle_attempts integer DEFAULT 0,
	created_at datetime, updated_at datetime, deleted_at datetime)`

const orderItemsDDL = `CREATE TABLE order_items (
	id text PRIMARY KEY, order_id text, menu_item_id text, name text, price real DEFAULT 0,
	quantity integer DEFAULT 0, subtotal real DEFAULT 0, notes text, modifiers text DEFAULT '[]',
	is_cancelled integer DEFAULT 0, cancelled_reason text, cancelled_at datetime,
	refund_id text, refund_amount real DEFAULT 0, created_at datetime)`

// outbox_events / audit_logs get inserted (not just read) on the allowed path.
// id defaults to a valid UUID literal and numeric default-tagged columns get a
// concrete default so GORM's post-insert reads never scan NULL.
const outboxDDL = `CREATE TABLE outbox_events (
	id text DEFAULT '00000000-0000-0000-0000-000000000000',
	subject text, msg_id text, aggregate_type text, aggregate_id text, payload text,
	status text, attempts integer DEFAULT 0, last_error text, next_retry_at datetime,
	created_at datetime, updated_at datetime, published_at datetime)`

const auditDDL = `CREATE TABLE audit_logs (
	id text DEFAULT '00000000-0000-0000-0000-000000000000',
	user_id text, action text, entity_type text, entity_id text,
	old_value text, new_value text, ip_address text, user_agent text,
	correlation_id text, created_at datetime)`

func setupChefOrderDB(t *testing.T) (*gorm.DB, uuid.UUID, uuid.UUID) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: glogger.Default.LogMode(glogger.Silent)})
	require.NoError(t, err)
	for _, s := range []string{
		`CREATE TABLE chef_profiles (address_line1_enc text DEFAULT '', address_line2_enc text DEFAULT '', id text PRIMARY KEY, user_id text, business_name text, is_active integer DEFAULT 1)`,
		ordersDDL, orderItemsDDL, outboxDDL, auditDDL,
	} {
		require.NoError(t, db.Exec(s).Error)
	}

	userID, chefID := uuid.New(), uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO chef_profiles (id, user_id, business_name, is_active) VALUES (?,?,?,1)`,
		chefID.String(), userID.String(), "Test Kitchen").Error)

	orig := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = orig })
	return db, userID, chefID
}

// seedReadyOrder inserts a `ready` order with the given fulfilment type, the
// exact pre-Delivered state a chef would be transitioning from.
func seedReadyOrder(t *testing.T, db *gorm.DB, chefID uuid.UUID, ft string) uuid.UUID {
	t.Helper()
	orderID, custID := uuid.New(), uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO orders
		(id, order_number, customer_id, chef_id, status, payment_status, fulfillment_type,
		 subtotal, total, currency, created_at, updated_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`,
		orderID.String(), "ORD-"+orderID.String()[:8], custID.String(), chefID.String(),
		"ready", "completed", ft, 100.0, 100.0, "INR", time.Now(), time.Now()).Error)
	return orderID
}

func postStatus(t *testing.T, userID, orderID uuid.UUID, status string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("userID", userID); c.Next() })
	r.POST("/chef/orders/:orderId/status", (&ChefHandler{}).UpdateOrderStatus)

	b, _ := json.Marshal(map[string]any{"status": status})
	req := httptest.NewRequest(http.MethodPost, "/chef/orders/"+orderID.String()+"/status", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func orderStatus(t *testing.T, db *gorm.DB, orderID uuid.UUID) string {
	t.Helper()
	var status string
	require.NoError(t, db.Raw(`SELECT status FROM orders WHERE id = ?`, orderID.String()).Scan(&status).Error)
	return status
}

// TestUpdateOrderStatus_BlocksDeliveryOrder — a 3PL `delivery` order chef→Delivered
// is rejected with a structured JSON error and the order stays `ready` (the
// release side-effects are never reached).
func TestUpdateOrderStatus_BlocksDeliveryOrder(t *testing.T) {
	db, userID, chefID := setupChefOrderDB(t)
	orderID := seedReadyOrder(t, db, chefID, "delivery")

	w := postStatus(t, userID, orderID, "delivered")
	require.Equal(t, http.StatusForbidden, w.Code, w.Body.String())

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.NotEmpty(t, body["error"], "structured error code expected")
	require.NotEmpty(t, body["message"], "human-readable message expected")

	require.Equal(t, "ready", orderStatus(t, db, orderID), "blocked order must not be marked delivered")
}

// TestUpdateOrderStatus_UnsetFulfillmentBlocked — empty fulfilment normalizes to
// `delivery` (GORM default) and is blocked (deny-by-default).
func TestUpdateOrderStatus_UnsetFulfillmentBlocked(t *testing.T) {
	db, userID, chefID := setupChefOrderDB(t)
	orderID := seedReadyOrder(t, db, chefID, "")

	w := postStatus(t, userID, orderID, "delivered")
	require.Equal(t, http.StatusForbidden, w.Code, w.Body.String())
	require.Equal(t, "ready", orderStatus(t, db, orderID))
}

// TestUpdateOrderStatus_AllowsChefDelivery — the chef self-delivery handoff stays
// allowed and the order is marked delivered.
func TestUpdateOrderStatus_AllowsChefDelivery(t *testing.T) {
	db, userID, chefID := setupChefOrderDB(t)
	orderID := seedReadyOrder(t, db, chefID, "chef_delivery")

	w := postStatus(t, userID, orderID, "delivered")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Equal(t, "delivered", orderStatus(t, db, orderID))
}

// TestUpdateOrderStatus_AllowsPickup — a pickup handoff stays allowed.
func TestUpdateOrderStatus_AllowsPickup(t *testing.T) {
	db, userID, chefID := setupChefOrderDB(t)
	orderID := seedReadyOrder(t, db, chefID, "pickup")

	w := postStatus(t, userID, orderID, "delivered")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Equal(t, "delivered", orderStatus(t, db, orderID))
}

// TestUpdateOrderStatus_ReStamp_PreservesAdvancedHold proves the chef status
// persist no longer clobbers the payout-hold columns (#460 race 1). A chef
// re-submitting `delivered` on an already-delivered order must NOT revert a hold
// that a concurrent customer-confirm advanced in the load→persist window. We
// simulate that concurrent confirm with a one-shot Before-update callback that
// flips the row to release_eligible (fresh NewDB session) just before the
// handler's persist UPDATE. Under the old full-row tx.Save(&order) the load-time
// `awaiting` overwrites it; the targeted Updates leaves the hold columns untouched.
func TestUpdateOrderStatus_ReStamp_PreservesAdvancedHold(t *testing.T) {
	db, userID, chefID := setupChefOrderDB(t)
	orderID, custID := uuid.New(), uuid.New()
	past := time.Now().Add(-time.Hour)
	require.NoError(t, db.Exec(`INSERT INTO orders
		(id, order_number, customer_id, chef_id, status, payment_status, fulfillment_type,
		 subtotal, total, currency, delivered_at, payout_hold_status, created_at, updated_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		orderID.String(), "ORD-"+orderID.String()[:8], custID.String(), chefID.String(),
		"delivered", "completed", "chef_delivery", 100.0, 100.0, "INR", past,
		string(models.PayoutHoldAwaitingConfirmation), time.Now(), time.Now()).Error)

	injected := false
	require.NoError(t, db.Callback().Update().Before("gorm:update").Register("inject_confirm", func(tx *gorm.DB) {
		if injected {
			return
		}
		injected = true
		tx.Session(&gorm.Session{NewDB: true}).Exec(
			`UPDATE orders SET payout_hold_status = ?, customer_confirmed_at = ? WHERE id = ?`,
			string(models.PayoutHoldReleaseEligible), time.Now(), orderID.String())
	}))

	w := postStatus(t, userID, orderID, "delivered")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var hold string
	require.NoError(t, db.Raw(`SELECT payout_hold_status FROM orders WHERE id = ?`, orderID.String()).Scan(&hold).Error)
	require.Equal(t, string(models.PayoutHoldReleaseEligible), hold,
		"chef status persist must not clobber a concurrently-advanced payout hold")
}

// TestUpdateOrderStatus_WritesAuditOnAllowedDelivered — every chef-initiated
// Delivered transition writes an AuditLog row capturing actor, order id, and
// fulfilment type (repudiation control, T-391-02).
func TestUpdateOrderStatus_WritesAuditOnAllowedDelivered(t *testing.T) {
	db, userID, chefID := setupChefOrderDB(t)
	orderID := seedReadyOrder(t, db, chefID, "chef_delivery")

	w := postStatus(t, userID, orderID, "delivered")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var row struct {
		UserID   string
		Action   string
		NewValue string
	}
	require.NoError(t, db.Raw(
		`SELECT user_id, action, new_value FROM audit_logs WHERE entity_id = ? AND action = 'chef.order.delivered'`,
		orderID.String()).Scan(&row).Error)

	require.Equal(t, "chef.order.delivered", row.Action, "an audit row for the delivered transition must exist")
	require.Equal(t, userID.String(), row.UserID, "audit row must record the chef actor")
	require.Contains(t, row.NewValue, "chef_delivery", "audit row must capture the fulfilment type")
}
