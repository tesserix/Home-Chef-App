package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
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
