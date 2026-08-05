package services

import (
	"strings"
	"testing"

	"github.com/homechef/api/config"
	"github.com/homechef/api/models"
)

func TestParseFuelPrice_RejectsImplausibleReadings(t *testing.T) {
	// The band is the gate every automatically-obtained number must clear before
	// it can move a customer's fee.
	cases := []struct {
		raw  string
		want bool
	}{
		{"104.21", true},
		{"94.5", true},
		{"49.99", false},   // below any real pump price
		{"201", false},     // above any real pump price
		{"9876543", false}, // a phone number, not a price
		{"", false},
		{"abc", false},
	}
	for _, c := range cases {
		if _, ok := parseFuelPrice(c.raw); ok != c.want {
			t.Errorf("parseFuelPrice(%q) ok = %v, want %v", c.raw, ok, c.want)
		}
	}
}

func TestParseSLMFuelReply(t *testing.T) {
	cases := []struct {
		reply string
		want  float64
		ok    bool
	}{
		{"104.21", 104.21, true},
		{" 98.70 ", 98.70, true},
		{"₹105.40", 105.40, true},
		{"The price is 102.35 per litre", 102.35, true},
		{"NONE", 0, false},
		{"none", 0, false},
		{"", 0, false},
		{"I cannot determine that", 0, false},
		{"5.00", 0, false}, // parses, but outside the pump band
	}
	for _, c := range cases {
		got, ok := parseSLMFuelReply(c.reply)
		if ok != c.ok {
			t.Errorf("parseSLMFuelReply(%q) ok = %v, want %v", c.reply, ok, c.ok)
			continue
		}
		if ok && got != c.want {
			t.Errorf("parseSLMFuelReply(%q) = %v, want %v", c.reply, got, c.want)
		}
	}
}

func TestParseFuelFromMarkup(t *testing.T) {
	page := `<html><body><div class="hero">Petrol price today</div>
	<span>Rs. 103.44</span> per litre in Bengaluru</body></html>`
	got, ok := parseFuelFromMarkup(page)
	if !ok {
		t.Fatal("expected a price from the markup")
	}
	if got != 103.44 {
		t.Fatalf("price = %v, want 103.44", got)
	}
}

// TestParseFuelFromMarkup_RealSourceShape uses the actual shape of the configured
// source (bankbazaar's city petrol table), so a markup change that breaks the
// regex is caught here rather than degrading the fuel factor to neutral in silence.
func TestParseFuelFromMarkup_RealSourceShape(t *testing.T) {
	page := `<div><h2>Petrol Price in Indian Cities today</h2>
	<table><tr><th>City</th><th>Petrol Price</th></tr>
	<tr><td>Agra</td><td>&#8377; 101.54 (0)</td></tr>
	<tr><td>Ahmedabad</td><td>&#8377; 101.83 (0)</td></tr>
	<tr><td>Aligarh</td><td>&#8377; 98 ( 4.02 &#9660; )</td></tr>
	</table></div>`
	got, ok := parseFuelFromMarkup(strings.ReplaceAll(page, "&#8377;", "₹"))
	if !ok {
		t.Fatal("expected a price from the source's table markup")
	}
	if got != 101.54 {
		t.Fatalf("price = %v, want the first city's petrol price 101.54", got)
	}
}

// The day-over-day change column ("( 4.02 ▼ )") sits right beside the price and
// must never be mistaken for one — it would collapse the fuel index to nothing.
func TestParseFuelFromMarkup_IgnoresChangeColumn(t *testing.T) {
	if price, ok := parseFuelFromMarkup(`<td>Aligarh</td><td>( 4.02 ▼ )</td>`); ok {
		t.Fatalf("parsed the change column as a price: %v", price)
	}
}

func TestParseFuelFromMarkup_IgnoresScriptNoise(t *testing.T) {
	// Script bodies carry version numbers and timestamps that look like prices.
	page := `<html><script>var v="1.20.30"; var t=1699999999;</script><body>no price here</body></html>`
	if _, ok := parseFuelFromMarkup(page); ok {
		t.Fatal("expected no price from a page without one")
	}
}

// TestSurgeChargedOnlyWhenEnabled pins the money decision: the same conditions
// must move the charged fee when surge-charging is on, and leave it alone when off.
func TestSurgeChargedOnlyWhenEnabled(t *testing.T) {
	// Rates kept modest so the surged fee stays under the platform's own ceiling
	// (MaxChefDeliveryFee) — this test is about the surge arithmetic, and the
	// ceiling is pinned separately.
	chef := models.ChefProfile{
		Latitude: 12.90, Longitude: 77.50,
		SelfDeliveryBaseFee: 20, SelfDeliveryPerKm: 3, OffersSelfDelivery: true,
	}
	dropLat, dropLng := 12.97, 77.59

	neutral := QuoteOrderDeliveryFeeCtx(chef, models.FulfillmentChefDelivery, dropLat, dropLng, "", "IN", 1.0)
	surged := QuoteOrderDeliveryFeeCtx(chef, models.FulfillmentChefDelivery, dropLat, dropLng, "", "IN", 1.5)

	if surged <= neutral {
		t.Fatalf("surged fee %v should exceed neutral %v", surged, neutral)
	}
	// The flat base is not a driving cost, so only the distance part surges.
	wantSurged := chef.SelfDeliveryBaseFee + (neutral-chef.SelfDeliveryBaseFee)*1.5
	if diff := surged - wantSurged; diff > 0.01 || diff < -0.01 {
		t.Fatalf("surged fee = %v, want %v (surge on distance only)", surged, wantSurged)
	}
}

func TestSurgeNeverExceedsChefMaxFee(t *testing.T) {
	chef := models.ChefProfile{
		Latitude: 12.90, Longitude: 77.50,
		SelfDeliveryBaseFee: 20, SelfDeliveryPerKm: 10,
		SelfDeliveryMaxFee: 60, OffersSelfDelivery: true,
	}
	// An extreme multiplier must still be bounded by the chef's own cap, so a bad
	// signal cannot run away with a customer's money.
	fee := QuoteOrderDeliveryFeeCtx(chef, models.FulfillmentChefDelivery, 12.97, 77.59, "", "IN", maxSurgeMultiplier)
	if fee > 60 {
		t.Fatalf("fee %v exceeded the chef's max of 60", fee)
	}
}

// TestFreeZoneCostsNothing pins the rule the customer is promised: inside the
// chef's free-delivery radius there is NO delivery charge at all — not the
// distance component, not the flat base, and no surge on top.
func TestFreeZoneCostsNothing(t *testing.T) {
	chef := models.ChefProfile{
		Latitude: 12.9716, Longitude: 77.5946,
		SelfDeliveryBaseFee: 40, SelfDeliveryPerKm: 15,
		SelfDeliveryFreeRadiusKm: 50, OffersSelfDelivery: true,
	}
	// A drop ~5 km away, well inside the 50 km free radius.
	b := SelfDeliveryBreakdownAt(chef, 12.9352, 77.6245, NeutralSurge(), maxSurgeMultiplier)

	if !b.WithinFreeZone {
		t.Fatal("expected the drop to be inside the free radius")
	}
	if b.Fee != 0 {
		t.Fatalf("free-zone fee = %v, want 0 (base fee must be waived too)", b.Fee)
	}
}

// A chef who set no free radius has no free zone, so a zero-distance drop still
// pays the flat base rather than being treated as free.
func TestNoFreeRadiusIsNotAFreeZone(t *testing.T) {
	chef := models.ChefProfile{
		Latitude: 18.5, Longitude: 73.8,
		SelfDeliveryBaseFee: 30, SelfDeliveryFreeRadiusKm: 0,
	}
	b := SelfDeliveryBreakdownAt(chef, 18.5, 73.8, NeutralSurge(), 1.0)
	if b.WithinFreeZone {
		t.Fatal("a chef with no free radius must not report a free zone")
	}
	if b.Fee != 30 {
		t.Fatalf("fee = %v, want the flat base 30", b.Fee)
	}
}

func TestResolveChargeSurge_NeutralWhenChargingDisabled(t *testing.T) {
	prev := config.AppConfig
	t.Cleanup(func() { config.AppConfig = prev })
	config.AppConfig = &config.Config{DeliverySurgeChargeEnabled: false}

	chef := models.ChefProfile{}
	if got := ResolveChargeSurge(t.Context(), "", chef, 12.97, 77.59, "IN"); got != 1.0 {
		t.Fatalf("charge surge = %v, want 1.0 while disabled", got)
	}
}
