package services

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/homechef/api/models"
)

// gstLines is what a caller actually renders: the intra decision this package
// owns, fed into the one split in models/pricing.go.
func gstLines(t *testing.T, tax, rate float64, sellerState, buyerState string) []models.TaxLine {
	t.Helper()
	return models.BuildTaxLines(
		models.OrderPricing{Tax: tax},
		models.PricingInput{
			Rates:      models.TaxRates{Food: rate, Service: rate, Delivery: rate},
			Country:    "IN",
			IntraState: IsIntraStateSupply(sellerState, buyerState),
		})
}

func TestGSTLines(t *testing.T) {
	t.Run("intra-state → CGST+SGST, exact halves", func(t *testing.T) {
		lines := gstLines(t, 11.00, 5, "Maharashtra", "maharashtra ")
		require.Len(t, lines, 2)
		require.Equal(t, models.TaxLineCGST, lines[0].Code)
		require.Equal(t, models.TaxLineSGST, lines[1].Code)
		require.InDelta(t, 5.50, lines[0].Amount, 1e-9)
		require.InDelta(t, 5.50, lines[1].Amount, 1e-9)
		require.InDelta(t, 11.00, lines[0].Amount+lines[1].Amount, 1e-9)
		require.InDelta(t, 2.5, lines[0].Rate, 1e-9)
		require.Equal(t, "CGST (2.5%)", lines[0].Label)
	})

	t.Run("odd amount splits with no rounding drift", func(t *testing.T) {
		lines := gstLines(t, 11.01, 5, "KA", "KA")
		require.InDelta(t, 11.01, lines[0].Amount+lines[1].Amount, 1e-9)
	})

	t.Run("inter-state → IGST at the full rate", func(t *testing.T) {
		lines := gstLines(t, 11.00, 5, "Odisha", "Maharashtra")
		require.Len(t, lines, 1)
		require.Equal(t, models.TaxLineIGST, lines[0].Code)
		require.InDelta(t, 11.00, lines[0].Amount, 1e-9)
		require.Equal(t, "IGST (5%)", lines[0].Label)
	})

	t.Run("unknown state defaults to intra (kitchen place-of-supply)", func(t *testing.T) {
		require.True(t, IsIntraStateSupply("", "Maharashtra"))
		require.True(t, IsIntraStateSupply("Maharashtra", ""))
	})
}
