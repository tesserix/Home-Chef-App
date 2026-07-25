package handlers

// loyalty_admin_test.go — HTTP-level check for the admin points grant/adjust
// endpoint (#40 Task 8): posting a grant returns 200 with the resulting
// balance. Reuses the loyalty handler-test harness (setupLoyaltyHandlerDB) —
// the ledger math itself is pinned in the services package.

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/homechef/api/database"
	"github.com/homechef/api/services"
)

func TestGrantLoyaltyPoints_Credit(t *testing.T) {
	setupLoyaltyHandlerDB(t)
	adminID := uuid.New()
	targetUserID := uuid.New()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("userID", adminID); c.Next() })
	r.POST("/admin/loyalty/grant", NewAdminHandler().GrantLoyaltyPoints)

	body, _ := json.Marshal(map[string]any{
		"userId": targetUserID.String(),
		"points": 250,
		"reason": "goodwill",
	})
	req := httptest.NewRequest(http.MethodPost, "/admin/loyalty/grant", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 250.0, resp["pointsBalance"])

	acct, err := services.LoyaltyBalance(database.DB, targetUserID)
	require.NoError(t, err)
	assert.Equal(t, 250.0, acct.Balance)
}
