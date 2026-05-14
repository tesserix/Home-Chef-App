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
	"golang.org/x/oauth2"

	"github.com/homechef/auth-bff/internal/apiclient"
	"github.com/homechef/auth-bff/internal/autologin"
	"github.com/homechef/auth-bff/internal/config"
	gippkg "github.com/homechef/auth-bff/internal/gip"
	"github.com/homechef/auth-bff/internal/headerproxy"
	oidcpkg "github.com/homechef/auth-bff/internal/oidc"
	"github.com/homechef/auth-bff/internal/productregistry"
	"github.com/homechef/auth-bff/internal/session"
)

func main() {
	_ = godotenv.Load(".env.local")

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	reg, err := productregistry.Load(cfg.ProductsConfigPath)
	if err != nil {
		log.Fatalf("registry: %v", err)
	}

	verifier, err := gippkg.New(gippkg.Config{
		ProjectID: cfg.GIPProjectID,
		Leeway:    10 * time.Second,
	})
	if err != nil {
		log.Fatalf("gip verifier: %v", err)
	}

	mgr, err := session.NewManager(session.Config{
		EncryptKey:   cfg.SessionEncryptKey,
		MaxAge:       cfg.SessionMaxAge,
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

	oauthByApp, oidcByApp, err := buildOAuthMaps(reg, cfg.GIPProjectID)
	if err != nil {
		log.Fatalf("oauth build: %v", err)
	}

	r := gin.Default()
	r.GET("/healthz", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })

	oidcH := &oidcpkg.Handlers{
		Registry:    reg,
		OAuthByApp:  oauthByApp,
		OIDCByApp:   oidcByApp,
		GIPVerifier: verifier,
		API:         api,
		Sessions:    mgr,
		StateStore:  oidcpkg.NewMemStateStore(),
	}
	oidcH.Register(r)

	autologin.NewHandler(&autologin.Deps{
		GIP: verifier, Sessions: mgr, Registry: reg, API: api,
	}).Register(r)

	(&session.Handler{Mgr: mgr}).Register(r)

	srv := &http.Server{Addr: ":" + cfg.HTTPPort, Handler: r, ReadHeaderTimeout: 5 * time.Second}
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

// buildOAuthMaps walks the product registry and constructs an *oauth2.Config
// and a discovered *oidc.Provider per app, keyed by app name.
//
// For each tenant, the OIDC issuer is "https://securetoken.google.com/{project_id}".
// The OAuth2 client secret is read from the env var named in app.ClientSecretEnv.
// The redirect URL is canonical: https://<callbackHost or first https host>/<callbackPath>.
// In dev (no https hosts), use http://<first localhost host>/<callbackPath>.
func buildOAuthMaps(reg *productregistry.Registry, projectID string) (map[string]*oauth2.Config, map[string]*oidc.Provider, error) {
	oauthByApp := make(map[string]*oauth2.Config)
	oidcByApp := make(map[string]*oidc.Provider)
	issuer := "https://securetoken.google.com/" + projectID
	provider, err := oidc.NewProvider(context.Background(), issuer)
	if err != nil {
		return nil, nil, fmt.Errorf("oidc discovery for %s: %w", issuer, err)
	}
	for _, p := range reg.Products {
		for _, a := range p.Apps {
			clientSecret := os.Getenv(a.ClientSecretEnv)
			redirectURL := pickRedirectURL(a)
			oauthByApp[a.Name] = &oauth2.Config{
				ClientID:     a.OAuthClientID,
				ClientSecret: clientSecret,
				Endpoint:     provider.Endpoint(),
				RedirectURL:  redirectURL,
				Scopes:       []string{oidc.ScopeOpenID, "email", "profile"},
			}
			oidcByApp[a.Name] = provider
		}
	}
	return oauthByApp, oidcByApp, nil
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
