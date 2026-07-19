package services

import (
	"context"
	"errors"
	"testing"
)

func TestIsValidEmailFormat(t *testing.T) {
	valid := []string{"a@b.co", "Chef.Name@example.com", "  user@domain.io  "}
	for _, e := range valid {
		if !IsValidEmailFormat(e) {
			t.Errorf("expected %q valid", e)
		}
	}
	invalid := []string{"", "nope", "no@domain", "@domain.com", "user@.com", "a b@c.com", "x@y.z"}
	for _, e := range invalid {
		if IsValidEmailFormat(e) {
			t.Errorf("expected %q invalid", e)
		}
	}
}

func TestNormalizeEmail(t *testing.T) {
	if got := NormalizeEmail("  Foo@Bar.COM "); got != "foo@bar.com" {
		t.Errorf("got %q", got)
	}
}

func TestGenerateOTP(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 200; i++ {
		c, err := generateOTP()
		if err != nil {
			t.Fatal(err)
		}
		if len(c) != 6 {
			t.Fatalf("want 6 digits, got %q", c)
		}
		for _, r := range c {
			if r < '0' || r > '9' {
				t.Fatalf("non-digit in %q", c)
			}
		}
		seen[c] = true
	}
	if len(seen) < 100 {
		t.Errorf("suspiciously low entropy: %d distinct of 200", len(seen))
	}
}

// Redis is not connected in unit tests: request rejects bad email before any
// Redis call, rejects valid email as unavailable, and the gate fails open.
func TestEmailOTP_NoRedis(t *testing.T) {
	ctx := context.Background()
	if err := RequestEmailOTP(ctx, "u1", "bad-email", "Sam"); !errors.Is(err, ErrOTPInvalidEmail) {
		t.Errorf("want ErrOTPInvalidEmail, got %v", err)
	}
	if err := RequestEmailOTP(ctx, "u1", "sam@fe3dr.com", "Sam"); !errors.Is(err, ErrOTPUnavailable) {
		t.Errorf("want ErrOTPUnavailable, got %v", err)
	}
	if err := VerifyEmailOTP(ctx, "u1", "sam@fe3dr.com", "123456"); !errors.Is(err, ErrOTPUnavailable) {
		t.Errorf("want ErrOTPUnavailable, got %v", err)
	}
	if !IsEmailOTPVerified(ctx, "u1", "sam@fe3dr.com") {
		t.Error("gate must fail open when Redis is unavailable")
	}
}
