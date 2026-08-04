package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// order_issue_food_tax_test.go — D-19. A food complaint may reclaim the GST on the
// FOOD supply and nothing else. Every case here is pinned to the order that found
// the defect (HC26080405191298: 320 food, tax_food 16.00, tax_service 2.44,
// tax_delivery 1.96 — 20.40 of tax in total), because the over-request was paid by
// an AUTO refund with no human in the loop.

func TestIssueRefundClaimsFoodGSTOnly(t *testing.T) {
	const (
		subtotal    = 320.0
		taxFood     = 16.0
		taxService  = 2.44
		taxDelivery = 1.96
		orderTax    = taxFood + taxService + taxDelivery // 20.40
		platformFee = 13.53
		orderTotal  = subtotal + platformFee + orderTax
	)

	t.Run("whole order reported", func(t *testing.T) {
		basis := ChefTaxOf(orderTax, taxFood, taxService)
		got := ComputeIssueRefund(subtotal, basis, orderTotal, 0, []float64{subtotal})
		assert.InDelta(t, 336.0, got, 0.001, "food plus its own GST")
		// The defect: passing the whole order tax over-requests by exactly the GST on
		// the platform fee and the delivery fee.
		wrong := ComputeIssueRefund(subtotal, orderTax, orderTotal, 0, []float64{subtotal})
		assert.InDelta(t, 4.40, wrong-got, 0.001, "tax_service + tax_delivery")
	})

	t.Run("one line of several", func(t *testing.T) {
		// 120 of the 320 subtotal → 120 + 16.00*(120/320) = 126.00.
		basis := ChefTaxOf(orderTax, taxFood, taxService)
		got := ComputeIssueRefund(subtotal, basis, orderTotal, 0, []float64{120})
		assert.InDelta(t, 126.0, got, 0.001)
	})

	t.Run("the apportioned shares sum to the food GST, never past it", func(t *testing.T) {
		basis := ChefTaxOf(orderTax, taxFood, taxService)
		lines := []float64{100, 150, 70} // = the whole 320 subtotal
		var sum float64
		for _, l := range lines {
			sum += LineRefundAmount(l, subtotal, basis)
		}
		assert.InDelta(t, subtotal+taxFood, sum, 0.001,
			"reporting every line reclaims the food GST exactly once and no fee GST")
	})
}

func TestIssueRefundTaxBasisBySnapshot(t *testing.T) {
	t.Run("per-supply snapshot yields the food component", func(t *testing.T) {
		assert.Equal(t, 16.0, ChefTaxOf(20.40, 16.0, 2.44))
	})
	t.Run("food fully discounted away claims no GST", func(t *testing.T) {
		// TaxFood 0 with a non-zero service leg still proves the snapshot exists, so
		// the claim gets 0 tax — not the fee's GST.
		assert.Zero(t, ChefTaxOf(4.40, 0, 2.44))
		assert.InDelta(t, 100.0, ComputeIssueRefund(100, ChefTaxOf(4.40, 0, 2.44), 118.0, 0, []float64{100}), 0.001)
	})
	t.Run("orders predating the split keep the figure they were charged", func(t *testing.T) {
		// No snapshot: Tax was food GST alone and their refunds were reconciled
		// against it, so restating them years later would be the bug.
		assert.Equal(t, 25.0, ChefTaxOf(25.0, 0, 0))
	})
}
