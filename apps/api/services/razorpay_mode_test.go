package services

import (
	"testing"

	"github.com/homechef/api/models"
)

// The two slots must be genuinely independent. A shared cache would mean the
// first chef to transact decides which credentials everyone else uses — the
// exact failure this feature exists to prevent.
func TestRazorpaySlotsAreIndependent(t *testing.T) {
	t.Cleanup(InvalidateRazorpay)

	SetRazorpayClientFor(models.ChefModeLive, &RazorpayClient{keyID: "rzp_live_AAA", keySecret: "s1"})
	SetRazorpayClientFor(models.ChefModeTest, &RazorpayClient{keyID: "rzp_test_BBB", keySecret: "s2"})

	if got := GetRazorpayFor(models.ChefModeLive).GetKeyID(); got != "rzp_live_AAA" {
		t.Fatalf("live slot = %q, want rzp_live_AAA", got)
	}
	if got := GetRazorpayFor(models.ChefModeTest).GetKeyID(); got != "rzp_test_BBB" {
		t.Fatalf("test slot = %q, want rzp_test_BBB", got)
	}

	// Invalidating one slot must not evict the other — saving test keys in the
	// admin UI must not knock a healthy live gateway offline.
	InvalidateRazorpayFor(models.ChefModeTest)
	if GetRazorpayFor(models.ChefModeLive).GetKeyID() != "rzp_live_AAA" {
		t.Fatal("invalidating test must not evict the live client")
	}
}

// An unrecognised mode must resolve to live, matching NormalizeMode. A typo in
// a caller must never silently fall through to sandbox credentials, which would
// take no real money while appearing to succeed.
func TestUnknownModeUsesLiveSlot(t *testing.T) {
	t.Cleanup(InvalidateRazorpay)
	SetRazorpayClientFor(models.ChefModeLive, &RazorpayClient{keyID: "rzp_live_AAA"})
	if GetRazorpayFor("garbage").GetKeyID() != "rzp_live_AAA" {
		t.Fatal("an unknown mode must use the live slot")
	}
}

// GetRazorpay() is retained for the non-chef-scoped call sites (admin gateway
// status, reconciliation, wallet top-ups, platform subscriptions) and must keep
// meaning exactly what it meant before: the live slot.
func TestLegacyGetRazorpayIsLiveSlot(t *testing.T) {
	t.Cleanup(InvalidateRazorpay)
	SetRazorpayClientFor(models.ChefModeLive, &RazorpayClient{keyID: "rzp_live_AAA"})
	if GetRazorpay().GetKeyID() != "rzp_live_AAA" {
		t.Fatal("GetRazorpay() must be the live slot")
	}
}

// Each slot must read its own Secret Manager names. Crossing them would make
// the admin UI's Test card silently overwrite the live credentials.
func TestRazorpaySecretNamesPerSlot(t *testing.T) {
	id, secret, webhook := razorpaySecretNames(models.ChefModeLive)
	if id != SecretRazorpayKeyID || secret != SecretRazorpayKeySecret || webhook != SecretRazorpayWebhookSecret {
		t.Fatalf("live slot resolved to (%s,%s,%s)", id, secret, webhook)
	}
	id, secret, webhook = razorpaySecretNames(models.ChefModeTest)
	if id != SecretRazorpayTestKeyID || secret != SecretRazorpayTestKeySecret || webhook != SecretRazorpayTestWebhookSecret {
		t.Fatalf("test slot resolved to (%s,%s,%s)", id, secret, webhook)
	}
	// The live slot names must be the pre-existing ones, or every credential
	// already configured in production silently stops being found.
	if SecretRazorpayKeyID != "prod-homechef-razorpay-key-id" {
		t.Fatalf("live key-id secret name changed to %q — existing prod config would break", SecretRazorpayKeyID)
	}
}
