package services

// apple_signin.go — the service-account side of Sign in with Apple.
//
// Why this exists: App Review guideline 5.1.1(v) requires that an app offering
// Sign in with Apple revoke the user's Apple tokens when they delete their
// account. Deleting the Identity Platform user (services/gip_admin.go) removes
// OUR credential, but Apple keeps its own record of the grant — the app stays
// listed under Settings → Apple ID → Sign in with Apple, and a reviewer who
// deletes the test account and looks there will see it and reject the build.
//
// Revocation needs a refresh token, and a refresh token only ever exists if we
// exchanged the one-shot authorization code Apple hands the client at sign-in.
// The mobile apps therefore post that code to /v1/auth/apple/link right after a
// successful sign-in; we exchange it here and keep the refresh token encrypted
// on the user row until deletion.
//
// Every entry point degrades to a no-op when the app is not configured for
// Apple (empty team/key/client id or key material), so local dev, tests and the
// Android-only path are unaffected.
//
// Two apps, two client ids: a native authorization code is bound to the App ID
// that issued it, so Apple's /auth/token and /auth/revoke reject a
// customer-app code presented with the vendor app's client_id and vice versa,
// even though both apps share one Team ID and one Sign in with Apple private
// key (config.AppConfig.AppleTeamID / AppleSignInPrivateKey). Every function
// here that talks to Apple therefore takes the user's models.UserRole and
// resolves the matching client_id via appleClientIDForRole — RoleCustomer for
// apps/mobile-customer, RoleChef for apps/mobile-vendor (that role is set at
// account creation from the GIP business pool, not deferred until kitchen
// approval — see auth-bff's defaultRoleForPool). Any other role has no client
// id mapped and degrades to ErrAppleNotConfigured, the same safe no-op as a
// fully unconfigured deployment, rather than guessing and sending a wrong
// client_id to Apple.

import (
	"context"
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"gorm.io/gorm"

	"github.com/homechef/api/config"
	"github.com/homechef/api/models"
)

// Apple's endpoints. Vars rather than consts so tests can point them at an
// httptest server; nothing else reassigns them.
var (
	appleTokenURL  = "https://appleid.apple.com/auth/token"
	appleRevokeURL = "https://appleid.apple.com/auth/revoke"
)

const (
	appleAudience = "https://appleid.apple.com"

	// appleClientSecretTTL is how long the generated client_secret JWT is valid.
	// Apple caps it at 6 months; we mint one per call, so minutes are plenty and
	// a short life limits the blast radius if one ever leaks into a log.
	appleClientSecretTTL = 5 * time.Minute
)

// ErrAppleNotConfigured means the Sign in with Apple service account is not set
// up. Callers treat it as "nothing to do", never as a failure.
var ErrAppleNotConfigured = errors.New("apple: sign in with apple is not configured")

// appleHTTPClient is overridable in tests.
var appleHTTPClient = &http.Client{Timeout: 15 * time.Second}

// AppleSignInConfigured reports whether the shared Apple service-account
// material is set AND at least one app has a client id — a coarse "is Sign in
// with Apple wired up for anything at all" check.
//
// Nil-safe: AppConfig is only populated by config.Load, and the deletion path
// this feeds must degrade to "no revocation" rather than panic in any context
// that skips it.
//
// This does not tell you whether a SPECIFIC app is configured — two apps have
// two independent client ids, and one can be set while the other is not. Call
// sites that know which user/app they are acting for must use
// AppleSignInConfiguredForRole instead; this is kept for the coarse checks
// (docs, tests) that predate the per-app split.
func AppleSignInConfigured() bool {
	cfg := config.AppConfig
	if !appleSharedConfigured(cfg) {
		return false
	}
	return cfg.AppleServicesClientID != "" || cfg.AppleServicesClientIDVendor != ""
}

// AppleSignInConfiguredForRole reports whether enough is set to talk to Apple
// on behalf of the app the given role belongs to.
func AppleSignInConfiguredForRole(role models.UserRole) bool {
	if !appleSharedConfigured(config.AppConfig) {
		return false
	}
	_, ok := appleClientIDForRole(role)
	return ok
}

// appleSharedConfigured checks the pieces every app shares: team id, key id,
// and the private key material. Nil-safe for the same reason as
// AppleSignInConfigured above.
func appleSharedConfigured(cfg *config.Config) bool {
	if cfg == nil {
		return false
	}
	return cfg.AppleTeamID != "" &&
		cfg.AppleKeyID != "" &&
		len(cfg.AppleSignInPrivateKey) > 0
}

// appleClientIDForRole resolves which Apple client_id (== that app's bundle
// id) to present to Apple for a given user's role.
//
// RoleCustomer maps to apps/mobile-customer, RoleChef to apps/mobile-vendor —
// the only two apps with Sign in with Apple enabled (both app.json list
// usesAppleSignIn: true). Every other role (delivery, admin, fleet manager)
// has no Sign in with Apple integration and returns ok=false, matching an
// unconfigured deployment rather than guessing a client_id that would be
// wrong for that user and get rejected by Apple as invalid_client.
func appleClientIDForRole(role models.UserRole) (clientID string, ok bool) {
	cfg := config.AppConfig
	if cfg == nil {
		return "", false
	}
	switch role {
	case models.RoleCustomer:
		if cfg.AppleServicesClientID != "" {
			return cfg.AppleServicesClientID, true
		}
	case models.RoleChef:
		if cfg.AppleServicesClientIDVendor != "" {
			return cfg.AppleServicesClientIDVendor, true
		}
	}
	return "", false
}

// appleClientSecret mints the ES256-signed JWT Apple accepts in place of a
// static client secret. Apple requires: iss=team id, aud=appleid.apple.com,
// sub=client id, and the key id in the header.
//
// clientID is the caller's already-resolved per-app client id (see
// appleClientIDForRole) — the client_secret's sub claim must equal whatever
// client_id accompanies it in the same request, or Apple answers
// invalid_client.
func appleClientSecret(clientID string) (string, error) {
	cfg := config.AppConfig
	if !appleSharedConfigured(cfg) {
		return "", ErrAppleNotConfigured
	}

	key, err := parseApplePrivateKey(cfg.AppleSignInPrivateKey)
	if err != nil {
		return "", err
	}

	now := time.Now()
	token := jwt.NewWithClaims(jwt.SigningMethodES256, jwt.RegisteredClaims{
		Issuer:    cfg.AppleTeamID,
		Subject:   clientID,
		Audience:  jwt.ClaimStrings{appleAudience},
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(appleClientSecretTTL)),
	})
	token.Header["kid"] = cfg.AppleKeyID

	signed, err := token.SignedString(key)
	if err != nil {
		return "", fmt.Errorf("apple: sign client secret: %w", err)
	}
	return signed, nil
}

// parseApplePrivateKey decodes the .p8 PKCS#8 ECDSA key Apple issues.
func parseApplePrivateKey(raw []byte) (*ecdsa.PrivateKey, error) {
	block, _ := pem.Decode(raw)
	if block == nil {
		return nil, fmt.Errorf("apple: private key is not valid PEM")
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("apple: parse private key: %w", err)
	}
	key, ok := parsed.(*ecdsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("apple: private key is %T, want ECDSA", parsed)
	}
	return key, nil
}

// ExchangeAppleAuthCode trades the one-shot authorization code the client got
// from Apple for a refresh token, which is the only credential /auth/revoke
// accepts on a long-lived basis.
//
// role selects which app's client_id to present — the authorization code was
// minted for a specific bundle id, and role must be the role of the user who
// produced it (see the file-level comment for why a wrong client_id fails
// with invalid_client rather than silently working).
//
// The authorization code expires in ~5 minutes and is single-use, so this must
// be called promptly after sign-in and must not be retried with the same code.
func ExchangeAppleAuthCode(ctx context.Context, authCode string, role models.UserRole) (string, error) {
	if authCode == "" {
		return "", fmt.Errorf("apple: authorization code is required")
	}
	clientID, ok := appleClientIDForRole(role)
	if !ok {
		return "", ErrAppleNotConfigured
	}

	secret, err := appleClientSecret(clientID)
	if err != nil {
		return "", err
	}

	form := url.Values{
		"client_id":     {clientID},
		"client_secret": {secret},
		"code":          {authCode},
		"grant_type":    {"authorization_code"},
	}

	body, err := applePostForm(ctx, appleTokenURL, form)
	if err != nil {
		return "", err
	}

	// Only refresh_token is read; id_token and access_token are deliberately
	// discarded — identity is already established by Identity Platform, and
	// holding more Apple credentials than revocation needs is a liability.
	var out struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := decodeAppleJSON(body, &out); err != nil {
		return "", err
	}
	if out.RefreshToken == "" {
		return "", fmt.Errorf("apple: token exchange returned no refresh token")
	}
	return out.RefreshToken, nil
}

// RevokeAppleRefreshToken revokes the user's grant with Apple. This is the call
// guideline 5.1.1(v) is actually asking for.
//
// role selects which app's client_id to present — must be the role of the
// user the refresh token was issued to (see the file-level comment).
//
// Idempotent in practice: Apple answers 200 for an already-revoked token, and
// an invalid_grant response means the grant is gone, which is the desired end
// state — both report success so the deletion path and the purge sweeper do not
// wedge retrying.
func RevokeAppleRefreshToken(ctx context.Context, refreshToken string, role models.UserRole) error {
	if refreshToken == "" {
		return nil // nothing was ever stored — e.g. an Android or email signup
	}
	clientID, ok := appleClientIDForRole(role)
	if !ok {
		return ErrAppleNotConfigured
	}

	secret, err := appleClientSecret(clientID)
	if err != nil {
		return err
	}

	form := url.Values{
		"client_id":       {clientID},
		"client_secret":   {secret},
		"token":           {refreshToken},
		"token_type_hint": {"refresh_token"},
	}

	if _, err := applePostForm(ctx, appleRevokeURL, form); err != nil {
		if strings.Contains(err.Error(), "invalid_grant") {
			return nil // already revoked or expired — the grant is gone either way
		}
		return err
	}
	return nil
}

// LinkAppleGrant exchanges a fresh authorization code and stores the resulting
// refresh token against the user, so deletion can revoke it later.
//
// The authorization code is bound to the app the user signed in on, which
// this resolves from user.Role rather than any request header — the request
// body only ever carried authorizationCode (see the handler), and role is
// durably known for the authenticated user by the time this is called. See
// the file-level comment for why the client_id must match.
//
// Best-effort by design: a user who signs in successfully must not be blocked
// because Apple's token endpoint was briefly unhappy. A failure here degrades
// to "we cannot revoke on delete", which is logged and visible, rather than a
// failed login.
func LinkAppleGrant(ctx context.Context, db *gorm.DB, user *models.User, authCode string) error {
	refreshToken, err := ExchangeAppleAuthCode(ctx, authCode, user.Role)
	if err != nil {
		return err
	}

	if err := db.WithContext(ctx).Model(&models.User{}).
		Where("id = ?", user.ID).
		Update("apple_refresh_token_enc", models.EncryptedString(refreshToken)).Error; err != nil {
		return fmt.Errorf("apple: persist refresh token: %w", err)
	}
	user.AppleRefreshTokenEnc = models.EncryptedString(refreshToken)
	return nil
}

// RevokeAppleGrantForUser revokes the user's Apple grant if one was ever
// recorded. Best-effort and always safe to call: it no-ops for users who never
// used Sign in with Apple and for deployments with no Apple key configured for
// this user's app (user.Role — see the file-level comment).
//
// Errors are logged, never returned — every caller is on the deletion path,
// where failing the user's deletion request because Apple was unreachable is
// the worse outcome. The purge sweeper calls this again before erasing.
func RevokeAppleGrantForUser(ctx context.Context, user *models.User) {
	token := user.AppleRefreshTokenEnc.String()
	if token == "" {
		return
	}
	if err := RevokeAppleRefreshToken(ctx, token, user.Role); err != nil {
		if errors.Is(err, ErrAppleNotConfigured) {
			// Nothing wired up for this user's app — the startup warning
			// (config.warnIfAppleSignInIncomplete) already surfaced this once;
			// logging it again on every deletion would just be noise.
			return
		}
		log.Printf("apple: token revocation failed for user=%s: %v", user.ID, err)
		return
	}
	log.Printf("apple: revoked Sign in with Apple grant for user=%s", user.ID)
}

// decodeAppleJSON unmarshals an Apple response body into out.
func decodeAppleJSON(body []byte, out any) error {
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("apple: decode response: %w", err)
	}
	return nil
}

// applePostForm posts a form to Apple and returns the body on success.
func applePostForm(ctx context.Context, endpoint string, form url.Values) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("apple: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := appleHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("apple: request %s: %w", endpoint, err)
	}
	defer resp.Body.Close()

	payload, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<10))
	if resp.StatusCode != http.StatusOK {
		// The body carries Apple's machine-readable reason (invalid_client,
		// invalid_grant, …) and is the only thing that makes a production
		// failure diagnosable. It never contains the user's tokens — those are
		// request-side only.
		return nil, fmt.Errorf("apple: %s: status %d: %s",
			endpoint, resp.StatusCode, strings.TrimSpace(string(payload)))
	}
	return payload, nil
}
