package models

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// EffectiveDeliveryFee is the figure every refund/settlement path must use after
// the chef's #703 reduction, so a later cancellation can't double-refund the
// delivery portion the chef already refunded. (#708)
func TestEffectiveDeliveryFee(t *testing.T) {
	// No chef adjustment → the charged fee stands.
	o := Order{DeliveryFee: 50}
	require.Equal(t, 50.0, o.EffectiveDeliveryFee())

	// Chef lowered it to 20 at accept → effective = 20 (the 30 diff was already
	// refunded by #703, so a cancellation must not refund it again).
	final := 20.0
	o.DeliveryFeeFinal = &final
	require.Equal(t, 20.0, o.EffectiveDeliveryFee())

	// Chef made it free (within their zone) → effective = 0.
	zero := 0.0
	o.DeliveryFeeFinal = &zero
	require.Equal(t, 0.0, o.EffectiveDeliveryFee())
}
