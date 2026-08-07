package services

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/homechef/api/models"
)

// #1029 — the app offered "Tip chef" on every delivered order and only found out
// at submit that the money could not reach anyone. These pin the predicate that
// now gates the entry point to the SAME rules the tip handler enforces.

func deliveredOrder(provider string) *models.Order {
	return &models.Order{
		Status:          models.OrderStatusDelivered,
		PaymentProvider: provider,
	}
}

func withRider(o *models.Order, acct string) *models.Order {
	id := uuid.New()
	o.Delivery = &models.Delivery{DeliveryPartnerID: &id}
	o.Delivery.DeliveryPartner.RazorpayAccountID = acct
	return o
}

func TestTipEligibility_CashfreeNeedsAnActiveVendor(t *testing.T) {
	// The exact production state behind #1029: a delivered Cashfree order whose
	// chef has no Easy Split vendor registration.
	o := deliveredOrder(string(models.PaymentProviderCashfree))
	require.False(t, TipEligibilityFor(o).Chef, "no vendor id must not advertise a chef tip")
	require.False(t, TipEligibilityFor(o).Any())

	o.Chef.CashfreeVendorID = "vend_1"
	o.Chef.CashfreeVendorStatus = "PENDING"
	require.False(t, TipEligibilityFor(o).Chef, "a registered-but-inactive vendor still cannot be paid")

	o.Chef.CashfreeVendorStatus = CashfreeVendorActive
	require.True(t, TipEligibilityFor(o).Chef)
	require.True(t, TipEligibilityFor(o).Any())
}

func TestTipEligibility_CashfreeStatusIsCaseInsensitive(t *testing.T) {
	o := deliveredOrder(string(models.PaymentProviderCashfree))
	o.Chef.CashfreeVendorID = "vend_1"
	o.Chef.CashfreeVendorStatus = "active"
	require.True(t, TipEligibilityFor(o).Chef, "handler uses EqualFold; the flag must too")
}

func TestTipEligibility_CashfreeOffersNoRiderTipOnAThirdPartyDelivery(t *testing.T) {
	// A DeliveryPartner carries only a Razorpay linked account, so Easy Split has
	// no route to them — the screen must not show the rider section.
	o := withRider(deliveredOrder(string(models.PaymentProviderCashfree)), "acc_rider")
	o.Chef.CashfreeVendorID = "vend_1"
	o.Chef.CashfreeVendorStatus = CashfreeVendorActive
	require.True(t, TipEligibilityFor(o).Chef)
	require.False(t, TipEligibilityFor(o).Rider)
}

// #1080 — the chef delivers their own orders today, so the rider leg has a real
// destination: the chef's own vendor account. Hiding it advertised less than the
// platform can do; showing it when the handler refuses is the #1029 dead end.
func TestTipEligibility_CashfreeOffersARiderTipWhenTheChefDelivered(t *testing.T) {
	o := deliveredOrder(string(models.PaymentProviderCashfree))
	o.FulfillmentType = models.FulfillmentChefDelivery
	o.Chef.CashfreeVendorID = "vend_1"
	o.Chef.CashfreeVendorStatus = CashfreeVendorActive

	require.True(t, TipEligibilityFor(o).Rider)
	require.True(t, TipEligibilityFor(o).Chef)
}

// The rider leg is the chef's vendor, so a dormant vendor kills both legs — the
// screen must not offer a rider tip the handler will refuse.
func TestTipEligibility_ChefDeliveredRiderLegNeedsTheVendorToo(t *testing.T) {
	o := deliveredOrder(string(models.PaymentProviderCashfree))
	o.FulfillmentType = models.FulfillmentChefDelivery
	o.Chef.CashfreeVendorID = "vend_1"
	o.Chef.CashfreeVendorStatus = "PENDING"

	require.False(t, TipEligibilityFor(o).Rider)
	require.False(t, TipEligibilityFor(o).Chef)
}

// A pickup order has no delivery leg at all, so there is nobody to tip as rider.
func TestTipEligibility_PickupOffersNoRiderTip(t *testing.T) {
	o := deliveredOrder(string(models.PaymentProviderCashfree))
	o.FulfillmentType = models.FulfillmentPickup
	o.Chef.CashfreeVendorID = "vend_1"
	o.Chef.CashfreeVendorStatus = CashfreeVendorActive

	require.False(t, TipEligibilityFor(o).Rider)
	require.True(t, TipEligibilityFor(o).Chef)
}

// A tip is a NEW charge, minted on Cashfree whatever gateway the order it thanks
// was stamped with (#1103). So a historical Razorpay order is judged by the chef's
// Easy Split vendor, which a chef left on the old rail does not have.
func TestTipEligibility_RazorpayOrderIsJudgedByTheCashfreeVendor(t *testing.T) {
	o := deliveredOrder(string(models.PaymentProviderRazorpay))
	withRider(o, "acc_rider")
	require.False(t, TipEligibilityFor(o).Chef, "no vendor, no Cashfree tip")
	require.False(t, TipEligibilityFor(o).Rider)

	o.Chef.CashfreeVendorID = "vend_1"
	o.Chef.CashfreeVendorStatus = CashfreeVendorActive
	require.True(t, TipEligibilityFor(o).Chef, "the chef's vendor is what the tip actually reaches")
	require.False(t, TipEligibilityFor(o).Rider, "a third-party rider still has no Easy Split route")
}

func TestTipEligibility_OnlyDeliveredOrdersAreTippable(t *testing.T) {
	// Mirrors the handler's first guard: "You can only tip after the order is
	// delivered". An in-flight order must not show the entry point at all.
	for _, st := range []models.OrderStatus{
		models.OrderStatusPending,
		models.OrderStatusPreparing,
		models.OrderStatusDelivering,
		models.OrderStatusCancelled,
		models.OrderStatusRefunded,
	} {
		o := deliveredOrder(string(models.PaymentProviderCashfree))
		o.Status = st
		o.FulfillmentType = models.FulfillmentChefDelivery
		o.Chef.CashfreeVendorID = "vend_1"
		o.Chef.CashfreeVendorStatus = CashfreeVendorActive
		require.False(t, TipEligibilityFor(o).Any(), "status %s must not be tippable", st)
	}
}

func TestTipEligibility_NilOrderIsNotTippable(t *testing.T) {
	require.False(t, TipEligibilityFor(nil).Any())
}
