//go:build integration

package internal_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/homechef/auth-bff/internal/apiclient"
	"github.com/homechef/auth-bff/internal/autologin"
	"github.com/homechef/auth-bff/internal/headerproxy"
	"github.com/homechef/auth-bff/internal/productregistry"
	"github.com/homechef/auth-bff/internal/session"
	"github.com/homechef/auth-bff/internal/zitadel"
)

const projectID = "388810586143588367"

// oidcServer serves an OIDC discovery document + JWKS for a test RSA key and
// signs test id_tokens with it — a stand-in for auth.tesserix.app.
type oidcServer struct {
	priv *rsa.PrivateKey
	srv  *httptest.Server
	kid  string
}

func newOIDCServer(t *testing.T) *oidcServer {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	o := &oidcServer{priv: priv, kid: "test-kid"}
	o.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"issuer":                                o.srv.URL,
				"jwks_uri":                              o.srv.URL + "/oauth/v2/keys",
				"authorization_endpoint":                o.srv.URL + "/oauth/v2/authorize",
				"token_endpoint":                        o.srv.URL + "/oauth/v2/token",
				"id_token_signing_alg_values_supported": []string{"RS256"},
			})
		case "/oauth/v2/keys":
			eBytes := big.NewInt(int64(priv.E)).Bytes()
			_ = json.NewEncoder(w).Encode(map[string]any{
				"keys": []map[string]string{{
					"kty": "RSA", "kid": o.kid, "alg": "RS256", "use": "sig",
					"n": base64.RawURLEncoding.EncodeToString(priv.N.Bytes()),
					"e": base64.RawURLEncoding.EncodeToString(eBytes),
				}},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(o.srv.Close)
	return o
}

func (o *oidcServer) signIDToken(t *testing.T, aud, sub, email string) string {
	t.Helper()
	now := time.Now()
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"iss":            o.srv.URL,
		"aud":            []string{aud},
		"sub":            sub,
		"email":          email,
		"email_verified": true,
		"name":           "Test User",
		"iat":            now.Unix(),
		"exp":            now.Add(time.Hour).Unix(),
	})
	tok.Header["kid"] = o.kid
	signed, err := tok.SignedString(o.priv)
	require.NoError(t, err)
	return signed
}

// fakeAPI is an apps/api stand-in that verifies the HMAC signature.
type fakeAPI struct {
	srv      *httptest.Server
	signer   *headerproxy.Signer
	calls    atomic.Int32
	lastBody []byte
}

func newFakeAPI(t *testing.T, hmacKey []byte) *fakeAPI {
	t.Helper()
	api := &fakeAPI{signer: headerproxy.NewSigner(headerproxy.SignerConfig{Key: hmacKey, Window: time.Minute})}
	mux := http.NewServeMux()
	mux.HandleFunc("/internal/users/upsert", func(w http.ResponseWriter, r *http.Request) {
		body := new(bytes.Buffer)
		if _, err := body.ReadFrom(r.Body); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		api.lastBody = body.Bytes()
		api.calls.Add(1)
		if _, err := api.signer.Verify(r, api.lastBody); err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(apiclient.UpsertUserResponse{UserID: "user-id-from-api"})
	})
	api.srv = httptest.NewServer(mux)
	t.Cleanup(api.srv.Close)
	return api
}

func buildStack(t *testing.T, oidcSrv *oidcServer) (*gin.Engine, *fakeAPI) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	hmacKey := make([]byte, 32)
	_, _ = rand.Read(hmacKey)
	sessKey := make([]byte, 32)
	_, _ = rand.Read(sessKey)
	api := newFakeAPI(t, hmacKey)

	verifier, err := zitadel.New(context.Background(), zitadel.Config{
		Issuer: oidcSrv.srv.URL, ProjectID: projectID,
	})
	require.NoError(t, err)
	mgr, err := session.NewManager(session.Config{
		EncryptKey: sessKey, MaxAge: time.Hour, CookieName: "hc_session", Secure: false,
	})
	require.NoError(t, err)
	reg, err := productregistry.Load("../homechef-products.yaml")
	require.NoError(t, err)
	signer := headerproxy.NewSigner(headerproxy.SignerConfig{Key: hmacKey, Window: time.Minute})

	r := gin.New()
	autologin.NewHandler(&autologin.Deps{
		Verifier: verifier, Sessions: mgr, Registry: reg, API: apiclient.New(api.srv.URL, signer),
	}).Register(r)
	(&session.Handler{Mgr: mgr}).Register(r)
	return r, api
}

func TestEndToEnd_MobileAutoLogin(t *testing.T) {
	oidcSrv := newOIDCServer(t)
	r, api := buildStack(t, oidcSrv)

	// --- act 1: POST /auth/auto-login with a valid signed Zitadel id_token ---
	idTok := oidcSrv.signIDToken(t, projectID, "zitadel-sub-123", "user@example.com")
	bodyJSON, _ := json.Marshal(map[string]string{"id_token": idTok, "pool": "customer"})
	req := httptest.NewRequest("POST", "/auth/auto-login", bytes.NewReader(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, 200, w.Code, "auto-login body: %s", w.Body.String())
	var resp autologin.Response
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp.SessionToken)
	assert.Equal(t, "user-id-from-api", resp.User.ID)
	assert.Equal(t, "user@example.com", resp.User.Email)

	// --- assert: the API received one HMAC-signed provider/subject upsert ---
	require.Equal(t, int32(1), api.calls.Load())
	var upsert apiclient.UpsertUserRequest
	require.NoError(t, json.Unmarshal(api.lastBody, &upsert))
	assert.Equal(t, "zitadel", upsert.Provider)
	assert.Equal(t, "zitadel-sub-123", upsert.Subject)
	assert.Equal(t, "customer", upsert.AuthPool)
	assert.Equal(t, "user@example.com", upsert.Email)

	// --- act 2: GET /auth/session with the minted token ---
	req2 := httptest.NewRequest("GET", "/auth/session", nil)
	req2.Header.Set("Authorization", "Bearer "+resp.SessionToken)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	require.Equal(t, 200, w2.Code)
	assert.Contains(t, w2.Body.String(), "user-id-from-api")
	assert.Contains(t, w2.Body.String(), "user@example.com")
}

func TestEndToEnd_MobileAutoLogin_WrongAudience_Rejected(t *testing.T) {
	oidcSrv := newOIDCServer(t)
	r, api := buildStack(t, oidcSrv)

	// Token minted for a different Zitadel project — audience check must fail.
	idTok := oidcSrv.signIDToken(t, "some-other-project", "sub", "x@y.com")
	bodyJSON, _ := json.Marshal(map[string]string{"id_token": idTok, "pool": "customer"})
	req := httptest.NewRequest("POST", "/auth/auto-login", bytes.NewReader(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, 401, w.Code, "body: %s", w.Body.String())
	assert.Equal(t, int32(0), api.calls.Load(), "apps/api should not have been called")
}

func TestEndToEnd_MobileAutoLogin_DisallowedPoolBlocked(t *testing.T) {
	oidcSrv := newOIDCServer(t)
	r, api := buildStack(t, oidcSrv)

	req := httptest.NewRequest("POST", "/auth/auto-login",
		strings.NewReader(`{"id_token":"any","pool":"superuser"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, 403, w.Code)
	assert.Equal(t, int32(0), api.calls.Load())
}
