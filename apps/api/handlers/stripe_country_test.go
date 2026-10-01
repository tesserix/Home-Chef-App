package handlers

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestPaymentProviderCannotOverrideCountry(t *testing.T) {
	for _, tc := range []struct{ country, requested string }{
		{"IN", "stripe"}, {"", "stripe"}, {"AU", "cashfree"}, {"NZ", "cashfree"}, {"US", "stripe"},
	} {
		t.Run(tc.country+tc.requested, func(t *testing.T) {
			db := setupPayDB(t)
			user := payUser(t, db, "chef")
			chef := payChef(t, db, user)
			require.NoError(t, db.Exec("UPDATE chef_profiles SET payout_country = ?, stripe_account_id = 'acct_fixture' WHERE id = ?", tc.country, chef).Error)
			response := callPay(user, http.MethodPut, "/chef/payment-provider", func(r *gin.Engine, _ *PaymentHandler) {
				r.PUT("/chef/payment-provider", NewStripeConnectHandler().SetPaymentProvider)
			}, map[string]string{"provider": tc.requested})
			require.Equal(t, http.StatusUnprocessableEntity, response.Code, response.Body.String())
			require.Contains(t, response.Body.String(), "market_provider_mismatch")
			var provider string
			require.NoError(t, db.Raw("SELECT payment_provider FROM chef_profiles WHERE id = ?", chef).Scan(&provider).Error)
			require.Equal(t, "cashfree", provider)
		})
	}
}

func TestStripeConnectRejectsCountryOverrideBeforeGatewayCall(t *testing.T) {
	for _, tc := range []struct{ stored, requested string }{
		{"IN", "AU"}, {"", "AU"}, {"AU", "NZ"}, {"NZ", "US"}, {"US", "US"},
	} {
		t.Run(tc.stored+tc.requested, func(t *testing.T) {
			db := setupPayDB(t)
			user := payUser(t, db, "chef")
			chef := payChef(t, db, user)
			require.NoError(t, db.Exec("UPDATE chef_profiles SET payout_country = ? WHERE id = ?", tc.stored, chef).Error)
			response := callPay(user, http.MethodPost, "/chef/stripe/connect", func(r *gin.Engine, _ *PaymentHandler) {
				r.POST("/chef/stripe/connect", NewStripeConnectHandler().CreateStripeConnectAccount)
			}, map[string]string{"country": tc.requested})
			require.Equal(t, http.StatusUnprocessableEntity, response.Code, response.Body.String())
			require.Contains(t, response.Body.String(), "market_provider_mismatch")
		})
	}
}
