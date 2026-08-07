package models

import (
	"slices"
	"strings"
)

// payment_provider.go — the payment-gateway vocabulary.
//
// The provider was once a bare string compared inline at ~40 sites, in three
// different ways: `== "<gateway>"`, `EqualFold(p, "stripe")`, and
// `switch p { … default: /* the INR one */ }`. Two of those shapes are traps the
// moment a second INR gateway exists: a guard that names one gateway silently
// stops refunding orders taken on the other, and a `default` arm sends them down
// a branch that reads an id the gateway never issued.
//
// So the provider string is a closed vocabulary with predicates that say what a
// provider CAN DO rather than which one it IS. Call sites ask "does this order
// settle through a gateway?" instead of naming one, and adding a gateway becomes
// a change to this file plus one adapter.
const (
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

// PreferredChefPaymentProvider is the gateway a NEW chef is created with, and
// what an unstamped value normalizes to — there is one INR gateway, so the two
// questions have the same answer.
const PreferredChefPaymentProvider = PaymentProviderCashfree

// NormalizeProvider coerces a stored or user-supplied provider to a known value.
//
// Rows stamped with the retired INR gateway keep that string in the database —
// it is the factual record of which rail took the money (#1122) — but it is no
// longer vocabulary here (#1132). Nothing in this codebase can call that gateway,
// so recognising the name would only let one of its ids be presented to Cashfree.
// It normalizes to cashfree like any other unknown value, and the refund path
// then finds no cashfree order id and settles to store credit.
func NormalizeProvider(p string) string {
	switch strings.ToLower(strings.TrimSpace(p)) {
	case PaymentProviderStripe:
		return PaymentProviderStripe
	case PaymentProviderWallet:
		return PaymentProviderWallet
	default:
		return PaymentProviderCashfree
	}
}

// IsKnownProvider reports whether the STORED string names a provider this
// codebase can still act on.
//
// NormalizeProvider coerces, which is right for reading a row and wrong for
// deciding whether to act on one: coercion alone would send an order from the
// retired INR gateway into the Cashfree verify leg carrying an id Cashfree never
// issued. Ask this before acting; ask NormalizeProvider to interpret (#1132).
func IsKnownProvider(p string) bool {
	switch strings.ToLower(strings.TrimSpace(p)) {
	case PaymentProviderCashfree, PaymentProviderStripe, PaymentProviderWallet:
		return true
	}
	return false
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
	return []string{PaymentProviderCashfree, PaymentProviderStripe}
}

// IsSelectableChefProvider validates a provider submitted by an admin or chef.
// Deliberately strict — unlike NormalizeProvider it does NOT coerce, because
// silently saving "cashfre" as cashfree would hide a typo the operator needs
// to see.
func IsSelectableChefProvider(p string) bool {
	return slices.Contains(SelectableChefProviders(), strings.ToLower(strings.TrimSpace(p)))
}

// UsesINRPaise reports whether this provider's money is denominated in INR and
// converted through the paise helpers (ToPaise/FromPaise), as opposed to
// Stripe's currency-aware minor units (ToMinor/FromMinor, where the divisor is
// 1, 100 or 1000 depending on the currency).
//
// This is the predicate that `inr := !EqualFold(provider, "stripe")` was really
// reaching for. Cashfree is INR, so it answers true — but note that Cashfree's
// WIRE format is rupees-as-decimal, not paise; that conversion lives in the
// Cashfree client, not here.
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
// <> ”, see database.go's postMigrate) and are read by the reconciliation
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
//   - Cashfree refunds an ORDER: POST /orders/{order_id}/refunds. Its payment id
//     exists, but no refund endpoint takes it.
//   - Stripe refunds a PaymentIntent.
//   - Wallet has nothing at a gateway; a wallet "refund" is a ledger credit.
//
// A row from the retired INR gateway normalizes to cashfree and carries no
// cashfree order id, so it answers "" — the refund settles to store credit
// rather than presenting a foreign id to Cashfree (#1132).
//
// Callers that only need the yes/no should use GatewayRefundable.
func (o *Order) GatewayRefundReference() string {
	if o == nil {
		return ""
	}
	switch NormalizeProvider(o.PaymentProvider) {
	case PaymentProviderStripe:
		return o.StripePaymentIntentID
	case PaymentProviderWallet:
		return ""
	default:
		return o.GatewayOrderID
	}
}

// GatewayRefundable reports whether a refund can actually be issued to this
// order's original payment method.
//
// This replaces the `order.PaymentProvider != "<gateway>" || order.GatewayPaymentID == ""`
// guard that appeared at five call sites. It was correct while one INR gateway
// existed and actively wrong the moment a second one did: a Cashfree order has a
// perfectly refundable payment, and the old test answered "no" — routing the
// customer's money to store credit at best, blocking the refund at worst. Ask
// this instead of naming a provider.
func (o *Order) GatewayRefundable() bool {
	return o.GatewayRefundReference() != ""
}
