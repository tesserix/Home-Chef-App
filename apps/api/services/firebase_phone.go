package services

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
)

// Server-side verification of the phone factor.
//
// The client drives Firebase phone verification — Google sends the SMS, checks
// the code, and links the number to the signed-in Firebase user. The client then
// posts a FRESH Firebase ID token, whose claims now carry the verified phone
// number, and we check it here.
//
// The check is not optional decoration. Everything before this point happened on
// a device we do not control, so a client could simply claim "phone verified".
// The signature on this token is what makes the phone factor a factor at all,
// which is why an unverifiable token is refused rather than warned about.
//
// The Admin SDK is used rather than hand-rolled JWT verification on purpose: it
// pins the algorithm, tracks Google's rotating public keys, and checks issuer and
// audience. Getting any one of those wrong turns the second factor into a
// formality, and it is not a thing to reimplement to save a dependency.

var (
	ErrPhoneVerifyUnavailable = errors.New("phone verification is temporarily unavailable")
	ErrPhoneTokenInvalid      = errors.New("could not verify that phone number")
	ErrPhoneTokenNoPhone      = errors.New("that sign-in has no verified phone number")
	ErrPhoneTokenWrongUser    = errors.New("that verification belongs to a different account")
)

var (
	firebaseAuthOnce   sync.Once
	firebaseAuthClient *auth.Client
	firebaseAuthErr    error
)

// firebaseAuth lazily builds the Admin SDK client. Built on first use rather
// than at boot so an API without Firebase configured still starts — the phone
// channel simply stays unavailable.
func firebaseAuth(ctx context.Context) (*auth.Client, error) {
	firebaseAuthOnce.Do(func() {
		cfg := AppConfigForFirebase()
		if cfg.ProjectID == "" {
			firebaseAuthErr = fmt.Errorf("firebase project id not configured")
			return
		}
		// Application Default Credentials only. In GKE that resolves through
		// Workload Identity to the app's GCP service account, so no key material
		// is mounted, shipped in an env var, or able to leak from a config dump.
		// (option.WithCredentialsJSON is deprecated for exactly that risk.)
		app, err := firebase.NewApp(ctx, &firebase.Config{ProjectID: cfg.ProjectID})
		if err != nil {
			firebaseAuthErr = err
			return
		}
		firebaseAuthClient, firebaseAuthErr = app.Auth(ctx)
	})
	return firebaseAuthClient, firebaseAuthErr
}

// FirebaseConfig is the slice of app config this file needs, passed in so the
// services package does not import config (which imports services).
type FirebaseConfig struct {
	ProjectID string
	// TenantID scopes verification to one GIP tenant. Empty verifies against the
	// project root.
	TenantID string
}

// appConfigForFirebase is set at boot by main. A function value rather than a
// direct config import keeps the dependency one-way.
var appConfigForFirebase = func() FirebaseConfig { return FirebaseConfig{} }

// SetFirebaseConfigProvider wires the config lookup. Called once from main.
func SetFirebaseConfigProvider(f func() FirebaseConfig) { appConfigForFirebase = f }

// AppConfigForFirebase returns the currently wired Firebase config.
func AppConfigForFirebase() FirebaseConfig { return appConfigForFirebase() }

// VerifiedPhone is the outcome of a successful check.
type VerifiedPhone struct {
	// E164 is the number Google verified, in +<country><number> form.
	E164 string
	// FirebaseUID is the account the number is attached to.
	FirebaseUID string
}

// VerifyFirebasePhoneToken checks a Firebase ID token and extracts the verified
// phone number.
//
// expectedFirebaseUID binds the token to the caller's own account. Without it a
// user could verify a phone on any Firebase account they control — including one
// they made seconds earlier — and enrol that number as a factor on someone
// else's HomeChef account.
func VerifyFirebasePhoneToken(ctx context.Context, idToken, expectedFirebaseUID string) (VerifiedPhone, error) {
	idToken = strings.TrimSpace(idToken)
	if idToken == "" {
		return VerifiedPhone{}, ErrPhoneTokenInvalid
	}
	client, err := firebaseAuth(ctx)
	if err != nil {
		log.Printf("firebase-phone: client unavailable: %v", err)
		return VerifiedPhone{}, ErrPhoneVerifyUnavailable
	}

	// VerifyIDToken checks signature, algorithm, issuer, audience and expiry.
	tok, err := client.VerifyIDToken(ctx, idToken)
	if err != nil {
		return VerifiedPhone{}, ErrPhoneTokenInvalid
	}
	if expectedFirebaseUID != "" && tok.UID != expectedFirebaseUID {
		return VerifiedPhone{}, ErrPhoneTokenWrongUser
	}

	phone := PhoneFromFirebaseClaims(tok.Claims)
	if phone == "" {
		return VerifiedPhone{}, ErrPhoneTokenNoPhone
	}
	return VerifiedPhone{E164: phone, FirebaseUID: tok.UID}, nil
}

// PhoneFromFirebaseClaims digs the verified number out of a token's claims.
//
// Exported so it can be tested without a live Firebase project — the claim shape
// is the fiddly part, and it is nested two levels inside an untyped map.
//
// Prefers firebase.identities.phone[0], which only Google writes and only for a
// number it actually verified, over the top-level phone_number convenience
// claim. Falls back to the latter when identities is absent.
func PhoneFromFirebaseClaims(claims map[string]any) string {
	if fb, ok := claims["firebase"].(map[string]any); ok {
		if ids, ok := fb["identities"].(map[string]any); ok {
			if phones, ok := ids["phone"].([]any); ok && len(phones) > 0 {
				if p, ok := phones[0].(string); ok && p != "" {
					return NormalizeE164(p)
				}
			}
		}
	}
	if p, ok := claims["phone_number"].(string); ok && p != "" {
		return NormalizeE164(p)
	}
	return ""
}

// NormalizeE164 strips spaces, dashes and brackets so the same number compares
// equal however the client formatted it. Google already returns E.164, but the
// value also arrives from other paths.
func NormalizeE164(phone string) string {
	var b strings.Builder
	for _, r := range strings.TrimSpace(phone) {
		switch {
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		// Keep a leading '+' only when nothing has been kept yet — keying on the
		// INPUT index instead drops it from "(+91) 98765 43210", and a number
		// stored without its '+' never compares equal to the same number with one.
		case r == '+' && b.Len() == 0:
			b.WriteRune(r)
		}
	}
	return b.String()
}
