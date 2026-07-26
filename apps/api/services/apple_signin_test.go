package services

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/golang-jwt/jwt/v4"

	"github.com/homechef/api/config"
)

// testApplePrivateKey generates a throwaway P-256 key in the PKCS#8 PEM shape
// Apple's .p8 files use.
func testApplePrivateKey(t *testing.T) []byte {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatalf("marshal key: %v", err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})
}

// configureApple wires a complete Apple service account for the duration of a
// test and restores whatever was there before. AppConfig is a pointer that only
// config.Load populates, so tests must supply their own.
func configureApple(t *testing.T) {
	t.Helper()
	prev := config.AppConfig
	t.Cleanup(func() { config.AppConfig = prev })

	config.AppConfig = &config.Config{
		AppleTeamID:           "TEAM123456",
		AppleKeyID:            "KEY1234567",
		AppleServicesClientID: "com.tesserix.homechef.customer",
		AppleSignInPrivateKey: testApplePrivateKey(t),
	}
}

// swapAppleClient points both Apple endpoints at a test server and returns a
// function restoring the originals.
func swapAppleClient(t *testing.T, srv *httptest.Server) func() {
	t.Helper()
	prevToken, prevRevoke := appleTokenURL, appleRevokeURL
	appleTokenURL = srv.URL + "/auth/token"
	appleRevokeURL = srv.URL + "/auth/revoke"
	return func() {
		appleTokenURL, appleRevokeURL = prevToken, prevRevoke
	}
}

func TestAppleSignInConfigured(t *testing.T) {
	prev := config.AppConfig
	t.Cleanup(func() { config.AppConfig = prev })

	// Nil config — the deletion path must not panic just because nothing loaded.
	config.AppConfig = nil
	if AppleSignInConfigured() {
		t.Fatal("expected unconfigured with a nil AppConfig")
	}

	config.AppConfig = &config.Config{}
	if AppleSignInConfigured() {
		t.Fatal("expected unconfigured with no Apple settings")
	}

	configureApple(t)
	if !AppleSignInConfigured() {
		t.Fatal("expected configured once all four settings are present")
	}

	// A partial configuration must not count — minting a client secret without
	// key material would fail at call time instead of degrading to a no-op.
	config.AppConfig.AppleSignInPrivateKey = nil
	if AppleSignInConfigured() {
		t.Fatal("expected unconfigured when the private key is missing")
	}
}

func TestAppleClientSecretClaims(t *testing.T) {
	configureApple(t)

	secret, err := appleClientSecret()
	if err != nil {
		t.Fatalf("appleClientSecret: %v", err)
	}

	// Parse without verifying the signature — the claims and the kid header are
	// what Apple validates the request against, and they are what regress.
	parser := jwt.NewParser()
	claims := jwt.RegisteredClaims{}
	token, _, err := parser.ParseUnverified(secret, &claims)
	if err != nil {
		t.Fatalf("parse client secret: %v", err)
	}

	if got := token.Header["kid"]; got != "KEY1234567" {
		t.Errorf("kid header = %v, want KEY1234567", got)
	}
	if token.Method.Alg() != "ES256" {
		t.Errorf("alg = %s, want ES256", token.Method.Alg())
	}
	if claims.Issuer != "TEAM123456" {
		t.Errorf("iss = %s, want the team id", claims.Issuer)
	}
	if claims.Subject != "com.tesserix.homechef.customer" {
		t.Errorf("sub = %s, want the client id", claims.Subject)
	}
	if len(claims.Audience) != 1 || claims.Audience[0] != appleAudience {
		t.Errorf("aud = %v, want [%s]", claims.Audience, appleAudience)
	}
	if claims.ExpiresAt == nil || !claims.ExpiresAt.After(claims.IssuedAt.Time) {
		t.Error("client secret must expire after it was issued")
	}
}

func TestRevokeAppleRefreshTokenEmptyTokenIsNoop(t *testing.T) {
	configureApple(t)

	// No stored token — e.g. an Android user, or an account created before the
	// grant was captured. Must not call Apple and must not error.
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	restore := swapAppleClient(t, srv)
	defer restore()

	if err := RevokeAppleRefreshToken(context.Background(), ""); err != nil {
		t.Fatalf("expected no error for empty token, got %v", err)
	}
	if called {
		t.Fatal("empty token must not reach Apple")
	}
}

func TestRevokeAppleRefreshTokenNotConfigured(t *testing.T) {
	configureApple(t)
	config.AppConfig.AppleSignInPrivateKey = nil

	err := RevokeAppleRefreshToken(context.Background(), "some-refresh-token")
	if err != ErrAppleNotConfigured {
		t.Fatalf("want ErrAppleNotConfigured, got %v", err)
	}
}

func TestRevokeAppleRefreshTokenSendsExpectedForm(t *testing.T) {
	configureApple(t)

	var gotForm url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		gotForm = r.PostForm
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	restore := swapAppleClient(t, srv)
	defer restore()

	if err := RevokeAppleRefreshToken(context.Background(), "rt-abc"); err != nil {
		t.Fatalf("RevokeAppleRefreshToken: %v", err)
	}

	if got := gotForm.Get("token"); got != "rt-abc" {
		t.Errorf("token = %q, want rt-abc", got)
	}
	if got := gotForm.Get("token_type_hint"); got != "refresh_token" {
		t.Errorf("token_type_hint = %q, want refresh_token", got)
	}
	if got := gotForm.Get("client_id"); got != "com.tesserix.homechef.customer" {
		t.Errorf("client_id = %q", got)
	}
	if gotForm.Get("client_secret") == "" {
		t.Error("client_secret must be present")
	}
}

// An already-revoked or expired grant comes back as invalid_grant. The end state
// Apple asks for is "the grant is gone", which is already true — so this must
// report success, or the deletion path and the purge sweeper retry forever.
func TestRevokeAppleRefreshTokenTreatsInvalidGrantAsSuccess(t *testing.T) {
	configureApple(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"invalid_grant"}`))
	}))
	defer srv.Close()
	restore := swapAppleClient(t, srv)
	defer restore()

	if err := RevokeAppleRefreshToken(context.Background(), "stale-token"); err != nil {
		t.Fatalf("invalid_grant must be treated as success, got %v", err)
	}
}

func TestRevokeAppleRefreshTokenSurfacesRealFailures(t *testing.T) {
	configureApple(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"invalid_client"}`))
	}))
	defer srv.Close()
	restore := swapAppleClient(t, srv)
	defer restore()

	err := RevokeAppleRefreshToken(context.Background(), "rt")
	if err == nil {
		t.Fatal("invalid_client is a real misconfiguration and must surface")
	}
	if !strings.Contains(err.Error(), "invalid_client") {
		t.Errorf("error should quote Apple's reason, got %v", err)
	}
}

func TestExchangeAppleAuthCodeReturnsRefreshToken(t *testing.T) {
	configureApple(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		if r.PostForm.Get("grant_type") != "authorization_code" {
			t.Errorf("grant_type = %q", r.PostForm.Get("grant_type"))
		}
		if r.PostForm.Get("code") != "auth-code-1" {
			t.Errorf("code = %q", r.PostForm.Get("code"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"refresh_token":"rt-xyz","access_token":"at","id_token":"it"}`))
	}))
	defer srv.Close()
	restore := swapAppleClient(t, srv)
	defer restore()

	got, err := ExchangeAppleAuthCode(context.Background(), "auth-code-1")
	if err != nil {
		t.Fatalf("ExchangeAppleAuthCode: %v", err)
	}
	if got != "rt-xyz" {
		t.Errorf("refresh token = %q, want rt-xyz", got)
	}
}

// Apple can answer 200 with no refresh_token if the code was already consumed.
// Storing an empty string would silently disable revocation for that user, so
// this must be an error the caller logs.
func TestExchangeAppleAuthCodeRejectsMissingRefreshToken(t *testing.T) {
	configureApple(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"access_token":"at"}`))
	}))
	defer srv.Close()
	restore := swapAppleClient(t, srv)
	defer restore()

	if _, err := ExchangeAppleAuthCode(context.Background(), "code"); err == nil {
		t.Fatal("expected an error when Apple returns no refresh token")
	}
}

func TestParseApplePrivateKeyRejectsGarbage(t *testing.T) {
	if _, err := parseApplePrivateKey([]byte("not a pem block")); err == nil {
		t.Fatal("expected an error for non-PEM input")
	}
	// A valid PEM carrying the wrong key type must also be rejected rather than
	// panicking on the type assertion.
	if _, err := parseApplePrivateKey(pem.EncodeToMemory(&pem.Block{
		Type: "PRIVATE KEY", Bytes: []byte("junk"),
	})); err == nil {
		t.Fatal("expected an error for an unparseable PKCS#8 body")
	}
}

func TestApplePrivateKeyRoundTripsBase64(t *testing.T) {
	// config.Load stores the key base64-decoded; make sure a real .p8 survives
	// that trip, since a silent decode failure disables revocation entirely.
	pemBytes := testApplePrivateKey(t)
	encoded := base64.StdEncoding.EncodeToString(pemBytes)
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if _, err := parseApplePrivateKey(decoded); err != nil {
		t.Fatalf("parse round-tripped key: %v", err)
	}
}
