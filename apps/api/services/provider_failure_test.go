package services

// provider_failure_test.go — #393. A 3PL delivery that terminally fails or is returned
// (RTO) must FREEZE the order's money for admin fault resolution, exactly like an
// own-fleet courier failure — not silently stamp a status and strand the order. Both 3PL
// webhook paths (the generic external_delivery_id flow and the Shadowfax order-number
// flow) route failed/returned through TerminalizeDeliveryFailure. The provider status is
// unstructured relative to our fault taxonomy, so the freeze uses FailureOther →
// FaultAmbiguous: the admin confirms the concrete fault before any money moves.

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

func setupProviderFailureDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	require.NoError(t, err)
	for _, s := range []string{
		`CREATE TABLE delivery_providers (id TEXT PRIMARY KEY, name TEXT DEFAULT '', code TEXT,
			webhook_secret TEXT DEFAULT '', status_mapping TEXT DEFAULT '{}', is_enabled INTEGER DEFAULT 1,
			is_active INTEGER DEFAULT 1, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME)`,
		`CREATE TABLE deliveries (id TEXT PRIMARY KEY, order_id TEXT, provider_id TEXT,
			external_delivery_id TEXT DEFAULT '', status TEXT DEFAULT 'pending', assignment_type TEXT DEFAULT 'manual',
			picked_up_at DATETIME, delivered_at DATETIME, cancelled_at DATETIME, cancel_reason TEXT DEFAULT '',
			failure_reason TEXT DEFAULT '', external_tracking_url TEXT DEFAULT '', provider_status TEXT DEFAULT '',
			rider_name TEXT DEFAULT '', rider_phone TEXT DEFAULT '', rider_latitude REAL DEFAULT 0,
			rider_longitude REAL DEFAULT 0, assigned_at DATETIME, created_at DATETIME, updated_at DATETIME)`,
		`CREATE TABLE orders (id TEXT PRIMARY KEY, order_number TEXT DEFAULT '', customer_id TEXT,
			chef_id TEXT, status TEXT, razorpay_order_id TEXT DEFAULT '', total REAL DEFAULT 0,
			payout_hold_status TEXT DEFAULT '', delivery_id TEXT, refund_amount REAL DEFAULT 0,
			created_at DATETIME, updated_at DATETIME, deleted_at DATETIME)`,
		`CREATE TABLE order_issues (id TEXT PRIMARY KEY, order_id TEXT, chef_id TEXT, customer_id TEXT,
			reason TEXT, description TEXT, photo_urls TEXT, affected_item_ids TEXT, requested_amount REAL DEFAULT 0,
			refund_amount REAL DEFAULT 0, status TEXT DEFAULT 'pending', resolved_by TEXT, resolved_at DATETIME,
			refund_txn_id TEXT, created_at DATETIME, updated_at DATETIME)`,
		`CREATE TABLE meal_plan_days (id TEXT PRIMARY KEY, meal_plan_id TEXT, order_id TEXT, status TEXT,
			payout_transfer_id TEXT DEFAULT '', price REAL DEFAULT 0, payout_hold_status TEXT DEFAULT '',
			delivered_at DATETIME, refund_txn_id TEXT, created_at DATETIME, updated_at DATETIME)`,
		`CREATE TABLE group_orders (id TEXT PRIMARY KEY, order_id TEXT, status TEXT, payout_hold_status TEXT DEFAULT '',
			created_at DATETIME, updated_at DATETIME)`,
		`CREATE TABLE outbox_events (id TEXT PRIMARY KEY, subject TEXT, msg_id TEXT, aggregate_type TEXT,
			aggregate_id TEXT, payload TEXT, status TEXT, attempts INT, last_error TEXT,
			next_retry_at DATETIME, created_at DATETIME, updated_at DATETIME, published_at DATETIME)`,
	} {
		require.NoError(t, db.Exec(s).Error)
	}
	prev := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = prev })
	return db
}

func seedProvider(t *testing.T, db *gorm.DB, code, statusMapping string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO delivery_providers (id, code, status_mapping) VALUES (?,?,?)`,
		id.String(), code, statusMapping).Error)
	return id
}

func seed3PLOrderAndDelivery(t *testing.T, db *gorm.DB, providerID uuid.UUID, externalID, orderNumber string) (uuid.UUID, uuid.UUID) {
	t.Helper()
	orderID, delID := uuid.New(), uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO orders (id, order_number, status, razorpay_order_id, chef_id, customer_id)
		VALUES (?,?,?,?,?,?)`, orderID.String(), orderNumber, string(models.OrderStatusDelivering),
		"order_rzp_"+orderID.String()[:8], uuid.NewString(), uuid.NewString()).Error)
	require.NoError(t, db.Exec(`INSERT INTO deliveries (id, order_id, provider_id, external_delivery_id, status, assignment_type)
		VALUES (?,?,?,?,?,?)`, delID.String(), orderID.String(), providerID.String(), externalID,
		string(models.DeliveryInTransit), string(models.AssignmentThirdParty)).Error)
	return orderID, delID
}

func loadHold(t *testing.T, db *gorm.DB, orderID uuid.UUID) models.PayoutHoldStatus {
	t.Helper()
	var s string
	require.NoError(t, db.Raw(`SELECT payout_hold_status FROM orders WHERE id = ?`, orderID.String()).Scan(&s).Error)
	return models.PayoutHoldStatus(s)
}

// ── generic 3PL webhook (external_delivery_id + StatusMapping) ────────────────

func TestHandleProviderWebhook_FailedFreezesMoney(t *testing.T) {
	db := setupProviderFailureDB(t)
	pid := seedProvider(t, db, "borzo", `{"failed":"failed"}`)
	orderID, _ := seed3PLOrderAndDelivery(t, db, pid, "BRZ-123", "HC-1")

	err := NewProviderService().HandleProviderWebhook("borzo", []byte(`{"delivery_id":"BRZ-123","status":"failed","reason":"customer not reachable"}`))
	require.NoError(t, err)

	require.Equal(t, models.PayoutHoldDisputed, loadHold(t, db, orderID), "money frozen for admin resolution")
	require.Equal(t, 1, countOrderIssues(t, db, orderID, models.IssueDeliveryFailed, models.IssuePending))
	require.Equal(t, 1, countOutbox(t, db, SubjectDeliveryFailed))
}

func TestHandleProviderWebhook_ReturnedRTOFreezes(t *testing.T) {
	db := setupProviderFailureDB(t)
	pid := seedProvider(t, db, "borzo", `{"rto":"returned"}`)
	orderID, _ := seed3PLOrderAndDelivery(t, db, pid, "BRZ-9", "HC-9")

	err := NewProviderService().HandleProviderWebhook("borzo", []byte(`{"delivery_id":"BRZ-9","status":"rto"}`))
	require.NoError(t, err)
	require.Equal(t, models.PayoutHoldDisputed, loadHold(t, db, orderID), "RTO freezes like a failure")
	require.Equal(t, 1, countOrderIssues(t, db, orderID, models.IssueDeliveryFailed, models.IssuePending))
}

func TestHandleProviderWebhook_FailedIdempotent(t *testing.T) {
	db := setupProviderFailureDB(t)
	pid := seedProvider(t, db, "borzo", `{"failed":"failed"}`)
	orderID, _ := seed3PLOrderAndDelivery(t, db, pid, "BRZ-2", "HC-2")

	for i := 0; i < 2; i++ {
		require.NoError(t, NewProviderService().HandleProviderWebhook("borzo", []byte(`{"delivery_id":"BRZ-2","status":"failed"}`)))
	}
	require.Equal(t, 1, countOrderIssues(t, db, orderID, models.IssueDeliveryFailed, models.IssuePending), "no duplicate dispute on re-fire")
	require.Equal(t, 1, countOutbox(t, db, SubjectDeliveryFailed))
}

func TestHandleProviderWebhook_DeliveredStillParks(t *testing.T) {
	// Regression: the delivered path must keep parking the hold at awaiting (not disputed).
	db := setupProviderFailureDB(t)
	pid := seedProvider(t, db, "borzo", `{"delivered":"delivered"}`)
	orderID, _ := seed3PLOrderAndDelivery(t, db, pid, "BRZ-3", "HC-3")

	require.NoError(t, NewProviderService().HandleProviderWebhook("borzo", []byte(`{"delivery_id":"BRZ-3","status":"delivered"}`)))
	require.Equal(t, models.PayoutHoldAwaitingConfirmation, loadHold(t, db, orderID))
	require.Equal(t, 0, countOrderIssues(t, db, orderID, models.IssueDeliveryFailed, models.IssuePending))
}

// #594: a 3PL `cancelled` AFTER pickup (food already collected by the rider) is
// effectively a failed delivery and must freeze the money for admin resolution; a
// PRE-pickup cancel is a normal cancellation and must NOT freeze.
func TestHandleProviderWebhook_CancelledAfterPickupFreezes(t *testing.T) {
	db := setupProviderFailureDB(t)
	pid := seedProvider(t, db, "borzo", `{"cancelled":"cancelled"}`)
	orderID, delID := seed3PLOrderAndDelivery(t, db, pid, "BRZ-C1", "HC-C1")
	require.NoError(t, db.Exec(`UPDATE deliveries SET picked_up_at = ? WHERE id = ?`, time.Now(), delID.String()).Error)

	err := NewProviderService().HandleProviderWebhook("borzo",
		[]byte(`{"delivery_id":"BRZ-C1","status":"cancelled","reason":"rider cancelled after pickup"}`))
	require.NoError(t, err)
	require.Equal(t, models.PayoutHoldDisputed, loadHold(t, db, orderID), "post-pickup cancel freezes like a failure")
	require.Equal(t, 1, countOrderIssues(t, db, orderID, models.IssueDeliveryFailed, models.IssuePending))
	require.Equal(t, 1, countOutbox(t, db, SubjectDeliveryFailed))
}

func TestHandleProviderWebhook_CancelledBeforePickupNoFreeze(t *testing.T) {
	db := setupProviderFailureDB(t)
	pid := seedProvider(t, db, "borzo", `{"cancelled":"cancelled"}`)
	orderID, _ := seed3PLOrderAndDelivery(t, db, pid, "BRZ-C2", "HC-C2") // picked_up_at NULL

	err := NewProviderService().HandleProviderWebhook("borzo",
		[]byte(`{"delivery_id":"BRZ-C2","status":"cancelled","reason":"cancelled before pickup"}`))
	require.NoError(t, err)
	require.NotEqual(t, models.PayoutHoldDisputed, loadHold(t, db, orderID), "pre-pickup cancel is a normal cancellation, not a freeze")
	require.Equal(t, 0, countOrderIssues(t, db, orderID, models.IssueDeliveryFailed, models.IssuePending))
	require.Equal(t, 0, countOutbox(t, db, SubjectDeliveryFailed))
}

func TestHandleProviderWebhook_CancelledAfterPickupIdempotent(t *testing.T) {
	db := setupProviderFailureDB(t)
	pid := seedProvider(t, db, "borzo", `{"cancelled":"cancelled"}`)
	orderID, delID := seed3PLOrderAndDelivery(t, db, pid, "BRZ-C3", "HC-C3")
	require.NoError(t, db.Exec(`UPDATE deliveries SET picked_up_at = ? WHERE id = ?`, time.Now(), delID.String()).Error)

	for i := 0; i < 2; i++ {
		require.NoError(t, NewProviderService().HandleProviderWebhook("borzo",
			[]byte(`{"delivery_id":"BRZ-C3","status":"cancelled"}`)))
	}
	require.Equal(t, 1, countOrderIssues(t, db, orderID, models.IssueDeliveryFailed, models.IssuePending), "no duplicate dispute on re-fire")
}

func TestHandleProviderWebhook_CancelledAfterDeliveredDoesNotFreeze(t *testing.T) {
	// A late/replayed `cancelled` webhook AFTER the delivery already delivered (hold parked
	// at awaiting_confirmation) must NOT dispute a correctly-delivered order.
	db := setupProviderFailureDB(t)
	pid := seedProvider(t, db, "borzo", `{"cancelled":"cancelled"}`)
	orderID, delID := seed3PLOrderAndDelivery(t, db, pid, "BRZ-C4", "HC-C4")
	require.NoError(t, db.Exec(`UPDATE deliveries SET picked_up_at = ?, delivered_at = ?, status = 'delivered' WHERE id = ?`,
		time.Now(), time.Now(), delID.String()).Error)
	require.NoError(t, db.Exec(`UPDATE orders SET payout_hold_status = ? WHERE id = ?`,
		string(models.PayoutHoldAwaitingConfirmation), orderID.String()).Error)

	err := NewProviderService().HandleProviderWebhook("borzo",
		[]byte(`{"delivery_id":"BRZ-C4","status":"cancelled","reason":"stale webhook after delivery"}`))
	require.NoError(t, err)
	require.Equal(t, models.PayoutHoldAwaitingConfirmation, loadHold(t, db, orderID), "a delivered order's parked hold is not disputed by a late cancel")
	require.Equal(t, 0, countOrderIssues(t, db, orderID, models.IssueDeliveryFailed, models.IssuePending))
}

// ── Shadowfax webhook (order_number + status vocabulary) ──────────────────────

func TestHandleShadowfaxWebhook_RTOFreezes(t *testing.T) {
	db := setupProviderFailureDB(t)
	pid := seedProvider(t, db, "shadowfax", `{}`)
	orderID, _ := seed3PLOrderAndDelivery(t, db, pid, "SFX-1", "HCSF-1")

	err := NewProviderService().HandleProviderWebhook("shadowfax", []byte(`{"order_id":"HCSF-1","status":"rto"}`))
	require.NoError(t, err)
	require.Equal(t, models.PayoutHoldDisputed, loadHold(t, db, orderID))
	require.Equal(t, 1, countOrderIssues(t, db, orderID, models.IssueDeliveryFailed, models.IssuePending))
	require.Equal(t, 1, countOutbox(t, db, SubjectDeliveryFailed))
}

func TestHandleShadowfaxWebhook_LostFreezes(t *testing.T) {
	db := setupProviderFailureDB(t)
	pid := seedProvider(t, db, "shadowfax", `{}`)
	orderID, _ := seed3PLOrderAndDelivery(t, db, pid, "SFX-2", "HCSF-2")

	require.NoError(t, NewProviderService().HandleProviderWebhook("shadowfax", []byte(`{"order_id":"HCSF-2","status":"lost"}`)))
	require.Equal(t, models.PayoutHoldDisputed, loadHold(t, db, orderID))
	require.Equal(t, 1, countOrderIssues(t, db, orderID, models.IssueDeliveryFailed, models.IssuePending))
}

func TestHandleShadowfaxWebhook_DeliveredStillParks(t *testing.T) {
	db := setupProviderFailureDB(t)
	pid := seedProvider(t, db, "shadowfax", `{}`)
	orderID, _ := seed3PLOrderAndDelivery(t, db, pid, "SFX-3", "HCSF-3")

	require.NoError(t, NewProviderService().HandleProviderWebhook("shadowfax", []byte(`{"order_id":"HCSF-3","status":"delivered"}`)))
	require.Equal(t, models.PayoutHoldAwaitingConfirmation, loadHold(t, db, orderID))
}
