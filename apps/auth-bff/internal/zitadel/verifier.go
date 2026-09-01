// Package zitadel verifies OIDC id_tokens issued by the Zitadel instance at
// auth.tesserix.app. It replaces the former Google Identity Platform verifier.
package zitadel

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/coreos/go-oidc/v3/oidc"
)

type Config struct {
	// Issuer is the Zitadel instance URL, e.g. https://auth.tesserix.app.
	Issuer string
	// ProjectID is the Zitadel project whose audience tokens must carry
	// (granted by the urn:zitadel:iam:org:project:id:<id>:aud scope).
	ProjectID string
	// ExtraAudiences additionally accepted in aud (e.g. per-app client IDs).
	ExtraAudiences []string
	// HTTPClient overrides the discovery/JWKS client (tests).
	HTTPClient *http.Client
}

type VerifiedToken struct {
	Subject       string
	Email         string
	EmailVerified bool
	Name          string
	Picture       string
	Claims        map[string]any
}

var ErrAudienceMismatch = errors.New("token audience does not include the Zitadel project")

type Verifier struct {
	verifier  *oidc.IDTokenVerifier
	projectID string
	extraAud  map[string]struct{}
}

func New(ctx context.Context, cfg Config) (*Verifier, error) {
	if cfg.Issuer == "" || cfg.ProjectID == "" {
		return nil, errors.New("zitadel verifier requires Issuer and ProjectID")
	}
	if cfg.HTTPClient != nil {
		ctx = oidc.ClientContext(ctx, cfg.HTTPClient)
	}
	provider, err := oidc.NewProvider(ctx, cfg.Issuer)
	if err != nil {
		return nil, fmt.Errorf("zitadel discovery for %s: %w", cfg.Issuer, err)
	}
	extra := make(map[string]struct{}, len(cfg.ExtraAudiences))
	for _, a := range cfg.ExtraAudiences {
		extra[a] = struct{}{}
	}
	return &Verifier{
		verifier: provider.Verifier(&oidc.Config{
			SkipClientIDCheck:    true, // audience is checked against the project below
			SupportedSigningAlgs: []string{oidc.RS256, oidc.ES256, oidc.EdDSA},
		}),
		projectID: cfg.ProjectID,
		extraAud:  extra,
	}, nil
}

// Verify checks signature, issuer, expiry and audience, and extracts the
// standard profile claims.
func (v *Verifier) Verify(ctx context.Context, raw string) (*VerifiedToken, error) {
	idt, err := v.verifier.Verify(ctx, raw)
	if err != nil {
		return nil, err
	}
	if !v.audienceOK(idt.Audience) {
		return nil, ErrAudienceMismatch
	}
	var claims map[string]any
	if err := idt.Claims(&claims); err != nil {
		return nil, fmt.Errorf("decode claims: %w", err)
	}
	tok := &VerifiedToken{Subject: idt.Subject, Claims: claims}
	tok.Email, _ = claims["email"].(string)
	tok.Name, _ = claims["name"].(string)
	tok.Picture, _ = claims["picture"].(string)
	tok.EmailVerified, _ = claims["email_verified"].(bool)
	return tok, nil
}

func (v *Verifier) audienceOK(aud []string) bool {
	for _, a := range aud {
		if a == v.projectID {
			return true
		}
		if _, ok := v.extraAud[a]; ok {
			return true
		}
	}
	return false
}

// ProjectAudScope returns the Zitadel scope that adds projectID to the token
// audience — without it a client's own token is not valid for this backend.
func ProjectAudScope(projectID string) string {
	return "urn:zitadel:iam:org:project:id:" + projectID + ":aud"
}
