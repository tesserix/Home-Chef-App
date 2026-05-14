package session

import (
	"crypto/rand"
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
