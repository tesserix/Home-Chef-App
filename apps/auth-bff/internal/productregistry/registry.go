package productregistry

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

var ErrUnknownHost = errors.New("unknown host")

type App struct {
	Name  string   `yaml:"name"`
	Hosts []string `yaml:"hosts"`
	// OAuthClientID is the Zitadel OIDC client id (public client, PKCE only —
	// there is no client secret). All web apps currently share one client.
	OAuthClientID    string   `yaml:"oauthClientId"`
	SessionCookie    string   `yaml:"sessionCookie"`
	CallbackPath     string   `yaml:"callbackPath"`
	CallbackHost     string   `yaml:"callbackHost"`
	PostLoginURL     string   `yaml:"postLoginUrl"`
	PostLogoutURL    string   `yaml:"postLogoutUrl"`
	AuthContext      string   `yaml:"authContext"`
	DefaultRole      string   `yaml:"defaultRole"`
	SignInMethods    []string `yaml:"signInMethods"`
	AllowedEmailsEnv string   `yaml:"allowedEmailsEnv"`
	AllowedOrigins   []string `yaml:"allowedOrigins"`
}

type Product struct {
	Name   string `yaml:"name"`
	Domain string `yaml:"domain"`
	Apps   []App  `yaml:"apps"`
}

type Registry struct {
	PlatformDomain string    `yaml:"platformDomain"`
	Products       []Product `yaml:"products"`
	// MobilePoolAllowlist names the auth pools mobile auto-login may mint
	// sessions for (customer/business/internal).
	MobilePoolAllowlist []string `yaml:"mobilePoolAllowlist"`

	hostIndex map[string]*App
}

func Load(path string) (*Registry, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var r Registry
	if err := yaml.Unmarshal(b, &r); err != nil {
		return nil, err
	}
	r.hostIndex = make(map[string]*App)
	for i := range r.Products {
		for j := range r.Products[i].Apps {
			a := &r.Products[i].Apps[j]
			for _, h := range a.Hosts {
				r.hostIndex[strings.ToLower(h)] = a
			}
		}
	}
	return &r, nil
}

func (r *Registry) ResolveByHost(host string) (*App, error) {
	if app, ok := r.hostIndex[strings.ToLower(host)]; ok {
		return app, nil
	}
	return nil, fmt.Errorf("%w: %s", ErrUnknownHost, host)
}

// SessionCookieForHost resolves the per-app session cookie name for host.
// It returns "" when host doesn't match any registered app, or when the
// matched app has no sessionCookie configured — callers (session.Manager's
// ResolveCookieName) treat "" as "no match" and fall back to their own
// default cookie name rather than erroring. Shaped as a plain
// func(string) string (matches session.CookieNameResolver) so it can be
// passed directly into session.Handler / apiproxy.Deps without either of
// those packages importing productregistry.
func (r *Registry) SessionCookieForHost(host string) string {
	app, err := r.ResolveByHost(host)
	if err != nil {
		return ""
	}
	return app.SessionCookie
}

func (r *Registry) IsMobilePoolAllowed(pool string) bool {
	for _, p := range r.MobilePoolAllowlist {
		if p == pool {
			return true
		}
	}
	return false
}

// ResolveByPool returns the first app registered with the given auth pool
// (authContext), or nil when no app matches. The mobile auto-login path
// resolves apps by pool (there is no request host), so it uses this instead
// of ResolveByHost.
func (r *Registry) ResolveByPool(pool string) *App {
	for i := range r.Products {
		for j := range r.Products[i].Apps {
			if r.Products[i].Apps[j].AuthContext == pool {
				return &r.Products[i].Apps[j]
			}
		}
	}
	return nil
}

// IsEmailAllowed reports whether email passes the app's admin email allowlist,
// resolved at call time from the env var named by AllowedEmailsEnv
// (comma-separated, case-insensitive, space-trimmed).
//
// The second return value reports whether an allowlist is actually configured.
// Callers fail CLOSED: admin/internal login is denied whenever allowed is false,
// INCLUDING when configured is false (unset/empty env). A missing allowlist must
// never grant admin access — the mesh does not strip inbound X-User-* headers,
// so this gate is the sole defense. configured is surfaced only so callers can
// log the unconfigured case distinctly ("set the env") from a real deny.
func (a *App) IsEmailAllowed(email string) (allowed, configured bool) {
	if a.AllowedEmailsEnv == "" {
		return false, false
	}
	want := strings.ToLower(strings.TrimSpace(email))
	for _, e := range strings.Split(os.Getenv(a.AllowedEmailsEnv), ",") {
		e = strings.ToLower(strings.TrimSpace(e))
		if e == "" {
			continue
		}
		configured = true
		if e == want {
			allowed = true
		}
	}
	return allowed, configured
}
