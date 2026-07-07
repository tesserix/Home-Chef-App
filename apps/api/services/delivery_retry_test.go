package services

// delivery_retry_test.go — #579 (owner's 2-attempt cap). A failed courier delivery gets
// ONE retry before terminal money resolution. The schema forbids a second Delivery row
// per order (order_id uniqueIndex, no soft-delete), so a retry MUST reuse the same row.
// These tests run against a real UNIQUE index on order_id (raw DDL — gorm.AutoMigrate
// emits invalid sqlite DDL for these uuid-defaulted models) so the constraint clash the
// naive "new row per retry" would hit is genuinely exercised end-to-end.

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

func setupDeliveryRetryDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	require.NoError(t, err)
	for _, s := range []string{
		`CREATE TABLE orders (id TEXT PRIMARY KEY, order_number TEXT DEFAULT '', customer_id TEXT,
			chef_id TEXT, status TEXT, razorpay_order_id TEXT DEFAULT '', total REAL DEFAULT 0,
			payout_hold_status TEXT DEFAULT '', delivery_id TEXT, refund_amount REAL DEFAULT 0,
			created_at DATETIME, updated_at DATETIME, deleted_at DATETIME)`,
		// order_id UNIQUE — the real constraint. No soft-delete, matching production.
		`CREATE TABLE deliveries (id TEXT PRIMARY KEY, order_id TEXT UNIQUE NOT NULL,
			delivery_partner_id TEXT, status TEXT DEFAULT 'pending', assignment_type TEXT DEFAULT 'manual',
			assigned_by_id TEXT, pickup_address_line1 TEXT DEFAULT '', pickup_address_city TEXT DEFAULT '',
			pickup_latitude REAL DEFAULT 0, pickup_longitude REAL DEFAULT 0,
			dropoff_address_line1 TEXT DEFAULT '', dropoff_address_city TEXT DEFAULT '',
			dropoff_latitude REAL DEFAULT 0, dropoff_longitude REAL DEFAULT 0,
			distance REAL DEFAULT 0, estimated_duration INTEGER DEFAULT 0, actual_duration INTEGER DEFAULT 0,
			delivery_fee REAL DEFAULT 0, tip REAL DEFAULT 0, total_payout REAL DEFAULT 0,
			attempt_number INTEGER DEFAULT 1, max_attempts INTEGER DEFAULT 3, failure_reason TEXT DEFAULT '',
			offer_expires_at DATETIME, provider_id TEXT, external_delivery_id TEXT DEFAULT '',
			external_tracking_id TEXT DEFAULT '', external_tracking_url TEXT DEFAULT '', provider_cost REAL DEFAULT 0,
			rider_name TEXT DEFAULT '', rider_phone TEXT DEFAULT '', rider_latitude REAL DEFAULT 0,
			rider_longitude REAL DEFAULT 0, provider_status TEXT DEFAULT '',
			cancel_reason TEXT DEFAULT '', cancelled_at DATETIME, picked_up_at DATETIME, delivered_at DATETIME,
			assigned_at DATETIME, created_at DATETIME, updated_at DATETIME)`,
		`CREATE TABLE order_issues (id TEXT PRIMARY KEY, order_id TEXT, chef_id TEXT, customer_id TEXT,
			reason TEXT, description TEXT, photo_urls TEXT, affected_item_ids TEXT,
			requested_amount REAL DEFAULT 0, refund_amount REAL DEFAULT 0, status TEXT DEFAULT 'pending',
			resolved_by TEXT, resolved_at DATETIME, refund_txn_id TEXT, created_at DATETIME, updated_at DATETIME)`,
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

func seedRetryOrder(t *testing.T, db *gorm.DB, status models.OrderStatus, gateway bool) uuid.UUID {
	t.Helper()
	id := uuid.New()
	rzp := ""
	if gateway {
		rzp = "order_rzp_" + id.String()[:8]
	}
	require.NoError(t, db.Exec(`INSERT INTO orders (id, status, razorpay_order_id, chef_id, customer_id, delivery_id)
		VALUES (?,?,?,?,?,?)`, id.String(), string(status), rzp, uuid.NewString(), uuid.NewString(), id.String()).Error)
	return id
}

func seedDelivery(t *testing.T, db *gorm.DB, orderID uuid.UUID, status models.DeliveryStatus, attempt int, partner *uuid.UUID) uuid.UUID {
	t.Helper()
	id := uuid.New()
	var p any
	if partner != nil {
		p = partner.String()
	}
	require.NoError(t, db.Exec(`INSERT INTO deliveries (id, order_id, delivery_partner_id, status, attempt_number, max_attempts, failure_reason)
		VALUES (?,?,?,?,?,?,?)`, id.String(), orderID.String(), p, string(status), attempt, MaxDeliveryAttempts, "").Error)
	return id
}

func loadDelivery(t *testing.T, db *gorm.DB, id uuid.UUID) models.Delivery {
	t.Helper()
	var d models.Delivery
	require.NoError(t, db.Raw(`SELECT id, order_id, delivery_partner_id, status, attempt_number, max_attempts, failure_reason
		FROM deliveries WHERE id = ?`, id.String()).Scan(&d).Error)
	return d
}

func loadRetryOrder(t *testing.T, db *gorm.DB, id uuid.UUID) models.Order {
	t.Helper()
	var o models.Order
	require.NoError(t, db.Raw(`SELECT id, status, delivery_id, payout_hold_status FROM orders WHERE id = ?`, id.String()).Scan(&o).Error)
	return o
}

func countDeliveries(t *testing.T, db *gorm.DB, orderID uuid.UUID) int {
	t.Helper()
	var n int
	require.NoError(t, db.Raw(`SELECT COUNT(*) FROM deliveries WHERE order_id = ?`, orderID.String()).Scan(&n).Error)
	return n
}

// ── the UNIQUE constraint is real (guards against a vacuous test) ─────────────

func TestDeliveryOrderIDUniqueEnforced(t *testing.T) {
	db := setupDeliveryRetryDB(t)
	orderID := seedRetryOrder(t, db, models.OrderStatusPickedUp, true)
	seedDelivery(t, db, orderID, models.DeliveryFailed, 1, nil)
	// A naive second row for the same order — what "new row per retry" would do — must
	// violate the unique index. This is why the retry has to REUSE the row.
	err := db.Exec(`INSERT INTO deliveries (id, order_id, status, attempt_number) VALUES (?,?,?,?)`,
		uuid.NewString(), orderID.String(), "pending", 2).Error
	require.Error(t, err, "order_id unique index must forbid a second delivery row")
}

// ── retry vs terminalize decision ────────────────────────────────────────────

func TestRetryOrTerminalize_FirstFailureRetries(t *testing.T) {
	db := setupDeliveryRetryDB(t)
	orderID := seedRetryOrder(t, db, models.OrderStatusPickedUp, true)
	partner := uuid.New()
	delID := seedDelivery(t, db, orderID, models.DeliveryAtDropoff, 1, &partner)
	del := loadDelivery(t, db, delID)
	del.Status = models.DeliveryFailed // handler mutates the struct BEFORE the call (row still at_dropoff)

	retried, err := RetryOrTerminalizeFailedDelivery(db, &del, models.FailureCustomerUnavailable, "courier")
	require.NoError(t, err)
	require.True(t, retried, "attempt 1 of 2 → re-dispatch")

	got := loadDelivery(t, db, delID)
	require.Equal(t, models.DeliveryPending, got.Status)
	require.Nil(t, got.DeliveryPartnerID, "partner cleared for re-offer")
	require.Equal(t, 2, got.AttemptNumber, "attempt bumped")
	require.Equal(t, "", got.FailureReason, "prior failure residue cleared")

	ord := loadRetryOrder(t, db, orderID)
	require.Equal(t, models.OrderStatusReady, ord.Status, "order re-opened for re-dispatch")
	require.Nil(t, ord.DeliveryID)
	require.Equal(t, models.PayoutHoldStatus(""), ord.PayoutHoldStatus, "no money frozen on a retry")
	require.Equal(t, 0, countOrderIssues(t, db, orderID, models.IssueDeliveryFailed, models.IssuePending), "no dispute on a retry")
	require.Equal(t, 1, countDeliveries(t, db, orderID), "still exactly one delivery row")
}

// #631: an order refunded/cancelled while its delivery was still in flight (the refund paths don't
// cancel the delivery) must NOT be reset to `ready` on a delivery failure — that un-terminalizes
// it, letting AcceptDelivery re-pick it and a fresh delivery cycle resurrect it to `delivered`
// (which folds into the weekly statement). The delivery may reset, but the order stays terminal.
func TestRetryOrTerminalize_RefundedOrderNotResurrected(t *testing.T) {
	db := setupDeliveryRetryDB(t)
	orderID := seedRetryOrder(t, db, models.OrderStatusRefunded, true)
	partner := uuid.New()
	delID := seedDelivery(t, db, orderID, models.DeliveryAtDropoff, 1, &partner)
	del := loadDelivery(t, db, delID)
	del.Status = models.DeliveryFailed

	_, err := RetryOrTerminalizeFailedDelivery(db, &del, models.FailureCustomerUnavailable, "courier")
	require.NoError(t, err)

	require.Equal(t, models.OrderStatusRefunded, loadRetryOrder(t, db, orderID).Status,
		"a refunded order is not reset to ready — no resurrection into a fresh delivery cycle")
}

func TestRetryOrTerminalize_CapReachedTerminalizes(t *testing.T) {
	db := setupDeliveryRetryDB(t)
	orderID := seedRetryOrder(t, db, models.OrderStatusPickedUp, true)
	partner := uuid.New()
	delID := seedDelivery(t, db, orderID, models.DeliveryAtDropoff, MaxDeliveryAttempts, &partner) // 2nd attempt
	del := loadDelivery(t, db, delID)

	retried, err := RetryOrTerminalizeFailedDelivery(db, &del, models.FailureCustomerUnavailable, "courier")
	require.NoError(t, err)
	require.False(t, retried, "cap reached → terminal money freeze")

	// Money frozen for admin resolution — an issue is opened and the hold disputed.
	require.Equal(t, 1, countOrderIssues(t, db, orderID, models.IssueDeliveryFailed, models.IssuePending))
	require.Equal(t, models.PayoutHoldDisputed, loadRetryOrder(t, db, orderID).PayoutHoldStatus)
	require.Equal(t, 1, countOutbox(t, db, SubjectDeliveryFailed))
	require.Equal(t, 1, countDeliveries(t, db, orderID))
}

func TestRetryOrTerminalize_ConcurrentDeliveredNotReset(t *testing.T) {
	// A failure report that races a delivered event must not resurrect the delivered row
	// back to pending — the guard excludes terminal states.
	db := setupDeliveryRetryDB(t)
	orderID := seedRetryOrder(t, db, models.OrderStatusDelivered, true)
	delID := seedDelivery(t, db, orderID, models.DeliveryDelivered, 1, nil)
	del := loadDelivery(t, db, delID)
	del.Status = models.DeliveryFailed // caller mutated in-memory (as the handler does)

	retried, err := RetryOrTerminalizeFailedDelivery(db, &del, models.FailureCustomerUnavailable, "courier")
	require.ErrorIs(t, err, ErrDeliveryStateChanged)
	require.False(t, retried)
	require.Equal(t, models.DeliveryDelivered, loadDelivery(t, db, delID).Status, "delivered row untouched")
	require.Equal(t, models.OrderStatusDelivered, loadRetryOrder(t, db, orderID).Status)
}

// ── assignment upsert (the constraint fix) ───────────────────────────────────

func buildAssignment(orderID uuid.UUID, partner uuid.UUID) *models.Delivery {
	return &models.Delivery{
		OrderID:           orderID,
		DeliveryPartnerID: &partner,
		Status:            models.DeliveryAssigned,
		AssignmentType:    models.AssignmentAuto,
		DeliveryFee:       40,
		Tip:               10,
		TotalPayout:       50,
	}
}

func TestAssignDeliveryForOrder_ReusesRetriedRow(t *testing.T) {
	db := setupDeliveryRetryDB(t)
	orderID := seedRetryOrder(t, db, models.OrderStatusReady, true)
	// A retry-reset row: pending, partner cleared, attempt already bumped to 2.
	retryRow := seedDelivery(t, db, orderID, models.DeliveryPending, 2, nil)

	partner := uuid.New()
	want := buildAssignment(orderID, partner)
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		return AssignDeliveryForOrder(tx, want)
	}))

	require.Equal(t, retryRow, want.ID, "reused the SAME row — no second insert")
	require.Equal(t, 1, countDeliveries(t, db, orderID), "still one row (unique constraint respected)")
	got := loadDelivery(t, db, retryRow)
	require.Equal(t, models.DeliveryAssigned, got.Status)
	require.NotNil(t, got.DeliveryPartnerID)
	require.Equal(t, partner, *got.DeliveryPartnerID)
	require.Equal(t, 2, got.AttemptNumber, "attempt counter preserved across re-dispatch")
}

func TestAssignDeliveryForOrder_FirstAssignmentCreates(t *testing.T) {
	db := setupDeliveryRetryDB(t)
	orderID := seedRetryOrder(t, db, models.OrderStatusReady, true)
	partner := uuid.New()
	want := buildAssignment(orderID, partner)

	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		return AssignDeliveryForOrder(tx, want)
	}))
	require.NotEqual(t, uuid.Nil, want.ID)
	require.Equal(t, 1, countDeliveries(t, db, orderID))
	got := loadDelivery(t, db, want.ID)
	require.Equal(t, 1, got.AttemptNumber, "first attempt")
	require.Equal(t, MaxDeliveryAttempts, got.MaxAttempts, "cap stamped on the row")
}

func TestAssignDeliveryForOrder_ReusesCancelledRow(t *testing.T) {
	// Latent bug this also fixes: a cancelled delivery leaves its row in place; a
	// re-dispatch to another driver must reuse it, not insert a second (which 500s today).
	db := setupDeliveryRetryDB(t)
	orderID := seedRetryOrder(t, db, models.OrderStatusReady, true)
	cancelled := seedDelivery(t, db, orderID, models.DeliveryCancelled, 1, nil)

	partner := uuid.New()
	want := buildAssignment(orderID, partner)
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		return AssignDeliveryForOrder(tx, want)
	}))
	require.Equal(t, cancelled, want.ID)
	require.Equal(t, 1, countDeliveries(t, db, orderID))
	require.Equal(t, models.DeliveryAssigned, loadDelivery(t, db, cancelled).Status)
}

func TestAssignDeliveryForOrder_RefusesActiveRow(t *testing.T) {
	// Two drivers racing to accept the same re-opened order: the reuse UPDATE is guarded
	// to re-dispatchable states, so it can't steal a row already assigned to a live
	// driver (the unique index doesn't protect an UPDATE the way it did tx.Create).
	db := setupDeliveryRetryDB(t)
	orderID := seedRetryOrder(t, db, models.OrderStatusReady, true)
	p1 := uuid.New()
	activeRow := seedDelivery(t, db, orderID, models.DeliveryAssigned, 1, &p1) // already owned

	p2 := uuid.New()
	want := buildAssignment(orderID, p2)
	err := db.Transaction(func(tx *gorm.DB) error { return AssignDeliveryForOrder(tx, want) })
	require.ErrorIs(t, err, ErrDeliveryStateChanged, "must not steal an active delivery")
	got := loadDelivery(t, db, activeRow)
	require.Equal(t, p1, *got.DeliveryPartnerID, "original driver keeps the delivery")
	require.Equal(t, models.DeliveryAssigned, got.Status)
}

// ── the full cycle the issue demands: failed → retry → re-accept → failed → terminal ─

func TestDeliveryRetry_FullCycleNeverClashes(t *testing.T) {
	db := setupDeliveryRetryDB(t)
	orderID := seedRetryOrder(t, db, models.OrderStatusPickedUp, true)
	p1 := uuid.New()
	delID := seedDelivery(t, db, orderID, models.DeliveryAtDropoff, 1, &p1)

	// attempt 1 fails → retry
	d := loadDelivery(t, db, delID)
	retried, err := RetryOrTerminalizeFailedDelivery(db, &d, models.FailureCustomerUnavailable, "courier")
	require.NoError(t, err)
	require.True(t, retried)

	// a second driver re-accepts → reuse the row (no unique clash)
	p2 := uuid.New()
	want := buildAssignment(orderID, p2)
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error { return AssignDeliveryForOrder(tx, want) }))
	require.Equal(t, delID, want.ID)
	require.Equal(t, 1, countDeliveries(t, db, orderID))

	// attempt 2 fails → terminal money freeze
	d2 := loadDelivery(t, db, delID)
	require.Equal(t, 2, d2.AttemptNumber)
	retried, err = RetryOrTerminalizeFailedDelivery(db, &d2, models.FailureCustomerUnavailable, "courier")
	require.NoError(t, err)
	require.False(t, retried, "cap reached on the 2nd attempt")
	require.Equal(t, models.PayoutHoldDisputed, loadRetryOrder(t, db, orderID).PayoutHoldStatus)
	require.Equal(t, 1, countDeliveries(t, db, orderID), "exactly one delivery row the whole cycle")
}
