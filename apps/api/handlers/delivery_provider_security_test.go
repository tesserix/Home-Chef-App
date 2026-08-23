package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateProvider_RejectsUnsafeAPIBaseURLBeforeDatabaseAccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/providers", NewDeliveryProviderHandler().CreateProvider)
	req := httptest.NewRequest(http.MethodPost, "/providers", strings.NewReader(
		`{"name":"Attacker","code":"attacker","apiBaseUrl":"http://169.254.169.254/latest/meta-data"}`,
	))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "public HTTPS URL")
}
