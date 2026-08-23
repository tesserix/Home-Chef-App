package oidc

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode"
)

// StateEntry is the data stored against an OAuth state value during the
// authorize -> callback round-trip. It carries the resolved app name, the
// nonce we'll later verify against the id_token, and an optional post-login
// return URL captured at /auth/login time.
//
// MarketingConsent is the DPDP §6 opt-in collected at the SPA — only the
// /auth/exchange path threads it through (callback flows are pure sign-in
// where consent isn't applicable). It is forwarded once to the API on
// first-user creation; subsequent logins ignore it.
type StateEntry struct {
	AppName          string
	Nonce            string
	ReturnTo         string
	MarketingConsent bool
	Created          time.Time
}

// StateManager creates and consumes browser-bound OAuth state. Implementations
// must support callbacks landing on a different service replica than login.
type StateManager interface {
	Begin(http.ResponseWriter, StateEntry) (string, error)
	Take(http.ResponseWriter, *http.Request, string) (StateEntry, bool)
}

const (
	stateTTL          = 5 * time.Minute
	stateHandleBytes  = 12
	stateBindingBytes = 32
	stateVersion      = "hc-oidc-state-v1"
)

// BrowserStateManager seals OAuth state into an authenticated envelope and
// binds it to a short-lived, host-only browser cookie. It has no process-local
// state, so login and callback may be handled by different replicas.
type BrowserStateManager struct {
	gcm    cipher.AEAD
	secure bool
	ttl    time.Duration
	now    func() time.Time
	random io.Reader
}

type browserStateEnvelope struct {
	Entry       StateEntry `json:"entry"`
	BindingHash []byte     `json:"binding_hash"`
	IssuedAt    int64      `json:"iat"`
	ExpiresAt   int64      `json:"exp"`
}

// NewBrowserStateManager derives an OAuth-state encryption key from the
// session key so the two ciphertext types remain cryptographically separated.
func NewBrowserStateManager(sessionKey []byte, secure bool) (*BrowserStateManager, error) {
	if l := len(sessionKey); l != 16 && l != 24 && l != 32 {
		return nil, fmt.Errorf("state key must be derived from a 16/24/32-byte session key, got %d", l)
	}
	mac := hmac.New(sha256.New, sessionKey)
	_, _ = mac.Write([]byte(stateVersion))
	key := mac.Sum(nil)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create state cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create state AEAD: %w", err)
	}
	return &BrowserStateManager{
		gcm:    gcm,
		secure: secure,
		ttl:    stateTTL,
		now:    time.Now,
		random: rand.Reader,
	}, nil
}

// Begin creates a browser-bound OAuth state value and writes its one-time
// verifier cookie. Each login gets a distinct cookie so parallel tabs work.
func (m *BrowserStateManager) Begin(w http.ResponseWriter, entry StateEntry) (string, error) {
	if entry.AppName == "" || entry.Nonce == "" {
		return "", errors.New("OAuth state requires app and nonce")
	}
	handleBytes := make([]byte, stateHandleBytes)
	if _, err := io.ReadFull(m.random, handleBytes); err != nil {
		return "", fmt.Errorf("generate state handle: %w", err)
	}
	binding := make([]byte, stateBindingBytes)
	if _, err := io.ReadFull(m.random, binding); err != nil {
		return "", fmt.Errorf("generate state binding: %w", err)
	}
	now := m.now().UTC()
	entry.Created = now
	envelope := browserStateEnvelope{
		Entry:       entry,
		BindingHash: bindingDigest(binding),
		IssuedAt:    now.Unix(),
		ExpiresAt:   now.Add(m.ttl).Unix(),
	}
	plaintext, err := json.Marshal(envelope)
	if err != nil {
		return "", fmt.Errorf("encode OAuth state: %w", err)
	}
	nonce := make([]byte, m.gcm.NonceSize())
	if _, err := io.ReadFull(m.random, nonce); err != nil {
		return "", fmt.Errorf("generate state nonce: %w", err)
	}
	handle := base64.RawURLEncoding.EncodeToString(handleBytes)
	sealed := m.gcm.Seal(nonce, nonce, plaintext, stateAAD(handle))
	http.SetCookie(w, &http.Cookie{
		Name:     m.cookieName(handle),
		Value:    base64.RawURLEncoding.EncodeToString(binding),
		Path:     "/auth/callback",
		MaxAge:   int(m.ttl.Seconds()),
		Expires:  now.Add(m.ttl),
		HttpOnly: true,
		Secure:   m.secure,
		SameSite: http.SameSiteLaxMode,
	})
	return handle + "." + base64.RawURLEncoding.EncodeToString(sealed), nil
}

// Take validates and consumes a browser-bound OAuth state value. The verifier
// cookie is expired before validation completes so a callback cannot be reused
// after either success or failure.
func (m *BrowserStateManager) Take(w http.ResponseWriter, r *http.Request, raw string) (StateEntry, bool) {
	handle, ciphertext, ok := splitState(raw)
	if !ok {
		return StateEntry{}, false
	}
	cookieName := m.cookieName(handle)
	cookie, err := r.Cookie(cookieName)
	if err != nil || cookie.Value == "" {
		return StateEntry{}, false
	}
	m.clearCookie(w, cookieName)
	binding, err := base64.RawURLEncoding.DecodeString(cookie.Value)
	if err != nil || len(binding) != stateBindingBytes {
		return StateEntry{}, false
	}
	blob, err := base64.RawURLEncoding.DecodeString(ciphertext)
	if err != nil || len(blob) < m.gcm.NonceSize()+m.gcm.Overhead() {
		return StateEntry{}, false
	}
	nonce, sealed := blob[:m.gcm.NonceSize()], blob[m.gcm.NonceSize():]
	plaintext, err := m.gcm.Open(nil, nonce, sealed, stateAAD(handle))
	if err != nil {
		return StateEntry{}, false
	}
	var envelope browserStateEnvelope
	if err := json.Unmarshal(plaintext, &envelope); err != nil {
		return StateEntry{}, false
	}
	now := m.now().Unix()
	if envelope.IssuedAt > now || now >= envelope.ExpiresAt ||
		envelope.ExpiresAt-envelope.IssuedAt != int64(m.ttl.Seconds()) ||
		envelope.Entry.AppName == "" || envelope.Entry.Nonce == "" ||
		len(envelope.BindingHash) != sha256.Size ||
		subtle.ConstantTimeCompare(envelope.BindingHash, bindingDigest(binding)) != 1 {
		return StateEntry{}, false
	}
	return envelope.Entry, true
}

func (m *BrowserStateManager) cookieName(handle string) string {
	prefix := "hc_oidc_"
	if m.secure {
		prefix = "__Secure-hc_oidc_"
	}
	return prefix + handle
}

func (m *BrowserStateManager) clearCookie(w http.ResponseWriter, name string) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/auth/callback",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   m.secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func splitState(raw string) (string, string, bool) {
	if len(raw) > 4096 || strings.Count(raw, ".") != 1 {
		return "", "", false
	}
	handle, ciphertext, _ := strings.Cut(raw, ".")
	decoded, err := base64.RawURLEncoding.DecodeString(handle)
	if err != nil || len(decoded) != stateHandleBytes || ciphertext == "" {
		return "", "", false
	}
	return handle, ciphertext, true
}

func stateAAD(handle string) []byte {
	return []byte(stateVersion + "." + handle)
}

func bindingDigest(binding []byte) []byte {
	digest := sha256.Sum256(binding)
	return digest[:]
}

// NewStateID returns a cryptographically random URL-safe identifier suitable
// for use as an OIDC nonce value.
func NewStateID() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// safeReturnTo accepts only a same-origin absolute path. Scheme-relative URLs,
// backslashes, control characters, and absolute URLs are deliberately dropped.
func safeReturnTo(raw string) string {
	if raw == "" || !strings.HasPrefix(raw, "/") || strings.HasPrefix(raw, "//") ||
		strings.Contains(raw, `\`) || strings.IndexFunc(raw, unicode.IsControl) >= 0 {
		return ""
	}
	u, err := url.ParseRequestURI(raw)
	if err != nil || u.IsAbs() || u.Host != "" || strings.HasPrefix(u.Path, "//") ||
		strings.Contains(u.Path, `\`) || strings.IndexFunc(u.Path, unicode.IsControl) >= 0 {
		return ""
	}
	return raw
}
