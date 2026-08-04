package handlers

// perline_cancel_food_tax_test.go — D-19, chef-cancel half. A cancelled FOOD line
// returns the food GST on that line and nothing else: the GST on the platform fee
// and on delivery belongs to supplies the customer still receives. The refund and
// the order-total reduction are one number, so the per-supply columns must move
// with it or Total stops equalling the sum of its parts.

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// snapshotTax stamps a per-supply GST snapshot on a seeded order (the sqlite
// harness seeds the pre-split shape: tax only).
func snapshotTax(t *testing.T, db *gorm.DB, orderID uuid.UUID, food, service, delivery float64) {
	t.Helper()
	require.NoError(t, db.Exec(`UPDATE orders SET tax = ?, tax_food = ?, tax_service = ?, tax_delivery = ? WHERE id = ?`,
		food+service+delivery, food, service, delivery, orderID.String()).Error)
}

func orderTaxSplit(t *testing.T, db *gorm.DB, orderID uuid.UUID) (tax, food, service, delivery float64) {
	t.Helper()
	var row struct{ Tax, TaxFood, TaxService, TaxDelivery float64 }
	require.NoError(t, db.Raw(`SELECT tax, tax_food, tax_service, tax_delivery FROM orders WHERE id = ?`,
		orderID.String()).Scan(&row).Error)
	return row.Tax, row.TaxFood, row.TaxService, row.TaxDelivery
}

func TestPerLineCancelRefundsFoodGSTOnly(t *testing.T) {
	db := setupPayDB(t)
	addIssueGuardTables(t, db)
	orderID, itemA, _ := reserveTwoLineOrder(t, db)
	// 900 of food, taxed 45 food + 8 fee + 7 delivery = 60.
	snapshotTax(t, db, orderID, 45, 8, 7)

	lineRefund, won := doReserve(t, db, orderID, itemA)
	require.True(t, won)
	// A is 500 of the 900 subtotal → 500 + 45*(500/900) = 525.00. On the whole-order
	// basis it would have been 500 + 60*(500/900) = 533.33 — 8.33 of the platform's
	// own GST handed back on supplies the customer still receives.
	require.InDelta(t, 525.0, lineRefund, 0.01, "food GST share only")

	tax, food, service, delivery := orderTaxSplit(t, db, orderID)
	require.InDelta(t, 35.0, tax, 0.01, "60 less the 25 of food GST refunded")
	require.InDelta(t, 20.0, food, 0.01, "the whole reduction lands on the food leg")
	require.InDelta(t, 8.0, service, 0.01, "the fee's GST is untouched")
	require.InDelta(t, 7.0, delivery, 0.01, "delivery's GST is untouched")
	require.InDelta(t, tax, food+service+delivery, 0.01, "the parts still sum to the whole")

	_, _, total, refundAmt := orderMoney(t, db, orderID)
	require.InDelta(t, lineRefund, refundAmt, 0.01, "refund_amount is exactly the line refund")
	require.InDelta(t, 1000-lineRefund, total, 0.01, "Total drops by exactly what was refunded")
}

// The gateway refused, so nothing moved: the per-supply split must come back intact.
func TestPerLineCancelReleaseRestoresFoodTax(t *testing.T) {
	db := setupPayDB(t)
	addIssueGuardTables(t, db)
	orderID, itemA, _ := reserveTwoLineOrder(t, db)
	snapshotTax(t, db, orderID, 45, 8, 7)

	lineRefund, won := doReserve(t, db, orderID, itemA)
	require.True(t, won)
	releaseOrderItemCancelReservation(db, orderID, itemA, lineRefund)

	tax, food, service, delivery := orderTaxSplit(t, db, orderID)
	require.InDelta(t, 60.0, tax, 0.01)
	require.InDelta(t, 45.0, food, 0.01)
	require.InDelta(t, 8.0, service, 0.01)
	require.InDelta(t, 7.0, delivery, 0.01)
}

// An order written before tax was split per supply has no snapshot to read, and its
// refunds were reconciled against the whole Tax — it must keep that basis.
func TestPerLineCancelPreSplitOrderKeepsWholeTaxBasis(t *testing.T) {
	db := setupPayDB(t)
	addIssueGuardTables(t, db)
	orderID, itemA, _ := reserveTwoLineOrder(t, db) // tax 100, no per-supply columns

	lineRefund, won := doReserve(t, db, orderID, itemA)
	require.True(t, won)
	require.InDelta(t, 555.5556, lineRefund, 0.01, "500 + 100*(500/900), unchanged")

	_, food, _, _ := orderTaxSplit(t, db, orderID)
	require.Zero(t, food, "no phantom snapshot is invented for a pre-split order")
}
