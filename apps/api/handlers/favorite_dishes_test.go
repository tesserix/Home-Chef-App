package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/homechef/api/database"
)

// favorite_dishes_test.go — #237. Pins the business logic of the dish-favorites
// handlers: existence check, duplicate rejection, the per-user cap, and that
// add/remove/ids round-trip correctly. The full ListFavoriteDishes preload
// touches pq.StringArray columns that SQLite can't model, so (like the existing
// chef-favorites code) the read-with-join path is left to integration; here we
// guarantee the decision logic that protects data integrity.

func setupFavDishDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	// menu_items: only the columns the handlers touch (id + soft-delete scope).
	require.NoError(t, db.Exec(`
		CREATE TABLE menu_items (
			id         TEXT PRIMARY KEY,
			chef_id    TEXT,
			name       TEXT,
			deleted_at DATETIME
		)
	`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE favorite_dishes (
			id           TEXT PRIMARY KEY,
			user_id      TEXT NOT NULL,
			menu_item_id TEXT NOT NULL,
			created_at   DATETIME
		)
	`).Error)
	require.NoError(t, db.Exec(
		`CREATE UNIQUE INDEX idx_favorite_dishes_user_item ON favorite_dishes(user_id, menu_item_id)`,
	).Error)
	return db
}

func seedDish(t *testing.T, db *gorm.DB, chefID uuid.UUID) uuid.UUID {
	t.Helper()
	id := uuid.New()
	require.NoError(t, db.Exec(
		`INSERT INTO menu_items (id, chef_id, name) VALUES (?, ?, ?)`,
		id.String(), chefID.String(), "Butter Chicken",
	).Error)
	return id
}

func favDishReq(t *testing.T, userID uuid.UUID, method, path string, register func(*gin.Engine, *FavoriteHandler), body any) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("userID", userID); c.Next() })
	register(r, NewFavoriteHandler())
	var rdr *bytes.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rdr = bytes.NewReader(b)
	} else {
		rdr = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, rdr)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestAddFavoriteDish(t *testing.T) {
	db := setupFavDishDB(t)
	database.DB = db
	uid := uuid.New()
	dishID := seedDish(t, db, uuid.New())

	add := func(menuItemID string) *httptest.ResponseRecorder {
		return favDishReq(t, uid, http.MethodPost, "/favorites/dishes",
			func(r *gin.Engine, h *FavoriteHandler) { r.POST("/favorites/dishes", h.AddFavoriteDish) },
			map[string]string{"menuItemId": menuItemID})
	}

	t.Run("adds a dish", func(t *testing.T) {
		w := add(dishID.String())
		assert.Equal(t, http.StatusCreated, w.Code)
		var n int64
		db.Raw(`SELECT count(*) FROM favorite_dishes WHERE user_id = ?`, uid.String()).Scan(&n)
		assert.Equal(t, int64(1), n)
	})

	t.Run("rejects duplicate", func(t *testing.T) {
		w := add(dishID.String())
		assert.Equal(t, http.StatusConflict, w.Code)
	})

	t.Run("rejects unknown dish", func(t *testing.T) {
		w := add(uuid.New().String())
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("rejects invalid uuid", func(t *testing.T) {
		w := add("not-a-uuid")
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestAddFavoriteDishCap(t *testing.T) {
	db := setupFavDishDB(t)
	database.DB = db
	uid := uuid.New()
	chefID := uuid.New()

	// Fill to the cap with distinct dishes.
	for i := 0; i < maxFavoriteDishes; i++ {
		d := seedDish(t, db, chefID)
		require.NoError(t, db.Exec(
			`INSERT INTO favorite_dishes (id, user_id, menu_item_id, created_at) VALUES (?, ?, ?, ?)`,
			uuid.New().String(), uid.String(), d.String(), time.Now(),
		).Error)
	}
	over := seedDish(t, db, chefID)

	w := favDishReq(t, uid, http.MethodPost, "/favorites/dishes",
		func(r *gin.Engine, h *FavoriteHandler) { r.POST("/favorites/dishes", h.AddFavoriteDish) },
		map[string]string{"menuItemId": over.String()})
	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestRemoveAndListFavoriteDishIDs(t *testing.T) {
	db := setupFavDishDB(t)
	database.DB = db
	uid := uuid.New()
	d1 := seedDish(t, db, uuid.New())
	d2 := seedDish(t, db, uuid.New())
	for _, d := range []uuid.UUID{d1, d2} {
		require.NoError(t, db.Exec(
			`INSERT INTO favorite_dishes (id, user_id, menu_item_id, created_at) VALUES (?, ?, ?, ?)`,
			uuid.New().String(), uid.String(), d.String(), time.Now(),
		).Error)
	}

	// IDs list returns both.
	w := favDishReq(t, uid, http.MethodGet, "/favorites/dishes/ids",
		func(r *gin.Engine, h *FavoriteHandler) { r.GET("/favorites/dishes/ids", h.ListFavoriteDishIDs) }, nil)
	require.Equal(t, http.StatusOK, w.Code)
	var idsResp struct {
		MenuItemIDs []string `json:"menuItemIds"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &idsResp))
	assert.Len(t, idsResp.MenuItemIDs, 2)

	// Remove d1.
	wd := favDishReq(t, uid, http.MethodDelete, fmt.Sprintf("/favorites/dishes/%s", d1),
		func(r *gin.Engine, h *FavoriteHandler) { r.DELETE("/favorites/dishes/:menuItemId", h.RemoveFavoriteDish) }, nil)
	assert.Equal(t, http.StatusOK, wd.Code)

	// Removing again → 404.
	wd2 := favDishReq(t, uid, http.MethodDelete, fmt.Sprintf("/favorites/dishes/%s", d1),
		func(r *gin.Engine, h *FavoriteHandler) { r.DELETE("/favorites/dishes/:menuItemId", h.RemoveFavoriteDish) }, nil)
	assert.Equal(t, http.StatusNotFound, wd2.Code)
}
