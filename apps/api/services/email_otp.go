package services

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"errors"
	"fmt"
	"log"
	"math/big"
	"regexp"
	"strings"
	"time"
)

var (
	ErrOTPUnavailable  = errors.New("email verification is temporarily unavailable")
	ErrOTPCooldown     = errors.New("please wait a moment before requesting another code")
	ErrOTPSendLimit    = errors.New("too many codes requested; try again later")
	ErrOTPInvalidEmail = errors.New("enter a valid email address")
	ErrOTPExpired      = errors.New("this code has expired; request a new one")
	ErrOTPMismatch     = errors.New("incorrect code")
	ErrOTPAttemptLimit = errors.New("too many incorrect attempts; request a new code")
)

const (
	emailOTPTTL            = 10 * time.Minute
	emailOTPVerifiedTTL    = 2 * time.Hour
	emailOTPResendCooldown = 60 * time.Second
	emailOTPSendWindow     = time.Hour
	emailOTPMaxSends       = 5
	emailOTPMaxAttempts    = 5
)

var emailFormatRe = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

func NormalizeEmail(e string) string { return strings.ToLower(strings.TrimSpace(e)) }

func IsValidEmailFormat(e string) bool {
	e = NormalizeEmail(e)
	return len(e) >= 6 && len(e) <= 254 && emailFormatRe.MatchString(e)
}

func otpCodeKey(uid, email string) string     { return fmt.Sprintf("email_otp:code:%s:%s", uid, email) }
func otpVerifiedKey(uid, email string) string { return fmt.Sprintf("email_otp:ok:%s:%s", uid, email) }
func otpAttemptsKey(uid, email string) string { return fmt.Sprintf("email_otp:att:%s:%s", uid, email) }
func otpCooldownKey(uid, email string) string { return fmt.Sprintf("email_otp:cd:%s:%s", uid, email) }
func otpSendsKey(uid, email string) string    { return fmt.Sprintf("email_otp:snd:%s:%s", uid, email) }

func generateOTP() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

// RequestEmailOTP generates a 6-digit code, stores it with a TTL and emails it.
// A per-(user,email) resend cooldown and hourly send cap throttle abuse.
func RequestEmailOTP(ctx context.Context, userID, email, firstName string) error {
	email = NormalizeEmail(email)
	if !IsValidEmailFormat(email) {
		return ErrOTPInvalidEmail
	}
	r := GetRedisClient()
	if r == nil || !r.IsConnected() {
		return ErrOTPUnavailable
	}

	ok, err := r.SetNX(ctx, otpCooldownKey(userID, email), "1", emailOTPResendCooldown)
	if err != nil {
		return ErrOTPUnavailable
	}
	if !ok {
		return ErrOTPCooldown
	}
	if sends, err := r.IncrAndExpire(ctx, otpSendsKey(userID, email), emailOTPSendWindow); err == nil && sends > emailOTPMaxSends {
		return ErrOTPSendLimit
	}

	code, err := generateOTP()
	if err != nil {
		return ErrOTPUnavailable
	}
	if err := r.Set(ctx, otpCodeKey(userID, email), code, emailOTPTTL); err != nil {
		return ErrOTPUnavailable
	}
	_ = r.Set(ctx, otpAttemptsKey(userID, email), "0", emailOTPTTL)

	if err := GetEmailService().SendEmailOTP(email, firstName, code); err != nil {
		log.Printf("email-otp: send failed for user=%s: %v", userID, err)
		return ErrOTPUnavailable
	}
	return nil
}

// VerifyEmailOTP checks a submitted code and, on success, writes the short-lived
// verified marker the onboarding gate reads. Wrong codes are rate-capped.
func VerifyEmailOTP(ctx context.Context, userID, email, code string) error {
	email = NormalizeEmail(email)
	code = strings.TrimSpace(code)
	if len(code) != 6 {
		return ErrOTPMismatch
	}
	r := GetRedisClient()
	if r == nil || !r.IsConnected() {
		return ErrOTPUnavailable
	}

	if attempts, err := r.IncrAndExpire(ctx, otpAttemptsKey(userID, email), emailOTPTTL); err == nil && attempts > emailOTPMaxAttempts {
		_ = r.Del(ctx, otpCodeKey(userID, email))
		return ErrOTPAttemptLimit
	}
	stored, err := r.Get(ctx, otpCodeKey(userID, email))
	if err != nil || stored == "" {
		return ErrOTPExpired
	}
	if subtle.ConstantTimeCompare([]byte(stored), []byte(code)) != 1 {
		return ErrOTPMismatch
	}
	if err := r.Set(ctx, otpVerifiedKey(userID, email), "1", emailOTPVerifiedTTL); err != nil {
		return ErrOTPUnavailable
	}
	_ = r.Del(ctx, otpCodeKey(userID, email))
	return nil
}

// IsEmailOTPVerified reports whether (userID,email) holds a live verified marker.
// Fails open on Redis outage so an infra failure can't block all onboarding.
func IsEmailOTPVerified(ctx context.Context, userID, email string) bool {
	email = NormalizeEmail(email)
	r := GetRedisClient()
	if r == nil || !r.IsConnected() {
		log.Printf("email-otp: Redis unavailable, allowing user=%s (failing open)", userID)
		return true
	}
	v, err := r.Get(ctx, otpVerifiedKey(userID, email))
	return err == nil && v == "1"
}
