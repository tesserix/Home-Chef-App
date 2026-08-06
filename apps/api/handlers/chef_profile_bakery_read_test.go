package handlers

// chef_profile_bakery_read_test.go — the opt-in has to survive a reload. The
// vendor app decides whether to show the cake configurator from what
// GET /chef/profile returns, so a field that saves but never comes back reads
// to the chef as the feature simply not existing (#1065).

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func getChefProfile(t *testing.T, userID uuid.UUID) map[string]any {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("userID", userID); c.Next() })
	r.GET("/chef/profile", (&ChefHandler{}).GetChefProfile)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/chef/profile", nil))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	return body
}

func TestGetChefProfile_ReturnsTheBakeryOptIn(t *testing.T) {
	db := setupChefGuardDB(t)
	userID, chefID := seedLiveKitchen(t, db)
	require.NoError(t, db.Exec(`UPDATE chef_profiles SET sells_bakery = 1 WHERE id = ?`, chefID.String()).Error)

	body := getChefProfile(t, userID)

	require.Equal(t, true, body["sellsBakery"])
	require.Equal(t, "kitchen", body["vertical"], "the store is still listed as a kitchen")
}

func TestGetChefProfile_MealsOnlyKitchenSellsNoBakery(t *testing.T) {
	db := setupChefGuardDB(t)
	userID, _ := seedLiveKitchen(t, db)

	require.Equal(t, false, getChefProfile(t, userID)["sellsBakery"])
}

func TestGetChefProfile_BakeryVerticalOffersBakeryWithoutTheOptIn(t *testing.T) {
	db := setupChefGuardDB(t)
	userID, chefID := seedLiveKitchen(t, db)
	require.NoError(t, db.Exec(`UPDATE chef_profiles SET vertical = 'bakery' WHERE id = ?`, chefID.String()).Error)

	body := getChefProfile(t, userID)

	require.Equal(t, true, body["sellsBakery"])
	require.Equal(t, "bakery", body["vertical"])
}
