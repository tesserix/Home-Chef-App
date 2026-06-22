//go:build integration

package internal_test

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strconv"
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
	"github.com/homechef/auth-bff/internal/gip"
	"github.com/homechef/auth-bff/internal/headerproxy"
	"github.com/homechef/auth-bff/internal/productregistry"
	"github.com/homechef/auth-bff/internal/session"
)

// jwkServer serves a JWK Set for the test RSA key and signs test id_tokens
// with the matching private key. Mirrors the test helper from
// internal/gip/verifier_test.go.
type jwkServer struct {
	priv *rsa.PrivateKey
	srv  *httptest.Server
	kid  string
}

func newJWKServer(t *testing.T) *jwkServer {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	j := &jwkServer{priv: priv, kid: "test-kid"}
	j.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		eBytes := big.NewInt(int64(priv.E)).Bytes()
		jwks := map[string]any{
			"keys": []map[string]any{{
				"kty": "RSA", "kid": j.kid, "use": "sig", "alg": "RS256",
				"n": base64.RawURLEncoding.EncodeToString(priv.N.Bytes()),
				"e": base64.RawURLEncoding.EncodeToString(eBytes),
			}},
		}
		_ = json.NewEncoder(w).Encode(jwks)
	}))
	return j
}

func (j *jwkServer) signIDToken(t *testing.T, projectID, tenantID, sub, email, provider string) string {
	t.Helper()
	claims := jwt.MapClaims{
		"iss":   "https://securetoken.google.com/" + projectID,
		"aud":   projectID,
		"sub":   sub,
		"email": email,
		"iat":   time.Now().Unix(),
		"exp":   time.Now().Add(time.Hour).Unix(),
		"firebase": map[string]any{
			"tenant":           tenantID,
			"sign_in_provider": provider,
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tok.Header["kid"] = j.kid
	signed, err := tok.SignedString(j.priv)
	require.NoError(t, err)
	return signed
}

// fakeAPI is an apps/api stand-in that verifies HMAC and returns a fixed user_id.
type fakeAPI struct {
	signer   *headerproxy.Signer
	srv      *httptest.Server
	calls    atomic.Int32
	lastBody []byte
}

func newFakeAPI(t *testing.T, hmacKey []byte) *fakeAPI {
	t.Helper()
	api := &fakeAPI{
		signer: headerproxy.NewSigner(headerproxy.SignerConfig{Key: hmacKey, Window: time.Minute}),
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/internal/users/upsert", func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "read body", http.StatusInternalServerError)
			return
		}
		api.lastBody = append([]byte(nil), body...)
		api.calls.Add(1)
		// Verify the BFF actually signed the request properly
		if _, err := api.signer.Verify(r, body); err != nil {
			http.Error(w, "hmac verify failed: "+err.Error(), http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(apiclient.UpsertUserResponse{UserID: "user-id-from-api"})
	})
	api.srv = httptest.NewServer(mux)
	return api
}

func TestEndToEnd_MobileAutoLogin(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// --- arrange ---
	jwks := newJWKServer(t)
	defer jwks.srv.Close()

	projectID := "tesseracthub-480811"
	tenantID := "HomeChef-Customer-rqg8a" // matches homechef-products.yaml
	hmacKey := make([]byte, 32)
	_, _ = rand.Read(hmacKey)

	sessKey := make([]byte, 32)
	_, _ = rand.Read(sessKey)

	api := newFakeAPI(t, hmacKey)
	defer api.srv.Close()

	verifier, err := gip.New(gip.Config{
		ProjectID: projectID,
		JWKSURL:   jwks.srv.URL,
		Leeway:    10 * time.Second,
	})
	require.NoError(t, err)

	mgr, err := session.NewManager(session.Config{
		EncryptKey: sessKey, MaxAge: time.Hour,
		CookieName: "hc_session", Secure: false,
	})
	require.NoError(t, err)

	reg, err := productregistry.Load("../homechef-products.yaml")
	require.NoError(t, err)

	signer := headerproxy.NewSigner(headerproxy.SignerConfig{Key: hmacKey, Window: time.Minute})
	apiClient := apiclient.New(api.srv.URL, signer)

	deps := &autologin.Deps{
		GIP: verifier, Sessions: mgr, Registry: reg, API: apiClient,
	}
	r := gin.New()
	autologin.NewHandler(deps).Register(r)
	(&session.Handler{Mgr: mgr}).Register(r)

	// --- act 1: POST /auth/auto-login with a valid signed GIP id_token ---
	idTok := jwks.signIDToken(t, projectID, tenantID, "gip-sub-123", "user@example.com", "google.com")
	body := map[string]string{"id_token": idTok, "expected_tenant_id": tenantID}
	bodyJSON, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/auth/auto-login", bytes.NewReader(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// --- assert 1: 200 + session token + user from API ---
	require.Equal(t, 200, w.Code, "auto-login body: %s", w.Body.String())

	var resp autologin.Response
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp.SessionToken)
	assert.Equal(t, "user-id-from-api", resp.User.ID)
	assert.Equal(t, "user@example.com", resp.User.Email)
	assert.Equal(t, "customer", resp.User.Pool)
	assert.Equal(t, "customer", resp.User.Role)

	// Verify the apps/api stand-in actually received an HMAC-signed call
	assert.Equal(t, int32(1), api.calls.Load())

	// Verify the body shape sent to apps/api
	var upsert apiclient.UpsertUserRequest
	require.NoError(t, json.Unmarshal(api.lastBody, &upsert))
	assert.Equal(t, "gip-sub-123", upsert.GIPUid)
	assert.Equal(t, tenantID, upsert.GIPTenantID)
	assert.Equal(t, "google.com", upsert.GIPProvider)
	assert.Equal(t, "customer", upsert.AuthPool)
	assert.Equal(t, "user@example.com", upsert.Email)

	// --- act 2: GET /auth/session with the bearer token from act 1 ---
	req2 := httptest.NewRequest("GET", "/auth/session", nil)
	req2.Header.Set("Authorization", "Bearer "+resp.SessionToken)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	require.Equal(t, 200, w2.Code, "session body: %s", w2.Body.String())
	var sess map[string]any
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &sess))
	assert.Equal(t, "user-id-from-api", sess["user_id"])
	assert.Equal(t, "user@example.com", sess["email"])
	assert.Equal(t, "customer", sess["role"])
	assert.Equal(t, "customer", sess["pool"])
}

func TestEndToEnd_MobileAutoLogin_WrongTenant_Rejected(t *testing.T) {
	gin.SetMode(gin.TestMode)

	jwks := newJWKServer(t)
	defer jwks.srv.Close()
	projectID := "p"
	hmacKey := make([]byte, 32)
	_, _ = rand.Read(hmacKey)
	sessKey := make([]byte, 32)
	_, _ = rand.Read(sessKey)

	api := newFakeAPI(t, hmacKey)
	defer api.srv.Close()

	verifier, err := gip.New(gip.Config{ProjectID: projectID, JWKSURL: jwks.srv.URL, Leeway: 10 * time.Second})
	require.NoError(t, err)
	mgr, _ := session.NewManager(session.Config{EncryptKey: sessKey, MaxAge: time.Hour})
	reg, _ := productregistry.Load("../homechef-products.yaml")
	signer := headerproxy.NewSigner(headerproxy.SignerConfig{Key: hmacKey, Window: time.Minute})
	apiClient := apiclient.New(api.srv.URL, signer)

	r := gin.New()
	autologin.NewHandler(&autologin.Deps{GIP: verifier, Sessions: mgr, Registry: reg, API: apiClient}).Register(r)

	// Token claims a Customer tenant but request says Business → tenant mismatch (401 invalid_token)
	idTok := jwks.signIDToken(t, projectID, "HomeChef-Customer-rqg8a", "sub", "x@y.com", "google.com")
	body := map[string]string{"id_token": idTok, "expected_tenant_id": "HomeChef-Business-8s8ql"}
	bodyJSON, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/auth/auto-login", bytes.NewReader(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, 401, w.Code, "body: %s", w.Body.String())
	assert.Equal(t, int32(0), api.calls.Load(), "apps/api should not have been called")
}

func TestEndToEnd_MobileAutoLogin_DisallowedTenantBlocked(t *testing.T) {
	gin.SetMode(gin.TestMode)
	jwks := newJWKServer(t)
	defer jwks.srv.Close()
	hmacKey := make([]byte, 32)
	_, _ = rand.Read(hmacKey)
	sessKey := make([]byte, 32)
	_, _ = rand.Read(sessKey)
	api := newFakeAPI(t, hmacKey)
	defer api.srv.Close()
	verifier, _ := gip.New(gip.Config{ProjectID: "p", JWKSURL: jwks.srv.URL, Leeway: 10 * time.Second})
	mgr, _ := session.NewManager(session.Config{EncryptKey: sessKey, MaxAge: time.Hour})
	reg, _ := productregistry.Load("../homechef-products.yaml")
	signer := headerproxy.NewSigner(headerproxy.SignerConfig{Key: hmacKey, Window: time.Minute})
	apiClient := apiclient.New(api.srv.URL, signer)

	r := gin.New()
	autologin.NewHandler(&autologin.Deps{GIP: verifier, Sessions: mgr, Registry: reg, API: apiClient}).Register(r)

	body := `{"id_token":"any","expected_tenant_id":"HomeChef-Unknown-zzzzz"}`
	req := httptest.NewRequest("POST", "/auth/auto-login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, 403, w.Code)
	assert.Equal(t, int32(0), api.calls.Load())
}

// Ensures the test package's unused-import helpers compile (some go versions
// can be picky in build-tagged files when an import only feeds an inline check).
var _ = fmt.Sprintf
var _ = strconv.Itoa
var _ = hmac.New
var _ = sha256.New
var _ = hex.EncodeToString
var _ = context.Background
