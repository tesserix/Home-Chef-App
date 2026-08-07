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
//     which reads an empty GatewayPaymentID and fails at the gateway.
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
	// the platform merchant account; the chef's share is split on release via
	// Easy Split (ADR-0003) and the remainder settles on the statement/payout
	// path. Nothing is split at capture.
	PaymentProviderCashfree = "cashfree"

	// PaymentProviderStripe — international. Connect destination charges.
	PaymentProviderStripe = "stripe"

	// PaymentProviderWallet — no gateway at all: the order was fully covered by
	// store credit, so there is nothing to capture and nothing to refund
	// externally. Never a chef's configured provider; only ever stamped on an
	// order at settlement.
	PaymentProviderWallet = "wallet"
)

// PreferredChefPaymentProvider is the gateway a NEW chef is created with. Since
// #1086 it is also what an unstamped value normalizes to — there is one INR
// gateway left, so the two questions have the same answer.
const PreferredChefPaymentProvider = PaymentProviderCashfree

// NormalizeProvider coerces a stored or user-supplied provider to a known value.
//
// Razorpay is now a RECOGNISED case rather than the catch-all (#1086). That order
// matters: a historical razorpay row must keep saying razorpay, because its refund
// has to go back to the gateway that took the money — while an empty or garbled
// value resolves to cashfree, the only INR gateway the platform still operates.
//
// The old fallback existed for rows predating the payment_provider column. There
// are none left: every provider column in production carries an explicit value.
func NormalizeProvider(p string) string {
	switch strings.ToLower(strings.TrimSpace(p)) {
	case PaymentProviderRazorpay:
		return PaymentProviderRazorpay
	case PaymentProviderStripe:
		return PaymentProviderStripe
	case PaymentProviderWallet:
		return PaymentProviderWallet
	default:
		return PaymentProviderCashfree
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
// fully-credit-covered order, never a configuration. Razorpay is excluded since
// #1086: it can still be READ off a historical order, but nothing may be newly
// configured onto it.
func SelectableChefProviders() []string {
	return []string{PaymentProviderCashfree, PaymentProviderStripe}
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

// GatewayOrderIDColumn / GatewayPaymentIDColumn name the one pair of columns
// every gateway's ids land in.
//
// Order.GatewayOrderID / .GatewayPaymentID (and MealPlan, GroupOrder, Tip,
// Catering, FeaturedListing) hold the GATEWAY's order and payment ids —
// whichever gateway that is; PaymentProvider says whose id it is.
//
// Sharing one pair is the point. They carry partial unique indexes (WHERE col
// <> '', see database.go's postMigrate) and are read by the reconciliation
// cron, the meal-plan advance lookup, the group-order settle path and the
// escrow ledger. A parallel cashfree_order_id on six models would fork every
// one of those queries — and a fork is precisely how a payment goes missing.
const (
	GatewayOrderIDColumn   = "gateway_order_id"
	GatewayPaymentIDColumn = "gateway_payment_id"
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
		return o.GatewayOrderID
	case PaymentProviderStripe:
		return o.StripePaymentIntentID
	case PaymentProviderWallet:
		return ""
	default:
		return o.GatewayPaymentID
	}
}

// GatewayRefundable reports whether a refund can actually be issued to this
// order's original payment method.
//
// This replaces the `order.PaymentProvider != "razorpay" || order.GatewayPaymentID == ""`
// guard that appeared at five call sites. That guard was correct when Razorpay
// was the only INR gateway and actively wrong the moment a second one existed: a
// Cashfree order has a perfectly refundable payment, and the old test would have
// answered "no" — routing the customer's money to store credit at best, and
// blocking the refund outright at worst. Ask this instead of naming a provider.
func (o *Order) GatewayRefundable() bool {
	return o.GatewayRefundReference() != ""
}
