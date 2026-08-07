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
//     which is Cashfree for every INR kitchen and Stripe for the international
//     ones. There is no third answer since #1086.
//
// The slot-health machinery below survives that removal because Cashfree
// picks sandbox-vs-production by HOSTNAME: test credentials authenticate against
// sandbox.cashfree.com and 401 against api.cashfree.com, so a slot can be fully
// configured and still not work. It no longer changes WHICH gateway is chosen —
// it names the failure for the operator instead of leaving a run of unexplained
// 500s on the checkout endpoint.

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
	// created since it was added. The stored value on an existing chef is
	// not a choice they made; it is the value every row had before there was
	// anything else to be, and treating it as a preference would leave the whole
	// existing estate on the old gateway forever.
	//
	// Safe to override because the ORDER, not the chef, is authoritative for
	// everything afterwards: whichever gateway takes the payment is stamped on
	// the order, and refunds, reconciliation and payout guards all read that. A
	// chef moving to Cashfree today does not disturb a single order taken
	// yesterday.
	//
	// There is no longer a fallback (#1086). An unusable slot now fails at the
	// Cashfree call — loudly, where the failure is — instead of quietly minting
	// an order on a gateway the platform is retiring and would have to refund
	// and reconcile separately for the rest of its life.
	if !cashfreeUsableFor(mode) {
		log.Printf("gateway-select: cashfree[%s] is unusable — this checkout will fail rather than fall back", mode)
	}
	return models.PaymentProviderCashfree
}

// cashfreeGatewayBreaker records a slot that has just failed to create an order,
// so the next checkout does not pay the same round-trip to discover it again.
//
// This exists because of a real, current state: credentials can be present and
// still not work. Cashfree separates sandbox from production by hostname, so a
// live slot holding test credentials resolves to a perfectly valid client that
// then 401s. Without the breaker EVERY checkout would spend a gateway round-trip
// rediscovering that, and log a line per order that would bury real failures.
//
// Deliberately short: this is a circuit breaker, not a health cache. A slot that
// starts working (real live keys are entered) must be picked up within a minute
// or two without a deploy or a restart.
var cashfreeGatewayBreaker sync.Map // mode -> time.Time (when the cooldown ends)

const cashfreeGatewayCooldown = 90 * time.Second

// cashfreeUsableFor reports whether Cashfree should be tried for this mode.
//
// "Configured" is NOT the same as "will work": a slot holding the wrong
// environment's credentials resolves to a perfectly valid client that then 401s.
// So a slot is only usable once it has been PROVED usable: presence, then the
// breaker, then a real health check whose result is cached for the cooldown
// window. The check costs one cheap authenticated request per mode per 90s, not
// one per checkout.
func cashfreeUsableFor(mode string) bool {
	c := GetCashfreeFor(mode)
	if c == nil {
		return false
	}
	if until, ok := cashfreeGatewayBreaker.Load(mode); ok {
		if t, _ := until.(time.Time); time.Now().Before(t) {
			return false
		}
		// Cooldown lapsed. Do NOT optimistically assume recovery — re-probe, so
		// a persistently broken slot stays unusable instead of flapping back to
		// "usable" every 90s and misreporting the aggregator each time.
		cashfreeGatewayBreaker.Delete(mode)
	}
	if healthy, known := cashfreeHealth.Load(mode); known {
		if h, _ := healthy.(cashfreeHealthResult); time.Now().Before(h.until) {
			return h.ok
		}
	}
	ok := c.HealthCheck() == nil
	cashfreeHealth.Store(mode, cashfreeHealthResult{ok: ok, until: time.Now().Add(cashfreeGatewayCooldown)})
	if !ok {
		log.Printf("gateway-select: cashfree[%s] failed its health check — treating as unusable", mode)
	}
	return ok
}

// cashfreeHealthResult caches one slot's probe outcome until `until`.
type cashfreeHealthResult struct {
	ok    bool
	until time.Time
}

var cashfreeHealth sync.Map // mode -> cashfreeHealthResult

// NoteCashfreeGatewayFailure opens the breaker for a mode after a failed order
// creation, so subsequent checkouts do not re-probe a slot known to be down.
//
// Called only from the create path, and only on a failure that has already been
// handled — it suppresses the repeated probe for later checkouts, never the
// outcome of the payment in front of us.
func NoteCashfreeGatewayFailure(mode string) {
	mode = models.NormalizeMode(mode)
	cashfreeGatewayBreaker.Store(mode, time.Now().Add(cashfreeGatewayCooldown))
	// Drop any cached "healthy" verdict: a real order-create failure is stronger
	// evidence than a probe that passed a moment earlier.
	cashfreeHealth.Delete(mode)
	log.Printf("gateway-select: cashfree[%s] marked unusable for %s after an order-create failure",
		mode, cashfreeGatewayCooldown)
}

// DefaultChefPaymentProvider is the provider to stamp on a newly created chef
// profile — the preferred one, which since #1086 is the only one an INR kitchen
// can charge on. It used to consult the slot's health and stamp the other gateway when
// Cashfree was unusable; there is nothing to stamp instead now, and a mode whose
// slot is unprovisioned is an operator problem to fix rather than a reason to
// create kitchens on a retiring gateway.
func DefaultChefPaymentProvider(_ string) string {
	return models.PreferredChefPaymentProvider
}
