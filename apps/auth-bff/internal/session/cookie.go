package session

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

type Config struct {
	EncryptKey   []byte
	MaxAge       time.Duration
	CookieName   string
	CookieDomain string
	Secure       bool
}

type Payload struct {
	UID       string `json:"uid"`
	Email     string `json:"email"`
	Pool      string `json:"pool"`
	Role      string `json:"role"`
	IssuedAt  int64  `json:"iat"`
	ExpiresAt int64  `json:"exp"`
}

type Manager struct {
	cfg Config
	gcm cipher.AEAD
}

func NewManager(cfg Config) (*Manager, error) {
	if l := len(cfg.EncryptKey); l != 16 && l != 24 && l != 32 {
		return nil, fmt.Errorf("encrypt key must be 16/24/32 bytes, got %d", l)
	}
	if cfg.MaxAge == 0 {
		cfg.MaxAge = 7 * 24 * time.Hour
	}
	if cfg.CookieName == "" {
		cfg.CookieName = "hc_session"
	}
	block, err := aes.NewCipher(cfg.EncryptKey)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &Manager{cfg: cfg, gcm: gcm}, nil
}

func (m *Manager) MaxAge() time.Duration { return m.cfg.MaxAge }
func (m *Manager) CookieName() string    { return m.cfg.CookieName }

// CookieNameResolver maps a request Host to the session cookie name that
// owns that host, so callers can isolate cookies per app without this
// package (or apiproxy) importing the product registry. Returning "" means
// "no match" — ResolveCookieName then falls back to the Manager's default.
type CookieNameResolver func(host string) string

// ResolveCookieName returns resolver(host) when resolver is non-nil and
// yields a non-empty name; otherwise it returns the Manager's configured
// default cookie name (cfg.CookieName). This centralizes the "unmatched
// host / no resolver" fallback for every caller that reads or writes the
// session cookie for a given request, so an unknown Host degrades to the
// historical single shared cookie instead of erroring.
func (m *Manager) ResolveCookieName(resolver CookieNameResolver, host string) string {
	if resolver != nil {
		if name := resolver(host); name != "" {
			return name
		}
	}
	return m.cfg.CookieName
}

func (m *Manager) Encode(p *Payload) (string, error) {
	plaintext, err := json.Marshal(p)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, m.gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	ct := m.gcm.Seal(nonce, nonce, plaintext, nil)
	return base64.RawURLEncoding.EncodeToString(ct), nil
}

func (m *Manager) Decode(raw string) (*Payload, error) {
	blob, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}
	if len(blob) < m.gcm.NonceSize() {
		return nil, errors.New("ciphertext too short")
	}
	nonce, ct := blob[:m.gcm.NonceSize()], blob[m.gcm.NonceSize():]
	plain, err := m.gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt: %w", err)
	}
	var p Payload
	if err := json.Unmarshal(plain, &p); err != nil {
		return nil, err
	}
	if time.Now().Unix() > p.ExpiresAt {
		return nil, errors.New("session expired")
	}
	return &p, nil
}

// SetCookie writes the session cookie under name. An empty name falls back
// to the Manager's configured default cookie name — callers that haven't
// resolved a per-app name (or that want the historical single cookie) can
// pass "" rather than duplicating the fallback themselves.
func (m *Manager) SetCookie(w http.ResponseWriter, name, value string) {
	if name == "" {
		name = m.cfg.CookieName
	}
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		Domain:   m.cfg.CookieDomain,
		MaxAge:   int(m.cfg.MaxAge.Seconds()),
		HttpOnly: true,
		Secure:   m.cfg.Secure,
		SameSite: http.SameSiteLaxMode,
	})
}

// Clear expires the session cookie under name, following the same
// empty-name fallback as SetCookie.
func (m *Manager) Clear(w http.ResponseWriter, name string) {
	if name == "" {
		name = m.cfg.CookieName
	}
	http.SetCookie(w, &http.Cookie{
		Name: name, Value: "", Path: "/", Domain: m.cfg.CookieDomain,
		MaxAge: -1, HttpOnly: true, Secure: m.cfg.Secure, SameSite: http.SameSiteLaxMode,
	})
}
