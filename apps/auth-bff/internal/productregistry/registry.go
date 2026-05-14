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
