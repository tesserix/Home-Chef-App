package services

import (
	"fmt"
	"strings"

	"github.com/homechef/api/models"
)

// ValidateStripeOrderPayment binds a Stripe capture to the stored order.
func ValidateStripeOrderPayment(order *models.Order, pi *StripePaymentIntent) error {
	if pi == nil || pi.ID == "" || order.StripePaymentIntentID != pi.ID || order.PaymentProvider != models.PaymentProviderStripe {
		return fmt.Errorf("payment does not belong to this order")
	}
	if pi.Status != "succeeded" {
		return fmt.Errorf("payment has not succeeded")
	}
	if pi.Livemode == (order.Mode == models.ChefModeTest) {
		return fmt.Errorf("payment environment does not match")
	}
	if order.Currency == "" || !strings.EqualFold(order.Currency, pi.Currency) {
		return fmt.Errorf("payment currency does not match")
	}
	due := ToMinor(order.Total-order.WalletApplied-order.LoyaltyApplied, order.Currency)
	if due <= 0 || pi.Amount != due || pi.AmountReceived != due {
		return fmt.Errorf("payment amount does not match")
	}
	return nil
}
