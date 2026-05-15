package oidc

import (
	"context"
	"net/http"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"

	"github.com/homechef/auth-bff/internal/apiclient"
	"github.com/homechef/auth-bff/internal/gip"
	"github.com/homechef/auth-bff/internal/productregistry"
	"github.com/homechef/auth-bff/internal/session"
)

// GIPVerifier is the surface of *gip.Verifier needed by the exchange handler.
// Defined as an interface so the handler can be tested without a live GIP.
type GIPVerifier interface {
	Verify(ctx context.Context, raw string, expectedTenantID string) (*gip.VerifiedToken, error)
}

// APIClient is the surface of *apiclient.Client needed to persist the user
// after successful authentication.
type APIClient interface {
	UpsertUser(ctx context.Context, req apiclient.UpsertUserRequest) (*apiclient.UpsertUserResponse, error)
}

// SessionWriter is the surface of *session.Manager needed to issue the
// browser session cookie.
type SessionWriter interface {
	Encode(*session.Payload) (string, error)
	SetCookie(w http.ResponseWriter, value string)
	MaxAge() time.Duration
}

// Handlers wires the BFF web auth surface: /auth/login, /auth/callback, and
// /auth/exchange. The OAuthByApp / OIDCByApp maps are populated by main.go
// based on homechef-products.yaml.
type Handlers struct {
	Registry    *productregistry.Registry
	OAuthByApp  map[string]*oauth2.Config
	OIDCByApp   map[string]*oidc.Provider
	GIPVerifier GIPVerifier
	API         APIClient
	Sessions    SessionWriter
	StateStore  StateStore
}

// Login starts the OIDC authorization-code flow. It resolves the per-host app
// from the registry, mints a fresh state + nonce, stores them, and redirects
// the user agent to GIP's authorize endpoint with the tenant-specific
// `tenantId` query parameter that GIP uses to scope the sign-in.
func (h *Handlers) Login(c *gin.Context) {
	app, err := h.Registry.ResolveByHost(c.Request.Host)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unknown_host"})
		return
	}
	cfg := h.OAuthByApp[app.Name]
	if cfg == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no_oauth_config"})
		return
	}
	state := NewStateID()
	nonce := NewStateID()
	h.StateStore.Put(state, StateEntry{
		AppName:  app.Name,
		Nonce:    nonce,
		ReturnTo: c.Query("return_to"),
	})
	url := cfg.AuthCodeURL(
		state,
		oidc.Nonce(nonce),
		oauth2.SetAuthURLParam("tenantId", app.GIPTenantID),
	)
	c.Redirect(http.StatusFound, url)
}

// Callback completes the OIDC authorization-code flow. It validates the
// returned state against the store, exchanges the authorization code for
// tokens, verifies the id_token via go-oidc, and then issues the browser
// session cookie via issueSession.
func (h *Handlers) Callback(c *gin.Context) {
	app, err := h.Registry.ResolveByHost(c.Request.Host)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unknown_host"})
		return
	}
	state := c.Query("state")
	entry, ok := h.StateStore.Take(state)
	if !ok {
		c.JSON(http.StatusForbidden, gin.H{"error": "bad_state"})
		return
	}
	cfg := h.OAuthByApp[app.Name]
	provider := h.OIDCByApp[app.Name]
	if cfg == nil || provider == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no_oauth_config"})
		return
	}
	tok, err := cfg.Exchange(c.Request.Context(), c.Query("code"))
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "exchange_failed"})
		return
	}
	rawID, _ := tok.Extra("id_token").(string)
	if rawID == "" {
		c.JSON(http.StatusBadGateway, gin.H{"error": "no_id_token"})
		return
	}
	verifier := provider.Verifier(&oidc.Config{ClientID: cfg.ClientID})
	idTok, err := verifier.Verify(c.Request.Context(), rawID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "id_token_invalid"})
		return
	}
	var claims map[string]any
	_ = idTok.Claims(&claims)
	h.issueSession(c, app, claims, entry)
}

// ExchangeRequest is the body shape POSTed by the SPA after a successful
// Firebase email/password sign-in. The browser has already obtained the
// id_token client-side; the BFF just verifies it and issues a session.
//
// MarketingConsent is the DPDP §6 opt-in captured by RegisterPage (CW-01b).
// It's optional — login flows omit it and it defaults to false, which the API
// only consumes on first-user creation (re-logins never flip the flag).
type ExchangeRequest struct {
	IDToken          string `json:"id_token" binding:"required"`
	MarketingConsent bool   `json:"marketing_consent"`
}

// Exchange handles the email/password sign-in path. The SPA signs in to
// Firebase client-side and POSTs the resulting id_token here; this handler
// verifies the token against the per-app GIP tenant and issues the session.
func (h *Handlers) Exchange(c *gin.Context) {
	app, err := h.Registry.ResolveByHost(c.Request.Host)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unknown_host"})
		return
	}
	var req ExchangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_body"})
		return
	}
	vt, err := h.GIPVerifier.Verify(c.Request.Context(), req.IDToken, app.GIPTenantID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "id_token_invalid"})
		return
	}
	h.issueSession(c, app, map[string]any(vt.Claims), StateEntry{
		MarketingConsent: req.MarketingConsent,
	})
}

// issueSession is the shared tail of both the Callback (GET) and Exchange
// (POST) paths. It upserts the user in apps/api, encodes the session payload
// into the encrypted cookie, and then either redirects (GET) or returns JSON
// (POST) depending on the request method.
func (h *Handlers) issueSession(c *gin.Context, app *productregistry.App, claims map[string]any, entry StateEntry) {
	pool := app.AuthContext
	role := app.DefaultRole
	if r, ok := claims["role"].(string); ok && r != "" {
		role = r
	}
	provider := ""
	if fb, ok := claims["firebase"].(map[string]any); ok {
		if p, ok := fb["sign_in_provider"].(string); ok {
			provider = p
		}
	}
	upsert, err := h.API.UpsertUser(c.Request.Context(), apiclient.UpsertUserRequest{
		GIPUid:           getStr(claims, "sub"),
		GIPTenantID:      app.GIPTenantID,
		GIPProvider:      provider,
		AuthPool:         pool,
		Email:            getStr(claims, "email"),
		Role:             role,
		MarketingConsent: entry.MarketingConsent,
	})
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "upsert_failed"})
		return
	}
	now := time.Now()
	exp := now.Add(h.Sessions.MaxAge())
	payload := &session.Payload{
		UID:       upsert.UserID,
		Email:     getStr(claims, "email"),
		Pool:      pool,
		Role:      role,
		IssuedAt:  now.Unix(),
		ExpiresAt: exp.Unix(),
	}
	enc, err := h.Sessions.Encode(payload)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "session_encode_failed"})
		return
	}
	h.Sessions.SetCookie(c.Writer, enc)
	if c.Request.Method == http.MethodGet {
		// Callback path → browser-friendly redirect.
		target := app.PostLoginURL
		if entry.ReturnTo != "" {
			target = entry.ReturnTo
		}
		c.Redirect(http.StatusFound, target)
		return
	}
	// Exchange path → JSON response for the SPA.
	c.JSON(http.StatusOK, gin.H{
		"user_id":    upsert.UserID,
		"email":      payload.Email,
		"role":       role,
		"pool":       pool,
		"expires_at": exp.Unix(),
	})
}

// Register binds the three auth endpoints onto the provided router.
func (h *Handlers) Register(r gin.IRouter) {
	r.GET("/auth/login", h.Login)
	r.GET("/auth/callback", h.Callback)
	r.POST("/auth/exchange", h.Exchange)
}

func getStr(m map[string]any, k string) string {
	if v, ok := m[k].(string); ok {
		return v
	}
	return ""
}
