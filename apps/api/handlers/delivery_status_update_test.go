package handlers

// delivery_status_update_test.go — #633. UpdateDeliveryStatus (own-fleet driver handler) had
// ZERO automated coverage; its #631 order-resurrection guards were proven only by tested analogs
// (shadowfax webhook / delivery_retry). This drives the handler end-to-end over the two
// deterministic, money-critical status paths — picked_up and cancelled — asserting:
//   - the #631 guard: a late own-fleet report on an already-terminal order (cancelled/refunded)
//     must NOT resurrect it (→ delivering / ready), which would fold it back into the weekly
//     statement; and
//   - the regression: on a LIVE order the same reports advance/reset it correctly (proving the
//     guard's `status NOT IN terminal` predicate doesn't block the normal path, AND that the
//     trailing Save(&delivery) doesn't clobber the guarded order write via its preloaded Order).
//
// The DELIVERED path is deliberately NOT driven here: it fans out fire-and-forget goroutines
// (subscription earnings / invoice / Stripe over the global database.DB) that race t.Cleanup, and
// the failed/returned path calls services that use Postgres now()/GREATEST (not sqlite-drivable).

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/homechef/api/database"
)

// setupDeliveryStatusDB creates the tables UpdateDeliveryStatus touches on the picked_up/cancelled
// paths. deliveries carries the FULL Delivery column set because the handler ends with
// database.DB.Save(&delivery) (a whole-struct update) — a column subset would error.
func setupDeliveryStatusDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	require.NoError(t, err)

	require.NoError(t, db.Exec(`CREATE TABLE delivery_partners (
		id TEXT PRIMARY KEY, user_id TEXT, total_deliveries INTEGER DEFAULT 0,
		current_latitude REAL DEFAULT 0, current_longitude REAL DEFAULT 0,
		stripe_account_id TEXT DEFAULT '', stripe_payouts_enabled INTEGER DEFAULT 0,
		created_at DATETIME, updated_at DATETIME
	)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE deliveries (
		id TEXT PRIMARY KEY, order_id TEXT, delivery_partner_id TEXT, status TEXT DEFAULT 'pending',
		pickup_address_line1 TEXT DEFAULT '', pickup_address_city TEXT DEFAULT '',
		pickup_latitude REAL DEFAULT 0, pickup_longitude REAL DEFAULT 0,
		dropoff_address_line1 TEXT DEFAULT '', dropoff_address_city TEXT DEFAULT '',
		dropoff_latitude REAL DEFAULT 0, dropoff_longitude REAL DEFAULT 0,
		distance REAL DEFAULT 0, estimated_duration INTEGER DEFAULT 0, actual_duration INTEGER DEFAULT 0,
		attempt_number INTEGER DEFAULT 1, max_attempts INTEGER DEFAULT 3, failure_reason TEXT DEFAULT '',
		assignment_type TEXT DEFAULT 'manual', assigned_by_id TEXT, offer_expires_at DATETIME,
		provider_id TEXT, external_delivery_id TEXT DEFAULT '', external_tracking_id TEXT DEFAULT '',
		external_tracking_url TEXT DEFAULT '', provider_cost REAL DEFAULT 0,
		rider_name TEXT DEFAULT '', rider_phone TEXT DEFAULT '',
		rider_latitude REAL DEFAULT 0, rider_longitude REAL DEFAULT 0, provider_status TEXT DEFAULT '',
		delivery_fee REAL DEFAULT 0, tip REAL DEFAULT 0, total_payout REAL DEFAULT 0,
		assigned_at DATETIME, picked_up_at DATETIME, delivered_at DATETIME,
		cancelled_at DATETIME, cancel_reason TEXT DEFAULT ''
	)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE orders (
		id TEXT PRIMARY KEY, order_number TEXT, customer_id TEXT, chef_id TEXT, delivery_id TEXT,
		status TEXT DEFAULT 'pending', payment_status TEXT DEFAULT 'pending',
		subtotal REAL DEFAULT 0, tax REAL DEFAULT 0, total REAL DEFAULT 0,
		razorpay_order_id TEXT DEFAULT '', razorpay_payment_id TEXT DEFAULT '',
		picked_up_at DATETIME, delivered_at DATETIME,
		created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
	)`).Error)
	// meal_plan_days / group_orders: only touched on the DELIVERED path (not driven), but present
	// so any stray classification probe is a clean empty read rather than a missing-table error.
	require.NoError(t, db.Exec(`CREATE TABLE meal_plan_days (id TEXT PRIMARY KEY, order_id TEXT, deleted_at DATETIME)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE group_orders (id TEXT PRIMARY KEY, order_id TEXT, deleted_at DATETIME)`).Error)

	prev := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = prev })
	return db
}

func seedDeliveryPartner(t *testing.T, db *gorm.DB) (userID, partnerID uuid.UUID) {
	t.Helper()
	userID, partnerID = uuid.New(), uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO delivery_partners (id, user_id, created_at, updated_at) VALUES (?,?,?,?)`,
		partnerID.String(), userID.String(), time.Now(), time.Now()).Error)
	return userID, partnerID
}

func seedStatusOrder(t *testing.T, db *gorm.DB, status string, deliveryID string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO orders (id, order_number, status, delivery_id, created_at) VALUES (?,?,?,?,?)`,
		id.String(), "ORD-"+id.String()[:6], status, deliveryID, time.Now()).Error)
	return id
}

func seedOwnFleetDelivery(t *testing.T, db *gorm.DB, orderID, partnerID uuid.UUID, status string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO deliveries (id, order_id, delivery_partner_id, status, assignment_type, assigned_at) VALUES (?,?,?,?, 'manual', ?)`,
		id.String(), orderID.String(), partnerID.String(), status, time.Now()).Error)
	return id
}

func postDeliveryStatus(userID uuid.UUID, deliveryID, body string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("userID", userID) })
	r.PUT("/deliveries/:id/status", (&DeliveryHandler{}).UpdateDeliveryStatus)
	req := httptest.NewRequest(http.MethodPut, "/deliveries/"+deliveryID+"/status", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func orderStatusOf(t *testing.T, db *gorm.DB, orderID uuid.UUID) (status string, deliveryID *string) {
	t.Helper()
	var row struct {
		Status     string
		DeliveryID *string
	}
	require.NoError(t, db.Raw(`SELECT status, delivery_id FROM orders WHERE id = ?`, orderID.String()).Scan(&row).Error)
	return row.Status, row.DeliveryID
}

func deliveryStatusOf(t *testing.T, db *gorm.DB, deliveryID uuid.UUID) string {
	t.Helper()
	var s string
	require.NoError(t, db.Raw(`SELECT status FROM deliveries WHERE id = ?`, deliveryID.String()).Scan(&s).Error)
	return s
}

// picked_up on a LIVE order advances it to delivering — the regression that also proves the
// trailing Save(&delivery) does NOT clobber the guarded order write via its preloaded Order.
func TestUpdateDeliveryStatus_PickedUp_LiveOrderAdvances(t *testing.T) {
	db := setupDeliveryStatusDB(t)
	userID, partnerID := seedDeliveryPartner(t, db)
	orderID := seedStatusOrder(t, db, "ready", "")
	deliveryID := seedOwnFleetDelivery(t, db, orderID, partnerID, "assigned")

	w := postDeliveryStatus(userID, deliveryID.String(), `{"status":"picked_up"}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	require.Equal(t, "picked_up", deliveryStatusOf(t, db, deliveryID))
	status, _ := orderStatusOf(t, db, orderID)
	require.Equal(t, "delivering", status, "a live order advances to delivering (guard passes; Save doesn't clobber)")
}

// #631 guard: picked_up on an already-CANCELLED order must NOT resurrect it to delivering.
func TestUpdateDeliveryStatus_PickedUp_TerminalOrderNotResurrected(t *testing.T) {
	db := setupDeliveryStatusDB(t)
	userID, partnerID := seedDeliveryPartner(t, db)
	orderID := seedStatusOrder(t, db, "cancelled", "")
	deliveryID := seedOwnFleetDelivery(t, db, orderID, partnerID, "assigned")

	w := postDeliveryStatus(userID, deliveryID.String(), `{"status":"picked_up"}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	require.Equal(t, "picked_up", deliveryStatusOf(t, db, deliveryID))
	status, _ := orderStatusOf(t, db, orderID)
	require.Equal(t, "cancelled", status, "#631: a late picked_up must not resurrect a cancelled order into the statement")
}

// cancelled on a LIVE order resets it to ready and clears the delivery assignment (regression).
func TestUpdateDeliveryStatus_Cancelled_LiveOrderReset(t *testing.T) {
	db := setupDeliveryStatusDB(t)
	userID, partnerID := seedDeliveryPartner(t, db)
	// order is delivering with this delivery assigned.
	orderID := uuid.New()
	deliveryID := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO orders (id, order_number, status, delivery_id, created_at) VALUES (?,?,?,?,?)`,
		orderID.String(), "ORD-live", "delivering", deliveryID.String(), time.Now()).Error)
	require.NoError(t, db.Exec(`INSERT INTO deliveries (id, order_id, delivery_partner_id, status, assignment_type, assigned_at) VALUES (?,?,?,?, 'manual', ?)`,
		deliveryID.String(), orderID.String(), partnerID.String(), "assigned", time.Now()).Error)

	w := postDeliveryStatus(userID, deliveryID.String(), `{"status":"cancelled","cancelReason":"driver bailed"}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	require.Equal(t, "cancelled", deliveryStatusOf(t, db, deliveryID))
	status, delID := orderStatusOf(t, db, orderID)
	require.Equal(t, "ready", status, "a live order is reset to ready so another driver can pick up")
	require.True(t, delID == nil || *delID == "", "the delivery assignment is cleared")
}

// #631 guard: cancelling the delivery on an already-REFUNDED order must NOT reset it to ready
// (which AcceptDelivery matches on) — that would let a fresh delivery cycle resurrect it.
func TestUpdateDeliveryStatus_Cancelled_TerminalOrderNotReset(t *testing.T) {
	db := setupDeliveryStatusDB(t)
	userID, partnerID := seedDeliveryPartner(t, db)
	orderID := uuid.New()
	deliveryID := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO orders (id, order_number, status, delivery_id, created_at) VALUES (?,?,?,?,?)`,
		orderID.String(), "ORD-refunded", "refunded", deliveryID.String(), time.Now()).Error)
	require.NoError(t, db.Exec(`INSERT INTO deliveries (id, order_id, delivery_partner_id, status, assignment_type, assigned_at) VALUES (?,?,?,?, 'manual', ?)`,
		deliveryID.String(), orderID.String(), partnerID.String(), "assigned", time.Now()).Error)

	w := postDeliveryStatus(userID, deliveryID.String(), `{"status":"cancelled","cancelReason":"x"}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	status, delID := orderStatusOf(t, db, orderID)
	require.Equal(t, "refunded", status, "#631: a delivery cancel must not reset a refunded order to ready")
	require.NotNil(t, delID)
	require.Equal(t, deliveryID.String(), *delID, "the terminal order's delivery assignment is left intact")
}

// Cheap guard coverage: unknown delivery → 404; a terminal delivery state → 400; a failed status
// without a structured failureReason → 400.
func TestUpdateDeliveryStatus_Guards(t *testing.T) {
	db := setupDeliveryStatusDB(t)
	userID, partnerID := seedDeliveryPartner(t, db)

	// Unknown delivery id.
	require.Equal(t, http.StatusNotFound,
		postDeliveryStatus(userID, uuid.NewString(), `{"status":"picked_up"}`).Code)

	// Terminal delivery state (delivered) has no valid transitions → 400.
	orderID := seedStatusOrder(t, db, "delivered", "")
	delivered := seedOwnFleetDelivery(t, db, orderID, partnerID, "delivered")
	require.Equal(t, http.StatusBadRequest,
		postDeliveryStatus(userID, delivered.String(), `{"status":"cancelled"}`).Code)

	// failed without a structured failureReason → 400.
	o2 := seedStatusOrder(t, db, "delivering", "")
	atDropoff := seedOwnFleetDelivery(t, db, o2, partnerID, "at_dropoff")
	require.Equal(t, http.StatusBadRequest,
		postDeliveryStatus(userID, atDropoff.String(), `{"status":"failed"}`).Code)
}
