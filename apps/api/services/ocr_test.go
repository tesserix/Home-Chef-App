package services

import "testing"

func TestFSSAIRegex(t *testing.T) {
	cases := map[string]string{
		"Lic. No. 12345678901234 issued":            "12345678901234",
		"FSSAI 1 0 0 19 0 22 000123":                "", // spaced digits don't form 14 contiguous
		"No fourteen digit here 1234567":            "",
		"prefix99887766554433 and 9988776655443322": "", // 16 digits, not a clean 14-digit word
	}
	for text, want := range cases {
		got := fssaiRe.FindString(text)
		if got != want {
			t.Errorf("fssai(%q): got %q, want %q", text, got, want)
		}
	}
}

func TestExtractExpiry(t *testing.T) {
	cases := []struct {
		name string
		text string
		want string
	}{
		{"valid-upto numeric", "FSSAI License\nValid Upto 15/06/2027\nIssued 01/06/2025", "2027-06-15"},
		{"validity month-name", "Validity: 09 Aug 2028", "2028-08-09"},
		{"dash format", "Valid till 31-12-2026", "2026-12-31"},
		{"two-digit year", "valid upto 05/03/27", "2027-03-05"},
		{"no keyword picks latest", "Printed 01/01/2024 then 01/01/2029", "2029-01-01"},
		{"impossible date skipped", "valid upto 31/02/2027 and 28/02/2027", "2027-02-28"},
		{"none", "no dates on this page", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := extractExpiry(c.text); got != c.want {
				t.Errorf("extractExpiry: got %q, want %q", got, c.want)
			}
		})
	}
}
