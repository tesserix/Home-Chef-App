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

// sign mirrors apps/auth-bff/internal/headerproxy/signer.go:compute exactly.
// Used by tests to simulate BFF signing without importing the other module.
func sign(method, path string, body []byte, ts string, key []byte) string {
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
	r.Header.Set(HdrSignature, sign(r.Method, r.URL.Path, body, tsStr, key))
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
