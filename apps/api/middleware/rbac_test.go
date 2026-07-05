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
