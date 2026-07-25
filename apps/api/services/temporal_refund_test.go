package services

// temporal_refund_test.go — the durable deferred chef-cancel gateway-refund
// retry flow (temporal_refund.go). Reuses the sqlite harness + gateway stub
// from deferred_cancel_refund_test.go (same package) since both exercise the
// identical "pending:gateway-retry:<paise>" sentinel contract.

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/homechef/api/config"
)

// TestStartDeferredRefundFlow_NoOpWhenTemporalDown verifies the producer is a
// safe no-op in the default unit-test state: temporalRT is nil (never set via
// SetTemporalRuntime), so deferredRefundFlowActive() is false and none of
// these calls should touch Temporal or panic — the cron backstop is what
// picks up the deferred refund instead.
func TestStartDeferredRefundFlow_NoOpWhenTemporalDown(t *testing.T) {
	StartDeferredRefundFlow(uuid.New(), "pay_test", 50000)
}

// TestStartDeferredRefundFlow_NoOpOnZeroOrEmptyPayment guards the two other
// no-op conditions even if Temporal were somehow up: nothing to refund, or no
// payment id to refund against. Since temporalRT is nil in this test binary
// regardless, this mostly documents the guard order, but still exercises the
// function without panicking on the zero/empty inputs.
func TestStartDeferredRefundFlow_NoOpOnZeroOrEmptyPayment(t *testing.T) {
	StartDeferredRefundFlow(uuid.New(), "pay_test", 0)
	StartDeferredRefundFlow(uuid.New(), "", 50000)
}

// TestGatewayRefundForWorkflow_CallsGatewayWithStableKeyAndAmount pins the
// activity implementation the worker wires onto workflows.GatewayRefundFunc:
// it must call the injected Razorpay stub with the SAME stable idempotency
// key (RefundFullIdempotencyKey) the cron uses, and the exact paise amount
// asked for — the double-refund guarantee depends on this key never drifting.
func TestGatewayRefundForWorkflow_CallsGatewayWithStableKeyAndAmount(t *testing.T) {
	setupDeferredCancelRefundDB(t)
	gotAmount, gotKey, calls := withDeferredRefundGateway(t)
	orderID := uuid.New()

	refundID, err := GatewayRefundForWorkflow(context.Background(), orderID, "pay_abc123", 305400)

	require.NoError(t, err)
	require.Equal(t, "rfnd_healed", refundID)
	require.Equal(t, 1, *calls)
	require.Equal(t, 305400, *gotAmount)
	require.Equal(t, normalizeIdempotencyKey(RefundFullIdempotencyKey(orderID)), *gotKey,
		"must reuse the SAME stable idempotency key the cron uses — dedups a lost-response success instead of double-refunding")
}

// TestGatewayRefundForWorkflow_NilGateway_ReturnsError verifies a nil
// Razorpay client surfaces as an error (not a silent empty id) so Temporal's
// activity retry policy actually retries instead of treating "no client" as
// "nothing to refund".
func TestGatewayRefundForWorkflow_NilGateway_ReturnsError(t *testing.T) {
	// GetRazorpay() with no cached client falls through to a live Secret Manager
	// fetch, which needs a non-nil config.AppConfig for its dev-fallback check —
	// set an empty one so the fetch fails cleanly (no real credentials) instead
	// of panicking on a nil config in this test binary. Same gotcha documented in
	// handlers/chef_order_cancel_deferred_test.go's TestCancelOrder_GatewayNil_CancelsAndDefers.
	prevCfg := config.AppConfig
	config.AppConfig = &config.Config{Environment: "test"}
	t.Cleanup(func() { config.AppConfig = prevCfg })
	SetRazorpayClient(nil)
	t.Cleanup(func() { SetRazorpayClient(nil) })

	_, err := GatewayRefundForWorkflow(context.Background(), uuid.New(), "pay_x", 1000)
	require.Error(t, err)
}

// TestPersistDeferredRefundID_ReplacesSentinel is the happy path: a deferred
// sentinel is replaced with the real gateway refund id.
func TestPersistDeferredRefundID_ReplacesSentinel(t *testing.T) {
	db := setupDeferredCancelRefundDB(t)
	addOutboxEventsTable(t, db)
	orderID := seedDeferredCancelOrder(t, db, "pending:gateway-retry:50000", time.Now().Add(-time.Hour))

	require.NoError(t, PersistDeferredRefundID(context.Background(), orderID, "rfnd_from_workflow"))

	require.Equal(t, "rfnd_from_workflow", deferredRefundIDOf(t, db, orderID))
}

// TestPersistDeferredRefundID_NoOpWhenNotASentinel — a concurrent actor (the
// cron, or a duplicate workflow run) already healed the row: refund_id no
// longer matches the sentinel prefix, so this call must NOT clobber it with a
// stale/different id.
func TestPersistDeferredRefundID_NoOpWhenNotASentinel(t *testing.T) {
	db := setupDeferredCancelRefundDB(t)
	addOutboxEventsTable(t, db)
	orderID := seedDeferredCancelOrder(t, db, "rfnd_already_real", time.Now().Add(-time.Hour))

	require.NoError(t, PersistDeferredRefundID(context.Background(), orderID, "rfnd_from_workflow"))

	require.Equal(t, "rfnd_already_real", deferredRefundIDOf(t, db, orderID),
		"a non-sentinel refund_id must never be overwritten by a racing workflow run")
}

// TestPersistDeferredRefundID_StagesOutboxEvent — once the sentinel is
// healed, an orders.updated event is staged through the existing outbox so
// the customer's client refreshes.
func TestPersistDeferredRefundID_StagesOutboxEvent(t *testing.T) {
	db := setupDeferredCancelRefundDB(t)
	addOutboxEventsTable(t, db)
	orderID := seedDeferredCancelOrder(t, db, "pending:gateway-retry:50000", time.Now().Add(-time.Hour))

	require.NoError(t, PersistDeferredRefundID(context.Background(), orderID, "rfnd_from_workflow"))

	var subjects []string
	require.NoError(t, db.Raw(`SELECT subject FROM outbox_events WHERE aggregate_id = ?`, orderID.String()).
		Pluck("subject", &subjects).Error)
	require.Equal(t, []string{SubjectOrderUpdated}, subjects)
}

// addOutboxEventsTable adds the outbox table to the deferred-cancel-refund
// harness DB so PersistDeferredRefundID's notify step can be observed.
func addOutboxEventsTable(t *testing.T, db *gorm.DB) {
	t.Helper()
	require.NoError(t, db.Exec(`CREATE TABLE outbox_events (
		id TEXT PRIMARY KEY, subject TEXT, aggregate_type TEXT, aggregate_id TEXT,
		msg_id TEXT, payload TEXT, status TEXT, attempts INTEGER DEFAULT 0,
		last_error TEXT, next_retry_at DATETIME, published_at DATETIME,
		created_at DATETIME, updated_at DATETIME
	)`).Error)
}
