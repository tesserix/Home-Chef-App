package services

import "testing"

func TestParsePhotonGeocode_TopResult(t *testing.T) {
	// Photon geometry is [lon, lat].
	body := []byte(`{"features":[{"geometry":{"coordinates":[72.7967,19.1499]}},{"geometry":{"coordinates":[1,2]}}]}`)
	lat, lng, ok := parsePhotonGeocode(body)
	if !ok || lat != 19.1499 || lng != 72.7967 {
		t.Fatalf("want 19.1499,72.7967 ok; got %v,%v ok=%v", lat, lng, ok)
	}
}

func TestParsePhotonGeocode_Empty(t *testing.T) {
	if _, _, ok := parsePhotonGeocode([]byte(`{"features":[]}`)); ok {
		t.Fatal("empty features must return ok=false")
	}
}

func TestGeocodeAddress_BlankQuery(t *testing.T) {
	if _, _, ok := GeocodeAddress("   "); ok {
		t.Fatal("blank query must return ok=false without a network call")
	}
}
