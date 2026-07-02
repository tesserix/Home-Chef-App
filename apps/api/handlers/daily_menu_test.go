package handlers

// daily_menu_test.go — HTTP-level checks for the per-date tiffin menu (#405).
// The defining behaviour vs the weekly menu is MULTIPLE dishes per (date, slot),
// so that is the headline assertion. Also covers validation, the publish gate on
// the public read, and the range parser. In-memory SQLite with hand-written DDL
// (the pq.StringArray text[] columns don't AutoMigrate on SQLite).

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/homechef/api/database"
)

func setupDailyMenuDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	stmts := []string{
		`CREATE TABLE chef_profiles (id text PRIMARY KEY, user_id text, business_name text, slug text,
			is_active integer DEFAULT 1, created_at datetime, updated_at datetime, deleted_at datetime)`,
		`CREATE TABLE daily_menus (id text PRIMARY KEY, chef_id text, date datetime, is_published integer DEFAULT 0,
			published_at datetime, created_at datetime, updated_at datetime)`,
		`CREATE TABLE daily_menu_items (id text PRIMARY KEY, daily_menu_id text, chef_id text, date datetime,
			slot text, variant text, name text, description text, price real DEFAULT 0, image_url text,
			dietary_tags text DEFAULT '{}', allergens text DEFAULT '{}', menu_item_id text,
			is_combo integer DEFAULT 0, combo_components text DEFAULT '{}', sort_order integer DEFAULT 0,
			created_at datetime, updated_at datetime)`,
		`CREATE TABLE outbox_events (id text PRIMARY KEY, subject text, msg_id text, aggregate_type text,
			aggregate_id text, payload text, status text DEFAULT 'pending', attempts integer DEFAULT 0,
			last_error text, next_retry_at datetime, created_at datetime, updated_at datetime, published_at datetime)`,
	}
	for _, s := range stmts {
		require.NoError(t, db.Exec(s).Error)
	}
	orig := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = orig })
	return db
}

// seedChef inserts a chef row and returns (userID, chefID).
func seedDailyChef(t *testing.T, db *gorm.DB) (uuid.UUID, uuid.UUID) {
	t.Helper()
	userID, chefID := uuid.New(), uuid.New()
	require.NoError(t, db.Exec(
		`INSERT INTO chef_profiles (id, user_id, business_name, is_active) VALUES (?,?,?,1)`,
		chefID.String(), userID.String(), "Dum Alooo Kitchen").Error)
	return userID, chefID
}

// chefReq drives a request through a gin engine with userID injected (mirrors the
// bffAuth contract) and the daily-menu routes registered.
func chefReq(t *testing.T, userID uuid.UUID, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("userID", userID); c.Next() })
	h := &ChefHandler{}
	r.PUT("/chef/daily-menu/:date", h.PutDailyMenu)
	r.GET("/chef/daily-menu", h.GetMyDailyMenu)
	r.GET("/chefs/:id/daily-menu", h.GetPublicDailyMenu)

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

func dishesResp(t *testing.T, w *httptest.ResponseRecorder) []map[string]any {
	t.Helper()
	var out struct {
		Days []struct {
			Date        string           `json:"date"`
			IsPublished bool             `json:"isPublished"`
			Items       []map[string]any `json:"items"`
		} `json:"days"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out))
	if len(out.Days) == 0 {
		return nil
	}
	return out.Days[0].Items
}

// The headline: a chef can list MULTIPLE dishes in one (date, slot) — impossible
// in the weekly menu (one dish per weekday×slot×variant).
func TestPutAndGetDailyMenu_MultiDishPerSlot(t *testing.T) {
	db := setupDailyMenuDB(t)
	userID, _ := seedDailyChef(t, db)

	body := dailyMenuUpsertRequest{Items: []dailyMenuItemInput{
		{Slot: "lunch", Variant: "veg", Name: "Rice", Price: 20},
		{Slot: "lunch", Variant: "veg", Name: "Dal", Price: 30},
		{Slot: "lunch", Variant: "veg", Name: "Bhindi", Price: 40},
		{Slot: "lunch", Variant: "nonveg", Name: "Chicken Curry", Price: 120},
		{Slot: "dinner", Variant: "veg", Name: "Roti", Price: 15},
	}}
	w := chefReq(t, userID, http.MethodPut, "/chef/daily-menu/2026-07-10", body)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	w = chefReq(t, userID, http.MethodGet, "/chef/daily-menu?from=2026-07-10&to=2026-07-10", nil)
	require.Equal(t, http.StatusOK, w.Code)
	items := dishesResp(t, w)
	require.Len(t, items, 5)
	lunch := 0
	for _, it := range items {
		if it["slot"] == "lunch" {
			lunch++
		}
	}
	require.Equal(t, 4, lunch, "four dishes in the same (date, lunch) slot")

	// Replace-all: a second PUT with fewer dishes replaces the day.
	w = chefReq(t, userID, http.MethodPut, "/chef/daily-menu/2026-07-10",
		dailyMenuUpsertRequest{Items: []dailyMenuItemInput{{Slot: "lunch", Variant: "veg", Name: "Khichdi", Price: 50}}})
	require.Equal(t, http.StatusOK, w.Code)
	w = chefReq(t, userID, http.MethodGet, "/chef/daily-menu?from=2026-07-10&to=2026-07-10", nil)
	require.Len(t, dishesResp(t, w), 1)
}

func TestDailyMenuValidation(t *testing.T) {
	db := setupDailyMenuDB(t)
	userID, _ := seedDailyChef(t, db)

	cases := []struct {
		name string
		body dailyMenuUpsertRequest
		path string
	}{
		{"bad slot", dailyMenuUpsertRequest{Items: []dailyMenuItemInput{{Slot: "brunch", Variant: "veg", Name: "X", Price: 1}}}, "/chef/daily-menu/2026-07-10"},
		{"empty name", dailyMenuUpsertRequest{Items: []dailyMenuItemInput{{Slot: "lunch", Variant: "veg", Name: "", Price: 1}}}, "/chef/daily-menu/2026-07-10"},
		{"negative price", dailyMenuUpsertRequest{Items: []dailyMenuItemInput{{Slot: "lunch", Variant: "veg", Name: "X", Price: -5}}}, "/chef/daily-menu/2026-07-10"},
		{"publish with no dishes", dailyMenuUpsertRequest{IsPublished: true}, "/chef/daily-menu/2026-07-10"},
		{"bad date", dailyMenuUpsertRequest{Items: []dailyMenuItemInput{{Slot: "lunch", Variant: "veg", Name: "X", Price: 1}}}, "/chef/daily-menu/not-a-date"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := chefReq(t, userID, http.MethodPut, tc.path, tc.body)
			require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
		})
	}
}

func TestDailyMenuPublishGatesPublicRead(t *testing.T) {
	db := setupDailyMenuDB(t)
	userID, chefID := seedDailyChef(t, db)

	// Draft (unpublished) date.
	require.Equal(t, http.StatusOK, chefReq(t, userID, http.MethodPut, "/chef/daily-menu/2026-07-11",
		dailyMenuUpsertRequest{Items: []dailyMenuItemInput{{Slot: "lunch", Variant: "veg", Name: "Draft", Price: 10}}}).Code)
	// Published date.
	require.Equal(t, http.StatusOK, chefReq(t, userID, http.MethodPut, "/chef/daily-menu/2026-07-12",
		dailyMenuUpsertRequest{IsPublished: true, Items: []dailyMenuItemInput{{Slot: "lunch", Variant: "veg", Name: "Live", Price: 10}}}).Code)

	// Public read over the range returns ONLY the published date.
	w := chefReq(t, userID, http.MethodGet, "/chefs/"+chefID.String()+"/daily-menu?from=2026-07-11&to=2026-07-12", nil)
	require.Equal(t, http.StatusOK, w.Code)
	var out struct {
		Days []struct {
			Date  string           `json:"date"`
			Items []map[string]any `json:"items"`
		} `json:"days"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out))
	require.Len(t, out.Days, 1, "only the published date is public")
	require.Equal(t, "2026-07-12", out.Days[0].Date)
}

// A chef can bundle the day's dishes into a priced thali that round-trips, and a
// malformed thali (too few components / no price) is rejected. (#406)
func TestDailyMenuCombo(t *testing.T) {
	db := setupDailyMenuDB(t)
	userID, _ := seedDailyChef(t, db)

	body := dailyMenuUpsertRequest{Items: []dailyMenuItemInput{
		{Slot: "lunch", Variant: "veg", Name: "Rice", Price: 20},
		{Slot: "lunch", Variant: "veg", Name: "Dal", Price: 30},
		{Slot: "lunch", Variant: "veg", Name: "Veg Combo", Price: 90, IsCombo: true,
			ComboComponents: []string{"Rice", "Dal", "Bhindi", "Chutney", "Papad"}},
	}}
	require.Equal(t, http.StatusOK, chefReq(t, userID, http.MethodPut, "/chef/daily-menu/2026-07-15", body).Code)

	w := chefReq(t, userID, http.MethodGet, "/chef/daily-menu?from=2026-07-15&to=2026-07-15", nil)
	items := dishesResp(t, w)
	require.Len(t, items, 3)
	var thali map[string]any
	for _, it := range items {
		if it["isCombo"] == true {
			thali = it
		}
	}
	require.NotNil(t, thali, "the thali is returned")
	require.Equal(t, float64(90), thali["price"])
	require.Len(t, thali["comboComponents"], 5)

	// Malformed thali: fewer than two components → 400.
	bad := dailyMenuUpsertRequest{Items: []dailyMenuItemInput{
		{Slot: "lunch", Variant: "veg", Name: "Half Combo", Price: 50, IsCombo: true, ComboComponents: []string{"Rice"}},
	}}
	require.Equal(t, http.StatusBadRequest, chefReq(t, userID, http.MethodPut, "/chef/daily-menu/2026-07-15", bad).Code)
	// Thali with no price → 400.
	bad2 := dailyMenuUpsertRequest{Items: []dailyMenuItemInput{
		{Slot: "lunch", Variant: "veg", Name: "Free Combo", Price: 0, IsCombo: true, ComboComponents: []string{"Rice", "Dal"}},
	}}
	require.Equal(t, http.StatusBadRequest, chefReq(t, userID, http.MethodPut, "/chef/daily-menu/2026-07-15", bad2).Code)
}

func TestParseDailyRangeDefault(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, "/x", nil) // no from/to
	from, to, ok := parseDailyRange(c)
	require.True(t, ok)
	require.Equal(t, 14, int(to.Sub(from).Hours()/24), "default window is today..+14d")
}
