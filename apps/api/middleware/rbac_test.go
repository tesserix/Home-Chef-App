package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/homechef/api/models"
)

// setPool seeds the auth_pool context key the way BFFAuth does, for pool tests.
func poolRouter(pools ...models.AuthPool) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/admin/x",
		func(c *gin.Context) {
			if v := c.Query("pool"); v != "" {
				c.Set(CtxAuthPool, v)
			}
			c.Next()
		},
		RequirePool(pools...),
		func(c *gin.Context) { c.Status(http.StatusOK) },
	)
	return r
}

// payoutPermRouter mimics the /admin/payouts/* wiring: it pre-seeds a cached
// staffMember (as BFFAuth+an earlier chain link would) and then applies the
// SPManagePayouts gate, so the test exercises the permission decision without a
// DB round-trip.
func payoutPermRouter(role models.StaffRole) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/admin/payouts/:aggType/:id/release",
		func(c *gin.Context) {
			c.Set("userID", uuid.New())
			c.Set("staffMember", &models.StaffMember{StaffRole: role, IsActive: true})
			c.Next()
		},
		RequireStaffPermission(models.SPManagePayouts),
		func(c *gin.Context) { c.Status(http.StatusOK) },
	)
	return r
}

func TestRequireStaffPermission_PayoutsSuperAdminAllowed(t *testing.T) {
	r := payoutPermRouter(models.StaffRoleSuperAdmin)
	req := httptest.NewRequest("POST", "/admin/payouts/order/abc/release", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestRequireStaffPermission_PayoutsNonFinanceRejected(t *testing.T) {
	// A plain admin-role staff can reach the admin panel but must be 403'd on the
	// money-release surface — SPManagePayouts is super_admin-only.
	r := payoutPermRouter(models.StaffRoleAdmin)
	req := httptest.NewRequest("POST", "/admin/payouts/order/abc/release", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestRequirePool_InternalAllowed(t *testing.T) {
	r := poolRouter(models.PoolInternal)
	req := httptest.NewRequest("POST", "/admin/x?pool=internal", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestRequirePool_WrongPoolRejected(t *testing.T) {
	// An admin-role user who signed in through the customer pool must not reach
	// an internal-pool-gated endpoint.
	r := poolRouter(models.PoolInternal)
	req := httptest.NewRequest("POST", "/admin/x?pool=customer", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestRequirePool_MissingPoolRejected(t *testing.T) {
	// Fails closed: no pool on the context → deny.
	r := poolRouter(models.PoolInternal)
	req := httptest.NewRequest("POST", "/admin/x", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusForbidden, w.Code)
}

// consoleAdminRouter mimics an admin arriving from the tesserix.app console:
// the signature has been verified upstream, so the signed email is on the
// context, but there is NO "userID" — the console signs with its own OIDC
// subject, which is not a HomeChef users.id and often not a UUID at all, so
// applyBFFIdentity never sets the legacy key. No staffMember is pre-seeded and
// no user row exists, so the DB is never consulted on this path.
func consoleAdminRouter(email string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/admin/payouts/blocked-chefs",
		func(c *gin.Context) {
			if email != "" {
				c.Set(CtxUserEmail, email)
			}
			c.Next()
		},
		RequireStaffPermission(models.SPManagePayouts),
		func(c *gin.Context) { c.Status(http.StatusOK) },
	)
	return r
}

func serveConsole(t *testing.T, email string) int {
	t.Helper()
	w := httptest.NewRecorder()
	consoleAdminRouter(email).ServeHTTP(w, httptest.NewRequest("GET", "/admin/payouts/blocked-chefs", nil))
	return w.Code
}

func TestRequireStaffPermission_ConsoleSuperAdminWithoutUserIDAllowed(t *testing.T) {
	// Regression: this returned 401 for every console admin, which took out all
	// of Payouts, Payout Setup, Staff and Fleet while the unguarded admin
	// endpoints kept working. Identity comes from the signature, not from a
	// resolvable HomeChef user id.
	require.Equal(t, http.StatusOK, serveConsole(t, "samyak.rout@gmail.com"))
}

func TestRequireStaffPermission_ConsoleSuperAdminEmailIsCaseInsensitive(t *testing.T) {
	require.Equal(t, http.StatusOK, serveConsole(t, "  Samyak.Rout@Gmail.com  "))
}

func TestRequireStaffPermission_ConsoleUnlistedEmailRejected(t *testing.T) {
	// A signature-verified admin who is not on the super-admin allowlist still
	// has no staff record, so the gate must deny.
	require.Equal(t, http.StatusForbidden, serveConsole(t, "someone.else@fe3dr.com"))
}

func TestRequireStaffPermission_NoIdentityFailsClosed(t *testing.T) {
	// Neither a user id nor a signed email: deny. Guards against the fallback
	// turning an empty email into super-admin access.
	require.Equal(t, http.StatusForbidden, serveConsole(t, ""))
}
