package services

import (
	"log"

	"github.com/homechef/api/models"
)

// gateway_select.go — which gateway takes a NEW payment.
//
// Two separate questions live here, and they must not be merged:
//
//  1. What gateway should a NEW kitchen be created with? →
//     models.PreferredChefPaymentProvider (Cashfree).
//  2. What gateway should THIS checkout actually use? → SelectCheckoutGateway,
//     which honours the chef's configured choice but degrades rather than failing
//     when that gateway has no usable credentials for the order's mode.
//
// The degradation is not defensive padding; it is load-bearing given how Cashfree
// separates environments. Cashfree picks sandbox-vs-production by HOSTNAME, so a
// set of test credentials authenticates against sandbox.cashfree.com and returns
// 401 against api.cashfree.com. A platform whose live Cashfree slot is not yet
// provisioned with real live keys — the normal state while the merchant account is
// still in review — would therefore 503 every live checkout the moment Cashfree
// became the default. Falling back to Razorpay keeps real money flowing while the
// live slot is finished, which is exactly what "Cashfree preferred, Razorpay still
// an option" has to mean in practice.

// SelectCheckoutGateway resolves the provider for a new payment on an order in
// the given mode.
//
// `configured` is the chef's stored PaymentProvider. The returned value is what
// the checkout must actually use AND what gets stamped on the order — the two can
// never disagree, or a refund would later be routed to a gateway that never took
// the payment.
func SelectCheckoutGateway(configured, mode string) string {
	provider := models.NormalizeProvider(configured)

	switch provider {
	case models.PaymentProviderCashfree:
		if GetCashfreeFor(mode) != nil {
			return models.PaymentProviderCashfree
		}
		// Loud on purpose. A silent downgrade would mean the platform quietly
		// stopped using its preferred gateway and nobody noticed until a
		// reconciliation looked odd.
		log.Printf("gateway-select: cashfree[%s] has no credentials — falling back to razorpay for this checkout", mode)
		return models.PaymentProviderRazorpay

	case models.PaymentProviderStripe:
		// No fallback. A Stripe chef is settling in a non-INR currency from a
		// Connect country; Razorpay cannot serve them at all, so degrading would
		// charge the wrong currency rather than fail honestly.
		return models.PaymentProviderStripe

	default:
		return models.PaymentProviderRazorpay
	}
}

// DefaultChefPaymentProvider is the provider to stamp on a newly created chef
// profile.
//
// It returns the preferred gateway only when that gateway is actually configured
// for the chef's mode; otherwise Razorpay. Stamping a provider the platform cannot
// serve would leave a brand-new kitchen unable to take a payment until an admin
// noticed — and the chef, not the admin, is the one who sees the failure.
//
// Note this reads the LIVE slot for a live chef and the TEST slot for a test one,
// so a platform with only sandbox Cashfree credentials creates test kitchens on
// Cashfree and live kitchens on Razorpay. That is the correct behaviour, not a
// compromise: it is precisely the state of a merchant account still in review.
func DefaultChefPaymentProvider(mode string) string {
	if models.PreferredChefPaymentProvider == models.PaymentProviderCashfree &&
		GetCashfreeFor(mode) != nil {
		return models.PaymentProviderCashfree
	}
	return models.PaymentProviderRazorpay
}
