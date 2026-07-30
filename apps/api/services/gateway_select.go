package services

import (
	"log"
	"sync"
	"time"

	"github.com/homechef/api/models"
)

// gateway_select.go — which gateway takes a NEW payment.
//
// Two separate questions live here, and they must not be merged:
//
//  1. What gateway should a NEW kitchen be created with? →
//     models.PreferredChefPaymentProvider (Cashfree).
//  2. What gateway should THIS checkout actually use? → SelectCheckoutGateway,
//     which prefers Cashfree for every INR kitchen and degrades to Razorpay
//     rather than failing when Cashfree has no usable credentials for the mode.
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

	// Stripe is decided by the chef and never overridden. A Stripe chef settles
	// in a non-INR currency from a Connect country; Cashfree cannot serve them at
	// all, so preferring it would charge the wrong currency rather than fail
	// honestly.
	if provider == models.PaymentProviderStripe {
		return models.PaymentProviderStripe
	}

	// CASHFREE IS THE PLATFORM DEFAULT for every INR kitchen — not only for ones
	// created since it was added. The stored `razorpay` on an existing chef is
	// not a choice they made; it is the value every row had before there was
	// anything else to be, and treating it as a preference would leave the whole
	// existing estate on the old gateway forever.
	//
	// Safe to override because the ORDER, not the chef, is authoritative for
	// everything afterwards: whichever gateway takes the payment is stamped on
	// the order, and refunds, reconciliation and payout guards all read that. A
	// chef moving to Cashfree today does not disturb a single order taken
	// yesterday.
	if cashfreeUsableFor(mode) {
		return models.PaymentProviderCashfree
	}

	// Loud on purpose. A silent downgrade would mean the platform quietly
	// stopped using its preferred gateway and nobody noticed until a
	// reconciliation looked odd.
	log.Printf("gateway-select: cashfree[%s] unusable — falling back to razorpay for this checkout", mode)
	return models.PaymentProviderRazorpay
}

// cashfreeGatewayBreaker records a slot that has just failed to create an order,
// so the next checkout does not pay the same round-trip to discover it again.
//
// This exists because of a real, current state: credentials can be present and
// still not work. Cashfree separates sandbox from production by hostname, so a
// live slot holding test credentials resolves to a perfectly valid client that
// then 401s. With Cashfree as the platform default, EVERY checkout would spend a
// gateway round-trip discovering that before falling back — added latency on the
// customer's critical path, and an error log line per order that would bury real
// failures.
//
// Deliberately short: this is a circuit breaker, not a health cache. A slot that
// starts working (real live keys are entered) must be picked up within a minute
// or two without a deploy or a restart.
var cashfreeGatewayBreaker sync.Map // mode -> time.Time (when the cooldown ends)

const cashfreeGatewayCooldown = 90 * time.Second

// cashfreeUsableFor reports whether Cashfree should be tried for this mode.
func cashfreeUsableFor(mode string) bool {
	if GetCashfreeFor(mode) == nil {
		return false
	}
	if until, ok := cashfreeGatewayBreaker.Load(mode); ok {
		if t, _ := until.(time.Time); time.Now().Before(t) {
			return false
		}
		cashfreeGatewayBreaker.Delete(mode)
	}
	return true
}

// NoteCashfreeGatewayFailure opens the breaker for a mode after a failed order
// creation, so subsequent checkouts skip straight to the fallback.
//
// Called only from the create path, and only on a failure that has already been
// handled — it changes which gateway later customers are offered, never the
// outcome of the payment in front of us.
func NoteCashfreeGatewayFailure(mode string) {
	mode = models.NormalizeMode(mode)
	cashfreeGatewayBreaker.Store(mode, time.Now().Add(cashfreeGatewayCooldown))
	log.Printf("gateway-select: cashfree[%s] marked unusable for %s after an order-create failure",
		mode, cashfreeGatewayCooldown)
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
		cashfreeUsableFor(mode) {
		return models.PaymentProviderCashfree
	}
	return models.PaymentProviderRazorpay
}
