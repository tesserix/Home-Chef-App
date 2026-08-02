package handlers

// chef_cancellation_freeze_test.go — #475: while a customer's cancellation
// request is awaiting the chef, the order is FROZEN.
//
// The customer's screen says "Cancellation requested — waiting for the chef";
// if the chef can still advance the order it reaches Ready and goes out the
// door under a request the customer believes is open, and the refund tier the
// chef later picks is judged against a stage they only reached AFTER being
// asked to stop. The two apps disagree about the same order and the money
// follows the wrong one.
//
// The vendor app greys the stage button on the same condition, but the app is
// not the enforcement — a stale screen, a retried request or any other client
// bypasses it. These tests pin the SERVER guard, and the detail field the app
// reads to explain itself.
//
// Reuses setupChefOrderDB / postStatus / orderStatus from
// chef_delivered_gate_test.go — same package, same in-memory SQLite harness.

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

// cancellationRequestsDDL covers only the columns pendingCancellationFor reads
// or filters on; the handler never writes this table.
const cancellationRequestsDDL = `CREATE TABLE cancellation_requests (
	id text PRIMARY KEY,
	order_id text, customer_id text, chef_id text,
	status text, customer_reason text, vendor_reason text, refund_destination text,
	vendor_respond_by datetime, resolved_at datetime,
	created_at datetime, updated_at datetime)`

// seedAcceptedOrder inserts an order mid-lifecycle — the state a customer's
// cancellation request actually lands on (pre-accept requests auto-refund and
// never reach the vendor).
func seedAcceptedOrder(t *testing.T, db *gorm.DB, chefID uuid.UUID, status string) uuid.UUID {
	t.Helper()
	orderID, custID := uuid.New(), uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO orders
		(id, order_number, customer_id, chef_id, status, payment_status, fulfillment_type,
		 subtotal, total, currency, created_at, updated_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`,
		orderID.String(), "ORD-"+orderID.String()[:8], custID.String(), chefID.String(),
		status, "completed", "chef_delivery", 100.0, 100.0, "INR", time.Now(), time.Now()).Error)
	return orderID
}

func seedCancellationRequest(t *testing.T, db *gorm.DB, orderID, chefID uuid.UUID, status, reason string) uuid.UUID {
	t.Helper()
	reqID := uuid.New()
	respondBy := time.Now().Add(30 * time.Minute)
	require.NoError(t, db.Exec(`INSERT INTO cancellation_requests
		(id, order_id, customer_id, chef_id, status, customer_reason, vendor_respond_by, created_at, updated_at)
		VALUES (?,?,?,?,?,?,?,?,?)`,
		reqID.String(), orderID.String(), uuid.New().String(), chefID.String(),
		status, reason, respondBy, time.Now(), time.Now()).Error)
	return reqID
}

// setupCancellationDB is setupChefOrderDB plus the cancellation table, and the
// two tables GetOrderDetail preloads (Customer, Items.MenuItem) — a missing
// preload target makes the whole lookup error, which the handler reports as a
// 404 rather than as the schema problem it is.
func setupCancellationDB(t *testing.T) (*gorm.DB, uuid.UUID, uuid.UUID) {
	t.Helper()
	db, userID, chefID := setupChefOrderDB(t)
	for _, s := range []string{
		cancellationRequestsDDL,
		`CREATE TABLE users (id text PRIMARY KEY, first_name text, last_name text,
			email text, phone text, role text, created_at datetime, updated_at datetime, deleted_at datetime)`,
		`CREATE TABLE menu_items (id text PRIMARY KEY, chef_id text, name text,
			is_veg integer DEFAULT 0, created_at datetime, updated_at datetime, deleted_at datetime)`,
	} {
		require.NoError(t, db.Exec(s).Error)
	}
	return db, userID, chefID
}

// TestUpdateOrderStatus_FrozenByPendingCancellation is the core guard: a chef
// cannot move an order on while the customer's request awaits them.
func TestUpdateOrderStatus_FrozenByPendingCancellation(t *testing.T) {
	db, userID, chefID := setupCancellationDB(t)
	orderID := seedAcceptedOrder(t, db, chefID, "preparing")
	seedCancellationRequest(t, db, orderID, chefID, string(models.CancelReqPendingVendor), "ordered by mistake")

	w := postStatus(t, userID, orderID, "ready")
	require.Equal(t, http.StatusConflict, w.Code, w.Body.String())

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, "cancellation_pending", body["error"])
	require.NotEmpty(t, body["message"], "the chef needs to be told what to do instead")

	require.Equal(t, "preparing", orderStatus(t, db, orderID),
		"a frozen order must not advance")
}

// TestUpdateOrderStatus_FrozenAtEveryStage — the freeze is not specific to one
// transition. Whichever stage the request lands on, the next one is refused.
func TestUpdateOrderStatus_FrozenAtEveryStage(t *testing.T) {
	for _, tc := range []struct{ from, to string }{
		{"accepted", "preparing"},
		{"preparing", "ready"},
		{"ready", "picked_up"},
	} {
		t.Run(tc.from+"->"+tc.to, func(t *testing.T) {
			db, userID, chefID := setupCancellationDB(t)
			orderID := seedAcceptedOrder(t, db, chefID, tc.from)
			seedCancellationRequest(t, db, orderID, chefID, string(models.CancelReqPendingVendor), "changed my mind")

			w := postStatus(t, userID, orderID, tc.to)
			require.Equal(t, http.StatusConflict, w.Code, w.Body.String())
			require.Equal(t, tc.from, orderStatus(t, db, orderID))
		})
	}
}

// TestUpdateOrderStatus_ResolvedRequestDoesNotFreeze — only pending_vendor
// freezes. Once the request is settled the order moves again, or a declined
// cancellation would strand the kitchen forever.
func TestUpdateOrderStatus_ResolvedRequestDoesNotFreeze(t *testing.T) {
	for _, status := range []models.CancellationRequestStatus{
		models.CancelReqApproved,
		models.CancelReqAutoRefunded,
		models.CancelReqResolved,
		models.CancelReqNotAllowed,
	} {
		t.Run(string(status), func(t *testing.T) {
			db, userID, chefID := setupCancellationDB(t)
			orderID := seedAcceptedOrder(t, db, chefID, "preparing")
			seedCancellationRequest(t, db, orderID, chefID, string(status), "settled")

			w := postStatus(t, userID, orderID, "ready")
			require.Equal(t, http.StatusOK, w.Code, w.Body.String())
			require.Equal(t, "ready", orderStatus(t, db, orderID))
		})
	}
}

// TestUpdateOrderStatus_NoRequestUnaffected — the guard must not cost anything
// to the overwhelming majority of orders, which have no cancellation row at all.
func TestUpdateOrderStatus_NoRequestUnaffected(t *testing.T) {
	db, userID, chefID := setupCancellationDB(t)
	orderID := seedAcceptedOrder(t, db, chefID, "preparing")

	w := postStatus(t, userID, orderID, "ready")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Equal(t, "ready", orderStatus(t, db, orderID))
}

// TestUpdateOrderStatus_FreezeIsPerOrder — a request against one order must not
// freeze the chef's other orders. "for that specific order" is the whole point:
// a kitchen-wide freeze during a rush would be a worse bug than the one fixed.
func TestUpdateOrderStatus_FreezeIsPerOrder(t *testing.T) {
	db, userID, chefID := setupCancellationDB(t)
	frozen := seedAcceptedOrder(t, db, chefID, "preparing")
	untouched := seedAcceptedOrder(t, db, chefID, "preparing")
	seedCancellationRequest(t, db, frozen, chefID, string(models.CancelReqPendingVendor), "mistake")

	require.Equal(t, http.StatusConflict, postStatus(t, userID, frozen, "ready").Code)

	w := postStatus(t, userID, untouched, "ready")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Equal(t, "ready", orderStatus(t, db, untouched))
	require.Equal(t, "preparing", orderStatus(t, db, frozen))
}

// TestUpdateOrderStatus_RestampingSameStatusStillNoOp — the guard is scoped to
// TRANSITIONS. A client re-sending the status it is already on (a retry, a
// double-tap) must stay the idempotent no-op it is everywhere else, not start
// erroring the moment a cancellation is open.
func TestUpdateOrderStatus_RestampingSameStatusStillNoOp(t *testing.T) {
	db, userID, chefID := setupCancellationDB(t)
	orderID := seedAcceptedOrder(t, db, chefID, "preparing")
	seedCancellationRequest(t, db, orderID, chefID, string(models.CancelReqPendingVendor), "mistake")

	w := postStatus(t, userID, orderID, "preparing")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Equal(t, "preparing", orderStatus(t, db, orderID))
}

// TestPendingCancellationFor_ChefFacingSubset pins what the vendor app is handed:
// enough to name the request, and none of the refund snapshot — those paise are
// all zero until the chef picks a tier, and showing a chef "₹0 refund" before
// they have decided would be a lie the screen cannot walk back.
func TestPendingCancellationFor_ChefFacingSubset(t *testing.T) {
	db, _, chefID := setupCancellationDB(t)
	orderID := seedAcceptedOrder(t, db, chefID, "preparing")
	reqID := seedCancellationRequest(t, db, orderID, chefID,
		string(models.CancelReqPendingVendor), "ordered the wrong dish")

	got := pendingCancellationFor(orderID)
	require.NotNil(t, got)
	require.Equal(t, reqID, got.ID)
	require.Equal(t, "ordered the wrong dish", got.Reason)
	require.False(t, got.RequestedAt.IsZero(), "the banner sorts and ages on this")
	require.NotNil(t, got.RespondBy, "the chef is on a clock and must see it")

	require.Nil(t, pendingCancellationFor(uuid.New()), "an unrelated order is never frozen")
}

// TestGetOrderDetail_SurfacesPendingCancellation — the app cannot grey a button
// for a reason the API never sends. Present while pending, absent once resolved.
func TestGetOrderDetail_SurfacesPendingCancellation(t *testing.T) {
	db, userID, chefID := setupCancellationDB(t)
	orderID := seedAcceptedOrder(t, db, chefID, "preparing")
	seedCancellationRequest(t, db, orderID, chefID,
		string(models.CancelReqPendingVendor), "running late, cancel please")

	body := getOrderDetail(t, userID, orderID)
	cr, ok := body["cancellationRequested"].(map[string]any)
	require.True(t, ok, "detail must carry the pending cancellation: %s", body)
	require.Equal(t, "running late, cancel please", cr["reason"])

	// Resolve it — the field must disappear, not linger and freeze the screen.
	require.NoError(t, db.Exec(`UPDATE cancellation_requests SET status = ? WHERE order_id = ?`,
		string(models.CancelReqApproved), orderID.String()).Error)

	body = getOrderDetail(t, userID, orderID)
	require.Nil(t, body["cancellationRequested"],
		"a settled request must not keep the order frozen")
}

func getOrderDetail(t *testing.T, userID, orderID uuid.UUID) map[string]any {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("userID", userID); c.Next() })
	r.GET("/chef/orders/:orderId", (&ChefHandler{}).GetOrderDetail)

	req := httptest.NewRequest(http.MethodGet, "/chef/orders/"+orderID.String(), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	return body
}
