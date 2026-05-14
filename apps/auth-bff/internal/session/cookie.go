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

func (m *Manager) SetCookie(w http.ResponseWriter, value string) {
	http.SetCookie(w, &http.Cookie{
		Name:     m.cfg.CookieName,
		Value:    value,
		Path:     "/",
		Domain:   m.cfg.CookieDomain,
		MaxAge:   int(m.cfg.MaxAge.Seconds()),
		HttpOnly: true,
		Secure:   m.cfg.Secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func (m *Manager) Clear(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name: m.cfg.CookieName, Value: "", Path: "/", Domain: m.cfg.CookieDomain,
		MaxAge: -1, HttpOnly: true, Secure: m.cfg.Secure, SameSite: http.SameSiteLaxMode,
	})
}
