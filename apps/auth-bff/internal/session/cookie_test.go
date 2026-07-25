package session

import (
	"crypto/rand"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func freshKey(t *testing.T) []byte {
	k := make([]byte, 32)
	_, err := rand.Read(k)
	require.NoError(t, err)
	return k
}

func TestCookie_RoundTrip(t *testing.T) {
	mgr, err := NewManager(Config{EncryptKey: freshKey(t), MaxAge: time.Hour})
	require.NoError(t, err)

	p := &Payload{UID: "u1", Email: "a@b.com", Pool: "customer", Role: "customer", IssuedAt: time.Now().Unix(), ExpiresAt: time.Now().Add(time.Hour).Unix()}
	enc, err := mgr.Encode(p)
	require.NoError(t, err)
	require.NotEmpty(t, enc)

	got, err := mgr.Decode(enc)
	require.NoError(t, err)
	assert.Equal(t, p.UID, got.UID)
	assert.Equal(t, p.Email, got.Email)
}

func TestCookie_Tampered_Rejected(t *testing.T) {
	mgr, _ := NewManager(Config{EncryptKey: freshKey(t), MaxAge: time.Hour})
	p := &Payload{UID: "u1", IssuedAt: time.Now().Unix(), ExpiresAt: time.Now().Add(time.Hour).Unix()}
	enc, _ := mgr.Encode(p)
	tampered := enc[:len(enc)-2] + "AA"
	_, err := mgr.Decode(tampered)
	require.Error(t, err)
}

func TestCookie_Expired_Rejected(t *testing.T) {
	mgr, _ := NewManager(Config{EncryptKey: freshKey(t), MaxAge: time.Hour})
	p := &Payload{UID: "u1", IssuedAt: time.Now().Add(-2 * time.Hour).Unix(), ExpiresAt: time.Now().Add(-time.Hour).Unix()}
	enc, _ := mgr.Encode(p)
	_, err := mgr.Decode(enc)
	require.Error(t, err)
}

func TestCookie_FreshNonceEachCall(t *testing.T) {
	mgr, _ := NewManager(Config{EncryptKey: freshKey(t), MaxAge: time.Hour})
	p := &Payload{UID: "u1", IssuedAt: time.Now().Unix(), ExpiresAt: time.Now().Add(time.Hour).Unix()}
	a, _ := mgr.Encode(p)
	b, _ := mgr.Encode(p)
	assert.NotEqual(t, a, b, "same payload should produce different ciphertext (fresh nonce)")
}

// --- ResolveCookieName / SetCookie(name) / Clear(name) ---
//
// These cover the fallback contract every caller (session.Handler,
// apiproxy.Deps) relies on: a resolver that matches wins, a resolver that
// doesn't match (or a nil resolver) falls back to the Manager's configured
// default cookie name — never an error, never an empty cookie name.

func TestResolveCookieName_ResolverMatch_Wins(t *testing.T) {
	mgr, err := NewManager(Config{EncryptKey: freshKey(t), MaxAge: time.Hour, CookieName: "hc_session"})
	require.NoError(t, err)
	resolver := func(host string) string {
		if host == "vendors.fe3dr.com" {
			return "hc_vendor_session"
		}
		return ""
	}
	assert.Equal(t, "hc_vendor_session", mgr.ResolveCookieName(resolver, "vendors.fe3dr.com"))
}

func TestResolveCookieName_ResolverNoMatch_FallsBackToDefault(t *testing.T) {
	mgr, err := NewManager(Config{EncryptKey: freshKey(t), MaxAge: time.Hour, CookieName: "hc_session"})
	require.NoError(t, err)
	resolver := func(string) string { return "" }
	assert.Equal(t, "hc_session", mgr.ResolveCookieName(resolver, "unknown.example.com"))
}

func TestResolveCookieName_NilResolver_FallsBackToDefault(t *testing.T) {
	mgr, err := NewManager(Config{EncryptKey: freshKey(t), MaxAge: time.Hour, CookieName: "hc_session"})
	require.NoError(t, err)
	assert.Equal(t, "hc_session", mgr.ResolveCookieName(nil, "vendors.fe3dr.com"))
}

func TestSetCookie_EmptyName_FallsBackToDefault(t *testing.T) {
	mgr, err := NewManager(Config{EncryptKey: freshKey(t), MaxAge: time.Hour, CookieName: "hc_session"})
	require.NoError(t, err)
	w := httptest.NewRecorder()
	mgr.SetCookie(w, "", "enc-value")
	assert.Contains(t, w.Header().Get("Set-Cookie"), "hc_session=enc-value")
}

func TestSetCookie_ExplicitName_Honored(t *testing.T) {
	mgr, err := NewManager(Config{EncryptKey: freshKey(t), MaxAge: time.Hour, CookieName: "hc_session"})
	require.NoError(t, err)
	w := httptest.NewRecorder()
	mgr.SetCookie(w, "hc_admin_session", "enc-value")
	setCookie := w.Header().Get("Set-Cookie")
	assert.Contains(t, setCookie, "hc_admin_session=enc-value")
	assert.NotContains(t, setCookie, "hc_session=enc-value")
}

func TestClear_ExplicitName_Honored(t *testing.T) {
	mgr, err := NewManager(Config{EncryptKey: freshKey(t), MaxAge: time.Hour, CookieName: "hc_session"})
	require.NoError(t, err)
	w := httptest.NewRecorder()
	mgr.Clear(w, "hc_vendor_session")
	setCookie := w.Header().Get("Set-Cookie")
	assert.Contains(t, setCookie, "hc_vendor_session=")
	assert.NotContains(t, setCookie, "hc_session=")
}
