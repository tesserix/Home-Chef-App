package handlers

import (
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
)

// chef_order_totals.go — pure money-math helpers for order cancellation and
// partial refunds. Extracted from the cancel handlers so the recompute logic
// (the part #8 cares about: subtotal/tax/total recompute after a line is
// pulled) is unit-testable without a DB or the Razorpay gateway.

// lineRefundAmount returns the refund owed for cancelling a single order line:
// the line's subtotal plus its proportional share of the order tax
// (orderTax * lineSubtotal/orderSubtotal). The tax share is computed against
// the order subtotal captured at refund time so concurrent per-line cancels
// split the original tax consistently. A zero/negative order subtotal yields
// just the line subtotal (no tax share).
func lineRefundAmount(lineSubtotal, orderSubtotal, orderTax float64) float64 {
	// Canonical math lives in services so chef per-line cancels and order-issue
	// refunds (#37) settle identically — no duplicated formula.
	return services.LineRefundAmount(lineSubtotal, orderSubtotal, orderTax)
}

// recomputeOrderTotals returns the order subtotal, tax, and total after
// excluding cancelled line items. Tax is scaled from the order's current tax
// by the ratio of the surviving subtotal to the current subtotal, preserving
// the effective tax rate. Delivery/service fees, tip, and discount pass
// through unchanged. A zero/negative current subtotal yields zero tax.
func recomputeOrderTotals(items []models.OrderItem, curSubtotal, curTax, deliveryFee, serviceFee, tip, discount float64) (subtotal, tax, total float64) {
	for _, it := range items {
		if !it.IsCancelled {
			subtotal += it.Subtotal
		}
	}
	if curSubtotal > 0 {
		tax = curTax * (subtotal / curSubtotal)
	}
	total = subtotal + deliveryFee + serviceFee + tax + tip - discount
	return subtotal, tax, total
}
