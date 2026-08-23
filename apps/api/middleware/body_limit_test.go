package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBodyLimit_RejectsOversizedRequestBeforeHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	called := false
	r := gin.New()
	r.Use(BodyLimit(4))
	r.POST("/upload", func(c *gin.Context) {
		called = true
		c.Status(http.StatusOK)
	})
	req := httptest.NewRequest(http.MethodPost, "/upload", strings.NewReader("12345"))
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusRequestEntityTooLarge, w.Code)
	assert.False(t, called)
}
