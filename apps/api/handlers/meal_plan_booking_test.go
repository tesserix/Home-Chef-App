package handlers

// meal_plan_booking_test.go — HTTP-level checks for the two CreateMealPlan guard
// paths added by #406 (booking from the per-date menu) and #409 (one active plan
// per customer+chef+week). Both are enforced BEFORE the GORM insert, so they run
// cleanly on SQLite (the models' gen_random_uuid() default can't create on SQLite;
// the happy-path create is covered by e2e). Array-column + uuid-default tables are
// hand-DDL'd.

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

const bookDate = "2027-03-15"
const bookDate2 = "2027-03-16"

func setupBookingDB(t *testing.T) (*gorm.DB, uuid.UUID, uuid.UUID) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	for _, s := range []string{
		`CREATE TABLE chef_profiles (id text PRIMARY KEY, user_id text, business_name text, slug text,
			payout_country text DEFAULT '', is_active integer DEFAULT 1, deleted_at datetime)`,
		`CREATE TABLE weekly_menus (id text PRIMARY KEY, chef_id text, is_published integer DEFAULT 0,
			published_at datetime, created_at datetime, updated_at datetime)`,
		`CREATE TABLE weekly_menu_items (id text PRIMARY KEY, chef_id text, day_of_week integer, slot text,
			variant text, name text, description text, price real DEFAULT 0, image_url text,
			dietary_tags text DEFAULT '{}', allergens text DEFAULT '{}', menu_item_id text,
			created_at datetime, updated_at datetime)`,
		`CREATE TABLE daily_menus (id text PRIMARY KEY, chef_id text, date datetime, is_published integer DEFAULT 0,
			published_at datetime, created_at datetime, updated_at datetime)`,
		`CREATE TABLE daily_menu_items (id text PRIMARY KEY, daily_menu_id text, chef_id text, date datetime,
			slot text, variant text, name text, description text, price real DEFAULT 0, image_url text,
			dietary_tags text DEFAULT '{}', allergens text DEFAULT '{}', menu_item_id text,
			is_combo integer DEFAULT 0, combo_components text DEFAULT '{}', sort_order integer DEFAULT 0,
			created_at datetime, updated_at datetime)`,
		`CREATE TABLE meal_plans (id text PRIMARY KEY, meal_plan_number text, customer_id text, chef_id text,
			status text, start_date datetime, end_date datetime, subtotal real DEFAULT 0, tax real DEFAULT 0,
			total real DEFAULT 0, currency text, escrow_payment_id text, razorpay_order_id text,
			chef_respond_by datetime, customer_approve_by datetime, confirmed_at datetime, cancelled_at datetime,
			cancel_reason text, created_at datetime, updated_at datetime)`,
	} {
		require.NoError(t, db.Exec(s).Error)
	}

	userID, chefID := uuid.New(), uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO chef_profiles (id, user_id, business_name, payout_country, is_active) VALUES (?,?,?,?,1)`,
		chefID.String(), userID.String(), "Dum Alooo Kitchen", "").Error)
	require.NoError(t, db.Exec(`INSERT INTO weekly_menus (id, chef_id, is_published) VALUES (?,?,1)`, uuid.NewString(), chefID.String()).Error)
	for dow := 0; dow < 7; dow++ {
		require.NoError(t, db.Exec(`INSERT INTO weekly_menu_items (id, chef_id, day_of_week, slot, variant, name, price) VALUES (?,?,?,?,?,?,?)`,
			uuid.NewString(), chefID.String(), dow, "lunch", "veg", "Rajma Chawal", 100.0).Error)
	}

	orig := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = orig })
	return db, userID, chefID
}

func mealPlanReq(t *testing.T, userID uuid.UUID, body any) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("userID", userID); c.Next() })
	h := &MealPlanHandler{}
	r.POST("/meal-plans", h.CreateMealPlan)
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/meal-plans", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// #406: a dailyMenuItemId not on the chef's PUBLISHED daily menu for that day is
// rejected (the per-date resolution guard).
func TestCreateMealPlan_RejectsUnpublishedDailyItem(t *testing.T) {
	_, userID, chefID := setupBookingDB(t)
	w := mealPlanReq(t, userID, map[string]any{
		"chefId": chefID.String(),
		"days":   []map[string]any{{"date": bookDate, "slot": "lunch", "dailyMenuItemId": uuid.NewString()}},
	})
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
}

// #406: an unpublished daily item is also rejected (only published dates are bookable).
func TestCreateMealPlan_RejectsDraftDailyItem(t *testing.T) {
	db, userID, chefID := setupBookingDB(t)
	menuID, itemID := uuid.NewString(), uuid.NewString()
	// is_published = 0 (draft)
	require.NoError(t, db.Exec(`INSERT INTO daily_menus (id, chef_id, date, is_published) VALUES (?,?,?,0)`, menuID, chefID.String(), bookDate).Error)
	require.NoError(t, db.Exec(`INSERT INTO daily_menu_items (id, daily_menu_id, chef_id, date, slot, variant, name, price) VALUES (?,?,?,?,?,?,?,?)`,
		itemID, menuID, chefID.String(), bookDate, "lunch", "veg", "Draft Combo", 90.0).Error)

	w := mealPlanReq(t, userID, map[string]any{
		"chefId": chefID.String(),
		"days":   []map[string]any{{"date": bookDate, "slot": "lunch", "dailyMenuItemId": itemID}},
	})
	require.Equal(t, http.StatusBadRequest, w.Code, "a draft daily item is not bookable")
}

// #409: a new plan overlapping an existing live plan for the same chef is blocked.
func TestCreateMealPlan_BlocksDuplicateOverlappingPlan(t *testing.T) {
	db, userID, chefID := setupBookingDB(t)
	// An existing pending plan spanning all of March 2027 (brackets bookDate2).
	require.NoError(t, db.Exec(`INSERT INTO meal_plans (id, meal_plan_number, customer_id, chef_id, status, start_date, end_date, total)
		VALUES (?,?,?,?,?,?,?,?)`,
		uuid.NewString(), "MP-existing", userID.String(), chefID.String(), "pending_chef", "2027-03-01", "2027-03-31", 200.0).Error)

	// A new request inside that range (bookDate2 = 2027-03-16) → 409 duplicate_plan.
	w := mealPlanReq(t, userID, map[string]any{
		"chefId": chefID.String(),
		"days":   []map[string]any{{"date": bookDate2, "slot": "lunch", "variant": "veg"}},
	})
	require.Equal(t, http.StatusConflict, w.Code, w.Body.String())
	var out map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out))
	require.Equal(t, "duplicate_plan", out["code"])
}

// #409: a NON-overlapping request (a later week) is NOT blocked by an existing plan.
func TestCreateMealPlan_AllowsNonOverlappingPlan(t *testing.T) {
	db, userID, chefID := setupBookingDB(t)
	require.NoError(t, db.Exec(`INSERT INTO meal_plans (id, meal_plan_number, customer_id, chef_id, status, start_date, end_date, total)
		VALUES (?,?,?,?,?,?,?,?)`,
		uuid.NewString(), "MP-existing", userID.String(), chefID.String(), "pending_chef", "2027-01-01", "2027-01-05", 200.0).Error)

	// bookDate (March) doesn't overlap January → passes the duplicate guard. It then
	// reaches the GORM insert which SQLite can't run (uuid default), so we only assert
	// it's NOT a 409 duplicate_plan.
	w := mealPlanReq(t, userID, map[string]any{
		"chefId": chefID.String(),
		"days":   []map[string]any{{"date": bookDate, "slot": "lunch", "variant": "veg"}},
	})
	require.NotEqual(t, http.StatusConflict, w.Code, "a non-overlapping plan must not hit the duplicate guard")
}
