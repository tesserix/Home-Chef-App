package gip

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrTenantMismatch = errors.New("firebase.tenant claim mismatch")
	ErrInvalidIssuer  = errors.New("invalid issuer")
	ErrInvalidAud     = errors.New("invalid audience")
)

type Config struct {
	ProjectID string
	JWKSURL   string // override only for tests; prod uses default
	Leeway    time.Duration
}

type Verifier struct {
	cfg    Config
	jwks   keyfunc.Keyfunc
	expIss string
}

type VerifiedToken struct {
	UID      string
	Email    string
	TenantID string
	Provider string // from firebase.sign_in_provider claim
	// Name is the user's display name from the "name" claim (empty if absent).
	Name string
	// Picture is the avatar URL from the "picture" claim (empty if absent).
	Picture string
	// EmailVerified reflects the "email_verified" claim. GIP tokens from
	// Google/Apple are always true; only unverified password signups are false.
	EmailVerified bool
	Claims        jwt.MapClaims
}

const defaultJWKSURL = "https://www.googleapis.com/service_accounts/v1/jwk/securetoken@system.gserviceaccount.com"

func New(cfg Config) (*Verifier, error) {
	jwksURL := cfg.JWKSURL
	if jwksURL == "" {
		jwksURL = defaultJWKSURL
	}
	k, err := keyfunc.NewDefaultCtx(context.Background(), []string{jwksURL})
	if err != nil {
		return nil, fmt.Errorf("init JWKS: %w", err)
	}
	if cfg.Leeway == 0 {
		cfg.Leeway = 10 * time.Second
	}
	return &Verifier{
		cfg:    cfg,
		jwks:   k,
		expIss: "https://securetoken.google.com/" + cfg.ProjectID,
	}, nil
}

func (v *Verifier) Verify(ctx context.Context, raw string, expectedTenantID string) (*VerifiedToken, error) {
	claims := jwt.MapClaims{}
	tok, err := jwt.ParseWithClaims(raw, claims, v.jwks.Keyfunc,
		jwt.WithLeeway(v.cfg.Leeway),
		jwt.WithIssuer(v.expIss),
		jwt.WithAudience(v.cfg.ProjectID),
		jwt.WithValidMethods([]string{"RS256"}),
	)
	if err != nil {
		return nil, fmt.Errorf("parse token: %w", err)
	}
	if !tok.Valid {
		return nil, errors.New("invalid token")
	}

	fb, _ := claims["firebase"].(map[string]any)
	tenantID, _ := fb["tenant"].(string)
	if tenantID != expectedTenantID {
		return nil, ErrTenantMismatch
	}
	provider, _ := fb["sign_in_provider"].(string)
	uid, _ := claims["sub"].(string)
	email, _ := claims["email"].(string)
	name, _ := claims["name"].(string)
	picture, _ := claims["picture"].(string)
	emailVerified, _ := claims["email_verified"].(bool)

	return &VerifiedToken{
		UID: uid, Email: email, TenantID: tenantID, Provider: provider,
		Name: name, Picture: picture, EmailVerified: emailVerified, Claims: claims,
	}, nil
}
