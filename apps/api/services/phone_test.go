package services

import "testing"

func TestIsValidPhone(t *testing.T) {
	valid := []string{"9876543210", "6000000000", "7123456789", "8999999999"}
	invalid := []string{"", "1234567890", "98765", "98765432100", "9876abc210", "0987654321", "5876543210"}
	for _, p := range valid {
		if !IsValidPhone("IN", p) {
			t.Errorf("IsValidPhone(IN,%q) = false, want true", p)
		}
	}
	for _, p := range invalid {
		if IsValidPhone("IN", p) {
			t.Errorf("IsValidPhone(IN,%q) = true, want false", p)
		}
	}
	// Unknown country falls back to the India rule.
	if !IsValidPhone("ZZ", "9876543210") {
		t.Error("unknown country should fall back to IN rule")
	}
}
