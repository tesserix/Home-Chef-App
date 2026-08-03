package services

import (
	"testing"

	"github.com/homechef/api/models"
)

func TestFormatMoney(t *testing.T) {
	cases := []struct {
		name   string
		amount float64
		want   string
	}{
		// The live defect: a ₹303.48 charge was announced as "₹306" — wrong figure
		// AND rounded. This covers the rounding half.
		{"paise are not rounded away", 303.48, "₹303.48"},
		{"the order total that was wrongly reported", 305.6508, "₹305.65"},
		{"whole rupees carry no decimals", 780, "₹780"},
		{"a trailing zero in the paise still shows two digits", 780.5, "₹780.50"},
		{"single-digit paise pad", 780.05, "₹780.05"},
		{"indian grouping past a thousand", 1234, "₹1,234"},
		{"indian grouping pairs above that", 123456, "₹1,23,456"},
		{"indian grouping with paise", 1121.42, "₹1,121.42"},
		{"zero", 0, "₹0"},
		{"negative refunds keep their sign", -41.07, "-₹41.07"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := FormatMoney(tc.amount); got != tc.want {
				t.Errorf("FormatMoney(%v) = %q, want %q", tc.amount, got, tc.want)
			}
		})
	}
}

// FormatMoney must agree with apps/mobile-customer/lib/format.ts, so the same
// value reads identically in a push and in the app.
func TestFormatMoney_MatchesClientFormatter(t *testing.T) {
	// Values taken from the cancellation drive, with the client's rendering.
	for _, tc := range []struct {
		amount float64
		want   string
	}{
		{377.07, "₹377.07"},
		{343.47, "₹343.47"},
		{41.07, "₹41.07"},
		{132.22, "₹132.22"},
		{16.77, "₹16.77"},
		{393.8424, "₹393.84"},
	} {
		if got := FormatMoney(tc.amount); got != tc.want {
			t.Errorf("FormatMoney(%v) = %q, want %q", tc.amount, got, tc.want)
		}
	}
}

func TestOrderCapturePaise(t *testing.T) {
	cases := []struct {
		name  string
		order models.Order
		want  int
	}{
		{
			// The order behind #934: ₹305.6508 total, ₹2.17 wallet → ₹303.48 charged,
			// reported as ₹306.
			name:  "credit shrinks the capture below the total",
			order: models.Order{Total: 305.6508, WalletApplied: 2.17},
			want:  30348,
		},
		{
			// From the cancellation drive: wallet + loyalty both applied.
			name:  "wallet and loyalty both come off",
			order: models.Order{Total: 393.8424, WalletApplied: 255.30, LoyaltyApplied: 0.45},
			want:  13809,
		},
		{
			name:  "no credit means capture equals total",
			order: models.Order{Total: 592.2735},
			want:  59227,
		},
		{
			name:  "a fully credit-funded order captures nothing",
			order: models.Order{Total: 100, WalletApplied: 100},
			want:  0,
		},
		{
			name:  "credit beyond the total never yields a negative capture",
			order: models.Order{Total: 100, WalletApplied: 150},
			want:  0,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := OrderCapturePaise(&tc.order); got != tc.want {
				t.Errorf("OrderCapturePaise() = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestOrderCapturePaise_NilOrder(t *testing.T) {
	if got := OrderCapturePaise(nil); got != 0 {
		t.Errorf("OrderCapturePaise(nil) = %d, want 0", got)
	}
}
