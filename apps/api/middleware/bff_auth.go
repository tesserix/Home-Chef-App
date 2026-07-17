package middleware

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
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

	// ctxBFFResolved marks that identity resolution has already run for this
	// request (see BFFIdentify). Its value is irrelevant — presence alone tells
	// BFFAuth not to redo the work. Resolution is expensive on the Bearer path
	// (one HTTP round-trip to the BFF's /auth/session per call), so without
	// this marker adding BFFIdentify would double it on every request.
	ctxBFFResolved = "bff_identity_resolved"
)

var (
	ErrBFFMissingSignature  = errors.New("missing X-Internal-Auth")
	ErrBFFStaleTimestamp    = errors.New("stale X-Auth-Ts")
	ErrBFFSignatureMismatch = errors.New("HMAC mismatch")
	// ErrBFFBodyRead is returned when the request body can't be read for
	// re-hashing. Kept distinct so BFFAuth can answer 400 (malformed request)
	// rather than 401 (bad credentials), as it did before resolution was
	// split out of enforcement.
	ErrBFFBodyRead = errors.New("body_read_failed")
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

	// BFFSessionURL, when non-empty, enables a Bearer-token fallback path.
	// If the HMAC verification fails and the request carries an
	// `Authorization: Bearer <token>` header, the middleware will validate
	// the token by calling `<BFFSessionURL>` (typically
	// "http://auth-bff:8090/auth/session") and populate the identity from
	// that response. Intended for mobile/SPA clients that hold a BFF
	// session token directly instead of routing every API call through the
	// BFF proxy. Leave empty in production deployments where every request
	// is expected to come via the BFF and be HMAC-signed.
	BFFSessionURL string
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

// hasBFFCredentials reports whether the request carries anything we could
// authenticate: the BFF's HMAC signature, or a Bearer token for the
// /auth/session fallback that mobile clients use (they call the API directly
// rather than through the BFF proxy, so they never carry a signature).
func hasBFFCredentials(r *http.Request) bool {
	if r.Header.Get(HdrSignature) != "" {
		return true
	}
	return strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ")
}

// BFFIdentify resolves the caller's identity and populates the identity context
// keys, WITHOUT enforcing anything: no credentials or bad credentials simply
// continue anonymously. Enforcement stays entirely with BFFAuth on the
// protected route groups, which still 401s an unidentified caller.
//
// Why this exists: Gin runs group middleware in registration order, so anything
// attached to the /api/v1 group runs BEFORE the per-subgroup BFFAuth. That left
// RateLimitRedis — which keys per-user budgets off c.Get("userID") — unable to
// ever see a user, silently demoting every authenticated caller to the
// per-IP/unauthenticated budget. Registering BFFIdentify ahead of the limiter
// on the same group makes the caller visible in time.
//
// It deliberately does NOT hydrate the User row or check suspension: that stays
// in BFFAuth so a suspended account is still rejected at the enforcement point,
// and so public routes don't pay for a DB lookup.
func BFFIdentify(cfg BFFAuthConfig) gin.HandlerFunc {
	if cfg.Window == 0 {
		cfg.Window = 60 * time.Second
	}
	bearerClient := &http.Client{Timeout: 5 * time.Second}
	return func(c *gin.Context) {
		if !hasBFFCredentials(c.Request) {
			c.Next()
			return
		}
		id, err := resolveBFFIdentity(c, cfg, bearerClient)
		if err != nil {
			// Bad or unverifiable credentials: stay anonymous and let BFFAuth
			// produce the 401 on protected routes. Marking resolution as done
			// would let a forged header skip enforcement, so we deliberately
			// leave the marker unset here.
			c.Next()
			return
		}
		applyBFFIdentity(c, id)
		c.Set(ctxBFFResolved, true)
		c.Next()
	}
}

// resolveBFFIdentity verifies the request's credentials and returns the identity
// they attest to. It never writes to the response — that is the caller's job —
// which is what lets BFFIdentify reuse it without 401ing public routes.
func resolveBFFIdentity(c *gin.Context, cfg BFFAuthConfig, bearerClient *http.Client) (*BFFIdentity, error) {
	// Read the body so we can re-hash it, then restore it for downstream handlers.
	var body []byte
	if c.Request.Body != nil {
		b, err := io.ReadAll(c.Request.Body)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrBFFBodyRead, err)
		}
		body = b
		c.Request.Body = io.NopCloser(bytes.NewReader(body))
	}
	id, err := verify(c.Request, body, cfg.HMACKey, cfg.Window)
	if err == nil {
		return id, nil
	}
	// HMAC path failed. Fall back to Bearer validation against the BFF if
	// configured (mobile/SPA clients). Production deployments that route
	// everything through the BFF proxy leave BFFSessionURL empty.
	if cfg.BFFSessionURL == "" {
		return nil, err
	}
	return verifyBearer(c.Request, cfg.BFFSessionURL, bearerClient)
}

// applyBFFIdentity writes the verified identity onto the Gin context, seeding
// both the new-style keys and the legacy aliases the pre-GIP handlers read.
func applyBFFIdentity(c *gin.Context, id *BFFIdentity) {
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
	bearerClient := &http.Client{Timeout: 5 * time.Second}
	return func(c *gin.Context) {
		// BFFIdentify (registered on the /api/v1 group so the rate limiter can
		// key by user) may already have verified and applied this request's
		// identity. Reuse it: re-verifying would mean a second /auth/session
		// round-trip on the Bearer path. The marker is only ever set after a
		// SUCCESSFUL verification, so bad credentials still fall through to the
		// resolve-and-401 path below. It lives in the Gin context, which no
		// client can influence.
		if _, resolved := c.Get(ctxBFFResolved); !resolved {
			id, err := resolveBFFIdentity(c, cfg, bearerClient)
			if err != nil {
				if errors.Is(err, ErrBFFBodyRead) {
					c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "body_read_failed"})
					return
				}
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "bff_auth_failed"})
				return
			}
			applyBFFIdentity(c, id)
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
	id := BFFIdentity{
		UserID: r.Header.Get(HdrUserID),
		Email:  r.Header.Get(HdrUserEmail),
		Role:   r.Header.Get(HdrUserRole),
		Pool:   r.Header.Get(HdrAuthPool),
	}
	// Signature check FIRST (constant time, no timing oracle on ts).
	// The signature binds the identity headers into the MAC so a signed request
	// can't be replayed with a swapped X-User-Role / X-Auth-Pool. There is no
	// legacy (identity-unbound) fallback: the auth-bff signs exclusively in this
	// format, so an OLD-format signature could only come from an attacker who
	// wants to assert unbound identity headers — reject it.
	want := compute(r.Method, r.URL.Path, body, ts, key, id)
	if !hmac.Equal([]byte(sig), []byte(want)) {
		return nil, ErrBFFSignatureMismatch
	}
	// Then check freshness window.
	d := time.Since(time.Unix(tsInt, 0))
	if d > window || d < -window {
		return nil, ErrBFFStaleTimestamp
	}
	return &id, nil
}

// compute builds the HMAC over the canonical message. The identity fields are
// appended after method/path/body/ts, in the exact order userID, email, role,
// pool.
//
// CRITICAL: this string MUST stay byte-identical to
// apps/auth-bff/internal/headerproxy/signer.go:compute — same header value
// order, same "\n" separators.
func compute(method, path string, body []byte, ts string, key []byte, id BFFIdentity) string {
	bodyHash := sha256.Sum256(body)
	m := hmac.New(sha256.New, key)
	fmt.Fprintf(m, "%s\n%s\n%s\n%s\n%s\n%s\n%s\n%s",
		method, path, hex.EncodeToString(bodyHash[:]), ts,
		id.UserID, id.Email, id.Role, id.Pool)
	return hex.EncodeToString(m.Sum(nil))
}

// verifyBearer validates a Bearer token by calling the BFF's /auth/session
// endpoint. Returns the BFF identity on success. Used as a fallback path for
// mobile/SPA clients that hold a session token directly instead of routing
// through the BFF's HMAC-signing proxy.
func verifyBearer(r *http.Request, bffSessionURL string, client *http.Client) (*BFFIdentity, error) {
	auth := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if !strings.HasPrefix(auth, prefix) {
		return nil, errors.New("no_bearer_token")
	}
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, bffSessionURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", auth)
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("bff_session_unreachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bff_session_status_%d", resp.StatusCode)
	}
	var body struct {
		UserID string `json:"user_id"`
		Email  string `json:"email"`
		Role   string `json:"role"`
		Pool   string `json:"pool"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("bff_session_decode: %w", err)
	}
	if body.UserID == "" {
		return nil, errors.New("bff_session_no_user_id")
	}
	return &BFFIdentity{
		UserID: body.UserID,
		Email:  body.Email,
		Role:   body.Role,
		Pool:   body.Pool,
	}, nil
}
