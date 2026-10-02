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

func TestIsValidPhoneAUNZ(t *testing.T) {
	for _, tc := range []struct {
		country, phone string
		valid          bool
	}{
		{"AU", "412345678", true}, {"AU", "0412345678", false}, {"AU", "9876543210", false},
		{"NZ", "21123456", true}, {"NZ", "2112345678", true}, {"NZ", "021123456", false},
	} {
		if got := IsValidPhone(tc.country, tc.phone); got != tc.valid {
			t.Errorf("%s %s got %v want %v", tc.country, tc.phone, got, tc.valid)
		}
	}
}

func TestCustomerPhoneCountry(t *testing.T) {
	for _, tc := range []struct {
		phoneCountry, addressCountry, want string
		ok                                 bool
	}{
		{"", "", "IN", true},
		{"au", "", "AU", true},
		{"", "NZ", "NZ", true},
		{"NZ", "AU", "NZ", true},
		{"US", "", "", false},
	} {
		got, ok := CustomerPhoneCountry(tc.phoneCountry, tc.addressCountry)
		if got != tc.want || ok != tc.ok {
			t.Errorf("CustomerPhoneCountry(%q,%q) = %q,%v want %q,%v", tc.phoneCountry, tc.addressCountry, got, ok, tc.want, tc.ok)
		}
	}
}

func TestInvalidPhoneMessage(t *testing.T) {
	for country, want := range map[string]string{
		"IN": "Enter a valid 10-digit mobile number",
		"AU": "Enter a valid 9-digit mobile number, without the leading 0",
		"NZ": "Enter a valid NZ mobile number, without the leading 0",
	} {
		if got := InvalidPhoneMessage(country); got != want {
			t.Errorf("InvalidPhoneMessage(%s) = %q want %q", country, got, want)
		}
	}
}
