package services

// cancellation_refund_ledger_test.go — #940. A cancellation refund must land on
// refund_transactions, or gateway reconciliation cannot see it and the ledger's
// unique-key double-refund guard does not cover it.

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services/orderrefund"
)

// seedCancelForLedger sets up an approved, unexecuted wallet cancellation.
func seedCancelForLedger(t *testing.T, total float64, paise int) (uuid.UUID, *models.CancellationRequest) {
	t.Helper()
	cust, chef := uuid.New(), uuid.New()
	oid, reqID := uuid.New(), uuid.New()
	require.NoError(t, database.DB.Exec(
		`INSERT INTO orders (id, customer_id, chef_id, status, payment_status, subtotal, total, refund_amount)
		 VALUES (?,?,?,?,?,?,?,0)`,
		oid.String(), cust.String(), chef.String(), "preparing", "completed", total, total).Error)
	require.NoError(t, database.DB.Exec(
		`INSERT INTO cancellation_requests (id, order_id, customer_id, chef_id, status, refund_destination, refund_total_paise, refund_executed)
		 VALUES (?,?,?,?,?,?,?,0)`,
		reqID.String(), oid.String(), cust.String(), chef.String(), "approved", "wallet", paise).Error)

	var cr models.CancellationRequest
	require.NoError(t, database.DB.First(&cr, "id = ?", reqID.String()).Error)
	return oid, &cr
}

func TestExecuteCancellationRefund_WritesRefundLedgerRow(t *testing.T) {
	db := setupSweepDB(t)
	oid, cr := seedCancelForLedger(t, 630, 25200)

	var order models.Order
	require.NoError(t, db.First(&order, "id = ?", oid.String()).Error)
	require.NoError(t, ExecuteCancellationRefund(&order, cr))

	var rows []models.RefundTransaction
	require.NoError(t, db.Find(&rows).Error)
	require.Len(t, rows, 1, "the cancellation refund must be on the ledger")

	row := rows[0]
	assert.Equal(t, oid, row.OrderID)
	assert.Equal(t, models.RefundTxnSucceeded, row.Status)
	assert.Equal(t, 252.0, row.Amount, "records what actually moved")
	assert.Equal(t, "customer cancellation", row.Reason)
	assert.Equal(t, orderrefund.ScopeFull, row.ScopeID)
	// A wallet-funded refund has no gateway leg — say so rather than claiming a
	// gateway refund with an empty id.
	assert.Equal(t, "wallet", row.Provider)
	assert.NotEmpty(t, row.ProviderRefundID)
	require.NotNil(t, row.CompletedAt)

	// THE key property: byte-identical to the key the coordinator would use for the
	// same logical refund, so the UNIQUE index mutually excludes the two paths.
	assert.Equal(t, orderrefund.IdempotencyKeyFor(oid, orderrefund.ScopeFull), row.IdempotencyKey)
}

// The sweep re-drives ExecuteCancellationRefund; a refund already on the ledger
// must not be recorded twice, and must not fail on the second pass.
func TestExecuteCancellationRefund_LedgerRowIsIdempotent(t *testing.T) {
	db := setupSweepDB(t)
	oid, cr := seedCancelForLedger(t, 630, 25200)

	var order models.Order
	require.NoError(t, db.First(&order, "id = ?", oid.String()).Error)
	require.NoError(t, ExecuteCancellationRefund(&order, cr))

	// Re-drive exactly as SweepCancellationRefunds would.
	var fresh models.Order
	require.NoError(t, db.First(&fresh, "id = ?", oid.String()).Error)
	require.NoError(t, ExecuteCancellationRefund(&fresh, cr))
	SweepCancellationRefunds()

	var count int64
	db.Model(&models.RefundTransaction{}).Count(&count)
	assert.EqualValues(t, 1, count, "one logical refund, one ledger row")
}

// A refund that moves no money records nothing — an empty row would read as a
// real refund during reconciliation.
func TestRecordCancellationRefundLedger_SkipsZero(t *testing.T) {
	db := setupSweepDB(t)
	require.NoError(t, recordCancellationRefundLedger(db, cancellationRefundLedgerInput{
		Order: &models.Order{ID: uuid.New()}, Amount: 0,
	}))
	var count int64
	db.Model(&models.RefundTransaction{}).Count(&count)
	assert.EqualValues(t, 0, count)
}

func TestRecordCancellationRefundLedger_PreservesOrderCurrency(t *testing.T) {
	for _, currency := range []string{"AUD", "NZD", "INR", ""} {
		t.Run(currency, func(t *testing.T) {
			db := setupSweepDB(t)
			order := &models.Order{ID: uuid.New(), Currency: currency, PaymentProvider: "stripe"}
			require.NoError(t, recordCancellationRefundLedger(db, cancellationRefundLedgerInput{
				Order: order, Amount: 10, GatewayRefundID: "re_test",
			}))
			var row models.RefundTransaction
			require.NoError(t, db.First(&row, "order_id = ?", order.ID).Error)
			want := currency
			if want == "" {
				want = "INR"
			}
			require.Equal(t, want, row.CurrencyCode)
			require.Equal(t, 10.0, row.Amount)
		})
	}
}
