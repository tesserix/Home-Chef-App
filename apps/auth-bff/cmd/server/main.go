package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"golang.org/x/oauth2"

	"github.com/homechef/auth-bff/internal/apiclient"
	"github.com/homechef/auth-bff/internal/apiproxy"
	"github.com/homechef/auth-bff/internal/audit"
	"github.com/homechef/auth-bff/internal/autologin"
	"github.com/homechef/auth-bff/internal/config"
	"github.com/homechef/auth-bff/internal/headerproxy"
	"github.com/homechef/auth-bff/internal/observability"
	"github.com/homechef/auth-bff/internal/obsmw"
	oidcpkg "github.com/homechef/auth-bff/internal/oidc"
	"github.com/homechef/auth-bff/internal/productregistry"
	"github.com/homechef/auth-bff/internal/ratelimit"
	"github.com/homechef/auth-bff/internal/session"
	"github.com/homechef/auth-bff/internal/tracing"
	"github.com/homechef/auth-bff/internal/zitadel"
)

// serviceName identifies this service in OpenTelemetry resources, the gin OTel
// middleware, and span attributes.
const serviceName = "auth-bff"

func main() {
	_ = godotenv.Load(".env.local")

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	// OpenTelemetry → in-cluster OTLP collector (traces + metrics). No-ops
	// when OTEL_EXPORTER_OTLP_ENDPOINT is unset (local dev).
	otelShutdown, oerr := observability.Init(context.Background(), serviceName)
	if oerr != nil {
		log.Printf("warning: observability init failed: %v", oerr)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = otelShutdown(shutdownCtx)
	}()

	// OpenTelemetry → Cloud Trace (same project as homechef-api when
	// GCP_PROJECT_ID is set). No-ops cleanly without creds/project.
	traceShutdown, terr := tracing.Init(
		context.Background(), cfg.TraceProjectID, cfg.Env, cfg.AppVersion, cfg.OTelSamplingRate,
	)
	if terr != nil {
		log.Printf("warning: tracing init failed: %v", terr)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = traceShutdown(shutdownCtx)
	}()

	reg, err := productregistry.Load(cfg.ProductsConfigPath)
	if err != nil {
		log.Fatalf("registry: %v", err)
	}

	verifier, err := zitadel.New(context.Background(), zitadel.Config{
		Issuer:    cfg.ZitadelIssuer,
		ProjectID: cfg.ZitadelProjectID,
	})
	if err != nil {
		log.Fatalf("zitadel verifier: %v", err)
	}

	mgr, err := session.NewManager(session.Config{
		EncryptKey: cfg.SessionEncryptKey,
		MaxAge:     cfg.SessionMaxAge,
		// Default/fallback cookie name for any Host the registry can't
		// resolve to an app (see productregistry.Registry.SessionCookieForHost
		// and session.Manager.ResolveCookieName). Each registered app gets
		// its own cookie name from homechef-products.yaml's sessionCookie
		// field instead — see reg.SessionCookieForHost wiring below.
		CookieName:   "hc_session",
		CookieDomain: cfg.SessionCookieDomain,
		Secure:       cfg.Env != "dev",
	})
	if err != nil {
		log.Fatalf("session manager: %v", err)
	}

	signer := headerproxy.NewSigner(headerproxy.SignerConfig{
		Key:    cfg.BFFInternalHMACKey,
		Window: 60 * time.Second,
	})
	api := apiclient.New(cfg.APIBaseURL, signer)

	provider, oauthByApp, err := buildOAuthMaps(reg, cfg.ZitadelIssuer, cfg.ZitadelProjectID)
	if err != nil {
		log.Fatalf("oauth build: %v", err)
	}
	stateManager, err := oidcpkg.NewBrowserStateManager(cfg.SessionEncryptKey, cfg.Env != "dev")
	if err != nil {
		log.Fatalf("oauth state manager: %v", err)
	}

	auditClient := audit.New(cfg.AuditEndpoint) // ok if cfg.AuditEndpoint is ""
	_ = auditClient                             // currently unused by handlers; placeholder for future emit calls

	r := gin.Default()
	// Correlation id first, then an OTel span per request, then echo the trace
	// id — so a login can be followed across auth-bff and the API in Cloud Trace.
	r.Use(obsmw.RequestID())
	r.Use(otelgin.Middleware(serviceName))
	r.Use(obsmw.TraceContext())
	r.GET("/healthz", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })

	oidcH := &oidcpkg.Handlers{
		Registry:     reg,
		OAuthByApp:   oauthByApp,
		Provider:     provider,
		API:          api,
		Sessions:     mgr,
		StateManager: stateManager,
	}

	// Rate-limit login-style endpoints: brute-force / resource-exhaustion targets.
	// Session-lifecycle endpoints (session/logout/refresh/csrf) are NOT
	// rate-limited because they're called by authenticated UI on every page nav.
	rl := ratelimit.New(5.0, 10) // 5 rps per IP, burst 10
	rateLimited := r.Group("/auth", rl.Middleware())
	rateLimited.GET("/login", oidcH.Login)
	rateLimited.GET("/callback", oidcH.Callback)

	autoH := autologin.NewHandler(&autologin.Deps{
		Verifier: verifier, Sessions: mgr, Registry: reg, API: api,
	})
	rateLimited.POST("/auto-login", autoH.PostHandler())

	(&session.Handler{Mgr: mgr, CookieForHost: reg.SessionCookieForHost}).Register(r)

	// Mobile clients hold a BFF session token but the upstream API only
	// accepts HMAC-signed requests. Catch-all /api/v1/* proxies validated
	// Bearer-token requests upstream, attaching the X-Internal-Auth + identity
	// headers via the existing headerproxy.Signer.
	apiProxyH := apiproxy.Handler(&apiproxy.Deps{
		APIBaseURL:    cfg.APIBaseURL,
		Sessions:      mgr,
		Signer:        signer,
		CookieForHost: reg.SessionCookieForHost,
	})
	r.Any("/api/v1/*proxyPath", apiProxyH)

	srv := &http.Server{
		Addr:              ":" + cfg.HTTPPort,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       2 * time.Minute,
		WriteTimeout:      2 * time.Minute,
		IdleTimeout:       time.Minute,
	}
	go func() {
		log.Printf("auth-bff listening on :%s (env=%s)", cfg.HTTPPort, cfg.Env)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}

// buildOAuthMaps discovers the Zitadel issuer once and constructs an
// *oauth2.Config per app, keyed by app name. All apps share the public PKCE
// client; only the redirect URL differs. The project-aud scope puts the
// Zitadel project id in the token audience so apps/api-facing verifiers accept it.
func buildOAuthMaps(reg *productregistry.Registry, issuer, projectID string) (*oidc.Provider, map[string]*oauth2.Config, error) {
	provider, err := oidc.NewProvider(context.Background(), issuer)
	if err != nil {
		return nil, nil, fmt.Errorf("oidc discovery for %s: %w", issuer, err)
	}
	oauthByApp := make(map[string]*oauth2.Config)
	for _, p := range reg.Products {
		for _, a := range p.Apps {
			oauthByApp[a.Name] = &oauth2.Config{
				ClientID:    a.OAuthClientID,
				Endpoint:    provider.Endpoint(),
				RedirectURL: pickRedirectURL(a),
				Scopes: []string{
					oidc.ScopeOpenID, "profile", "email",
					zitadel.ProjectAudScope(projectID),
				},
			}
		}
	}
	return provider, oauthByApp, nil
}

func pickRedirectURL(a productregistry.App) string {
	cb := a.CallbackPath
	if cb == "" {
		cb = "/auth/callback"
	}
	if a.CallbackHost != "" {
		return "https://" + a.CallbackHost + cb
	}
	for _, h := range a.Hosts {
		if !strings.HasPrefix(h, "localhost") {
			return "https://" + h + cb
		}
	}
	if len(a.Hosts) > 0 {
		return "http://" + a.Hosts[0] + cb
	}
	return ""
}
