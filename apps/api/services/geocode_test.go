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

// The regression that hid every kitchen: Photon returns ZERO features for a full
// Indian address carrying a flat number, so a single-shot lookup left lat/lng at
// 0 and the bounding-box filter in GetChefs dropped the chef for every located
// customer. The unit-less form MUST be attempted, and the full form must still be
// tried first so a resolvable precise address wins.
func TestGeocodeCandidates_DropsUnitLine(t *testing.T) {
	got := geocodeCandidates("F 104 Ashima Residency", "Kalarahanga, Patia", "Bhubaneswar", "Odisha", "751024")

	if len(got) == 0 {
		t.Fatal("expected candidates")
	}
	if want := "F 104 Ashima Residency, Kalarahanga, Patia, Bhubaneswar, Odisha, 751024"; got[0] != want {
		t.Fatalf("most precise form must be tried first\n got %q\nwant %q", got[0], want)
	}
	want := "Kalarahanga, Patia, Bhubaneswar, Odisha, 751024"
	for _, q := range got {
		if q == want {
			return
		}
	}
	t.Fatalf("no candidate dropped the unit line; want %q among %q", want, got)
}

// A sparse address must not produce blank or duplicate lookups — each would be a
// wasted round-trip on the boot backfill's hot path.
func TestGeocodeCandidates_NoBlanksOrDuplicates(t *testing.T) {
	got := geocodeCandidates("", "", "Bhubaneswar", "Odisha", "")

	seen := map[string]bool{}
	for _, q := range got {
		if q == "" {
			t.Fatalf("blank candidate in %q", got)
		}
		if seen[q] {
			t.Fatalf("duplicate candidate %q in %q", q, got)
		}
		seen[q] = true
	}
	if len(got) != 1 || got[0] != "Bhubaneswar, Odisha" {
		t.Fatalf("want exactly [Bhubaneswar, Odisha]; got %q", got)
	}
}
