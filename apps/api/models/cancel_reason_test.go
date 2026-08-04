package models

// cancel_reason_test.go — D-17. The chef picks a labelled button; the customer
// was shown the database value behind it ("this order was cancelled
// out_of_ingredient"). The column also carries free apology text from the
// platform's own void paths, so the mapping has to pass prose through untouched.

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCustomerCancelReason(t *testing.T) {
	t.Run("every chef-picked reason has customer wording", func(t *testing.T) {
		for _, r := range []CancelReason{
			CancelReasonOutOfIngredient, CancelReasonEquipmentFailure,
			CancelReasonCustomerRequest, CancelReasonOther,
		} {
			got := CustomerCancelReason(string(r))
			require.NotEqual(t, string(r), got, "%s still reaches the customer as the raw enum", r)
			assert.NotContains(t, got, "_", "%s reads as prose, not a database value", r)
		}
	})

	t.Run("the reported case", func(t *testing.T) {
		assert.Equal(t, "The chef ran out of an ingredient", CustomerCancelReason("out_of_ingredient"))
	})

	t.Run("platform apology text passes through", func(t *testing.T) {
		assert.Equal(t, "payment not completed", CustomerCancelReason("payment not completed"))
		assert.Empty(t, CustomerCancelReason(""))
	})

	t.Run("the order response resolves it", func(t *testing.T) {
		o := Order{CancelReason: string(CancelReasonOutOfIngredient)}
		assert.Equal(t, "The chef ran out of an ingredient", o.ToResponse().CancelReason)
	})
}
