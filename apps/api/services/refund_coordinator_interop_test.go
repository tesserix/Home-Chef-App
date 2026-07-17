package services

// refund_coordinator_interop_test.go — #690.
//
// The coordinator (#689) has to coexist with the paths it has not replaced yet.
// #690 migrates them ONE AT A TIME, which means that for the whole of that
// migration a coordinator refund and a legacy refund can be in flight for the
// SAME order at the same time. These tests pin whether the two mechanisms
// actually exclude each other.
//
// They live in `services` (not `orderrefund`) because they need BOTH sides, and
// orderrefund must not import services — services will import orderrefund as the
// call sites migrate, so the dependency only points one way.

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
	"github.com/homechef/api/services/orderrefund"
)

// interopGateway fakes the gateway and runs `during` at the exact moment the
// coordinator is mid-gateway-call — reservation committed, bookkeeping not yet.
// That instant is the whole question: it is the window a sibling path observes.
type interopGateway struct {
	calls  int
	during func()
}

func (g *interopGateway) RefundPayment(_ context.Context, _ string, _ int, _ string, _ map[string]string) (string, error) {
	g.calls++
	if g.during != nil {
		g.during()
	}
	return "rfnd_interop", nil
}

func seedCoordinatorOrder(t *testing.T, db *gorm.DB, total float64) uuid.UUID {
	t.Helper()
	id := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO orders (id, order_number, customer_id, chef_id, status, payment_status,
		payment_provider, razorpay_payment_id, total, refund_amount)
		VALUES (?,?,?,?,?,?,?,?,?,0)`,
		id.String(), "ORD-INTEROP", uuid.NewString(), uuid.NewString(), string(models.OrderStatusCancelled),
		string(models.PaymentCompleted), "razorpay", "pay_interop", total).Error)
	return id
}

// TestInterop_LegacyFullRefund_CannotRaceACoordinatorRefundInFlight is the gate
// on #690's incremental migration. If a legacy path can win its claim while a
// coordinator refund is mid-gateway, then migrating any single call site to the
// coordinator lets the migrated and un-migrated paths BOTH move real money for
// the same order — the exact double-refund #687 exists to prevent.
func TestInterop_LegacyFullRefund_CannotRaceACoordinatorRefundInFlight(t *testing.T) {
	db := setupCancelRefundDB(t)
	require.NoError(t, db.Exec(`CREATE TABLE refund_transactions (
		id TEXT PRIMARY KEY, order_id TEXT NOT NULL, provider TEXT NOT NULL,
		provider_payment_id TEXT, provider_refund_id TEXT,
		amount REAL NOT NULL, currency_code TEXT NOT NULL DEFAULT 'INR',
		status TEXT NOT NULL DEFAULT 'pending', reason TEXT,
		idempotency_key TEXT NOT NULL UNIQUE, scope_id TEXT NOT NULL,
		actor TEXT, failure_reason TEXT,
		created_at DATETIME, updated_at DATETIME, completed_at DATETIME)`).Error)

	orderID := seedCoordinatorOrder(t, db, 300)

	var (
		legacyAmount float64
		legacyWon    bool
	)
	gw := &interopGateway{during: func() {
		// A sibling full-refund path (chef reject, customer cancel, saga
		// compensation) fires while the coordinator's refund is at the gateway.
		legacyAmount, legacyWon, _ = ReserveFullRefund(db, orderID)
	}}

	coord := orderrefund.NewCoordinator(db, gw, true)
	res, err := coord.Refund(context.Background(), orderrefund.RefundCommand{
		OrderID: orderID, Reason: "cancellation", Actor: "customer", ScopeID: "cancel",
	})
	require.NoError(t, err)
	require.Equal(t, 300.0, res.Amount, "coordinator refunded the full order")
	require.Equal(t, 1, gw.calls)

	require.False(t, legacyWon,
		"a legacy full-refund path won its claim while the coordinator was mid-gateway — "+
			"both would move real money for order %s (legacy reserved ₹%.2f on top of the coordinator's ₹%.2f)",
		orderID, legacyAmount, res.Amount)
}
