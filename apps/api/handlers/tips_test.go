package handlers

import "testing"

// Unit tests for the post-delivery tip amount validation (#45).

func TestValidateTipAmounts(t *testing.T) {
	cases := []struct {
		name        string
		chef, rider float64
		ok          bool
	}{
		{"chef only", 50, 0, true},
		{"rider only", 0, 20, true},
		{"both", 50, 30, true},
		{"at minimum", 1, 0, true},
		{"below minimum", 0.5, 0, false},
		{"zero", 0, 0, false},
		{"negative chef", -10, 50, false},
		{"negative rider", 50, -10, false},
		{"at cap", 5000, 0, true},
		{"over cap", 4000, 2000, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateTipAmounts(tc.chef, tc.rider)
			if tc.ok && err != nil {
				t.Fatalf("expected valid (%v,%v), got error: %v", tc.chef, tc.rider, err)
			}
			if !tc.ok && err == nil {
				t.Fatalf("expected invalid (%v,%v), got no error", tc.chef, tc.rider)
			}
		})
	}
}
