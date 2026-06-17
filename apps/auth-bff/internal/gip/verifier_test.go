package gip

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testJWKServer struct {
	key *rsa.PrivateKey
	srv *httptest.Server
	kid string
}

func newJWKServer(t *testing.T) *testJWKServer {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	tj := &testJWKServer{key: priv, kid: "test-kid"}
	tj.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		eBytes := big.NewInt(int64(priv.E)).Bytes()
		jwk := map[string]any{
			"keys": []map[string]any{{
				"kty": "RSA",
				"kid": tj.kid,
				"use": "sig",
				"alg": "RS256",
				"n":   base64.RawURLEncoding.EncodeToString(priv.N.Bytes()),
				"e":   base64.RawURLEncoding.EncodeToString(eBytes),
			}},
		}
		_ = json.NewEncoder(w).Encode(jwk)
	}))
	return tj
}

func (tj *testJWKServer) signToken(t *testing.T, claims jwt.MapClaims) string {
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tok.Header["kid"] = tj.kid
	s, err := tok.SignedString(tj.key)
	require.NoError(t, err)
	return s
}

func TestVerifier_ValidToken_PassesAllChecks(t *testing.T) {
	tj := newJWKServer(t)
	defer tj.srv.Close()

	v, err := New(Config{
		ProjectID: "tesseracthub-480811",
		JWKSURL:   tj.srv.URL,
		Leeway:    10 * time.Second,
	})
	require.NoError(t, err)

	tok := tj.signToken(t, jwt.MapClaims{
		"iss":            "https://securetoken.google.com/tesseracthub-480811",
		"aud":            "tesseracthub-480811",
		"sub":            "user-123",
		"email":          "a@b.com",
		"name":           "Ada Lovelace",
		"picture":        "https://example.com/ada.png",
		"email_verified": true,
		"iat":            time.Now().Unix(),
		"exp":            time.Now().Add(time.Hour).Unix(),
		"firebase": map[string]any{
			"tenant":           "HomeChef-Customer-aaaaa",
			"sign_in_provider": "google.com",
		},
	})

	got, err := v.Verify(t.Context(), tok, "HomeChef-Customer-aaaaa")
	require.NoError(t, err)
	assert.Equal(t, "user-123", got.UID)
	assert.Equal(t, "a@b.com", got.Email)
	assert.Equal(t, "HomeChef-Customer-aaaaa", got.TenantID)
	assert.Equal(t, "google.com", got.Provider)
	assert.Equal(t, "Ada Lovelace", got.Name)
	assert.Equal(t, "https://example.com/ada.png", got.Picture)
	assert.True(t, got.EmailVerified)
}

func TestVerifier_MissingProfileClaims_DefaultToZeroValues(t *testing.T) {
	tj := newJWKServer(t)
	defer tj.srv.Close()

	v, err := New(Config{ProjectID: "p", JWKSURL: tj.srv.URL, Leeway: 10 * time.Second})
	require.NoError(t, err)

	// No name/picture/email_verified claims present.
	tok := tj.signToken(t, jwt.MapClaims{
		"iss": "https://securetoken.google.com/p", "aud": "p", "sub": "u",
		"email": "a@b.com",
		"iat":   time.Now().Unix(), "exp": time.Now().Add(time.Hour).Unix(),
		"firebase": map[string]any{"tenant": "T", "sign_in_provider": "password"},
	})

	got, err := v.Verify(t.Context(), tok, "T")
	require.NoError(t, err)
	assert.Equal(t, "", got.Name)
	assert.Equal(t, "", got.Picture)
	assert.False(t, got.EmailVerified)
}

func TestVerifier_WrongTenant_Rejects(t *testing.T) {
	tj := newJWKServer(t)
	defer tj.srv.Close()
	v, _ := New(Config{ProjectID: "p", JWKSURL: tj.srv.URL, Leeway: 10 * time.Second})

	tok := tj.signToken(t, jwt.MapClaims{
		"iss": "https://securetoken.google.com/p", "aud": "p", "sub": "u",
		"exp": time.Now().Add(time.Hour).Unix(), "iat": time.Now().Unix(),
		"firebase": map[string]any{"tenant": "OtherTenant-xxxxx"},
	})

	_, err := v.Verify(t.Context(), tok, "HomeChef-Customer-aaaaa")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrTenantMismatch)
}

func TestVerifier_Expired_Rejects(t *testing.T) {
	tj := newJWKServer(t)
	defer tj.srv.Close()
	v, _ := New(Config{ProjectID: "p", JWKSURL: tj.srv.URL, Leeway: 1 * time.Second})

	tok := tj.signToken(t, jwt.MapClaims{
		"iss": "https://securetoken.google.com/p", "aud": "p", "sub": "u",
		"exp": time.Now().Add(-time.Hour).Unix(), "iat": time.Now().Add(-2 * time.Hour).Unix(),
		"firebase": map[string]any{"tenant": "T"},
	})

	_, err := v.Verify(t.Context(), tok, "T")
	require.Error(t, err)
}

func TestVerifier_TamperedSignature_Rejects(t *testing.T) {
	tj := newJWKServer(t)
	defer tj.srv.Close()
	v, _ := New(Config{ProjectID: "p", JWKSURL: tj.srv.URL, Leeway: 1 * time.Second})

	tok := tj.signToken(t, jwt.MapClaims{
		"iss": "https://securetoken.google.com/p", "aud": "p", "sub": "u",
		"exp": time.Now().Add(time.Hour).Unix(), "iat": time.Now().Unix(),
		"firebase": map[string]any{"tenant": "T"},
	})
	tampered := tok[:len(tok)-3] + "AAA"

	_, err := v.Verify(t.Context(), tampered, "T")
	require.Error(t, err)
}
