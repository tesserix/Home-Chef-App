package services

// driver_tip_payout_test.go — #1081. The checkout-time DriverTip reached nobody
// on a chef-delivered order.
//
// order_payment_settle.go books DeliveryFee + DriverTip to the delivery
// partner's account. On a chef_delivery order there is no delivery partner, so
// that leg is dropped — and ChefNetPayoutFor never included the tip, so the
// money the customer paid to thank their driver was captured by the platform
// and settled to no one.

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/homechef/api/models"
)

func tippedOrder(ft models.FulfillmentType, driverTip float64) *models.Order {
	o := &models.Order{
		Subtotal:             1000,
		Tax:                  50,
		TaxFood:              50,
		DeliveryFee:          40,
		ChefTip:              30,
		DriverTip:            driverTip,
		CommissionRate:       0.15,
		FulfillmentType:      ft,
		Status:               models.OrderStatusDelivered,
		DeliveryAddressState: "Karnataka",
	}
	o.Chef.State = "Karnataka"
	return o
}

// The chef carried the leg, so the tip meant for the driver is the chef's.
func TestChefNetPayoutFor_IncludesTheDriverTipWhenTheChefDelivered(t *testing.T) {
	without := ChefNetPayoutFor(tippedOrder(models.FulfillmentChefDelivery, 0))
	with := ChefNetPayoutFor(tippedOrder(models.FulfillmentChefDelivery, 25))

	require.Greater(t, with, without, "a driver tip on a chef-delivered order must reach the chef")
	require.InDelta(t, 25.0*(1-RateTDS), with-without, 0.001,
		"the tip reaches the chef exactly as a chef tip does — no commission, TDS base")
}

// A chef tip and a driver tip on a chef-delivered order are the same money from
// the same customer to the same person, so they must settle identically. If they
// ever diverge, a chef's statement carries a difference nobody can explain.
func TestChefNetPayoutFor_TreatsBothTipsAlikeWhenTheChefDelivered(t *testing.T) {
	chefTipped := tippedOrder(models.FulfillmentChefDelivery, 0)
	chefTipped.ChefTip += 25
	driverTipped := tippedOrder(models.FulfillmentChefDelivery, 25)

	require.Equal(t, ChefNetPayoutFor(chefTipped), ChefNetPayoutFor(driverTipped))
}

// A third party carried it, so the tip is theirs and must not follow the food.
func TestChefNetPayoutFor_ExcludesTheDriverTipOnAThirdPartyDelivery(t *testing.T) {
	for _, ft := range []models.FulfillmentType{models.FulfillmentDelivery, models.FulfillmentPickup, ""} {
		without := ChefNetPayoutFor(tippedOrder(ft, 0))
		with := ChefNetPayoutFor(tippedOrder(ft, 25))
		require.Equal(t, without, with, "fulfilment %q must not pay the driver tip to the chef", ft)
	}
}

// Commission is on food revenue only. It must not start applying to a tip
// because the tip entered the chef's share.
func TestComputeOrderEarnings_DriverTipCarriesNoCommission(t *testing.T) {
	base := EarningsInput{
		ItemRevenue: 1000, Tax: 50, ChefTip: 30, DeliveryFee: 40,
		ChefEarnsDeliveryFee: true, CommissionRate: 0.15, DeliveryState: "Karnataka",
	}
	tipped := base
	tipped.DriverTip = 25

	plain := ComputeOrderEarnings(base, "Karnataka")
	withTip := ComputeOrderEarnings(tipped, "Karnataka")

	require.Equal(t, plain.PlatformCommission, withTip.PlatformCommission)
	require.Equal(t, plain.CGST, withTip.CGST)
	require.Equal(t, plain.SGST, withTip.SGST)
	require.InDelta(t, 25.0, withTip.Gross-plain.Gross, 0.001)
}

// A chef-delivered order has no delivery partner, so nothing must be booked to
// one — and the fee and tip the chef already earned must not be paid twice.
func TestOrderSettlements_DriverLegIsEmptyWhenTheChefDelivered(t *testing.T) {
	db := setupOrderSettlementsDB(t)
	order := tippedOrder(models.FulfillmentChefDelivery, 25)
	order.OrderNumber = "HC-DT1"
	order.Chef.RazorpayAccountID = "acc_chef"

	settlements := OrderSettlements(db, order)

	require.Len(t, settlements, 2)
	require.Equal(t, "acc_chef", settlements[0].Account)
	require.Zero(t, settlements[1].Amount,
		"the chef already earned the fee and the tip; booking them again pays twice")
}

// A 3PL leg is unchanged: the driver is still paid their fee and their tip.
func TestOrderSettlements_DriverLegKeepsTheTipOnAThirdPartyDelivery(t *testing.T) {
	db := setupOrderSettlementsDB(t)
	order := tippedOrder(models.FulfillmentDelivery, 25)
	order.OrderNumber = "HC-DT2"
	order.Chef.RazorpayAccountID = "acc_chef"
	order.Delivery = &models.Delivery{}
	order.Delivery.DeliveryPartner.RazorpayAccountID = "acc_driver"

	settlements := OrderSettlements(db, order)

	require.Equal(t, "acc_driver", settlements[1].Account)
	require.Equal(t, ToPaise(65), settlements[1].Amount, "fee ₹40 + tip ₹25")
}

// Money conservation, in paise: every rupee the customer paid is accounted for
// by exactly one party. A driver tip that reaches neither the chef nor a driver
// is the defect this issue names.
func TestDriverTip_IsConservedAcrossTheLegs(t *testing.T) {
	db := setupOrderSettlementsDB(t)
	order := tippedOrder(models.FulfillmentChefDelivery, 25)
	order.OrderNumber = "HC-DT3"
	order.Chef.RazorpayAccountID = "acc_chef"

	settlements := OrderSettlements(db, order)
	earnings := ComputeOrderEarnings(EarningsInput{
		ItemRevenue: order.Subtotal, Tax: ChefAttributableTax(order), ChefTip: order.ChefTip,
		DriverTip: order.DriverTip, DeliveryFee: order.EffectiveDeliveryFee(),
		ChefEarnsDeliveryFee: order.ChefEarnsDeliveryFee(), CommissionRate: order.CommissionRate,
		DeliveryState: order.DeliveryAddressState,
	}, order.Chef.State)

	chefPaise := settlements[0].Amount + settlements[1].Amount
	platformPaise := ToPaise(earnings.PlatformCommission) + ToPaise(earnings.TDS)
	tippedTotal := order.Subtotal + ChefAttributableTax(order) + order.ChefTip +
		order.EffectiveDeliveryFee() + order.DriverTip

	require.Equal(t, ToPaise(tippedTotal), chefPaise+platformPaise,
		"chef share + platform share must exhaust what the customer paid")
}

// The four consumers of ChefNetPayoutFor must agree on the driver tip, or the
// figure on a chef's statement will not be the figure that was transferred.
func TestDriverTip_StatementAndPayoutAgree(t *testing.T) {
	order := tippedOrder(models.FulfillmentChefDelivery, 25)

	row := statementOrderRow{
		ItemRevenue: order.Subtotal, Tax: order.Tax, TaxFood: order.TaxFood,
		DeliveryFee: order.DeliveryFee, ChefTip: order.ChefTip, DriverTip: order.DriverTip,
		DeliveryState: order.DeliveryAddressState, CommissionRate: order.CommissionRate,
		ChefState: order.Chef.State, FulfillmentType: string(models.FulfillmentChefDelivery),
	}
	fromStatement := ComputeOrderEarnings(row.earningsInput(DefaultCommissionRate), row.ChefState).NetPayout

	require.Equal(t, ChefNetPayoutFor(order), fromStatement,
		"the weekly statement and the payout must read one number")
}
