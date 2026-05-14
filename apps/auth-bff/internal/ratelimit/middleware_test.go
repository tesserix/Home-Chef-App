package ratelimit

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAllow_WithinBurst_Allowed(t *testing.T) {
	l := New(1.0, 3) // 1 rps, burst 3
	require.True(t, l.Allow("1.2.3.4"))
	require.True(t, l.Allow("1.2.3.4"))
	require.True(t, l.Allow("1.2.3.4"))
	// 4th in same burst → denied
	require.False(t, l.Allow("1.2.3.4"))
}

func TestAllow_DifferentKeys_Independent(t *testing.T) {
	l := New(1.0, 1) // 1 rps, burst 1
	require.True(t, l.Allow("1.1.1.1"))
	require.False(t, l.Allow("1.1.1.1")) // exhausted
	require.True(t, l.Allow("2.2.2.2"))  // separate key still allowed
}

func TestMiddleware_429AfterBurst(t *testing.T) {
	gin.SetMode(gin.TestMode)
	l := New(1.0, 2)
	r := gin.New()
	r.Use(l.Middleware())
	r.POST("/x", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })

	doReq := func(remoteAddr string) int {
		req := httptest.NewRequest(http.MethodPost, "/x", nil)
		req.RemoteAddr = remoteAddr + ":1234"
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w.Code
	}

	assert.Equal(t, http.StatusOK, doReq("9.9.9.9"))
	assert.Equal(t, http.StatusOK, doReq("9.9.9.9"))
	assert.Equal(t, http.StatusTooManyRequests, doReq("9.9.9.9"))
}
