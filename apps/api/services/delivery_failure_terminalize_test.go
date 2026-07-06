package services

// delivery_failure_terminalize_test.go — #393 chef self-delivery parity. The shared
// terminalize seam (courier + chef self-delivery both call it): freeze the money
// (RecordDeliveryFailure) and, on the FIRST terminalization, stage one delivery.failed
// notification. No money moves — the hold is only frozen to disputed.

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/homechef/api/models"
)

func TestTerminalizeDeliveryFailure_FreezesAndEmits(t *testing.T) {
	db := setupCrossguardDB(t)
	orderID, _ := seedCrossOrder(t, db, models.PayoutHoldNone, "delivering", nil)
	order := loadFailureOrder(t, db, orderID)

	froze, err := TerminalizeDeliveryFailure(db, order, models.FailureCustomerUnavailable, "chef_self_delivery", map[string]any{"self_delivery": true})
	require.NoError(t, err)
	require.True(t, froze)
	require.Equal(t, models.PayoutHoldDisputed, loadOrderHold(t, db, orderID))
	require.Equal(t, 1, countOrderIssues(t, db, orderID, models.IssueDeliveryFailed, models.IssuePending))
	require.Equal(t, 1, countOutbox(t, db, SubjectDeliveryFailed), "emits delivery.failed exactly once")
	// The reporter is persisted on the issue so the admin resolver can scrutinize a
	// chef's self-report (a self-interested party) more carefully than a courier report.
	var desc string
	db.Raw(`SELECT description FROM order_issues WHERE order_id = ?`, orderID.String()).Scan(&desc)
	require.Contains(t, desc, "reported_by=chef_self_delivery")
}

func TestTerminalizeDeliveryFailure_IdempotentNoDoubleEvent(t *testing.T) {
	db := setupCrossguardDB(t)
	orderID, _ := seedCrossOrder(t, db, models.PayoutHoldNone, "delivering", nil)
	order := loadFailureOrder(t, db, orderID)

	for i := 0; i < 2; i++ {
		_, err := TerminalizeDeliveryFailure(db, order, models.FailureCustomerUnavailable, "courier", nil)
		require.NoError(t, err)
	}
	require.Equal(t, 1, countOutbox(t, db, SubjectDeliveryFailed), "no duplicate notification on re-fire")
	require.Equal(t, 1, countOrderIssues(t, db, orderID, models.IssueDeliveryFailed, models.IssuePending))
}

func TestTerminalizeDeliveryFailure_NonGatewayOrderSkips(t *testing.T) {
	db := setupCrossguardDB(t)
	orderID, _ := seedCrossOrder(t, db, models.PayoutHoldNone, "delivering", nil)
	require.NoError(t, db.Exec(`UPDATE orders SET razorpay_order_id = '' WHERE id = ?`, orderID.String()).Error)
	order := loadFailureOrder(t, db, orderID)

	froze, err := TerminalizeDeliveryFailure(db, order, models.FailureCustomerUnavailable, "courier", nil)
	require.NoError(t, err)
	require.False(t, froze)
	require.Equal(t, 0, countOutbox(t, db, SubjectDeliveryFailed))
	require.Equal(t, models.PayoutHoldNone, loadOrderHold(t, db, orderID))
}
