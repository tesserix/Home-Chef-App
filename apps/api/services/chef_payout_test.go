package services

import (
	"math"
	"testing"

	"github.com/homechef/api/models"
)

func chefOrder(subtotal, deliveryFee, chefTip float64, ft models.FulfillmentType) *models.Order {
	return &models.Order{
		Subtotal: subtotal, DeliveryFee: deliveryFee, ChefTip: chefTip,
		FulfillmentType: ft, Currency: "INR",
	}
}

// The live order this whole change came from: a ₹320 order the customer paid
// ₹351.97 for. The chef is paid the food, and nothing else.
func TestComputeChefPayout_LiveOrder(t *testing.T) {
	b := ComputeChefPayout(chefOrder(320, 0, 0, models.FulfillmentDelivery), 0)
	if b.NetPayout != 320.00 {
		t.Fatalf("net = %v, want 320.00 (no commission, no GST, no TDS)", b.NetPayout)
	}
	if b.DeliveryFee != 0 {
		t.Fatalf("free-zone delivery must contribute 0, got %v", b.DeliveryFee)
	}
}

// The delivery fee is the chef's ONLY when they carried the leg.
func TestComputeChefPayout_DeliveryFeeOnlyWhenChefCarried(t *testing.T) {
	platform := ComputeChefPayout(chefOrder(320, 39.16, 0, models.FulfillmentDelivery), 0)
	if platform.DeliveryFee != 0 || platform.NetPayout != 320.00 {
		t.Fatalf("platform-carried leg must not pay the chef its fee: %+v", platform)
	}
	chefDrove := ComputeChefPayout(chefOrder(320, 39.16, 0, models.FulfillmentChefDelivery), 0)
	if chefDrove.DeliveryFee != 39.16 || chefDrove.NetPayout != 359.16 {
		t.Fatalf("chef-carried leg must pay the fee: %+v", chefDrove)
	}
	pickup := ComputeChefPayout(chefOrder(320, 0, 0, models.FulfillmentPickup), 0)
	if pickup.DeliveryFee != 0 || pickup.NetPayout != 320.00 {
		t.Fatalf("pickup has no delivery leg: %+v", pickup)
	}
}

// The chef can lower the fee at accept (#703) and the difference is refunded to
// the customer — so the chef is paid what they settled on, not what was charged.
func TestComputeChefPayout_UsesTheFinalDeliveryFee(t *testing.T) {
	o := chefOrder(320, 39.16, 0, models.FulfillmentChefDelivery)
	final := 20.00
	o.DeliveryFeeFinal = &final
	b := ComputeChefPayout(o, 0)
	if b.DeliveryFee != 20.00 || b.NetPayout != 340.00 {
		t.Fatalf("must pay the chef's final fee, not the charged one: %+v", b)
	}
}

func TestComputeChefPayout_TipAndPenalty(t *testing.T) {
	b := ComputeChefPayout(chefOrder(320, 0, 50, models.FulfillmentDelivery), 30)
	if b.ChefTip != 50 || b.Penalty != 30 || b.NetPayout != 340.00 {
		t.Fatalf("320 + 50 tip − 30 penalty = 340: %+v", b)
	}
}

// A penalty bigger than the order cannot make the chef owe money on it.
func TestComputeChefPayout_NeverNegative(t *testing.T) {
	b := ComputeChefPayout(chefOrder(100, 0, 0, models.FulfillmentDelivery), 500)
	if b.NetPayout != 0 {
		t.Fatalf("net = %v, want 0", b.NetPayout)
	}
}

// THE guarantee the owner asked for: the amount paid is exactly the sum of the
// lines shown, to the paise. No independent rounding of the total — that is how
// a receipt comes to out-sum its own lines.
func TestComputeChefPayout_NetIsExactlyTheSumOfItsLines(t *testing.T) {
	cases := [][4]float64{
		{320, 0, 0, 0},
		{320.55, 39.16, 50.45, 0},
		{99.99, 0.01, 0.01, 0},
		{1234.56, 78.91, 23.45, 67.89},
		{0.01, 0.01, 0.01, 0.01},
		{333.33, 66.67, 11.11, 22.22},
	}
	for _, c := range cases {
		o := chefOrder(c[0], c[1], c[2], models.FulfillmentChefDelivery)
		b := ComputeChefPayout(o, c[3])
		want := b.FoodAmount + b.DeliveryFee + b.ChefTip - b.Penalty
		if math.Abs(b.NetPayout-want) > 0.0000001 {
			t.Errorf("net %v != food %v + delivery %v + tip %v − penalty %v (= %v)",
				b.NetPayout, b.FoodAmount, b.DeliveryFee, b.ChefTip, b.Penalty, want)
		}
		// And it must be a clean paise figure — never 320.00000000000006.
		if r := math.Abs(b.NetPayout*100 - math.Round(b.NetPayout*100)); r > 0.0000001 {
			t.Errorf("net %v is not an exact paise amount", b.NetPayout)
		}
	}
}

// Nothing the platform charges may ever reach the chef's payout.
func TestComputeChefPayout_ExcludesPlatformFeeAndTax(t *testing.T) {
	o := chefOrder(320, 0, 0, models.FulfillmentDelivery)
	o.PlatformFee = 13.53 // platform fee — the product owner's cut
	o.Tax = 18.44         // food GST 16.00 + the platform's service GST 2.44
	o.TaxFood = 16.00
	o.TaxService = 2.44
	o.Total = 351.97
	o.CommissionRate = 0.06 // stored, but commission is not deducted

	b := ComputeChefPayout(o, 0)
	if b.NetPayout != 320.00 {
		t.Fatalf("net = %v, want 320.00 — platform fee, GST and commission must not appear", b.NetPayout)
	}
}
