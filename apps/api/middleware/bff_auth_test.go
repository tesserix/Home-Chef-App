package middleware

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// sign mirrors apps/auth-bff/internal/headerproxy/signer.go:compute exactly —
// the identity-bound canonical message. Used by tests to simulate BFF signing
// without importing the other module.
func sign(method, path string, body []byte, ts string, key []byte, id BFFIdentity) string {
	bodyHash := sha256.Sum256(body)
	m := hmac.New(sha256.New, key)
	_, _ = fmt.Fprintf(m, "%s\n%s\n%s\n%s\n%s\n%s\n%s\n%s",
		method, path, hex.EncodeToString(bodyHash[:]), ts,
		id.UserID, id.Email, id.Role, id.Pool)
	return hex.EncodeToString(m.Sum(nil))
}

// signLegacy mirrors the DEPRECATED pre-identity-binding format
// (method/path/body/ts only). Used to exercise the backward-compat fallback.
func signLegacy(method, path string, body []byte, ts string, key []byte) string {
	bodyHash := sha256.Sum256(body)
	m := hmac.New(sha256.New, key)
	_, _ = fmt.Fprintf(m, "%s\n%s\n%s\n%s", method, path, hex.EncodeToString(bodyHash[:]), ts)
	return hex.EncodeToString(m.Sum(nil))
}

func attachSigned(r *http.Request, body []byte, key []byte, id BFFIdentity, ts int64) {
	tsStr := strconv.FormatInt(ts, 10)
	r.Header.Set(HdrUserID, id.UserID)
	r.Header.Set(HdrUserEmail, id.Email)
	r.Header.Set(HdrUserRole, id.Role)
	r.Header.Set(HdrAuthPool, id.Pool)
	r.Header.Set(HdrAuthTs, tsStr)
	r.Header.Set(HdrSignature, sign(r.Method, r.URL.Path, body, tsStr, key, id))
}

func TestBFFAuth_ValidSignature_SetsContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	key := []byte("test-key-32-bytes-padding-padding!")
	r := gin.New()
	r.POST("/x", BFFAuth(BFFAuthConfig{HMACKey: key, Window: time.Minute}), func(c *gin.Context) {
		uid, _ := c.Get("user_id")
		email, _ := c.Get("user_email")
		role, _ := c.Get("user_role")
		pool, _ := c.Get("auth_pool")
		c.JSON(200, gin.H{"user_id": uid, "email": email, "role": role, "pool": pool})
	})

	body := []byte(`{"hello":"world"}`)
	req := httptest.NewRequest("POST", "/x", bytes.NewReader(body))
	attachSigned(req, body, key, BFFIdentity{UserID: "u1", Email: "a@b.com", Role: "customer", Pool: "customer"}, time.Now().Unix())

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, 200, w.Code, "body: %s", w.Body.String())
	assert.Contains(t, w.Body.String(), `"user_id":"u1"`)
	assert.Contains(t, w.Body.String(), `"role":"customer"`)
}

func TestBFFAuth_MissingHeader_401(t *testing.T) {
	gin.SetMode(gin.TestMode)
	key := []byte("test-key-32-bytes-padding-padding!")
	r := gin.New()
	r.POST("/x", BFFAuth(BFFAuthConfig{HMACKey: key, Window: time.Minute}), func(c *gin.Context) { c.Status(200) })

	req := httptest.NewRequest("POST", "/x", strings.NewReader(`{}`))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, 401, w.Code)
}

func TestBFFAuth_StaleTs_401(t *testing.T) {
	gin.SetMode(gin.TestMode)
	key := []byte("test-key-32-bytes-padding-padding!")
	r := gin.New()
	r.POST("/x", BFFAuth(BFFAuthConfig{HMACKey: key, Window: time.Second}), func(c *gin.Context) { c.Status(200) })

	body := []byte(`{}`)
	req := httptest.NewRequest("POST", "/x", bytes.NewReader(body))
	old := time.Now().Add(-time.Hour).Unix()
	attachSigned(req, body, key, BFFIdentity{UserID: "u1"}, old)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, 401, w.Code)
}

func TestBFFAuth_TamperedBody_401(t *testing.T) {
	gin.SetMode(gin.TestMode)
	key := []byte("test-key-32-bytes-padding-padding!")
	r := gin.New()
	r.POST("/x", BFFAuth(BFFAuthConfig{HMACKey: key, Window: time.Minute}), func(c *gin.Context) { c.Status(200) })

	body := []byte(`{"hello":"world"}`)
	req := httptest.NewRequest("POST", "/x", bytes.NewReader(body))
	// Sign with one body, then mutate the body so what's read on the server differs
	attachSigned(req, body, key, BFFIdentity{UserID: "u1"}, time.Now().Unix())
	req.Body = io.NopCloser(bytes.NewReader([]byte(`{"hello":"tampered"}`)))
	req.ContentLength = int64(len(`{"hello":"tampered"}`))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, 401, w.Code)
}

// TestBFFAuth_LegacySignature_Rejected proves the legacy (pre-identity-binding)
// signature path is gone: the auth-bff now signs exclusively in the
// identity-bound format, so an OLD-format signature — which lets a caller swap
// the X-User-* headers without breaking the MAC — must be rejected. Keeping the
// fallback would make the identity binding (and every role/pool check on top of
// it) only advisory.
func TestBFFAuth_LegacySignature_Rejected(t *testing.T) {
	gin.SetMode(gin.TestMode)
	key := []byte("test-key-32-bytes-padding-padding!")
	r := gin.New()
	r.POST("/x", BFFAuth(BFFAuthConfig{HMACKey: key, Window: time.Minute}), func(c *gin.Context) {
		c.Status(200)
	})

	body := []byte(`{"hello":"world"}`)
	req := httptest.NewRequest("POST", "/x", bytes.NewReader(body))
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	req.Header.Set(HdrUserID, "u1")
	req.Header.Set(HdrUserRole, "customer")
	req.Header.Set(HdrAuthTs, ts)
	req.Header.Set(HdrSignature, signLegacy(req.Method, req.URL.Path, body, ts, key))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, 401, w.Code, "legacy (identity-unbound) signatures must be rejected")
}

// TestBFFAuth_LegacySignature_CannotForgeAdmin is the attack the removal closes:
// an attacker who can mint a valid OLD-format signature sets X-User-Role: admin.
// With the fallback gone this must 401 — the MAC no longer covers the role header
// under any accepted format.
func TestBFFAuth_LegacySignature_CannotForgeAdmin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	key := []byte("test-key-32-bytes-padding-padding!")
	r := gin.New()
	r.POST("/x", BFFAuth(BFFAuthConfig{HMACKey: key, Window: time.Minute}), func(c *gin.Context) { c.Status(200) })

	body := []byte(`{}`)
	req := httptest.NewRequest("POST", "/x", bytes.NewReader(body))
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	req.Header.Set(HdrUserID, "attacker")
	req.Header.Set(HdrUserRole, "admin")
	req.Header.Set(HdrAuthTs, ts)
	req.Header.Set(HdrSignature, signLegacy(req.Method, req.URL.Path, body, ts, key))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, 401, w.Code, "a legacy signature must not be able to assert an unbound admin role")
}

// TestBFFAuth_TamperedRoleHeader_401 proves the new identity binding: a request
// signed as "customer" whose X-User-Role header is then escalated to "admin"
// must be rejected (neither the new nor the legacy signature covers the swap).
func TestBFFAuth_TamperedRoleHeader_401(t *testing.T) {
	gin.SetMode(gin.TestMode)
	key := []byte("test-key-32-bytes-padding-padding!")
	r := gin.New()
	r.POST("/x", BFFAuth(BFFAuthConfig{HMACKey: key, Window: time.Minute}), func(c *gin.Context) { c.Status(200) })

	body := []byte(`{}`)
	req := httptest.NewRequest("POST", "/x", bytes.NewReader(body))
	attachSigned(req, body, key, BFFIdentity{UserID: "u1", Email: "a@b.com", Role: "customer", Pool: "customer"}, time.Now().Unix())
	req.Header.Set(HdrUserRole, "admin") // escalate after signing

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, 401, w.Code, "swapping X-User-Role must break the identity-bound signature")
}

func TestBFFAuth_WrongKey_401(t *testing.T) {
	gin.SetMode(gin.TestMode)
	signerKey := []byte("client-key-aaaaaaaaaaaaaaaaaaaaaaaaaaa!")
	serverKey := []byte("server-key-bbbbbbbbbbbbbbbbbbbbbbbbbbb!")
	r := gin.New()
	r.POST("/x", BFFAuth(BFFAuthConfig{HMACKey: serverKey, Window: time.Minute}), func(c *gin.Context) { c.Status(200) })

	body := []byte(`{}`)
	req := httptest.NewRequest("POST", "/x", bytes.NewReader(body))
	attachSigned(req, body, signerKey, BFFIdentity{UserID: "u1"}, time.Now().Unix())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, 401, w.Code)
}
