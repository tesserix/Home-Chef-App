package services

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// hmacHex mirrors VerifyWebhookSignature's expected-signature computation so
// tests can produce a valid signature for an arbitrary payload + secret.
func hmacHex(payload []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}

// withRazorpayClient swaps the package-level razorpayClient for the duration of
// a test and restores it afterward, keeping the singleton clean across tests.
func withRazorpayClient(t *testing.T, c *RazorpayClient) {
	t.Helper()
	razorpayMu.Lock()
	prev := razorpayClient
	razorpayClient = c
	razorpayMu.Unlock()
	t.Cleanup(func() {
		razorpayMu.Lock()
		razorpayClient = prev
		razorpayMu.Unlock()
	})
}

func TestVerifyWebhookSignature_ValidSignature(t *testing.T) {
	secret := "whsec_test_razorpay"
	withRazorpayClient(t, &RazorpayClient{webhookSecret: secret, fetchedAt: time.Now()})

	payload := []byte(`{"event":"payment.captured","id":"pay_123"}`)
	sig := hmacHex(payload, secret)

	assert.True(t, VerifyWebhookSignature(payload, sig))
}

func TestVerifyWebhookSignature_BadSignature_Rejected(t *testing.T) {
	withRazorpayClient(t, &RazorpayClient{webhookSecret: "whsec_test_razorpay", fetchedAt: time.Now()})

	payload := []byte(`{"event":"payment.captured","id":"pay_123"}`)
	assert.False(t, VerifyWebhookSignature(payload, "deadbeefdeadbeef"))
	assert.False(t, VerifyWebhookSignature(payload, ""))
}

func TestVerifyWebhookSignature_TamperedPayload_Rejected(t *testing.T) {
	secret := "whsec_test_razorpay"
	withRazorpayClient(t, &RazorpayClient{webhookSecret: secret, fetchedAt: time.Now()})

	original := []byte(`{"amount":100}`)
	sig := hmacHex(original, secret)
	tampered := []byte(`{"amount":999999}`)

	require.True(t, VerifyWebhookSignature(original, sig))
	assert.False(t, VerifyWebhookSignature(tampered, sig))
}

func TestVerifyWebhookSignature_WrongSecret_Rejected(t *testing.T) {
	withRazorpayClient(t, &RazorpayClient{webhookSecret: "real_secret", fetchedAt: time.Now()})

	payload := []byte(`{"event":"payment.captured"}`)
	sigFromAttacker := hmacHex(payload, "attacker_secret")
	assert.False(t, VerifyWebhookSignature(payload, sigFromAttacker))
}

func TestVerifyWebhookSignature_NoSecretConfigured_Rejected(t *testing.T) {
	// Nil client: secret unconfigured -> reject (never fail-open on auth).
	withRazorpayClient(t, nil)
	assert.False(t, VerifyWebhookSignature([]byte(`{}`), hmacHex([]byte(`{}`), "x")))

	// Client present but empty secret -> still reject.
	withRazorpayClient(t, &RazorpayClient{webhookSecret: "", fetchedAt: time.Now()})
	assert.False(t, VerifyWebhookSignature([]byte(`{}`), "anything"))
}
