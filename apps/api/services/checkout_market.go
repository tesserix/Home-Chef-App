package services

import (
	"fmt"
	"strings"

	"github.com/homechef/api/internal/markets"
	"github.com/homechef/api/models"
)

// OrderSupplyCountry binds domestic AU/NZ tax selection to the kitchen's market.
func OrderSupplyCountry(chefCountry, addressCountry string, fulfillment models.FulfillmentType) (string, error) {
	chefCountry = strings.ToUpper(strings.TrimSpace(chefCountry))
	addressCountry = strings.ToUpper(strings.TrimSpace(addressCountry))
	if chefCountry == "AU" || chefCountry == "NZ" {
		if fulfillment != models.FulfillmentPickup && addressCountry != chefCountry {
			return "", fmt.Errorf("delivery address must be in the kitchen's country")
		}
		return chefCountry, nil
	}
	if addressCountry == "" {
		addressCountry = "IN"
	}
	return addressCountry, nil
}

// OrderCheckoutProvider keeps retries on their original gateway and routes new charges by market.
func OrderCheckoutProvider(order *models.Order) (string, error) {
	if order.StripePaymentIntentID != "" {
		if order.PaymentProvider != models.PaymentProviderStripe || order.GatewayOrderID != "" {
			return "", fmt.Errorf("conflicting payment gateway references")
		}
		return models.PaymentProviderStripe, nil
	}
	if order.GatewayOrderID != "" {
		if order.PaymentProvider != models.PaymentProviderCashfree || !strings.EqualFold(order.Currency, "INR") {
			return "", fmt.Errorf("existing payment gateway cannot be changed")
		}
		return models.PaymentProviderCashfree, nil
	}
	country := strings.ToUpper(strings.TrimSpace(order.Chef.PayoutCountry))
	if country == "" {
		country = "IN"
	}
	market, ok := markets.Lookup(country)
	if !ok {
		return "", fmt.Errorf("unsupported payment market")
	}
	if !strings.EqualFold(order.Currency, market.Currency) {
		return "", fmt.Errorf("order currency does not match payment market")
	}
	return market.PaymentProvider, nil
}
