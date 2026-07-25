package services

import (
	"context"
	"errors"
	"testing"
	"time"
)

// The reason this file exists: onboarding writes a long-lived "verified"
// marker, and if a login challenge could read it, every user who verified their
// email at signup would auto-pass 2FA forever. Purpose isolation is the whole
// security property, so it is the first thing tested.

func TestOTPPurposeIsolation_OnboardingCannotSatisfyLogin(t *testing.T) {
	mr := withMiniredis(t)
	ctx := context.Background()
	uid, email := "user-1", "chef@fe3dr.com"

	// Verify through the legacy onboarding path.
	if err := RequestEmailOTP(ctx, uid, email, "Chef"); err != nil {
		t.Fatalf("onboarding request: %v", err)
	}
	code, err := mr.Get(otpCodeKey(uid, email))
	if err != nil {
		t.Fatalf("onboarding code missing: %v", err)
	}
	if err := VerifyEmailOTP(ctx, uid, email, code); err != nil {
		t.Fatalf("onboarding verify: %v", err)
	}
	if !IsEmailOTPVerified(ctx, uid, email) {
		t.Fatal("onboarding should be verified")
	}

	// That marker must be invisible to a login challenge.
	if OTPVerified(ctx, PurposeMFALogin, uid, email) {
		t.Fatal("onboarding verification satisfied an MFA login challenge")
	}
	if OTPVerified(ctx, PurposeMFAEnroll, uid, email) {
		t.Fatal("onboarding verification satisfied an MFA enrollment challenge")
	}
}

func TestOTPPurposeIsolation_EnrollCannotSatisfyLogin(t *testing.T) {
	withMiniredis(t)
	ctx := context.Background()
	uid, email := "user-1", "chef@fe3dr.com"

	code, err := IssueOTP(ctx, PurposeMFAEnroll, uid, email)
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	if err := RedeemOTP(ctx, PurposeMFAEnroll, uid, email, code); err != nil {
		t.Fatalf("redeem: %v", err)
	}
	if !OTPVerified(ctx, PurposeMFAEnroll, uid, email) {
		t.Fatal("enroll should be verified")
	}
	if OTPVerified(ctx, PurposeMFALogin, uid, email) {
		t.Fatal("enrollment verification satisfied a login challenge")
	}
}

func TestOTPKeysAreDistinctPerPurpose(t *testing.T) {
	seen := map[string]OTPPurpose{}
	for _, p := range []OTPPurpose{PurposeOnboardingEmail, PurposeMFAEnroll, PurposeMFALogin} {
		k := otpKey(p, "code", "u", "s")
		if other, dup := seen[k]; dup {
			t.Fatalf("purposes %q and %q share key %q", other, p, k)
		}
		seen[k] = p
	}
}

// The legacy onboarding key shape must not move: a deploy that changes it
// invalidates every in-flight verification and locks users mid-signup.
func TestOnboardingKeysKeepLegacyShape(t *testing.T) {
	if got, want := otpKey(PurposeOnboardingEmail, "code", "u1", "a@b.com"), "email_otp:code:u1:a@b.com"; got != want {
		t.Fatalf("legacy code key drifted: got %q want %q", got, want)
	}
	if got, want := otpKey(PurposeOnboardingEmail, "ok", "u1", "a@b.com"), "email_otp:ok:u1:a@b.com"; got != want {
		t.Fatalf("legacy verified key drifted: got %q want %q", got, want)
	}
}

func TestOTPRedeem_WrongCodeThenAttemptCeiling(t *testing.T) {
	withMiniredis(t)
	ctx := context.Background()
	uid, sub := "user-1", "chef@fe3dr.com"

	code, err := IssueOTP(ctx, PurposeMFALogin, uid, sub)
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	bad := wrongCode(code)
	for i := 0; i < otpMaxAttempts; i++ {
		if err := RedeemOTP(ctx, PurposeMFALogin, uid, sub, bad); !errors.Is(err, ErrOTPMismatch) {
			t.Fatalf("attempt %d: want ErrOTPMismatch, got %v", i+1, err)
		}
	}
	// Ceiling reached — the challenge is destroyed, so even the right code fails.
	if err := RedeemOTP(ctx, PurposeMFALogin, uid, sub, code); !errors.Is(err, ErrOTPAttemptLimit) {
		t.Fatalf("want ErrOTPAttemptLimit, got %v", err)
	}
	if OTPVerified(ctx, PurposeMFALogin, uid, sub) {
		t.Fatal("must not be verified after exhausting attempts")
	}
}

func TestOTPIssue_ResendCooldown(t *testing.T) {
	mr := withMiniredis(t)
	ctx := context.Background()
	uid, sub := "user-1", "chef@fe3dr.com"

	if _, err := IssueOTP(ctx, PurposeMFALogin, uid, sub); err != nil {
		t.Fatalf("first issue: %v", err)
	}
	if _, err := IssueOTP(ctx, PurposeMFALogin, uid, sub); !errors.Is(err, ErrOTPCooldown) {
		t.Fatalf("want ErrOTPCooldown, got %v", err)
	}
	mr.FastForward(otpResendCooldown + time.Second)
	if _, err := IssueOTP(ctx, PurposeMFALogin, uid, sub); err != nil {
		t.Fatalf("after cooldown: %v", err)
	}
}

func TestOTPIssue_SendWindowCap(t *testing.T) {
	mr := withMiniredis(t)
	ctx := context.Background()
	uid, sub := "user-1", "chef@fe3dr.com"

	for i := 0; i < otpMaxSends; i++ {
		if _, err := IssueOTP(ctx, PurposeMFALogin, uid, sub); err != nil {
			t.Fatalf("send %d: %v", i+1, err)
		}
		mr.FastForward(otpResendCooldown + time.Second)
	}
	if _, err := IssueOTP(ctx, PurposeMFALogin, uid, sub); !errors.Is(err, ErrOTPSendLimit) {
		t.Fatalf("want ErrOTPSendLimit, got %v", err)
	}
}

func TestOTPExpiry(t *testing.T) {
	mr := withMiniredis(t)
	ctx := context.Background()
	uid, sub := "user-1", "chef@fe3dr.com"

	code, err := IssueOTP(ctx, PurposeMFALogin, uid, sub)
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	mr.FastForward(otpTTL + time.Second)
	if err := RedeemOTP(ctx, PurposeMFALogin, uid, sub, code); !errors.Is(err, ErrOTPExpired) {
		t.Fatalf("want ErrOTPExpired, got %v", err)
	}
}

// Onboarding fails OPEN on a Redis outage so infra trouble cannot block every
// signup. A login second factor must do the opposite: failing open there would
// disable 2FA for everyone the moment Redis blipped.
func TestOTPVerified_FailsClosedForLoginWhenRedisDown(t *testing.T) {
	prev := SetRedisClientForTest(nil)
	t.Cleanup(func() { SetRedisClientForTest(prev) })
	ctx := context.Background()

	if OTPVerified(ctx, PurposeMFALogin, "user-1", "chef@fe3dr.com") {
		t.Fatal("login challenge must fail closed when Redis is unavailable")
	}
	if OTPVerified(ctx, PurposeMFAEnroll, "user-1", "chef@fe3dr.com") {
		t.Fatal("enrollment must fail closed when Redis is unavailable")
	}
	if !IsEmailOTPVerified(ctx, "user-1", "chef@fe3dr.com") {
		t.Fatal("onboarding must keep failing open (unchanged behaviour)")
	}
}

func TestOTPCodeShape(t *testing.T) {
	withMiniredis(t)
	ctx := context.Background()
	code, err := IssueOTP(ctx, PurposeMFALogin, "u", "s")
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	if len(code) != 6 {
		t.Fatalf("want 6 digits, got %q", code)
	}
	for _, r := range code {
		if r < '0' || r > '9' {
			t.Fatalf("non-digit in code %q", code)
		}
	}
}
