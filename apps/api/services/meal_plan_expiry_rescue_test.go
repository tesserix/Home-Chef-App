package services

// meal_plan_expiry_rescue_test.go — #395·3 GAP 1. The critical money hole: the expiry
// sweep could flip a CAPTURED-but-unconfirmed plan to `expired`, after which the refund
// no-ops (blank escrow_payment_id) and the advance-reconcile can never re-select it —
// money taken, chef unpaid, no refund, permanent. rescueCapturedBeforeExpiry closes it:
// ask the gateway before expiring; if paid, confirm instead; if the gateway can't be
// reached, defer (never expire an unverified plan).

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/homechef/api/models"
)

// A captured plan about to expire is CONFIRMED, not expired (rescue returns true).
func TestRescueCapturedBeforeExpiry_ConfirmsCaptured(t *testing.T) {
	escrowFlag(t, true)
	db := setupConfirmAdvanceDB(t)
	var creates int32
	reconcileStub(t, "order_rescue", 35200, true, &creates) // gateway shows captured
	planID := seedStuckAdvancePlan(t, db, "order_rescue", []float64{160, 160}, time.Now())

	var plan models.MealPlan
	require.NoError(t, db.Preload("Days").First(&plan, "id = ?", planID).Error)

	rescued := rescueCapturedBeforeExpiry(&plan)
	require.True(t, rescued, "a captured plan must be rescued (not expired)")
	require.Equal(t, string(models.MealPlanConfirmed), planField(t, db, planID, "status"), "confirmed instead of expired")
	require.Equal(t, "pay_rec", planField(t, db, planID, "escrow_payment_id"))
	require.Equal(t, int32(2), atomic.LoadInt32(&creates), "chef payouts held")
}

// A gateway-confirmed UNPAID plan is not rescued (returns false → sweep expires it).
func TestRescueCapturedBeforeExpiry_UnpaidNotRescued(t *testing.T) {
	escrowFlag(t, true)
	db := setupConfirmAdvanceDB(t)
	var creates int32
	reconcileStub(t, "order_unpaid", 35200, false, &creates) // no captured payment
	planID := seedStuckAdvancePlan(t, db, "order_unpaid", []float64{160, 160}, time.Now())

	var plan models.MealPlan
	require.NoError(t, db.Preload("Days").First(&plan, "id = ?", planID).Error)

	rescued := rescueCapturedBeforeExpiry(&plan)
	require.False(t, rescued, "an unpaid plan is not rescued — safe to expire")
	require.Equal(t, string(models.MealPlanAwaitingCustomer), planField(t, db, planID, "status"), "untouched")
	require.Equal(t, int32(0), atomic.LoadInt32(&creates))
}
