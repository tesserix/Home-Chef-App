package services

// password_reset.go — self-service password reset for the mobile apps and
// portals.
//
// WHY THIS EXISTS AT ALL, rather than just calling Firebase's
// sendPasswordResetEmail from the client:
//
// Firebase happily mints the token but its delivery is unusable for this
// product. It sends from noreply@<project>.firebaseapp.com — a domain with no
// SPF/DKIM alignment to ours — so Gmail files it as spam ("previous messages
// from firebaseapp.com were marked as spam"). The message it composes is
// branded with the GCP project name ("Reset your password for Tesserix") and
// pastes a raw firebaseapp.com URL into the body, which reads as phishing to
// anyone who looks at it.
//
// So we split the job: Identity Platform stays the authority for minting and
// validating the credential, and we own delivery — our authenticated sender
// (noreply@tesserix.app, SPF+DKIM+DMARC verified), our branded template, and a
// link on our own domain.
//
// The link the user receives is OURS, not Firebase's. It carries an opaque
// single-use token that we exchange for the Firebase link at click time. That
// buys three things Firebase alone could not:
//
//   - A real 15-minute expiry. Firebase's own oobCode lives about an hour and
//     that is not configurable per request.
//   - Single use. The token is deleted the moment it is redeemed, so a link
//     sitting in an inbox (or a forwarded mail) cannot be replayed.
//   - A trustworthy hostname, which is most of why the mail lands at all.

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/homechef/api/config"
)

const (
	// PasswordResetTTL bounds how long a reset link works. Short enough that a
	// mail left open on a shared machine goes stale quickly, long enough for
	// someone to find the mail and choose a password without being rushed.
	PasswordResetTTL = 15 * time.Minute

	// Rate limits, keyed separately so one noisy address cannot lock out a
	// whole NAT'd office, and a bored attacker cannot walk the user table.
	passwordResetMaxPerEmail  = 3
	passwordResetEmailWindow  = 15 * time.Minute
	passwordResetMaxPerClient = 10
	passwordResetClientWindow = time.Hour
)

// ErrPasswordResetRateLimited means the caller asked too often. The HTTP layer
// still answers generically — see RequestPasswordReset.
var ErrPasswordResetRateLimited = errors.New("too many password reset requests")

// ErrResetTokenInvalid covers expired, unknown, and already-used tokens alike.
// They are deliberately indistinguishable to the caller.
var ErrResetTokenInvalid = errors.New("this reset link is no longer valid")

// TenantForApp maps a client app to the Identity Platform tenant its accounts
// live in.
//
// This matters more than it looks. GIP accounts are tenant-scoped, so asking
// for a reset in the wrong app finds nothing and sends nothing — which is
// exactly how a vendor resetting from the customer app ends up staring at a
// "check your inbox" message for a mail that was never going to arrive.
// Unknown apps resolve to the customer tenant, the largest audience.
func TenantForApp(app string) string {
	switch strings.ToLower(strings.TrimSpace(app)) {
	case "vendor", "chef", "business":
		return config.AppConfig.GIPBusinessTenantID
	case "delivery", "driver":
		return config.AppConfig.GIPDeliveryTenantID
	default:
		return config.AppConfig.GIPCustomerTenantID
	}
}

// resetLatestKey points at the most recent token minted for an address, so a
// re-request can retire the previous one.
//
// Identity Platform invalidates an earlier PASSWORD_RESET code as soon as a new
// one is minted for the same address — sound behaviour, but it means an older
// link in the inbox is dead while OUR wrapper token is still alive. Following
// it would bounce the user to the provider's bare "expired or already used"
// page. Retiring our own token in step keeps them on our page, which at least
// tells them what to do next.
func resetLatestKey(email string) string {
	sum := sha256.Sum256([]byte(strings.ToLower(email)))
	return "pwreset:latest:" + hex.EncodeToString(sum[:])
}

func resetTokenKey(token string) string {
	// The token is stored hashed. A Redis dump, a log line, or a support engineer
	// glancing at keys then yields nothing that can actually reset an account.
	sum := sha256.Sum256([]byte(token))
	return "pwreset:" + hex.EncodeToString(sum[:])
}

// RequestPasswordReset mints a reset link and emails it.
//
// ALWAYS returns nil for "we could not find that account". Reporting a miss
// would turn this endpoint into an account-enumeration oracle: anyone could
// discover which addresses are registered, and — because tenants are separate —
// even which app each address belongs to. Genuine infrastructure failures do
// return an error so they are visible in logs and metrics.
func RequestPasswordReset(ctx context.Context, email, app, clientIP string) error {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return nil
	}

	r := GetRedisClient()
	if r == nil || !r.IsConnected() {
		// No Redis means no single-use store and no rate limit. Refusing is the
		// safe direction: a reset flow with neither guard is worse than none.
		return fmt.Errorf("password reset unavailable: no session store")
	}

	// Rate limit before doing any work, so the expensive path cannot be used as
	// an amplifier either.
	if n, err := r.IncrAndExpire(ctx, "pwreset:rl:email:"+email, passwordResetEmailWindow); err == nil && n > passwordResetMaxPerEmail {
		return ErrPasswordResetRateLimited
	}
	if clientIP != "" {
		if n, err := r.IncrAndExpire(ctx, "pwreset:rl:ip:"+clientIP, passwordResetClientWindow); err == nil && n > passwordResetMaxPerClient {
			return ErrPasswordResetRateLimited
		}
	}

	tenantID := TenantForApp(app)
	firebaseLink, err := GenerateGIPPasswordResetLink(ctx, tenantID, email)
	if err != nil {
		if errors.Is(err, ErrGIPAccountNotFound) {
			// Deliberately silent to the caller. Logged without the address so
			// the logs themselves do not become the enumeration oracle.
			log.Printf("password-reset: no account in tenant %s for the requested address", tenantID)
			return nil
		}
		return fmt.Errorf("password reset: mint link: %w", err)
	}

	token, err := newResetToken()
	if err != nil {
		return fmt.Errorf("password reset: mint token: %w", err)
	}
	// Retire the previous link for this address before publishing the new one.
	// Only ever one live reset link per account.
	if prev, err := r.Get(ctx, resetLatestKey(email)); err == nil && prev != "" {
		_ = r.Del(ctx, prev)
	}
	if err := r.Set(ctx, resetTokenKey(token), firebaseLink, PasswordResetTTL); err != nil {
		return fmt.Errorf("password reset: store token: %w", err)
	}
	// Tracked for the same TTL — once the token is gone there is nothing to retire.
	_ = r.Set(ctx, resetLatestKey(email), resetTokenKey(token), PasswordResetTTL)

	if err := GetEmailService().SendPasswordResetLink(email, PasswordResetLinkURL(token), PasswordResetTTL); err != nil {
		// Drop the token rather than leave a live credential pointing at a mail
		// that never went out.
		_ = r.Del(ctx, resetTokenKey(token))
		return fmt.Errorf("password reset: send mail: %w", err)
	}
	return nil
}

// RedeemPasswordResetToken exchanges our opaque token for the Firebase link,
// consuming it so the same link can never be used twice.
func RedeemPasswordResetToken(ctx context.Context, token string) (string, error) {
	if strings.TrimSpace(token) == "" {
		return "", ErrResetTokenInvalid
	}
	r := GetRedisClient()
	if r == nil || !r.IsConnected() {
		return "", ErrResetTokenInvalid
	}

	key := resetTokenKey(token)
	link, err := r.Get(ctx, key)
	if err != nil || link == "" {
		return "", ErrResetTokenInvalid
	}
	// Consume BEFORE handing the link back. If the delete fails we refuse
	// rather than serve a token we cannot guarantee is single-use.
	if err := r.Del(ctx, key); err != nil {
		return "", ErrResetTokenInvalid
	}
	return link, nil
}

// PasswordResetLinkURL builds the customer-facing link. Points at our own API
// host, so the address in the mail is one the recipient recognises.
func PasswordResetLinkURL(token string) string {
	base := strings.TrimRight(config.AppConfig.PublicAPIBaseURL, "/")
	if base == "" {
		base = "https://api.fe3dr.com"
	}
	return base + "/api/v1/auth/password-reset/consume?t=" + token
}

// newResetToken returns 32 bytes of CSPRNG entropy, URL-safe. Guessing one
// inside the 15-minute window is not a realistic attack.
func newResetToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
