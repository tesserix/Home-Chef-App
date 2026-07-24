package services

// confirm_receipt_test.go — AutoConfirmOrderReceipt (Task 1 of the
// auto-confirm-delivery plan). Uses the same hand-DDL'd in-memory sqlite
// harness as payout_hold_test.go (setupHoldDB) since the models'
// gen_random_uuid() defaults can't run on sqlite; rows are inserted with raw
// SQL rather than db.Create.

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/homechef/api/models"
)

// An order still awaiting confirmation gets auto-confirmed: acted=true,
// status advances to release_eligible, and customer_confirmed_at is stamped.
func TestAutoConfirmOrderReceipt_AwaitingOrder_Confirms(t *testing.T) {
	db := setupHoldDB(t)
	id := seedRegularOrder(t, db, models.PayoutHoldAwaitingConfirmation)

	status, acted, err := AutoConfirmOrderReceipt(db, id)
	require.NoError(t, err)
	require.True(t, acted, "expected acted=true for an awaiting order")
	require.Equal(t, models.PayoutHoldReleaseEligible, status)

	reloaded := loadOrder(t, db, id)
	require.NotNil(t, reloaded.CustomerConfirmedAt, "expected customer_confirmed_at stamped")
}

// An order the customer already confirmed is left untouched: acted=false,
// no error.
func TestAutoConfirmOrderReceipt_AlreadyConfirmed_NoOp(t *testing.T) {
	db := setupHoldDB(t)
	id := uuid.New()
	now := time.Now()
	require.NoError(t, db.Exec(
		`INSERT INTO orders (id, customer_id, status, razorpay_order_id, payout_hold_status, customer_confirmed_at) VALUES (?,?,?,?,?,?)`,
		id.String(), uuid.NewString(), "delivered", "order_rzp_123", string(models.PayoutHoldReleaseEligible), now,
	).Error)

	_, acted, err := AutoConfirmOrderReceipt(db, id)
	require.NoError(t, err)
	require.False(t, acted, "expected acted=false for an already-confirmed order")
}
