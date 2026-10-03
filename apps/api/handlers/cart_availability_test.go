package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/homechef/api/database"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestCartAvailabilityScopesAddressAndRadius(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	previous := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = previous })
	require.NoError(t, db.Exec(`CREATE TABLE addresses (id text, user_id text, country text, latitude real, longitude real)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE chef_profiles (id text, mode text, is_active boolean, is_verified boolean, payout_country text, latitude real, longitude real, service_radius real)`).Error)
	user, other, address, near, far, overseas := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO addresses VALUES (?, ?, 'NZ', -36.844, 174.767)`, address, user).Error)
	for _, row := range []struct {
		id      uuid.UUID
		country string
		lat     float64
	}{{near, "NZ", -36.845}, {far, "NZ", -37.0}, {overseas, "IN", -36.845}} {
		require.NoError(t, db.Exec(`INSERT INTO chef_profiles VALUES (?, 'live', 1, 1, ?, ?, 174.767, 5)`, row.id, row.country, row.lat).Error)
	}
	call := func(owner uuid.UUID) *httptest.ResponseRecorder {
		r := gin.New()
		r.Use(func(c *gin.Context) { c.Set("userID", owner) })
		r.POST("/addresses/:id/cart-availability", NewAddressHandler().CartAvailability)
		body, _ := json.Marshal(map[string]any{"chefIds": []uuid.UUID{near, far, overseas}})
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/addresses/"+address.String()+"/cart-availability", strings.NewReader(string(body)))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		return w
	}
	w := call(user)
	require.Equal(t, 200, w.Code, w.Body.String())
	var result struct {
		ChefIDs []uuid.UUID `json:"chefIds"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result))
	require.Equal(t, []uuid.UUID{near}, result.ChefIDs)
	require.Equal(t, 404, call(other).Code)
	require.NoError(t, db.Exec(`UPDATE addresses SET latitude = 0, longitude = 0`).Error)
	w = call(user)
	require.Equal(t, 200, w.Code)
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result))
	require.Empty(t, result.ChefIDs)
	require.NoError(t, db.Exec("DROP TABLE addresses").Error)
	require.Equal(t, 500, call(user).Code)

}
