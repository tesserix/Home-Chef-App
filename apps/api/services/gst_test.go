package services

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSplitIndiaGST(t *testing.T) {
	t.Run("intra-state → CGST+SGST, exact halves", func(t *testing.T) {
		b := SplitIndiaGST(11.00, 5, "Maharashtra", "maharashtra ")
		require.True(t, b.Intra)
		require.InDelta(t, 5.50, b.CGST, 1e-9)
		require.InDelta(t, 5.50, b.SGST, 1e-9)
		require.InDelta(t, 11.00, b.CGST+b.SGST, 1e-9) // sums to the total, no drift
		require.InDelta(t, 2.5, b.CGSTRate, 1e-9)
		require.Equal(t, 0.0, b.IGST)
	})

	t.Run("odd amount splits with no rounding drift", func(t *testing.T) {
		b := SplitIndiaGST(11.01, 5, "KA", "KA")
		require.InDelta(t, 11.01, b.CGST+b.SGST, 1e-9)
	})

	t.Run("inter-state → IGST at the full rate", func(t *testing.T) {
		b := SplitIndiaGST(11.00, 5, "Odisha", "Maharashtra")
		require.False(t, b.Intra)
		require.InDelta(t, 11.00, b.IGST, 1e-9)
		require.InDelta(t, 5.0, b.IGSTRate, 1e-9)
		require.Equal(t, 0.0, b.CGST)
	})

	t.Run("unknown state defaults to intra (kitchen place-of-supply)", func(t *testing.T) {
		require.True(t, SplitIndiaGST(10, 5, "", "Maharashtra").Intra)
		require.True(t, SplitIndiaGST(10, 5, "Maharashtra", "").Intra)
	})
}
