package services

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
	"time"

	"github.com/homechef/api/config"
)

// The stated expiry in the email must be the one actually enforced. These
// drifting apart is how users end up told "15 minutes" while the link dies in
// five, or lives for an hour.
func TestPasswordResetTTLIsFifteenMinutes(t *testing.T) {
	if PasswordResetTTL != 15*time.Minute {
		t.Fatalf("PasswordResetTTL = %v, want 15m", PasswordResetTTL)
	}
	_, html := PasswordResetHTML("https://api.fe3dr.com/x", PasswordResetTTL)
	if !strings.Contains(html, "expires in 15 minutes") {
		t.Fatal("the email must state the same expiry the code enforces")
	}
}

// Any OTP on this platform is a live credential sitting in an inbox. Five
// minutes is the agreed ceiling.
func TestOTPLifetimesAreAtMostFiveMinutes(t *testing.T) {
	if emailOTPTTL > 5*time.Minute {
		t.Fatalf("emailOTPTTL = %v, want <= 5m", emailOTPTTL)
	}
	if otpTTL > 5*time.Minute {
		t.Fatalf("otpTTL = %v, want <= 5m", otpTTL)
	}
}

// The token is a credential. Storing it verbatim would mean a Redis dump — or
// anyone with KEYS access — could reset arbitrary accounts.
func TestResetTokensAreStoredHashed(t *testing.T) {
	token := "super-secret-token-value"
	key := resetTokenKey(token)

	if strings.Contains(key, token) {
		t.Fatal("the raw token must never appear in the Redis key")
	}
	sum := sha256.Sum256([]byte(token))
	if key != "pwreset:"+hex.EncodeToString(sum[:]) {
		t.Fatalf("unexpected key derivation: %s", key)
	}
	// Same token in, same key out — otherwise redemption could never find it.
	if resetTokenKey(token) != key {
		t.Fatal("key derivation must be deterministic")
	}
}

func TestResetTokensAreUnpredictable(t *testing.T) {
	seen := make(map[string]bool, 200)
	for i := 0; i < 200; i++ {
		tok, err := newResetToken()
		if err != nil {
			t.Fatalf("newResetToken: %v", err)
		}
		if len(tok) < 40 {
			t.Fatalf("token too short to resist guessing: %d chars", len(tok))
		}
		if seen[tok] {
			t.Fatal("newResetToken returned a duplicate — not CSPRNG-backed")
		}
		seen[tok] = true
	}
}

// Accounts are tenant-scoped in Identity Platform. Getting this mapping wrong
// is precisely the bug that made a vendor's reset request silently no-op: the
// address simply does not exist in the customer tenant.
func TestTenantForApp(t *testing.T) {
	restoreAppConfig(t)
	config.AppConfig = &config.Config{
		GIPCustomerTenantID: "cust",
		GIPBusinessTenantID: "biz",
		GIPDeliveryTenantID: "drv",
	}
	cases := map[string]string{
		"vendor": "biz", "chef": "biz", "business": "biz", "VENDOR": "biz", " vendor ": "biz",
		"delivery": "drv", "driver": "drv",
		"customer": "cust", "": "cust", "nonsense": "cust",
	}
	for in, want := range cases {
		if got := TenantForApp(in); got != want {
			t.Fatalf("TenantForApp(%q) = %q, want %q", in, got, want)
		}
	}
}

// The reset link must point at our own host. A raw firebaseapp.com URL in an
// email is both a deliverability problem and indistinguishable from phishing.
func TestResetLinkUsesOurOwnDomain(t *testing.T) {
	restoreAppConfig(t)
	config.AppConfig = &config.Config{PublicAPIBaseURL: "https://api.fe3dr.com"}
	url := PasswordResetLinkURL("tok123")

	if !strings.HasPrefix(url, "https://api.fe3dr.com/") {
		t.Fatalf("reset link must be on our domain, got %s", url)
	}
	if strings.Contains(url, "firebaseapp.com") {
		t.Fatal("the emailed link must never expose the provider URL")
	}
	if !strings.Contains(url, "t=tok123") {
		t.Fatalf("token missing from link: %s", url)
	}

	// A trailing slash on the configured base must not produce a double slash.
	config.AppConfig = &config.Config{PublicAPIBaseURL: "https://api.fe3dr.com/"}
	if strings.Contains(PasswordResetLinkURL("t"), "com//") {
		t.Fatal("trailing slash in the configured base must be handled")
	}
}

// The email is a security message. It must not leak the provider, and it must
// carry the anti-phishing guidance that makes a real one distinguishable from
// a fake.
func TestResetEmailIsBrandedAndSafe(t *testing.T) {
	subject, html := PasswordResetHTML("https://api.fe3dr.com/api/v1/auth/password-reset/consume?t=abc", 15*time.Minute)

	if !strings.Contains(subject, "Fe3dr") {
		t.Fatalf("subject must be branded for the product, got %q", subject)
	}
	if strings.Contains(html, "Tesserix") || strings.Contains(html, "firebaseapp.com") {
		t.Fatal("the email must not mention the GCP project or the provider domain")
	}
	if !strings.Contains(html, "only be used once") {
		t.Fatal("single-use must be stated")
	}
	if !strings.Contains(html, "never ask you for your password") {
		t.Fatal("anti-phishing guidance must be present in a security email")
	}
}

// Expired, unknown and already-used links must be indistinguishable, or the
// page becomes an oracle for probing tokens.
func TestExpiredPageRevealsNothing(t *testing.T) {
	page := PasswordResetExpiredHTML()
	for _, leak := range []string{"expired", "not found", "already used", "invalid token"} {
		if strings.Contains(strings.ToLower(page), leak) && leak != "expired" {
			t.Fatalf("the page distinguishes failure modes via %q", leak)
		}
	}
	if !strings.Contains(page, "no longer valid") {
		t.Fatal("the page should use one neutral message for every failure mode")
	}
	if !strings.Contains(page, "noindex") {
		t.Fatal("the page must not be indexable")
	}
}

// restoreAppConfig puts the process-global config back after a test mutates it.
// Without this, whichever test ran last silently dictates configuration for
// every test that follows — a trap this codebase has been bitten by before.
func restoreAppConfig(t *testing.T) {
	t.Helper()
	prev := config.AppConfig
	t.Cleanup(func() { config.AppConfig = prev })
}

// Only one reset link per address may be live at a time.
//
// Identity Platform kills an earlier code as soon as a newer one is minted, so
// an older link in the inbox is already dead on the provider side. If our
// wrapper token outlived it, following that link would bounce the user to the
// provider's bare "expired or already used" page instead of ours.
func TestLatestKeyIsPerAddressAndHashed(t *testing.T) {
	a := resetLatestKey("Chef@Example.com")
	b := resetLatestKey("chef@example.com")
	if a != b {
		t.Fatal("the pointer must be case-insensitive, or a re-request would not retire the old link")
	}
	if strings.Contains(a, "chef@example.com") {
		t.Fatal("the address must never appear in a Redis key")
	}
	if a == resetLatestKey("someone@else.com") {
		t.Fatal("two addresses must not share a pointer")
	}
	if !strings.HasPrefix(a, "pwreset:latest:") {
		t.Fatalf("unexpected key shape: %s", a)
	}
}
