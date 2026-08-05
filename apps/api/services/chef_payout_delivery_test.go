package services

// chef_payout_delivery_test.go — whose money the delivery fee is.
//
// A delivery order is created as plain `delivery` and the carrier is only chosen
// at Mark Ready, so between placing and ready the fulfilment type says nothing
// about who drives. What does say it is where the fee came from: a fee priced
// from the CHEF's own published rates is the chef's to earn, and the vendor app
// must show it while the chef is deciding on the order — not silently omit it
// and then add it after delivery.

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/homechef/api/models"
)

func pendingDeliveryOrder(source string) *models.Order {
	return &models.Order{
		Subtotal: 320, DeliveryFee: 39.15, Currency: "INR",
		FulfillmentType:   models.FulfillmentDelivery,
		Status:            models.OrderStatusPending,
		DeliveryFeeSource: source,
	}
}

func TestComputeChefPayout_ChefPricedFeeIsTheirsBeforeACarrierIsChosen(t *testing.T) {
	b := ComputeChefPayout(pendingDeliveryOrder(models.DeliveryFeeSourceChef), 0)
	require.Equal(t, 39.15, b.DeliveryFee)
	require.Equal(t, 359.15, b.NetPayout)
}

func TestComputeChefPayout_PlatformPricedFeeIsNeverTheChefs(t *testing.T) {
	b := ComputeChefPayout(pendingDeliveryOrder(models.DeliveryFeeSourcePlatform), 0)
	require.Zero(t, b.DeliveryFee)

	b = ComputeChefPayout(pendingDeliveryOrder(models.DeliveryFeeSourceProvider), 0)
	require.Zero(t, b.DeliveryFee)
}

func TestComputeChefPayout_RiderCarriesItOnceTheChefHandsItOver(t *testing.T) {
	// Mark Ready with "hand to a rider" leaves the type `delivery` and moves the
	// order past preparing — the fee stops being the kitchen's from there.
	o := pendingDeliveryOrder(models.DeliveryFeeSourceChef)
	o.Status = models.OrderStatusReady
	require.Zero(t, ComputeChefPayout(o, 0).DeliveryFee)

	// Same order marked ready as "I'll deliver" — the chef keeps it.
	o.FulfillmentType = models.FulfillmentChefDelivery
	require.Equal(t, 39.15, ComputeChefPayout(o, 0).DeliveryFee)
}

func TestComputeChefPayout_OrdersFromBeforeTheSourceColumnAreUnchanged(t *testing.T) {
	o := pendingDeliveryOrder("")
	require.Zero(t, ComputeChefPayout(o, 0).DeliveryFee)
}
