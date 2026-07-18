package handlers

// delivery_quote_test.go — the checkout quote endpoint. Pins the JSON contract
// the customer app reads (deliveryFee / pickupFee / pickupSaving) and that pickup
// is always free.

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	glogger "gorm.io/gorm/logger"

	"github.com/homechef/api/database"
	"github.com/homechef/api/services"
)

func setupQuoteDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: glogger.Default.LogMode(glogger.Silent)})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TABLE chef_profiles (
		id text PRIMARY KEY, user_id text, business_name text, is_active integer DEFAULT 1,
		offers_pickup integer DEFAULT 0, offers_self_delivery integer DEFAULT 0,
		self_delivery_base_fee real DEFAULT 0, self_delivery_free_radius_km real DEFAULT 0,
		self_delivery_per_km real DEFAULT 0, self_delivery_max_fee real DEFAULT 0,
		self_delivery_max_distance_km real DEFAULT 0,
		latitude real DEFAULT 0, longitude real DEFAULT 0, payout_country text DEFAULT 'IN')`).Error)
	orig := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = orig })
	return db
}

func quote(t *testing.T, chefID uuid.UUID, body string) map[string]any {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/chefs/:id/delivery-quote", (&OrderHandler{}).QuoteDeliveryFee)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/chefs/"+chefID.String()+"/delivery-quote", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var out map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out))
	return out
}

// Delivery costs the flat policy fee (no coords → fallback), pickup is free, and
// the saving equals the delivery fee — the exact numbers the incentive shows.
func TestQuoteEndpoint_ReturnsDeliveryFeeAndPickupSaving(t *testing.T) {
	db := setupQuoteDB(t)
	// Force a known policy fee via a seeded platform_settings-free override: the
	// service reads GetPlatformPolicy, whose default BaseDeliveryFee is 2.99.
	chefID := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO chef_profiles (id, user_id, offers_pickup) VALUES (?,?,1)`,
		chefID.String(), uuid.NewString()).Error)

	out := quote(t, chefID, `{"city":"Pune","country":"IN"}`)

	require.Equal(t, services.GetPlatformPolicy().BaseDeliveryFee, out["deliveryFee"],
		"delivery falls back to the flat policy fee without coords — the same as CreateOrder")
	require.Equal(t, 0.0, out["pickupFee"], "pickup is always free")
	require.Equal(t, out["deliveryFee"], out["pickupSaving"], "the saving is the whole delivery fee")
	require.Equal(t, true, out["offersPickup"])
}

func TestQuoteEndpoint_UnknownChef_Is404(t *testing.T) {
	setupQuoteDB(t)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/chefs/:id/delivery-quote", (&OrderHandler{}).QuoteDeliveryFee)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/chefs/"+uuid.NewString()+"/delivery-quote", strings.NewReader(`{}`)))
	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestQuoteEndpoint_BadChefID_Is400(t *testing.T) {
	setupQuoteDB(t)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/chefs/:id/delivery-quote", (&OrderHandler{}).QuoteDeliveryFee)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/chefs/not-a-uuid/delivery-quote", strings.NewReader(`{}`)))
	require.Equal(t, http.StatusBadRequest, w.Code)
}
