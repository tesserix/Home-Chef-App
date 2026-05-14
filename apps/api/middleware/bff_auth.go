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
	"github.com/google/uuid"
	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
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

// BFFAuthOptional behaves like BFFAuth when the signature header is present —
// it verifies the HMAC and populates the same context keys. When the header
// is absent it lets the request through anonymously, leaving the context
// keys unset. Use this for surfaces that have both a public and a logged-in
// shape (e.g. chef listings that show "favorite" status when the caller is
// authenticated but render fine when they're not).
//
// IMPORTANT: an invalid signature still 401s. Anonymous = no header at all.
func BFFAuthOptional(cfg BFFAuthConfig) gin.HandlerFunc {
	verifier := BFFAuth(cfg)
	return func(c *gin.Context) {
		if c.Request.Header.Get(HdrSignature) == "" {
			c.Next()
			return
		}
		verifier(c)
	}
}

// BFFAuth returns a Gin middleware that verifies the X-Internal-Auth HMAC
// signature applied by the BFF and, on success, populates the Gin context
// with the user identity headers.
//
// The wire format matches apps/auth-bff/internal/headerproxy/signer.go:compute.
// Any drift between the two will manifest as 401s with ErrBFFSignatureMismatch —
// the unit tests in this package replicate the BFF's compute() to catch drift.
//
// Backwards-compat: in addition to the new context keys (user_id, user_email,
// user_role, auth_pool) the middleware also sets the legacy keys (userID as
// uuid.UUID, userEmail as string, userRole as models.UserRole, user as
// *models.User) so handlers written against the pre-GIP AuthMiddleware keep
// working without per-handler edits.
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
		// New-style context keys (raw header values).
		c.Set(CtxUserID, id.UserID)
		c.Set(CtxUserEmail, id.Email)
		c.Set(CtxUserRole, id.Role)
		c.Set(CtxAuthPool, id.Pool)

		// Legacy aliases — typed exactly the way the old AuthMiddleware set
		// them so existing helpers (GetUserID/GetUserRole/GetUser) and any
		// handlers reading c.Get("userID") directly continue to compile and
		// behave the same. Parse failures fall through with empty values;
		// downstream handlers will treat that as "no user".
		if parsed, perr := uuid.Parse(id.UserID); perr == nil {
			c.Set("userID", parsed)
		}
		c.Set("userEmail", id.Email)
		if id.Role != "" {
			c.Set("userRole", models.UserRole(id.Role))
		}

		// Hydrate the User row from DB. This is the same lookup the old
		// AuthMiddleware did, kept here so handlers calling
		// middleware.GetUser(c) keep working. The cost is one indexed
		// primary-key SELECT per protected request — same as before.
		if uid, ok := c.Get("userID"); ok {
			if parsed, ok := uid.(uuid.UUID); ok && database.DB != nil {
				var user models.User
				if err := database.DB.First(&user, "id = ?", parsed).Error; err == nil {
					if !user.IsActive {
						c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Account is suspended"})
						return
					}
					c.Set("user", &user)
				}
			}
		}

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
