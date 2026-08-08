package services

import (
	"context"
	"crypto/subtle"
	"fmt"
	"log"
	"strings"
	"time"
)

// Purpose-namespaced one-time codes.
//
// The onboarding flow already had OTPs (email_otp.go), and the obvious move
// when adding a login second factor was to call the same functions. That would
// have been a hole: VerifyEmailOTP writes a two-hour "verified" marker keyed by
// (user, email), and a login challenge reading the same key would be satisfied
// by a code the user typed during signup — for every user, forever.
//
// So every entry is keyed by purpose, and a marker written under one purpose is
// invisible to another. PurposeOnboardingEmail deliberately keeps the original
// key shape: changing it on deploy would invalidate every in-flight signup
// verification.

type OTPPurpose string

const (
	// PurposeOnboardingEmail is the pre-existing signup email verification.
	// Its key shape is frozen for backwards compatibility.
	PurposeOnboardingEmail OTPPurpose = "onboarding_email"
	// PurposeMFAEnroll proves a channel belongs to the user while they are
	// switching two-factor on.
	PurposeMFAEnroll OTPPurpose = "mfa_enroll"
	// PurposeMFALogin is the login second factor.
	PurposeMFALogin OTPPurpose = "mfa_login"
)

const (
	// Matches emailOTPTTL — see the note there. Any OTP on this platform is
	// valid for at most five minutes.
	otpTTL            = 5 * time.Minute
	otpVerifiedTTL    = 2 * time.Hour
	otpResendCooldown = 60 * time.Second
	otpSendWindow     = time.Hour
	otpMaxSends       = 5
	otpMaxAttempts    = 5
)

// otpKey builds the Redis key for one (purpose, kind, user, subject, device)
// tuple. `kind` is one of code / ok / att / cd / snd. The subject is the email
// address or phone number the code was sent to, so enrolling a second channel
// cannot clobber the first one's challenge.
//
// An empty device is account-wide and keeps the pre-#1164 key shape, which is
// what clients that predate the X-Device-Id header send.
func otpKey(p OTPPurpose, kind, uid, subject, device string) string {
	if p == PurposeOnboardingEmail {
		// Frozen shape — see the file comment.
		return fmt.Sprintf("email_otp:%s:%s:%s", kind, uid, subject)
	}
	if device == "" {
		return fmt.Sprintf("otp:%s:%s:%s:%s", p, kind, uid, subject)
	}
	return fmt.Sprintf("otp:%s:%s:%s:%s:%s", p, kind, uid, subject, device)
}

// IssueOTP generates a code and stores it under the given purpose, applying the
// resend cooldown and the hourly send cap. It does NOT deliver the code —
// delivery differs per channel, and keeping it out means a delivery failure is
// the caller's to report rather than something swallowed here.
func IssueOTP(ctx context.Context, p OTPPurpose, uid, subject, device string) (string, error) {
	r := GetRedisClient()
	if r == nil || !r.IsConnected() {
		return "", ErrOTPUnavailable
	}

	ok, err := r.SetNX(ctx, otpKey(p, "cd", uid, subject, device), "1", otpResendCooldown)
	if err != nil {
		return "", ErrOTPUnavailable
	}
	if !ok {
		return "", ErrOTPCooldown
	}
	// The send cap is deliberately account-wide: a device id comes from the
	// client, so a per-device budget would let a rotating id mail bomb the
	// address this cap exists to protect.
	if sends, err := r.IncrAndExpire(ctx, otpKey(p, "snd", uid, subject, ""), otpSendWindow); err == nil && sends > otpMaxSends {
		return "", ErrOTPSendLimit
	}

	code, err := generateOTP()
	if err != nil {
		return "", ErrOTPUnavailable
	}
	if err := r.Set(ctx, otpKey(p, "code", uid, subject, device), code, otpTTL); err != nil {
		return "", ErrOTPUnavailable
	}
	// Reset the attempt counter with the new code, so a fresh challenge is not
	// born already at the ceiling from a previous one.
	_ = r.Set(ctx, otpKey(p, "att", uid, subject, device), "0", otpTTL)
	return code, nil
}

// RedeemOTP checks a submitted code and, on success, writes the verified marker.
// Attempts are counted before the comparison so a wrong-code flood burns the
// budget rather than probing indefinitely.
func RedeemOTP(ctx context.Context, p OTPPurpose, uid, subject, device, code string) error {
	code = strings.TrimSpace(code)
	if len(code) != 6 {
		return ErrOTPMismatch
	}
	r := GetRedisClient()
	if r == nil || !r.IsConnected() {
		return ErrOTPUnavailable
	}

	// The client reads its device id from the keychain asynchronously, so the
	// request that asked for the code can predate the header while the one
	// redeeming it carries it. Fall back to the account-wide slot rather than
	// telling the user their correct code is wrong.
	if device != "" {
		if v, err := r.Get(ctx, otpKey(p, "code", uid, subject, device)); err != nil || v == "" {
			device = ""
		}
	}

	if attempts, err := r.IncrAndExpire(ctx, otpKey(p, "att", uid, subject, device), otpTTL); err == nil && attempts > otpMaxAttempts {
		_ = r.Del(ctx, otpKey(p, "code", uid, subject, device))
		return ErrOTPAttemptLimit
	}
	stored, err := r.Get(ctx, otpKey(p, "code", uid, subject, device))
	if err != nil || stored == "" {
		return ErrOTPExpired
	}
	if subtle.ConstantTimeCompare([]byte(stored), []byte(code)) != 1 {
		return ErrOTPMismatch
	}
	// The verified marker stays account-wide. It records that the user proved
	// this channel, which is not a per-device fact, and the two readers of it
	// (enrollment, onboarding) are account-wide flows.
	if err := r.Set(ctx, otpKey(p, "ok", uid, subject, ""), "1", otpVerifiedTTL); err != nil {
		return ErrOTPUnavailable
	}
	_ = r.Del(ctx, otpKey(p, "code", uid, subject, device))
	return nil
}

// OTPVerified reports whether a live verified marker exists for this purpose.
//
// Fails CLOSED on a Redis outage. Onboarding fails open (see IsEmailOTPVerified)
// because blocking every signup on a cache blip is worse than letting an
// unverified address through. The trade runs the other way for a second factor:
// failing open would silently disable 2FA for every user the moment Redis
// blipped, which is precisely the attack window someone would wait for.
func OTPVerified(ctx context.Context, p OTPPurpose, uid, subject string) bool {
	r := GetRedisClient()
	if r == nil || !r.IsConnected() {
		log.Printf("otp: Redis unavailable, denying %s for user=%s (failing closed)", p, uid)
		return false
	}
	v, err := r.Get(ctx, otpKey(p, "ok", uid, subject, ""))
	return err == nil && v == "1"
}

// ClearOTPVerified drops a verified marker. Used when a factor is re-enrolled or
// removed, so a stale marker cannot vouch for a channel the user has dropped.
func ClearOTPVerified(ctx context.Context, p OTPPurpose, uid, subject string) {
	r := GetRedisClient()
	if r == nil || !r.IsConnected() {
		return
	}
	_ = r.Del(ctx, otpKey(p, "ok", uid, subject, ""))
}
