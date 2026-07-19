package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func withMiniredis(t *testing.T) *miniredis.Miniredis {
	t.Helper()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	prev := SetRedisClientForTest(client)
	t.Cleanup(func() {
		_ = client.Close()
		SetRedisClientForTest(prev)
	})
	return mr
}

func wrongCode(actual string) string {
	if actual == "111111" {
		return "222222"
	}
	return "111111"
}

func TestEmailOTP_RequestVerifyFlow(t *testing.T) {
	mr := withMiniredis(t)
	ctx := context.Background()
	uid, email := "user-1", "Chef@Fe3dr.com"
	norm := NormalizeEmail(email)

	if IsEmailOTPVerified(ctx, uid, email) {
		t.Fatal("must not be verified before request")
	}
	if err := RequestEmailOTP(ctx, uid, email, "Chef"); err != nil {
		t.Fatalf("request: %v", err)
	}
	code, err := mr.Get(otpCodeKey(uid, norm))
	if err != nil || len(code) != 6 {
		t.Fatalf("stored code missing: %q %v", code, err)
	}

	if err := VerifyEmailOTP(ctx, uid, email, wrongCode(code)); !errors.Is(err, ErrOTPMismatch) {
		t.Fatalf("wrong code: want ErrOTPMismatch, got %v", err)
	}
	if err := VerifyEmailOTP(ctx, uid, email, code); err != nil {
		t.Fatalf("verify correct: %v", err)
	}
	if !IsEmailOTPVerified(ctx, uid, email) {
		t.Fatal("must be verified after correct code")
	}
	// Code is consumed on success.
	if _, err := mr.Get(otpCodeKey(uid, norm)); err == nil {
		t.Error("code key should be deleted after successful verify")
	}
}

func TestEmailOTP_AttemptLockout(t *testing.T) {
	mr := withMiniredis(t)
	ctx := context.Background()
	uid, email := "user-2", "d@fe3dr.com"

	if err := RequestEmailOTP(ctx, uid, email, ""); err != nil {
		t.Fatalf("request: %v", err)
	}
	code, _ := mr.Get(otpCodeKey(uid, email))
	bad := wrongCode(code)

	for i := 0; i < emailOTPMaxAttempts; i++ {
		if err := VerifyEmailOTP(ctx, uid, email, bad); !errors.Is(err, ErrOTPMismatch) {
			t.Fatalf("attempt %d: want ErrOTPMismatch, got %v", i+1, err)
		}
	}
	// Next attempt is locked out — even the correct code is rejected.
	if err := VerifyEmailOTP(ctx, uid, email, code); !errors.Is(err, ErrOTPAttemptLimit) {
		t.Fatalf("want ErrOTPAttemptLimit, got %v", err)
	}
	if IsEmailOTPVerified(ctx, uid, email) {
		t.Fatal("must not be verified after lockout")
	}
}

func TestEmailOTP_ResendCooldown(t *testing.T) {
	withMiniredis(t)
	ctx := context.Background()
	uid, email := "user-3", "c@fe3dr.com"

	if err := RequestEmailOTP(ctx, uid, email, ""); err != nil {
		t.Fatalf("first request: %v", err)
	}
	if err := RequestEmailOTP(ctx, uid, email, ""); !errors.Is(err, ErrOTPCooldown) {
		t.Fatalf("want ErrOTPCooldown, got %v", err)
	}
}

func TestEmailOTP_Expiry(t *testing.T) {
	mr := withMiniredis(t)
	ctx := context.Background()
	uid, email := "user-4", "e@fe3dr.com"

	if err := RequestEmailOTP(ctx, uid, email, ""); err != nil {
		t.Fatalf("request: %v", err)
	}
	code, _ := mr.Get(otpCodeKey(uid, email))
	mr.FastForward(emailOTPTTL + time.Minute)
	if err := VerifyEmailOTP(ctx, uid, email, code); !errors.Is(err, ErrOTPExpired) {
		t.Fatalf("want ErrOTPExpired, got %v", err)
	}
}

func TestEmailOTP_SendCap(t *testing.T) {
	mr := withMiniredis(t)
	ctx := context.Background()
	uid, email := "user-5", "s@fe3dr.com"

	for i := 0; i < emailOTPMaxSends; i++ {
		if err := RequestEmailOTP(ctx, uid, email, ""); err != nil {
			t.Fatalf("send %d: %v", i+1, err)
		}
		mr.FastForward(emailOTPResendCooldown + time.Second) // clear cooldown, stay in send window
	}
	if err := RequestEmailOTP(ctx, uid, email, ""); !errors.Is(err, ErrOTPSendLimit) {
		t.Fatalf("want ErrOTPSendLimit, got %v", err)
	}
}
