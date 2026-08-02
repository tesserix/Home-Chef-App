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
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/homechef/api/config"
	"github.com/homechef/api/models"
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

// appleCustomerClientID / appleVendorClientID are the per-app client ids used
// throughout this file — real bundle ids, matching apps/mobile-customer's and
// apps/mobile-vendor's app.json, so a regression that mixed them up would
// produce recognizable wrong values in a test failure instead of opaque ones.
const (
	appleCustomerClientID = "com.tesserix.homechef.customer"
	appleVendorClientID   = "com.tesserix.homechef.vendor"
)

// configureApple wires a complete Apple service account — BOTH apps' client
// ids — for the duration of a test and restores whatever was there before.
// AppConfig is a pointer that only config.Load populates, so tests must
// supply their own.
func configureApple(t *testing.T) {
	t.Helper()
	prev := config.AppConfig
	t.Cleanup(func() { config.AppConfig = prev })

	config.AppConfig = &config.Config{
		AppleTeamID:                 "TEAM123456",
		AppleKeyID:                  "KEY1234567",
		AppleServicesClientID:       appleCustomerClientID,
		AppleServicesClientIDVendor: appleVendorClientID,
		AppleSignInPrivateKey:       testApplePrivateKey(t),
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
		t.Fatal("expected configured once shared settings + at least one client id are present")
	}

	// A partial configuration must not count — minting a client secret without
	// key material would fail at call time instead of degrading to a no-op.
	config.AppConfig.AppleSignInPrivateKey = nil
	if AppleSignInConfigured() {
		t.Fatal("expected unconfigured when the private key is missing")
	}
}

// Only one app's client id needs to be set for the coarse AppleSignInConfigured
// check to report true — the two apps are independently configurable, and this
// deliberately does NOT tell you which one.
func TestAppleSignInConfiguredWithOnlyOneAppClientID(t *testing.T) {
	prev := config.AppConfig
	t.Cleanup(func() { config.AppConfig = prev })

	config.AppConfig = &config.Config{
		AppleTeamID:           "TEAM123456",
		AppleKeyID:            "KEY1234567",
		AppleServicesClientID: appleCustomerClientID, // vendor left unset
		AppleSignInPrivateKey: testApplePrivateKey(t),
	}
	if !AppleSignInConfigured() {
		t.Fatal("expected configured — the customer client id alone is enough for the coarse check")
	}
}

// AppleSignInConfiguredForRole is the per-app check real call sites should
// use: unlike AppleSignInConfigured it must say NO for the app whose client
// id is missing, even while the other app is fully configured.
func TestAppleSignInConfiguredForRole(t *testing.T) {
	prev := config.AppConfig
	t.Cleanup(func() { config.AppConfig = prev })

	config.AppConfig = &config.Config{
		AppleTeamID:           "TEAM123456",
		AppleKeyID:            "KEY1234567",
		AppleServicesClientID: appleCustomerClientID, // vendor left unset
		AppleSignInPrivateKey: testApplePrivateKey(t),
	}

	if !AppleSignInConfiguredForRole(models.RoleCustomer) {
		t.Error("expected the customer app to be configured")
	}
	if AppleSignInConfiguredForRole(models.RoleChef) {
		t.Error("expected the vendor app to be NOT configured — its client id is unset")
	}
	if AppleSignInConfiguredForRole(models.RoleDelivery) {
		t.Error("delivery has no Sign in with Apple integration at all")
	}
}

// appleClientIDForRole is the resolver every Apple-facing call goes through.
// Only customer and chef map to an app; every other role must come back
// unresolved rather than falling back to either app's id.
func TestAppleClientIDForRole(t *testing.T) {
	configureApple(t)

	if id, ok := appleClientIDForRole(models.RoleCustomer); !ok || id != appleCustomerClientID {
		t.Errorf("RoleCustomer resolved to (%q, %v), want (%q, true)", id, ok, appleCustomerClientID)
	}
	if id, ok := appleClientIDForRole(models.RoleChef); !ok || id != appleVendorClientID {
		t.Errorf("RoleChef resolved to (%q, %v), want (%q, true)", id, ok, appleVendorClientID)
	}
	for _, role := range []models.UserRole{models.RoleDelivery, models.RoleAdmin, models.RoleFleetManager, ""} {
		if id, ok := appleClientIDForRole(role); ok {
			t.Errorf("role %q resolved to client id %q, want unresolved (no app owns this role)", role, id)
		}
	}
}

func TestAppleClientSecretClaims(t *testing.T) {
	configureApple(t)

	secret, err := appleClientSecret(appleCustomerClientID)
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
	if claims.Subject != appleCustomerClientID {
		t.Errorf("sub = %s, want the client id passed in", claims.Subject)
	}
	if len(claims.Audience) != 1 || claims.Audience[0] != appleAudience {
		t.Errorf("aud = %v, want [%s]", claims.Audience, appleAudience)
	}
	if claims.ExpiresAt == nil || !claims.ExpiresAt.After(claims.IssuedAt.Time) {
		t.Error("client secret must expire after it was issued")
	}
}

// The client secret's sub claim must track whichever client_id the caller
// resolved — this is what lets one shared private key serve two apps.
func TestAppleClientSecretUsesVendorClientID(t *testing.T) {
	configureApple(t)

	secret, err := appleClientSecret(appleVendorClientID)
	if err != nil {
		t.Fatalf("appleClientSecret: %v", err)
	}
	parser := jwt.NewParser()
	claims := jwt.RegisteredClaims{}
	if _, _, err := parser.ParseUnverified(secret, &claims); err != nil {
		t.Fatalf("parse client secret: %v", err)
	}
	if claims.Subject != appleVendorClientID {
		t.Errorf("sub = %s, want %s", claims.Subject, appleVendorClientID)
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

	if err := RevokeAppleRefreshToken(context.Background(), "", models.RoleCustomer); err != nil {
		t.Fatalf("expected no error for empty token, got %v", err)
	}
	if called {
		t.Fatal("empty token must not reach Apple")
	}
}

func TestRevokeAppleRefreshTokenNotConfigured(t *testing.T) {
	configureApple(t)
	config.AppConfig.AppleSignInPrivateKey = nil

	err := RevokeAppleRefreshToken(context.Background(), "some-refresh-token", models.RoleCustomer)
	if err != ErrAppleNotConfigured {
		t.Fatalf("want ErrAppleNotConfigured, got %v", err)
	}
}

// The core of the two-app fix: a role with no mapped client id (here, the
// vendor app's id is simply unset — the exact shape of "provisioned for one
// app but not the other" this bug produced in prod) must NOT fall back to
// the other app's client_id and must NOT reach Apple at all.
func TestRevokeAppleRefreshTokenWrongAppDoesNotSendBadClientID(t *testing.T) {
	prev := config.AppConfig
	t.Cleanup(func() { config.AppConfig = prev })
	config.AppConfig = &config.Config{
		AppleTeamID:           "TEAM123456",
		AppleKeyID:            "KEY1234567",
		AppleServicesClientID: appleCustomerClientID, // vendor deliberately unset
		AppleSignInPrivateKey: testApplePrivateKey(t),
	}

	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	restore := swapAppleClient(t, srv)
	defer restore()

	err := RevokeAppleRefreshToken(context.Background(), "vendor-rt", models.RoleChef)
	if err != ErrAppleNotConfigured {
		t.Fatalf("want ErrAppleNotConfigured for an app with no client id, got %v", err)
	}
	if called {
		t.Fatal("must not reach Apple at all when the app's client id is unresolved — " +
			"sending the customer client_id for a vendor grant would be a wrong client_id, not a safe no-op")
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

	if err := RevokeAppleRefreshToken(context.Background(), "rt-abc", models.RoleCustomer); err != nil {
		t.Fatalf("RevokeAppleRefreshToken: %v", err)
	}

	if got := gotForm.Get("token"); got != "rt-abc" {
		t.Errorf("token = %q, want rt-abc", got)
	}
	if got := gotForm.Get("token_type_hint"); got != "refresh_token" {
		t.Errorf("token_type_hint = %q, want refresh_token", got)
	}
	if got := gotForm.Get("client_id"); got != appleCustomerClientID {
		t.Errorf("client_id = %q, want %q", got, appleCustomerClientID)
	}
	if gotForm.Get("client_secret") == "" {
		t.Error("client_secret must be present")
	}
}

// Mirror of the above for the vendor app — the two client ids must not get
// crossed in either direction.
func TestRevokeAppleRefreshTokenSendsVendorClientID(t *testing.T) {
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

	if err := RevokeAppleRefreshToken(context.Background(), "rt-vendor", models.RoleChef); err != nil {
		t.Fatalf("RevokeAppleRefreshToken: %v", err)
	}
	if got := gotForm.Get("client_id"); got != appleVendorClientID {
		t.Errorf("client_id = %q, want %q", got, appleVendorClientID)
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

	if err := RevokeAppleRefreshToken(context.Background(), "stale-token", models.RoleCustomer); err != nil {
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

	err := RevokeAppleRefreshToken(context.Background(), "rt", models.RoleCustomer)
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
		if got := r.PostForm.Get("client_id"); got != appleCustomerClientID {
			t.Errorf("client_id = %q, want %q", got, appleCustomerClientID)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"refresh_token":"rt-xyz","access_token":"at","id_token":"it"}`))
	}))
	defer srv.Close()
	restore := swapAppleClient(t, srv)
	defer restore()

	got, err := ExchangeAppleAuthCode(context.Background(), "auth-code-1", models.RoleCustomer)
	if err != nil {
		t.Fatalf("ExchangeAppleAuthCode: %v", err)
	}
	if got != "rt-xyz" {
		t.Errorf("refresh token = %q, want rt-xyz", got)
	}
}

// The vendor-app mirror of the above, exercising the same code path with the
// other client id so a regression that hardcoded the customer id anywhere in
// the exchange path fails loudly.
func TestExchangeAppleAuthCodeUsesVendorClientID(t *testing.T) {
	configureApple(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		if got := r.PostForm.Get("client_id"); got != appleVendorClientID {
			t.Errorf("client_id = %q, want %q", got, appleVendorClientID)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"refresh_token":"rt-vendor"}`))
	}))
	defer srv.Close()
	restore := swapAppleClient(t, srv)
	defer restore()

	got, err := ExchangeAppleAuthCode(context.Background(), "auth-code-vendor", models.RoleChef)
	if err != nil {
		t.Fatalf("ExchangeAppleAuthCode: %v", err)
	}
	if got != "rt-vendor" {
		t.Errorf("refresh token = %q, want rt-vendor", got)
	}
}

// The exchange-side twin of TestRevokeAppleRefreshTokenWrongAppDoesNotSendBadClientID:
// a role with no configured client id must fail closed before ever building a
// request, not fall back to whichever id happens to be set.
func TestExchangeAppleAuthCodeWrongAppDoesNotSendBadClientID(t *testing.T) {
	prev := config.AppConfig
	t.Cleanup(func() { config.AppConfig = prev })
	config.AppConfig = &config.Config{
		AppleTeamID:           "TEAM123456",
		AppleKeyID:            "KEY1234567",
		AppleServicesClientID: appleCustomerClientID, // vendor deliberately unset
		AppleSignInPrivateKey: testApplePrivateKey(t),
	}

	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	restore := swapAppleClient(t, srv)
	defer restore()

	_, err := ExchangeAppleAuthCode(context.Background(), "auth-code-vendor", models.RoleChef)
	if err != ErrAppleNotConfigured {
		t.Fatalf("want ErrAppleNotConfigured, got %v", err)
	}
	if called {
		t.Fatal("must not reach Apple with an unresolved client id")
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

	if _, err := ExchangeAppleAuthCode(context.Background(), "code", models.RoleCustomer); err == nil {
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

// LinkAppleGrant and RevokeAppleGrantForUser are what the real call sites use
// (handlers/apple_signin.go's LinkGrant, handlers/account_lifecycle.go's
// Delete, and services/account_purge_cron.go's purgeOneAccount) — both take a
// *models.User rather than a bare role, so these exercise the actual
// discrimination path: user.Role, not a request header, since
// RevokeAppleGrantForUser in particular runs with no HTTP request in scope at
// all (the deletion handler calls it post-commit, the purge cron calls it from
// a batch sweep).

func TestLinkAppleGrantUsesUserRoleClientID(t *testing.T) {
	configureApple(t)
	db := setupAccountDB(t)

	userID := uuid.New()
	require.NoError(t, db.Exec(
		`INSERT INTO users (id, email, first_name, last_name, role) VALUES (?, ?, ?, ?, ?)`,
		userID.String(), "chef@example.com", "Chef", "Test", "chef",
	).Error)

	var gotForm url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		gotForm = r.PostForm
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"refresh_token":"rt-vendor-link"}`))
	}))
	defer srv.Close()
	restore := swapAppleClient(t, srv)
	defer restore()

	user := &models.User{ID: userID, Role: models.RoleChef}
	if err := LinkAppleGrant(context.Background(), db, user, "auth-code"); err != nil {
		t.Fatalf("LinkAppleGrant: %v", err)
	}

	if got := gotForm.Get("client_id"); got != appleVendorClientID {
		t.Errorf("client_id = %q, want %q — a chef's authorization code must be exchanged with the vendor app's id", got, appleVendorClientID)
	}
	if user.AppleRefreshTokenEnc.String() != "rt-vendor-link" {
		t.Errorf("in-memory user not updated, got %q", user.AppleRefreshTokenEnc.String())
	}

	var stored string
	require.NoError(t, db.Raw(`SELECT apple_refresh_token_enc FROM users WHERE id = ?`, userID.String()).Scan(&stored).Error)
	if stored != "rt-vendor-link" {
		t.Errorf("persisted refresh token = %q, want rt-vendor-link", stored)
	}
}

// This is the direct assertion that the deletion path reaches
// RevokeAppleGrantForUser with the right effect: both real callers
// (handlers/account_lifecycle.go:191 and
// services/account_purge_cron.go:127) call this exact function with the
// user row they already loaded, with no other app signal available.
func TestRevokeAppleGrantForUserUsesUserRoleClientID(t *testing.T) {
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

	user := &models.User{
		ID:                   uuid.New(),
		Role:                 models.RoleChef,
		AppleRefreshTokenEnc: models.EncryptedString("stored-vendor-rt"),
	}
	RevokeAppleGrantForUser(context.Background(), user)

	if got := gotForm.Get("client_id"); got != appleVendorClientID {
		t.Errorf("client_id = %q, want %q", got, appleVendorClientID)
	}
	if got := gotForm.Get("token"); got != "stored-vendor-rt" {
		t.Errorf("token = %q, want stored-vendor-rt", got)
	}
}

func TestRevokeAppleGrantForUserNoTokenIsNoop(t *testing.T) {
	configureApple(t)
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	restore := swapAppleClient(t, srv)
	defer restore()

	RevokeAppleGrantForUser(context.Background(), &models.User{ID: uuid.New(), Role: models.RoleCustomer})
	if called {
		t.Fatal("a user with no stored Apple refresh token must not reach Apple")
	}
}

// Same wrong-client-id guarantee as TestRevokeAppleRefreshTokenWrongAppDoesNotSendBadClientID,
// but through the actual deletion-path entry point.
func TestRevokeAppleGrantForUserUnresolvedRoleDoesNotSendBadClientID(t *testing.T) {
	prev := config.AppConfig
	t.Cleanup(func() { config.AppConfig = prev })
	config.AppConfig = &config.Config{
		AppleTeamID:           "TEAM123456",
		AppleKeyID:            "KEY1234567",
		AppleServicesClientID: appleCustomerClientID, // vendor deliberately unset
		AppleSignInPrivateKey: testApplePrivateKey(t),
	}

	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	restore := swapAppleClient(t, srv)
	defer restore()

	user := &models.User{
		ID:                   uuid.New(),
		Role:                 models.RoleChef,
		AppleRefreshTokenEnc: models.EncryptedString("rt"),
	}
	// Must not panic, must not call Apple, and — since this runs on the
	// deletion path — must not be surfaced as an error to the caller either.
	RevokeAppleGrantForUser(context.Background(), user)

	if called {
		t.Fatal("must not reach Apple with the wrong app's client id")
	}
}
