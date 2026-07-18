package services

// invoice_receipt_test.go — a TAX INVOICE is only for a completed, delivered
// sale. A refunded or not-yet-delivered order that the customer paid for gets a
// PAYMENT RECEIPT, never a tax invoice — issuing a tax invoice for a refunded
// order would claim a taxable sale that did not complete. This pins that legal
// distinction so a future edit can't quietly stamp "TAX INVOICE" on a refund.

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/homechef/api/models"
)

func TestOrderInvoiceIsTaxInvoice_OnlyForADeliveredUnrefundedSale(t *testing.T) {
	delivered := &models.Order{Status: models.OrderStatusDelivered, RefundAmount: 0}
	require.True(t, orderInvoiceIsTaxInvoice(delivered),
		"a delivered, unrefunded order is a completed sale → tax invoice")

	// A delivered order that was later part-refunded is no longer a clean sale.
	deliveredRefunded := &models.Order{Status: models.OrderStatusDelivered, RefundAmount: 50}
	require.False(t, orderInvoiceIsTaxInvoice(deliveredRefunded),
		"a refund means the sale did not complete as invoiced → receipt, not tax invoice")

	for _, s := range []models.OrderStatus{
		models.OrderStatusPending, models.OrderStatusAccepted, models.OrderStatusPreparing,
		models.OrderStatusReady, models.OrderStatusCancelled, models.OrderStatusRefunded,
	} {
		o := &models.Order{Status: s}
		require.False(t, orderInvoiceIsTaxInvoice(o),
			"%s is not a completed delivery → payment receipt, not a tax invoice", s)
	}
}
