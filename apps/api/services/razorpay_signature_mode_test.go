package services

import (
	"testing"

	"github.com/homechef/api/models"
)

func TestWebhookSignatureIdentifiesSigningMode(t *testing.T) {
	t.Cleanup(InvalidateRazorpay)
	SetRazorpayClientFor(models.ChefModeLive, &RazorpayClient{webhookSecret: "live-secret"})
	SetRazorpayClientFor(models.ChefModeTest, &RazorpayClient{webhookSecret: "test-secret"})

	payload := []byte(`{"event":"payment.captured"}`)

	ok, mode := VerifyWebhookSignatureMode(payload, hmacHex(payload, "live-secret"))
	if !ok || mode != models.ChefModeLive {
		t.Fatalf("live-signed webhook = (%v,%q), want (true,live)", ok, mode)
	}
	ok, mode = VerifyWebhookSignatureMode(payload, hmacHex(payload, "test-secret"))
	if !ok || mode != models.ChefModeTest {
		t.Fatalf("test-signed webhook = (%v,%q), want (true,test)", ok, mode)
	}
	if ok, _ := VerifyWebhookSignatureMode(payload, hmacHex(payload, "wrong")); ok {
		t.Fatal("a foreign-signed webhook must be rejected")
	}
}

// The interim production state: both slots hold the SAME test key until a real
// live key is issued. The live attempt then always wins, so the mode reported
// here is not trustworthy on its own — which is exactly why the webhook handler
// must also compare against the record's own mode. This test documents the
// behaviour so nobody "fixes" it later without understanding the consequence.
func TestIdenticalSecretsResolveToLive(t *testing.T) {
	t.Cleanup(InvalidateRazorpay)
	SetRazorpayClientFor(models.ChefModeLive, &RazorpayClient{webhookSecret: "same"})
	SetRazorpayClientFor(models.ChefModeTest, &RazorpayClient{webhookSecret: "same"})

	payload := []byte(`{"event":"payment.captured"}`)
	ok, mode := VerifyWebhookSignatureMode(payload, hmacHex(payload, "same"))
	if !ok || mode != models.ChefModeLive {
		t.Fatalf("identical secrets = (%v,%q), want (true,live)", ok, mode)
	}
}

// A test slot that is not configured yet must not break live webhooks.
func TestWebhookWorksWithOnlyLiveConfigured(t *testing.T) {
	t.Cleanup(InvalidateRazorpay)
	SetRazorpayClientFor(models.ChefModeLive, &RazorpayClient{webhookSecret: "live-secret"})
	SetRazorpayClientFor(models.ChefModeTest, nil)

	payload := []byte(`{"event":"payment.captured"}`)
	if ok, mode := VerifyWebhookSignatureMode(payload, hmacHex(payload, "live-secret")); !ok || mode != models.ChefModeLive {
		t.Fatalf("got (%v,%q), want (true,live)", ok, mode)
	}
}

func TestPaymentSignatureIsModeScoped(t *testing.T) {
	t.Cleanup(InvalidateRazorpay)
	SetRazorpayClientFor(models.ChefModeLive, &RazorpayClient{keySecret: "live-key-secret"})
	SetRazorpayClientFor(models.ChefModeTest, &RazorpayClient{keySecret: "test-key-secret"})

	oid, pid := "order_ABC1", "pay_ABC1"
	body := []byte(oid + "|" + pid)

	if !VerifyPaymentSignatureFor(models.ChefModeTest, oid, pid, hmacHex(body, "test-key-secret")) {
		t.Fatal("a test-mode payment must verify against the test key secret")
	}
	// The load-bearing assertion: a sandbox-signed payment must never be
	// accepted as a real one.
	if VerifyPaymentSignatureFor(models.ChefModeLive, oid, pid, hmacHex(body, "test-key-secret")) {
		t.Fatal("a test-signed payment must NOT verify against live credentials")
	}
	if !VerifyPaymentSignatureFor(models.ChefModeLive, oid, pid, hmacHex(body, "live-key-secret")) {
		t.Fatal("a live payment must verify against the live key secret")
	}
}
