package oidc

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"log"
	"net/http"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"

	"github.com/homechef/auth-bff/internal/apiclient"
	"github.com/homechef/auth-bff/internal/productregistry"
	"github.com/homechef/auth-bff/internal/session"
)

// ProviderName is recorded on every identity minted through this BFF.
const ProviderName = "zitadel"

// APIClient is the surface of *apiclient.Client needed to persist a user
// after successful authentication.
type APIClient interface {
	UpsertUser(ctx context.Context, req apiclient.UpsertUserRequest) (*apiclient.UpsertUserResponse, error)
}

// SessionWriter is the surface of *session.Manager needed to issue a
// browser session cookie.
type SessionWriter interface {
	Encode(*session.Payload) (string, error)
	SetCookie(w http.ResponseWriter, name, value string)
	MaxAge() time.Duration
}

// Handlers wires the BFF web auth surface: /auth/login and /auth/callback.
// All apps share the Zitadel hosted login; per-app oauth2 configs differ only
// by redirect URL. PKCE (S256) replaces client secrets — the Zitadel apps are
// public clients (OIDC_AUTH_METHOD_TYPE_NONE).
type Handlers struct {
	Registry     *productregistry.Registry
	OAuthByApp   map[string]*oauth2.Config
	Provider     *oidc.Provider
	API          APIClient
	Sessions     SessionWriter
	StateManager StateManager
}

// Login starts the OIDC authorization-code + PKCE flow against Zitadel.
// ?screen=register forwards prompt=create so Zitadel opens registration;
// ?marketing_consent=true carries the DPDP §6 opt-in collected pre-redirect.
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
	nonce, err := NewStateID()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "state_failed"})
		return
	}
	pkce := oauth2.GenerateVerifier()
	state, err := h.StateManager.Begin(c.Writer, StateEntry{
		AppName:          app.Name,
		Nonce:            nonce,
		ReturnTo:         safeReturnTo(c.Query("return_to")),
		MarketingConsent: c.Query("marketing_consent") == "true",
		CodeVerifier:     pkce,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "state_failed"})
		return
	}
	opts := []oauth2.AuthCodeOption{
		oidc.Nonce(nonce),
		oauth2.S256ChallengeOption(pkce),
	}
	if c.Query("screen") == "register" {
		opts = append(opts, oauth2.SetAuthURLParam("prompt", "create"))
	}
	c.Redirect(http.StatusFound, cfg.AuthCodeURL(state, opts...))
}

// Callback completes the code flow: state + PKCE + nonce checks, id_token
// verification, then session issuance.
func (h *Handlers) Callback(c *gin.Context) {
	app, err := h.Registry.ResolveByHost(c.Request.Host)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unknown_host"})
		return
	}
	state := c.Query("state")
	entry, ok := h.StateManager.Take(c.Writer, c.Request, state)
	if !ok {
		c.JSON(http.StatusForbidden, gin.H{"error": "bad_state"})
		return
	}
	if !stateMatchesApp(entry, app.Name) {
		c.JSON(http.StatusForbidden, gin.H{"error": "bad_state"})
		return
	}
	cfg := h.OAuthByApp[app.Name]
	if cfg == nil || h.Provider == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no_oauth_config"})
		return
	}
	tok, err := cfg.Exchange(c.Request.Context(), c.Query("code"), oauth2.VerifierOption(entry.CodeVerifier))
	if err != nil {
		log.Printf("oidc: token exchange failed for app %s: %v", app.Name, err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "exchange_failed"})
		return
	}
	rawID, _ := tok.Extra("id_token").(string)
	if rawID == "" {
		log.Printf("oidc: token response for app %s has no id_token", app.Name)
		c.JSON(http.StatusBadGateway, gin.H{"error": "no_id_token"})
		return
	}
	verifier := h.Provider.Verifier(&oidc.Config{ClientID: cfg.ClientID})
	idTok, err := verifier.Verify(c.Request.Context(), rawID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "id_token_invalid"})
		return
	}
	if !nonceMatches(entry, idTok.Nonce) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "id_token_invalid"})
		return
	}
	var claims map[string]any
	if err := idTok.Claims(&claims); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "id_token_invalid"})
		return
	}
	h.issueSession(c, app, claims, entry)
}

// issueSession upserts the user in apps/api, encodes the session payload into
// the encrypted cookie, then redirects to the app.
func (h *Handlers) issueSession(c *gin.Context, app *productregistry.App, claims map[string]any, entry StateEntry) {
	pool := app.AuthContext
	role := app.DefaultRole
	if r, ok := claims["role"].(string); ok && r != "" {
		role = r
	}
	// Admin allowlist enforcement. FAIL-CLOSED: an unconfigured/empty allowlist
	// denies admin login, and the reject happens BEFORE the upsert so a blocked
	// email never gets an admin row or session.
	if pool == "internal" || role == "admin" {
		email := getStr(claims, "email")
		if allowed, configured := app.IsEmailAllowed(email); !configured || !allowed {
			if !configured {
				log.Printf("oidc: admin allowlist %s unconfigured — denying admin login for %q (set the env to enable admin access)", app.AllowedEmailsEnv, email)
			} else {
				log.Printf("oidc: rejected admin login for %q — not in %s allowlist", email, app.AllowedEmailsEnv)
			}
			c.JSON(http.StatusForbidden, gin.H{"error": "email_not_allowed"})
			return
		}
		if !getBool(claims, "email_verified") {
			log.Printf("oidc: rejected admin login for %q — email is not verified", email)
			c.JSON(http.StatusForbidden, gin.H{"error": "email_not_verified"})
			return
		}
	}
	upsert, err := h.API.UpsertUser(c.Request.Context(), apiclient.UpsertUserRequest{
		Provider:         ProviderName,
		Subject:          getStr(claims, "sub"),
		AuthPool:         pool,
		Email:            getStr(claims, "email"),
		Name:             getStr(claims, "name"),
		Avatar:           getStr(claims, "picture"),
		EmailVerified:    getBool(claims, "email_verified"),
		Role:             role,
		MarketingConsent: entry.MarketingConsent,
		DeviceID:         browserDeviceID(c),
		Platform:         "web",
		DeviceLabel:      browserLabel(c.GetHeader("User-Agent")),
		IP:               c.ClientIP(),
	})
	if err != nil {
		log.Printf("oidc: user upsert failed for app %s: %v", app.Name, err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "upstream_error"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "session_failed"})
		return
	}
	h.Sessions.SetCookie(c.Writer, app.SessionCookie, enc)
	target := app.PostLoginURL
	if entry.ReturnTo != "" {
		target = entry.ReturnTo
	}
	c.Redirect(http.StatusFound, target)
}

// Register binds the auth endpoints onto the provided router.
func (h *Handlers) Register(r gin.IRouter) {
	r.GET("/auth/login", h.Login)
	r.GET("/auth/callback", h.Callback)
}

func getStr(m map[string]any, k string) string {
	if v, ok := m[k].(string); ok {
		return v
	}
	return ""
}

func getBool(m map[string]any, k string) bool {
	v, _ := m[k].(bool)
	return v
}

func stateMatchesApp(entry StateEntry, appName string) bool {
	return entry.AppName != "" && entry.AppName == appName
}

func nonceMatches(entry StateEntry, tokenNonce string) bool {
	if entry.Nonce == "" || tokenNonce == "" {
		return false
	}
	expected := sha256.Sum256([]byte(entry.Nonce))
	actual := sha256.Sum256([]byte(tokenNonce))
	return subtle.ConstantTimeCompare(expected[:], actual[:]) == 1
}
