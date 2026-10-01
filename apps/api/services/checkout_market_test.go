package services

import (
	"testing"

	"github.com/homechef/api/models"
	"github.com/stretchr/testify/require"
)

func TestOrderCheckoutProviderUsesMarketAndPreservesExistingPayments(t *testing.T) {
	for _, tc := range []struct {
		name, country, currency, configured, provider, intent, gateway, want string
		wantError                                                            bool
	}{
		{name: "India cannot select Stripe", country: "IN", currency: "INR", configured: "stripe", want: "cashfree"},
		{name: "Australia uses Stripe", country: "AU", currency: "AUD", configured: "cashfree", want: "stripe"},
		{name: "New Zealand uses Stripe", country: "NZ", currency: "NZD", configured: "cashfree", want: "stripe"},
		{name: "legacy India", currency: "INR", want: "cashfree"},
		{name: "wrong currency", country: "AU", currency: "INR", wantError: true},
		{name: "unknown market", country: "US", currency: "USD", wantError: true},
		{name: "existing intent survives chef edit", country: "IN", currency: "AUD", provider: "stripe", intent: "pi_existing", want: "stripe"},
		{name: "existing Cashfree survives chef edit", country: "AU", currency: "INR", provider: "cashfree", gateway: "cf_existing", want: "cashfree"},
		{name: "conflicting gateway IDs", country: "AU", currency: "AUD", provider: "stripe", intent: "pi_existing", gateway: "cf_existing", wantError: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			order := &models.Order{Currency: tc.currency, PaymentProvider: tc.provider, StripePaymentIntentID: tc.intent, GatewayOrderID: tc.gateway}
			order.Chef.PayoutCountry = tc.country
			order.Chef.PaymentProvider = tc.configured
			provider, err := OrderCheckoutProvider(order)
			if tc.wantError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, provider)
		})
	}
}

func TestOrderSupplyCountry(t *testing.T) {
	for _, tc := range []struct {
		name, chef, address, want string
		pickup, invalid           bool
	}{
		{name: "AU pickup uses kitchen", chef: "AU", pickup: true, want: "AU"},
		{name: "NZ pickup ignores customer location", chef: "NZ", address: "IN", pickup: true, want: "NZ"},
		{name: "AU delivery", chef: "AU", address: "au", want: "AU"},
		{name: "NZ delivery", chef: "nz", address: "NZ", want: "NZ"},
		{name: "AU cannot choose Indian tax", chef: "AU", address: "IN", invalid: true},
		{name: "NZ missing address country", chef: "NZ", invalid: true},
		{name: "legacy India", want: "IN"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fulfillment := models.FulfillmentDelivery
			if tc.pickup {
				fulfillment = models.FulfillmentPickup
			}
			got, err := OrderSupplyCountry(tc.chef, tc.address, fulfillment)
			if tc.invalid {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}
