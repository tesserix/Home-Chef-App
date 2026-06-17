package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// signDelivery mirrors verifyHMACSHA256's expected computation so tests can
// produce a valid X-Webhook-Signature value for a payload + provider secret.
func signDelivery(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

func TestVerifyHMACSHA256_ValidSignature(t *testing.T) {
	secret := "shadowfax_webhook_secret"
	body := []byte(`{"event":"order.delivered","awb":"SF12345"}`)
	assert.True(t, verifyHMACSHA256(body, signDelivery(body, secret), secret))
}

func TestVerifyHMACSHA256_BadSignature_Rejected(t *testing.T) {
	secret := "shadowfax_webhook_secret"
	body := []byte(`{"event":"order.delivered"}`)
	assert.False(t, verifyHMACSHA256(body, "not-a-real-signature", secret))
}

func TestVerifyHMACSHA256_TamperedBody_Rejected(t *testing.T) {
	secret := "shadowfax_webhook_secret"
	original := []byte(`{"status":"in_transit"}`)
	sig := signDelivery(original, secret)
	tampered := []byte(`{"status":"delivered"}`)

	require.True(t, verifyHMACSHA256(original, sig, secret))
	assert.False(t, verifyHMACSHA256(tampered, sig, secret))
}

func TestVerifyHMACSHA256_WrongSecret_Rejected(t *testing.T) {
	body := []byte(`{"event":"order.delivered"}`)
	sigFromAttacker := signDelivery(body, "attacker_secret")
	assert.False(t, verifyHMACSHA256(body, sigFromAttacker, "real_provider_secret"))
}

func TestVerifyHMACSHA256_EmptySignatureOrSecret_Rejected(t *testing.T) {
	body := []byte(`{}`)
	assert.False(t, verifyHMACSHA256(body, "", "secret"))
	assert.False(t, verifyHMACSHA256(body, signDelivery(body, "secret"), ""))
	assert.False(t, verifyHMACSHA256(body, "", ""))
}
