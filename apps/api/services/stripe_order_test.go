package services

import (
	"testing"

	"github.com/homechef/api/models"
	"github.com/stretchr/testify/require"
)

func TestValidateStripeOrderPaymentBindsCapturedFunds(t *testing.T) {
	for _, currency := range []string{"AUD", "NZD"} {
		t.Run(currency, func(t *testing.T) {
			order := &models.Order{Total: 50, Currency: currency, PaymentProvider: "stripe", StripePaymentIntentID: "pi_test"}
			good := StripePaymentIntent{ID: "pi_test", Amount: 5000, AmountReceived: 5000, Currency: currency, Status: "succeeded", Livemode: true}
			require.NoError(t, ValidateStripeOrderPayment(order, &good))
			for _, tc := range []struct {
				name   string
				mutate func(*StripePaymentIntent)
			}{
				{"wrong intent", func(p *StripePaymentIntent) { p.ID = "pi_other" }},
				{"wrong environment", func(p *StripePaymentIntent) { p.Livemode = false }},
				{"wrong currency", func(p *StripePaymentIntent) { p.Currency = "USD" }},
				{"wrong amount", func(p *StripePaymentIntent) { p.Amount = 100 }},
				{"not captured in full", func(p *StripePaymentIntent) { p.AmountReceived = 100 }},
				{"requires action", func(p *StripePaymentIntent) { p.Status = "requires_action" }},
			} {
				t.Run(tc.name, func(t *testing.T) {
					pi := good
					tc.mutate(&pi)
					require.Error(t, ValidateStripeOrderPayment(order, &pi))
				})
			}
		})
	}
}
