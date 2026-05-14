package middleware

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// Header names — must stay in lockstep with apps/auth-bff/internal/headerproxy/signer.go.
const (
	HdrUserID    = "X-User-Id"
	HdrUserEmail = "X-User-Email"
	HdrUserRole  = "X-User-Role"
	HdrAuthPool  = "X-Auth-Pool"
	HdrAuthTs    = "X-Auth-Ts"
	HdrSignature = "X-Internal-Auth"
)

// Gin context keys.
const (
	CtxUserID    = "user_id"
	CtxUserEmail = "user_email"
	CtxUserRole  = "user_role"
	CtxAuthPool  = "auth_pool"
)

var (
	ErrBFFMissingSignature  = errors.New("missing X-Internal-Auth")
	ErrBFFStaleTimestamp    = errors.New("stale X-Auth-Ts")
	ErrBFFSignatureMismatch = errors.New("HMAC mismatch")
)

type BFFIdentity struct {
	UserID string
	Email  string
	Role   string
	Pool   string
}

type BFFAuthConfig struct {
	HMACKey []byte
	Window  time.Duration // default 60s
}

// BFFAuth returns a Gin middleware that verifies the X-Internal-Auth HMAC
// signature applied by the BFF and, on success, populates the Gin context
// with the user identity headers.
//
// The wire format matches apps/auth-bff/internal/headerproxy/signer.go:compute.
// Any drift between the two will manifest as 401s with ErrBFFSignatureMismatch —
// the unit tests in this package replicate the BFF's compute() to catch drift.
func BFFAuth(cfg BFFAuthConfig) gin.HandlerFunc {
	if cfg.Window == 0 {
		cfg.Window = 60 * time.Second
	}
	return func(c *gin.Context) {
		// Read the body so we can re-hash it, then restore it for downstream handlers.
		var body []byte
		if c.Request.Body != nil {
			b, err := io.ReadAll(c.Request.Body)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "body_read_failed"})
				return
			}
			body = b
			c.Request.Body = io.NopCloser(bytes.NewReader(body))
		}
		id, err := verify(c.Request, body, cfg.HMACKey, cfg.Window)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "bff_auth_failed"})
			return
		}
		c.Set(CtxUserID, id.UserID)
		c.Set(CtxUserEmail, id.Email)
		c.Set(CtxUserRole, id.Role)
		c.Set(CtxAuthPool, id.Pool)
		c.Next()
	}
}

func verify(r *http.Request, body []byte, key []byte, window time.Duration) (*BFFIdentity, error) {
	sig := r.Header.Get(HdrSignature)
	if sig == "" {
		return nil, ErrBFFMissingSignature
	}
	ts := r.Header.Get(HdrAuthTs)
	tsInt, err := strconv.ParseInt(ts, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("bad X-Auth-Ts: %w", err)
	}
	// Signature check FIRST (constant time, no timing oracle on ts).
	want := compute(r.Method, r.URL.Path, body, ts, key)
	if !hmac.Equal([]byte(sig), []byte(want)) {
		return nil, ErrBFFSignatureMismatch
	}
	// Then check freshness window.
	d := time.Since(time.Unix(tsInt, 0))
	if d > window || d < -window {
		return nil, ErrBFFStaleTimestamp
	}
	return &BFFIdentity{
		UserID: r.Header.Get(HdrUserID),
		Email:  r.Header.Get(HdrUserEmail),
		Role:   r.Header.Get(HdrUserRole),
		Pool:   r.Header.Get(HdrAuthPool),
	}, nil
}

func compute(method, path string, body []byte, ts string, key []byte) string {
	bodyHash := sha256.Sum256(body)
	m := hmac.New(sha256.New, key)
	fmt.Fprintf(m, "%s\n%s\n%s\n%s", method, path, hex.EncodeToString(bodyHash[:]), ts)
	return hex.EncodeToString(m.Sum(nil))
}
