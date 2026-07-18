package handlers

import (
	"encoding/json"
	"testing"
)

func TestParseIndianAddressTail(t *testing.T) {
	cases := []struct {
		addr             string
		city, reg, postal string
	}{
		{"Asima Residency, Kalarahanga, Bhubaneswar, Odisha 751024", "Bhubaneswar", "Odisha", "751024"},
		{"Kalarahanga, Bhubaneswar, Odisha, 751024", "Bhubaneswar", "Odisha", "751024"},
		{"Bhubaneswar, Odisha", "Bhubaneswar", "Odisha", ""},
		{"Odisha", "", "Odisha", ""},
		{"", "", "", ""},
	}
	for _, tc := range cases {
		city, reg, postal := parseIndianAddressTail(tc.addr)
		if city != tc.city || reg != tc.reg || postal != tc.postal {
			t.Errorf("parseIndianAddressTail(%q) = (%q,%q,%q), want (%q,%q,%q)",
				tc.addr, city, reg, postal, tc.city, tc.reg, tc.postal)
		}
	}
}

func TestFlexFloatUnmarshal(t *testing.T) {
	var row struct {
		Lat flexFloat `json:"lat"`
		Lng flexFloat `json:"lng"`
	}
	// Mappls sends coordinates as strings in some payloads, numbers in others.
	if err := json.Unmarshal([]byte(`{"lat":"20.29","lng":85.82}`), &row); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if float64(row.Lat) != 20.29 || float64(row.Lng) != 85.82 {
		t.Errorf("got (%v,%v), want (20.29,85.82)", float64(row.Lat), float64(row.Lng))
	}
}

func TestMapplsToSuggestion(t *testing.T) {
	loc := mapplsLocation{
		PlaceName:    "Asima Residency",
		PlaceAddress: "Kalarahanga, Bhubaneswar, Odisha 751024",
		Latitude:     20.2961,
		Longitude:    85.8245,
	}
	s := mapplsToSuggestion(loc)
	if s.Lat == 0 || s.Lon == 0 {
		t.Fatalf("expected coordinates, got %+v", s)
	}
	if s.City != "Bhubaneswar" || s.Region != "Odisha" || s.Postal != "751024" {
		t.Errorf("bad parse: city=%q region=%q postal=%q", s.City, s.Region, s.Postal)
	}
	if s.Country != "IN" || s.Line1 != "Asima Residency" {
		t.Errorf("bad mapping: country=%q line1=%q", s.Country, s.Line1)
	}
}
