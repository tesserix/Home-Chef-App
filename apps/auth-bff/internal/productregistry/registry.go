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
	Name             string   `yaml:"name"`
	Hosts            []string `yaml:"hosts"`
	GIPTenantID      string   `yaml:"gipTenantId"`
	OAuthClientID    string   `yaml:"oauthClientId"`
	ClientSecretEnv  string   `yaml:"clientSecretEnv"`
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
	PlatformDomain        string    `yaml:"platformDomain"`
	Products              []Product `yaml:"products"`
	MobileTenantAllowlist []string  `yaml:"mobileTenantAllowlist"`

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

func (r *Registry) IsMobileTenantAllowed(tenantID string) bool {
	for _, t := range r.MobileTenantAllowlist {
		if t == tenantID {
			return true
		}
	}
	return false
}

// ResolveByTenant returns the first app registered with the given GIP tenant ID,
// or nil when no app matches. The mobile auto-login path resolves apps by tenant
// (there is no request host), so it uses this instead of ResolveByHost.
func (r *Registry) ResolveByTenant(tenantID string) *App {
	for i := range r.Products {
		for j := range r.Products[i].Apps {
			if r.Products[i].Apps[j].GIPTenantID == tenantID {
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
// When configured is false the caller should fail OPEN (allow + log) so an unset
// env var doesn't lock everyone out; when configured is true the caller MUST
// fail CLOSED whenever allowed is false.
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
