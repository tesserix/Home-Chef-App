package models

import (
	"slices"
	"strings"
)

// payment_provider.go — the payment-gateway vocabulary.
//
// Before this file the provider was a bare string compared inline at ~40 sites,
// in three different ways: `== "razorpay"`, `EqualFold(p, "stripe")`, and
// `switch p { … default: /* razorpay */ }`. Two of those three shapes are traps
// once a THIRD gateway exists:
//
//   - `!= PaymentProviderRazorpay → skip the refund` silently stops refunding a
//     Cashfree order. Not an error, not a log line — the customer just never
//     gets their money.
//   - `default: // razorpay` sends a Cashfree order down the Razorpay branch,
//     which reads an empty RazorpayPaymentID and fails at the gateway.
//
// So the provider string is now a closed vocabulary with predicates that say
// what a provider CAN DO rather than which one it IS. Call sites ask
// "does this order settle through a gateway?" instead of "is this razorpay?",
// and adding a fourth gateway becomes a change to this file plus one adapter.
const (
	// PaymentProviderRazorpay — India. Route linked accounts, payment-linked
	// transfers with the on_hold → release → reverse lifecycle the escrow paths
	// are built on.
	PaymentProviderRazorpay = "razorpay"

	// PaymentProviderCashfree — India. Cashfree PG. Captures the FULL order to
	// the platform merchant account; chef/rider money is settled through the
	// statement/payout path, NOT split at the gateway. See
	// ProviderSupportsGatewaySplit for why.
	PaymentProviderCashfree = "cashfree"

	// PaymentProviderStripe — international. Connect destination charges.
	PaymentProviderStripe = "stripe"

	// PaymentProviderWallet — no gateway at all: the order was fully covered by
	// store credit, so there is nothing to capture and nothing to refund
	// externally. Never a chef's configured provider; only ever stamped on an
	// order at settlement.
	PaymentProviderWallet = "wallet"
)

// PreferredChefPaymentProvider is the gateway a NEW chef is created with —
// Cashfree is the platform's first choice for India, with Razorpay remaining a
// fully supported option a chef or admin can switch to.
//
// This is deliberately SEPARATE from NormalizeProvider's fallback, and the
// distinction is the whole point:
//
//   - PreferredChefPaymentProvider answers "what should a NEW kitchen get?" →
//     cashfree.
//   - NormalizeProvider("") answers "what does an UNSTAMPED EXISTING ROW mean?" →
//     razorpay, because every row that predates the payment_provider column really
//     is a Razorpay order.
//
// Conflating them would silently reinterpret every historical order as Cashfree,
// and their refunds would be issued against a gateway that never took the money.
const PreferredChefPaymentProvider = PaymentProviderCashfree

// NormalizeProvider coerces a stored or user-supplied provider to a known value.
//
// An empty or unrecognised value becomes razorpay. That is not a guess — it is
// the pre-existing behaviour of every `if provider == "" { provider = "razorpay" }`
// this function replaces, and it is the right failure direction for the rows that
// predate the payment_provider column (all of which really are Razorpay orders).
//
// Note the asymmetry with NormalizeMode: mode fails toward live because a wrong
// "test" silently captures no money. Provider fails toward razorpay because that
// is what unstamped historical rows actually are, and a wrong answer here
// produces a loud gateway rejection rather than silent data loss.
func NormalizeProvider(p string) string {
	switch strings.ToLower(strings.TrimSpace(p)) {
	case PaymentProviderCashfree:
		return PaymentProviderCashfree
	case PaymentProviderStripe:
		return PaymentProviderStripe
	case PaymentProviderWallet:
		return PaymentProviderWallet
	default:
		return PaymentProviderRazorpay
	}
}

// IsGatewayProvider reports whether this provider involves an external payment
// gateway — i.e. there is money held at a third party that a refund has to be
// sent to, and a webhook that can arrive about it.
//
// False only for wallet, where the "refund" is a store-credit ledger entry.
func IsGatewayProvider(p string) bool {
	return NormalizeProvider(p) != PaymentProviderWallet
}

// SelectableChefProviders are the providers a chef (or an admin on their behalf)
// may actually be configured with. Wallet is excluded — it is an outcome of a
// fully-credit-covered order, never a configuration.
func SelectableChefProviders() []string {
	return []string{PaymentProviderRazorpay, PaymentProviderCashfree, PaymentProviderStripe}
}

// IsSelectableChefProvider validates a provider submitted by an admin or chef.
// Deliberately strict — unlike NormalizeProvider it does NOT coerce, because
// silently saving "razorpy" as razorpay would hide a typo that the operator
// needs to see.
func IsSelectableChefProvider(p string) bool {
	return slices.Contains(SelectableChefProviders(), strings.ToLower(strings.TrimSpace(p)))
}

// UsesINRPaise reports whether this provider's money is denominated in INR and
// converted through the paise helpers (ToPaise/FromPaise), as opposed to
// Stripe's currency-aware minor units (ToMinor/FromMinor, where the divisor is
// 1, 100 or 1000 depending on the currency).
//
// This is the predicate that `razorpay := !EqualFold(provider, "stripe")` was
// really reaching for. Cashfree is INR, so it answers true — but note that
// Cashfree's WIRE format is rupees-as-decimal, not paise; that conversion lives
// in the Cashfree client, not here.
func UsesINRPaise(p string) bool {
	switch NormalizeProvider(p) {
	case PaymentProviderStripe:
		return false
	default:
		return true
	}
}

// ProviderSupportsGatewaySplit reports whether this provider splits an order
// across chef/rider payees AT THE GATEWAY, producing a transfer object with the
// on_hold → release → reverse lifecycle that the escrow paths drive
// (meal_plan_escrow, group_order_payout, order_payout, escrow_ledger_reconcile,
// and the payout_hold state machine).
//
// TRUE for Razorpay only. Route gives every split a `transfer` with an id,
// `on_hold`, `amount_reversed`, a PATCH to release and a POST to reverse — the
// exact primitives that whole layer assumes.
//
// FALSE for Cashfree. Easy Split has no comparable object: it offers split-at-
// order or split-after-payment, a date-based "settlement eligibility date" for
// deferral, vendor-balance CREDIT/DEBIT transfers, and automatic pro-rata refund
// deduction once a split has settled. There is no transfer id to hold, release
// or partially reverse, so pretending otherwise would mean an escrow ledger that
// reconciles against nothing. Cashfree therefore captures the full amount to the
// platform and the chef/rider are paid through the ordinary statement/payout
// path — the same route a Razorpay order takes when
// ORDER_PAYOUT_AUTO_RELEASE_ENABLED is off.
//
// FALSE for Stripe: it uses destination charges + application fees, and its
// reversal is a flag on the refund (ReverseTransfer), not a transfer lifecycle.
//
// Every gateway-transfer call site must be guarded on this. A no-op is the
// CORRECT behaviour for a provider that answers false — but it must be a
// deliberate, logged no-op, never a silent fallthrough into the Razorpay branch.
func ProviderSupportsGatewaySplit(p string) bool {
	return NormalizeProvider(p) == PaymentProviderRazorpay
}

// GatewayOrderIDColumn / GatewayPaymentIDColumn document a deliberate reuse.
//
// Order.RazorpayOrderID / .RazorpayPaymentID (and MealPlan, GroupOrder, Tip,
// Catering, FeaturedListing) hold the GATEWAY's order and payment ids —
// whichever gateway that is. A Cashfree order's `order_id` and `cf_payment_id`
// go in the same two columns.
//
// The names are historical and the reuse is intentional, not laziness. Those
// columns carry partial unique indexes (WHERE col <> ”, see database.go's
// postMigrate), and they are read by the reconciliation cron, the meal-plan
// advance lookup, the group-order settle path and the escrow ledger. Adding a
// parallel cashfree_order_id to six models would fork every one of those
// queries — and a fork is precisely how a payment goes missing. One column plus
// PaymentProvider to say whose id it is keeps a single code path.
//
// A rename to gateway_order_id / gateway_payment_id is the right eventual
// cleanup; it is a mechanical migration + rename, deliberately not bundled with
// adding a gateway.
const (
	GatewayOrderIDColumn   = "razorpay_order_id"
	GatewayPaymentIDColumn = "razorpay_payment_id"
)

// GatewayRefundReference returns the id a refund must be issued against for this
// order, or "" when the order has no refundable gateway payment.
//
// Which id that is differs by gateway, and getting it wrong is a silent failure
// rather than a loud one:
//
//   - Razorpay refunds a PAYMENT: POST /payments/{payment_id}/refund.
//   - Cashfree refunds an ORDER:  POST /orders/{order_id}/refunds. Its payment id
//     exists, but no refund endpoint takes it.
//   - Stripe refunds a PaymentIntent.
//   - Wallet has nothing at a gateway; a wallet "refund" is a ledger credit.
//
// Callers that only need the yes/no should use GatewayRefundable.
func (o *Order) GatewayRefundReference() string {
	if o == nil {
		return ""
	}
	switch NormalizeProvider(o.PaymentProvider) {
	case PaymentProviderCashfree:
		return o.RazorpayOrderID
	case PaymentProviderStripe:
		return o.StripePaymentIntentID
	case PaymentProviderWallet:
		return ""
	default:
		return o.RazorpayPaymentID
	}
}

// GatewayRefundable reports whether a refund can actually be issued to this
// order's original payment method.
//
// This replaces the `order.PaymentProvider != "razorpay" || order.RazorpayPaymentID == ""`
// guard that appeared at five call sites. That guard was correct when Razorpay
// was the only INR gateway and actively wrong the moment a second one existed: a
// Cashfree order has a perfectly refundable payment, and the old test would have
// answered "no" — routing the customer's money to store credit at best, and
// blocking the refund outright at worst. Ask this instead of naming a provider.
func (o *Order) GatewayRefundable() bool {
	return o.GatewayRefundReference() != ""
}
