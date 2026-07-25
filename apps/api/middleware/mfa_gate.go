package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
)

// The two-factor gate.
//
// It runs after identity resolution and asks one question: has this user
// satisfied their second factor on this device? If not, every route outside the
// allowlist is refused, so there is no ordering in which a resource handler can
// run for an unchallenged session.
//
// Enforcement sits here rather than in auth-bff because mobile authenticates
// straight to this API with a Bearer token and only consults the BFF to resolve
// identity. A gate in the BFF would cover the web portals and leave mobile open.

const (
	// HdrDeviceToken carries either a remembered-device token or a
	// session-elevation token. One header, because the client should not have to
	// know which kind it holds — it stores what the verify call returned and
	// sends it back.
	HdrDeviceToken = "X-Device-Token"
	// HdrClientApp identifies which app is calling, so device trust stays
	// scoped: a token lifted from the vendor app must not vouch for admin.
	HdrClientApp = "X-Client-App"
)

// CtxMFAChallenged marks a request that passed the gate, for handlers that want
// to require a freshly-challenged session for a sensitive action.
const CtxMFAChallenged = "mfa_challenged"

// mfaExemptPrefixes are the paths that must stay reachable while a user is
// mid-challenge. Getting this list wrong in either direction is the classic bug
// in a feature like this: too narrow and the user cannot complete the challenge
// that unblocks them, too broad and the gate leaks.
var mfaExemptPrefixes = []string{
	"/api/v1/auth/mfa/",
	"/api/v1/auth/password-policy",
	"/health",
	"/healthz",
	"/readyz",
	"/metrics",
	"/version",
}

func isMFAExempt(path string) bool {
	for _, p := range mfaExemptPrefixes {
		if strings.HasPrefix(path, p) {
			return true
		}
	}
	return false
}

// ClientAppFrom resolves the calling app, defaulting to customer when the header
// is absent so an older client that predates this feature keeps working. The
// default is the least-privileged app, and it is only ever used to scope device
// trust — never to grant it.
//
// Exported because the verify handler must mint a token under exactly the scope
// the gate will later check it against; two independent readings of the header
// would drift.
func ClientAppFrom(c *gin.Context) string {
	app := strings.ToLower(strings.TrimSpace(c.GetHeader(HdrClientApp)))
	if services.ValidMFAApps[app] {
		return app
	}
	return "customer"
}

// MFAGate refuses requests from users who have two-factor on and have not
// satisfied it on this device.
func MFAGate(db *gorm.DB, enabled bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !enabled || isMFAExempt(c.Request.URL.Path) {
			c.Next()
			return
		}
		userID, ok := GetUserID(c)
		if !ok {
			// Unauthenticated requests are not this middleware's business; the
			// auth middleware has already decided, or the route is public.
			c.Next()
			return
		}

		app := ClientAppFrom(c)
		req, err := services.EvaluateMFA(
			c.Request.Context(), db, enabled, userID, app, c.GetHeader(HdrDeviceToken),
		)
		if err != nil {
			// A database failure must not become a way past the second factor.
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
				"error": "could not verify two-factor status",
			})
			return
		}
		if !req.Required {
			c.Set(CtxMFAChallenged, true)
			c.Next()
			return
		}

		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"error":    "mfa_required",
			"channels": req.Channels,
			"masked":   maskedHints(db, c, userID, req.Channels),
		})
	}
}

// maskedHints tells the client where a code would go, without disclosing the
// address. Best-effort: a lookup failure yields no hint rather than blocking the
// challenge, since the user can still pick a channel by name.
func maskedHints(db *gorm.DB, c *gin.Context, userID uuid.UUID, channels []services.MFAChannel) map[string]string {
	out := map[string]string{}
	var s models.UserMFASettings
	if err := db.Where("user_id = ?", userID).First(&s).Error; err != nil {
		return out
	}
	email, _ := c.Get(CtxUserEmail)
	emailStr, _ := email.(string)
	for _, ch := range channels {
		if _, masked, err := services.ChallengeSubject(s, emailStr, ch); err == nil {
			out[string(ch)] = masked
		}
	}
	return out
}
