package services

import (
	"strings"

	"github.com/homechef/api/models"
)

// tip_eligibility.go — can this order actually be tipped, and to whom (#1029).
//
// The tip screen used to be offered on EVERY delivered order. The customer
// picked an amount, committed, and only then got a 409 telling them the money
// could not reach anyone — "This chef's payout account isn't active yet". The
// entry point promised something the platform could not deliver, which is the
// same defect #875 avoided by hiding group orders rather than letting customers
// walk into a checkout that cannot complete.
//
// The preconditions are NOT a property of the chef alone: they depend on the
// gateway the ORDER was paid through, because each rail routes tips differently.
// So eligibility is computed per order, and it lives here — one predicate shared
// by the handler that enforces it and the response that advertises it. If these
// two ever disagreed we would be back to a dead-end CTA, just a subtler one.

// The TipEligibility type itself lives in models so OrderResponse can carry it
// without models importing services (the rule needs the Cashfree constants, so
// the PREDICATE has to live here).

// TipEligibilityFor mirrors, exactly, the guards in TipHandler.CreateOrderTip
// and createCashfreeTip. Order matters: the Cashfree branch is chosen by the
// order's payment provider before any chef check happens.
//
// Requires Chef preloaded, and Delivery + Delivery.DeliveryPartner preloaded to
// judge the rider leg; an unloaded association reads as "not eligible", which
// fails CLOSED (we hide an entry point that might have worked) rather than open
// (we advertise one that cannot).
func TipEligibilityFor(order *models.Order) models.TipEligibility {
	if order == nil {
		return models.TipEligibility{}
	}
	// Only a delivered order is tippable at all.
	if order.Status != models.OrderStatusDelivered {
		return models.TipEligibility{}
	}

	if models.NormalizeProvider(order.PaymentProvider) == models.PaymentProviderCashfree {
		// Easy Split routes a tip to a vendor account. The chef has one; a
		// DeliveryPartner has only a Razorpay linked account, so a third-party
		// delivery still has no rider route. When the chef carried the order
		// themselves the rider IS the chef, so both legs land on the one vendor
		// and stand or fall together (#1080, handlers/tips.go planCashfreeTip).
		vendorOK := order.Chef.CashfreeVendorID != "" &&
			strings.EqualFold(order.Chef.CashfreeVendorStatus, CashfreeVendorActive)
		return models.TipEligibility{
			Chef:  vendorOK,
			Rider: vendorOK && order.FulfillmentType == models.FulfillmentChefDelivery,
		}
	}

	// Razorpay Route: each leg needs its own linked account.
	return models.TipEligibility{
		Chef: order.Chef.RazorpayAccountID != "",
		Rider: order.Delivery != nil &&
			order.Delivery.DeliveryPartnerID != nil &&
			order.Delivery.DeliveryPartner.RazorpayAccountID != "",
	}
}
