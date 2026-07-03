package services

import "testing"

// TestParseCommissionRate covers the pure parse+validate layer that backs the
// runtime `payout.commission_rate` override. Only 0 < rate < 1 is accepted; any
// invalid/out-of-range value reports ok=false so GetCommissionRate falls back to
// the flat DefaultCommissionRate — a bad admin value never charges a nonsense rate.
func TestParseCommissionRate(t *testing.T) {
	tests := []struct {
		name   string
		raw    string
		want   float64
		wantOK bool
	}{
		{"six percent", "0.06", 0.06, true},
		{"eight percent override", "0.08", 0.08, true},
		{"empty", "", 0, false},
		{"zero", "0", 0, false},
		{"one", "1", 0, false},
		{"above one", "1.5", 0, false},
		{"non-numeric", "abc", 0, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := parseCommissionRate(tc.raw)
			if ok != tc.wantOK {
				t.Fatalf("parseCommissionRate(%q) ok = %v, want %v", tc.raw, ok, tc.wantOK)
			}
			if tc.wantOK && got != tc.want {
				t.Errorf("parseCommissionRate(%q) = %v, want %v", tc.raw, got, tc.want)
			}
		})
	}
}
