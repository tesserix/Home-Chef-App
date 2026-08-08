package services

import (
	"context"
	"errors"
	"testing"
	"time"
)

// #1164 finding 3. A login challenge used to be keyed by (user, subject) alone,
// so two handsets on one account fought over a single code: the second device to
// ask overwrote the first's code, and whichever asked within the minute was told
// to wait instead of being sent anything. The device that could not sign in
// looked dead rather than challenged.

const (
	deviceA = "device-a"
	deviceB = "device-b"
)

func TestOTPIssue_SecondDeviceIsChallengedIndependently(t *testing.T) {
	withMiniredis(t)
	ctx := context.Background()
	uid, sub := "user-1", "chef@fe3dr.com"

	codeA, err := IssueOTP(ctx, PurposeMFALogin, uid, sub, deviceA)
	if err != nil {
		t.Fatalf("device A issue: %v", err)
	}
	codeB, err := IssueOTP(ctx, PurposeMFALogin, uid, sub, deviceB)
	if err != nil {
		t.Fatalf("device B issue inside A's cooldown: %v", err)
	}
	if codeA == codeB {
		t.Fatal("both devices were handed the same code")
	}

	// A's code must survive B's challenge, and vice versa.
	if err := RedeemOTP(ctx, PurposeMFALogin, uid, sub, deviceA, codeA); err != nil {
		t.Fatalf("device A redeem after B challenged: %v", err)
	}
	if err := RedeemOTP(ctx, PurposeMFALogin, uid, sub, deviceB, codeB); err != nil {
		t.Fatalf("device B redeem: %v", err)
	}
}

// A code is a bearer credential for one device. Presenting it from another
// device must not pass, or the scoping would be cosmetic.
func TestOTPRedeem_CodeDoesNotCrossDevices(t *testing.T) {
	withMiniredis(t)
	ctx := context.Background()
	uid, sub := "user-1", "chef@fe3dr.com"

	codeA, err := IssueOTP(ctx, PurposeMFALogin, uid, sub, deviceA)
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	if _, err := IssueOTP(ctx, PurposeMFALogin, uid, sub, deviceB); err != nil {
		t.Fatalf("device B issue: %v", err)
	}
	if err := RedeemOTP(ctx, PurposeMFALogin, uid, sub, deviceB, codeA); !errors.Is(err, ErrOTPMismatch) {
		t.Fatalf("want ErrOTPMismatch replaying A's code on B, got %v", err)
	}
}

// The client's device id is read from the keychain asynchronously, so the very
// first request of a cold start can go out without the header and the next one
// with it. A code issued account-wide must therefore still redeem once the id
// has warmed, or the user is told their correct code is wrong.
func TestOTPRedeem_FallsBackToAnAccountWideCode(t *testing.T) {
	withMiniredis(t)
	ctx := context.Background()
	uid, sub := "user-1", "chef@fe3dr.com"

	code, err := IssueOTP(ctx, PurposeMFALogin, uid, sub, "")
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	if err := RedeemOTP(ctx, PurposeMFALogin, uid, sub, deviceA, code); err != nil {
		t.Fatalf("redeem once the device id warmed: %v", err)
	}
}

// The fallback must not become a bypass: with a device-scoped code on file, the
// account-wide slot is not consulted.
func TestOTPRedeem_ScopedCodeTakesPrecedence(t *testing.T) {
	withMiniredis(t)
	ctx := context.Background()
	uid, sub := "user-1", "chef@fe3dr.com"

	wide, err := IssueOTP(ctx, PurposeMFALogin, uid, sub, "")
	if err != nil {
		t.Fatalf("account-wide issue: %v", err)
	}
	if _, err := IssueOTP(ctx, PurposeMFALogin, uid, sub, deviceA); err != nil {
		t.Fatalf("device issue: %v", err)
	}
	if err := RedeemOTP(ctx, PurposeMFALogin, uid, sub, deviceA, wide); !errors.Is(err, ErrOTPMismatch) {
		t.Fatalf("want ErrOTPMismatch, got %v", err)
	}
}

func TestOTPIssue_ResendCooldownIsPerDevice(t *testing.T) {
	withMiniredis(t)
	ctx := context.Background()
	uid, sub := "user-1", "chef@fe3dr.com"

	if _, err := IssueOTP(ctx, PurposeMFALogin, uid, sub, deviceA); err != nil {
		t.Fatalf("first issue: %v", err)
	}
	if _, err := IssueOTP(ctx, PurposeMFALogin, uid, sub, deviceA); !errors.Is(err, ErrOTPCooldown) {
		t.Fatalf("same device wants ErrOTPCooldown, got %v", err)
	}
	if _, err := IssueOTP(ctx, PurposeMFALogin, uid, sub, deviceB); err != nil {
		t.Fatalf("other device must not inherit the cooldown: %v", err)
	}
}

func TestOTPRedeem_AttemptCeilingIsPerDevice(t *testing.T) {
	withMiniredis(t)
	ctx := context.Background()
	uid, sub := "user-1", "chef@fe3dr.com"

	codeA, err := IssueOTP(ctx, PurposeMFALogin, uid, sub, deviceA)
	if err != nil {
		t.Fatalf("device A issue: %v", err)
	}
	if _, err := IssueOTP(ctx, PurposeMFALogin, uid, sub, deviceB); err != nil {
		t.Fatalf("device B issue: %v", err)
	}
	bad := wrongCode(codeA)
	for i := 0; i < otpMaxAttempts; i++ {
		if err := RedeemOTP(ctx, PurposeMFALogin, uid, sub, deviceB, bad); !errors.Is(err, ErrOTPMismatch) {
			t.Fatalf("device B attempt %d: %v", i+1, err)
		}
	}
	if err := RedeemOTP(ctx, PurposeMFALogin, uid, sub, deviceA, codeA); err != nil {
		t.Fatalf("device A locked out by device B's wrong guesses: %v", err)
	}
}

// The hourly cap protects the user's inbox, so it stays account-wide: a device
// id is client-supplied, and per-device budgets would let a rotating id mail
// bomb the address.
func TestOTPIssue_SendCapStaysAccountWide(t *testing.T) {
	mr := withMiniredis(t)
	ctx := context.Background()
	uid, sub := "user-1", "chef@fe3dr.com"

	for i := 0; i < otpMaxSends; i++ {
		device := deviceA
		if i%2 == 1 {
			device = deviceB
		}
		if _, err := IssueOTP(ctx, PurposeMFALogin, uid, sub, device); err != nil {
			t.Fatalf("send %d: %v", i+1, err)
		}
		mr.FastForward(otpResendCooldown + time.Second)
	}
	if _, err := IssueOTP(ctx, PurposeMFALogin, uid, sub, deviceB); !errors.Is(err, ErrOTPSendLimit) {
		t.Fatalf("want ErrOTPSendLimit, got %v", err)
	}
}

// Clients that predate the device header send nothing, and must keep the
// pre-#1164 key shape rather than landing in a device-scoped bucket they can
// never address again.
func TestOTPKeys_UnscopedShapeIsUnchanged(t *testing.T) {
	if got, want := otpKey(PurposeMFALogin, "code", "u1", "a@b.com", ""), "otp:mfa_login:code:u1:a@b.com"; got != want {
		t.Fatalf("unscoped key drifted: got %q want %q", got, want)
	}
	if otpKey(PurposeMFALogin, "code", "u1", "a@b.com", deviceA) == otpKey(PurposeMFALogin, "code", "u1", "a@b.com", "") {
		t.Fatal("a device-scoped key collides with the unscoped one")
	}
	// Onboarding's shape is frozen and takes no device.
	if got, want := otpKey(PurposeOnboardingEmail, "code", "u1", "a@b.com", deviceA), "email_otp:code:u1:a@b.com"; got != want {
		t.Fatalf("legacy key drifted: got %q want %q", got, want)
	}
}
