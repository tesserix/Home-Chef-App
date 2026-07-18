package services

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/homechef/api/models"
)

// makeSurgeChef is a self-delivering chef with a distance-based fee (base 20,
// free 2 km, ₹10/km) so surge has a distance component to act on.
func makeSurgeChef(lat, lng float64) models.ChefProfile {
	return models.ChefProfile{
		Latitude: lat, Longitude: lng, OffersSelfDelivery: true,
		SelfDeliveryBaseFee: 20, SelfDeliveryFreeRadiusKm: 2, SelfDeliveryPerKm: 10,
	}
}

// fakeFuel is a test FuelIndexProvider returning a fixed multiplier.
type fakeFuel struct {
	mult float64
	ok   bool
	hits int
}

func (f *fakeFuel) FuelMultiplier(_ context.Context, _ string) (float64, bool) {
	f.hits++
	return f.mult, f.ok
}

const surgeLat, surgeLng = 12.9352, 77.6245

// fakeWeather is a test WeatherProvider returning a fixed multiplier.
type fakeWeather struct {
	mult float64
	ok   bool
	hits int
}

func (w *fakeWeather) WeatherMultiplier(_ context.Context, _, _ float64) (float64, bool) {
	w.hits++
	return w.mult, w.ok
}

// fakeTraffic is a test TrafficProvider returning a fixed multiplier.
type fakeTraffic struct {
	mult float64
	ok   bool
	hits int
}

func (tr *fakeTraffic) TrafficMultiplier(_ context.Context, _, _ float64) (float64, bool) {
	tr.hits++
	return tr.mult, tr.ok
}

func clearSurgeProviders() {
	SetFuelIndexProvider(nil)
	SetWeatherProvider(nil)
	SetTrafficProvider(nil)
}

func TestFuelSurge_DefaultsToNeutralWithoutProvider(t *testing.T) {
	clearSurgeProviders()
	s := CurrentSurge(context.Background(), "IN", surgeLat, surgeLng)
	require.Equal(t, 1.0, s.Fuel, "no provider → no surge")
	require.Equal(t, 1.0, s.Weather)
	require.Equal(t, 1.0, s.Traffic)
	require.Equal(t, 1.0, s.Combined)
}

// Traffic surge combines with the others; heavy traffic raises the estimate. Same
// neutral-default + clamp guarantees.
func TestTrafficSurge_UsesProviderClampsAndCombines(t *testing.T) {
	t.Cleanup(clearSurgeProviders)
	clearSurgeProviders()

	SetTrafficProvider(&fakeTraffic{mult: 1.4, ok: true})
	s := CurrentSurge(context.Background(), "IN", surgeLat, surgeLng)
	require.InDelta(t, 1.4, s.Traffic, 1e-9)
	require.InDelta(t, 1.4, s.Combined, 1e-9)

	// Light traffic never discounts; gridlock is capped.
	SetTrafficProvider(&fakeTraffic{mult: 0.6, ok: true})
	require.Equal(t, 1.0, CurrentSurge(context.Background(), "IN", surgeLat, surgeLng).Traffic)
	SetTrafficProvider(&fakeTraffic{mult: 9, ok: true})
	require.Equal(t, maxSurgeMultiplier, CurrentSurge(context.Background(), "IN", surgeLat, surgeLng).Traffic)

	// A provider that can't answer degrades to neutral.
	SetTrafficProvider(&fakeTraffic{ok: false})
	require.Equal(t, 1.0, CurrentSurge(context.Background(), "IN", surgeLat, surgeLng).Traffic)
}

// Weather surge multiplies alongside fuel; bad weather raises the estimate. Same
// neutral-default + clamp guarantees as fuel.
func TestWeatherSurge_UsesProviderClampsAndCombines(t *testing.T) {
	t.Cleanup(func() { SetFuelIndexProvider(nil); SetWeatherProvider(nil) })
	SetFuelIndexProvider(nil)

	SetWeatherProvider(&fakeWeather{mult: 1.2, ok: true})
	s := CurrentSurge(context.Background(), "IN", surgeLat, surgeLng)
	require.InDelta(t, 1.2, s.Weather, 1e-9)
	require.InDelta(t, 1.2, s.Combined, 1e-9)

	// Good weather never discounts; a storm-sized spike is capped.
	SetWeatherProvider(&fakeWeather{mult: 0.5, ok: true})
	require.Equal(t, 1.0, CurrentSurge(context.Background(), "IN", surgeLat, surgeLng).Weather)
	SetWeatherProvider(&fakeWeather{mult: 5, ok: true})
	require.Equal(t, maxSurgeMultiplier, CurrentSurge(context.Background(), "IN", surgeLat, surgeLng).Weather)

	// Fuel × weather combine, clamped to the overall cap.
	SetFuelIndexProvider(&fakeFuel{mult: 1.5, ok: true})
	SetWeatherProvider(&fakeWeather{mult: 1.5, ok: true})
	require.Equal(t, maxSurgeMultiplier, CurrentSurge(context.Background(), "IN", surgeLat, surgeLng).Combined,
		"1.5×1.5=2.25 clamped to the 2.0 cap")

	// A provider that can't answer degrades to neutral.
	SetWeatherProvider(&fakeWeather{ok: false})
	SetFuelIndexProvider(nil)
	require.Equal(t, 1.0, CurrentSurge(context.Background(), "IN", surgeLat, surgeLng).Weather)
}

func TestFuelSurge_UsesProviderAndClamps(t *testing.T) {
	t.Cleanup(func() { SetFuelIndexProvider(nil) })

	// A provider reporting a 1.3× fuel index surges the combined factor to 1.3.
	SetFuelIndexProvider(&fakeFuel{mult: 1.3, ok: true})
	s := CurrentSurge(context.Background(), "IN", surgeLat, surgeLng)
	require.InDelta(t, 1.3, s.Fuel, 1e-9)
	require.InDelta(t, 1.3, s.Combined, 1e-9)

	// Fuel cheaper than baseline never DISCOUNTS the chef's rate — clamped to 1.0.
	SetFuelIndexProvider(&fakeFuel{mult: 0.7, ok: true})
	require.Equal(t, 1.0, CurrentSurge(context.Background(), "IN", surgeLat, surgeLng).Fuel)

	// An absurd spike is capped so an estimate can't run away.
	SetFuelIndexProvider(&fakeFuel{mult: 99, ok: true})
	require.Equal(t, maxSurgeMultiplier, CurrentSurge(context.Background(), "IN", surgeLat, surgeLng).Fuel)

	// A provider that can't answer degrades to neutral, never an error.
	SetFuelIndexProvider(&fakeFuel{ok: false})
	require.Equal(t, 1.0, CurrentSurge(context.Background(), "IN", surgeLat, surgeLng).Fuel)
}

// The estimate applies surge to the DISTANCE component only (fuel is a driving
// cost, not the flat base), stays capped at the chef's max, and never falls below
// the un-surged charge basis.
func TestEstimateSelfDeliveryFeeBreakdown_AppliesFuelSurge(t *testing.T) {
	t.Cleanup(func() { SetFuelIndexProvider(nil) })
	const chefLat, chefLng = 12.9716, 77.5946
	const dropLat, dropLng = 12.9352, 77.6245

	chef := makeSurgeChef(chefLat, chefLng)

	SetFuelIndexProvider(nil)
	base := EstimateSelfDeliveryFeeBreakdown(context.Background(), chef, dropLat, dropLng, "IN")
	require.Equal(t, 1.0, base.SurgeMultiplier)
	// With no surge the estimate equals the charge basis.
	require.InDelta(t, ComputeSelfDeliveryFee(chef, dropLat, dropLng), base.Fee, 1e-9)

	SetFuelIndexProvider(&fakeFuel{mult: 1.5, ok: true})
	surged := EstimateSelfDeliveryFeeBreakdown(context.Background(), chef, dropLat, dropLng, "IN")
	require.InDelta(t, 1.5, surged.SurgeMultiplier, 1e-9)
	require.InDelta(t, 1.5, surged.FuelSurge, 1e-9)
	// Base fee is unchanged; only the distance component is surged.
	require.Equal(t, base.BaseFee, surged.BaseFee)
	require.InDelta(t, base.DistanceComponent*1.5, surged.DistanceComponent, 1e-6)
	require.Greater(t, surged.Fee, base.Fee, "surge raises the estimate")
}

func TestEstimateSelfDeliveryFeeBreakdown_SurgeStillCapped(t *testing.T) {
	t.Cleanup(func() { SetFuelIndexProvider(nil) })
	const chefLat, chefLng = 12.9716, 77.5946
	const dropLat, dropLng = 12.9352, 77.6245

	// Cap at 50; surge would push way past it → the estimate is still the cap.
	chef := makeSurgeChef(chefLat, chefLng)
	chef.SelfDeliveryMaxFee = 50
	SetFuelIndexProvider(&fakeFuel{mult: 2.0, ok: true})

	b := EstimateSelfDeliveryFeeBreakdown(context.Background(), chef, dropLat, dropLng, "IN")
	require.True(t, b.Capped)
	require.Equal(t, 50.0, b.Fee, "the cap is the true approx-max, surge or not")
}
