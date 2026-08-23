package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/homechef/api/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAllowedOrigins_ProductionRejectsUnsafeConfiguration(t *testing.T) {
	previous := config.AppConfig
	config.AppConfig = &config.Config{Environment: "production"}
	t.Cleanup(func() { config.AppConfig = previous })
	t.Setenv("CORS_ORIGINS", "*,http://evil.example,https://valid.fe3dr.com,https://evil.example/path")

	assert.Equal(t, []string{"https://valid.fe3dr.com"}, allowedOrigins())
}

func TestCORSMiddleware_AllowsTrustedAndDeniesForeignOrigins(t *testing.T) {
	previous := config.AppConfig
	config.AppConfig = &config.Config{Environment: "production"}
	t.Cleanup(func() { config.AppConfig = previous })
	t.Setenv("CORS_ORIGINS", "https://fe3dr.com")

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(cors.New(corsConfiguration()))
	r.GET("/resource", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	trusted := httptest.NewRequest(http.MethodOptions, "/resource", nil)
	trusted.Header.Set("Origin", "https://fe3dr.com")
	trusted.Header.Set("Access-Control-Request-Method", http.MethodGet)
	trustedResponse := httptest.NewRecorder()
	r.ServeHTTP(trustedResponse, trusted)
	require.Equal(t, http.StatusNoContent, trustedResponse.Code)
	assert.Equal(t, "https://fe3dr.com", trustedResponse.Header().Get("Access-Control-Allow-Origin"))

	foreign := httptest.NewRequest(http.MethodOptions, "/resource", nil)
	foreign.Header.Set("Origin", "https://evil.example")
	foreign.Header.Set("Access-Control-Request-Method", http.MethodGet)
	foreignResponse := httptest.NewRecorder()
	r.ServeHTTP(foreignResponse, foreign)
	assert.Empty(t, foreignResponse.Header().Get("Access-Control-Allow-Origin"))
}
