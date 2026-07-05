package handlers

// chef_orders_visibility_test.go — the "3 orders but queue is clear" bug: an
// order row is created BEFORE payment, and the three vendor surfaces disagreed
// on whether unpaid rows count. The dashboard's todayOrders counted ALL rows
// created today, while the orders queue/history and pendingOrders required
// payment completed — so the header said "3 Orders" while both tabs were empty
// (and the customer, whose own list has no payment filter, sat waiting on an
// order the chef could never see). These tests pin the single shared predicate:
// unpaid/abandoned checkouts appear NOWHERE in the chef app; the dashboard
// counters and the orders tab must agree exactly.
//
// Self-contained in-memory SQLite (mirrors payment_test.go's harness; distinct
// helper names). Not parallel — shares the global DB.

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/homechef/api/database"
	"github.com/homechef/api/services"
)

func setupChefVisDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	require.NoError(t, err)

	require.NoError(t, db.Exec(`CREATE TABLE users (
		id TEXT PRIMARY KEY, email TEXT, first_name TEXT DEFAULT '', last_name TEXT DEFAULT '',
		phone TEXT DEFAULT '', role TEXT DEFAULT 'customer', is_active INTEGER DEFAULT 1,
		created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
	)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE chef_profiles (
		id TEXT PRIMARY KEY, user_id TEXT, business_name TEXT DEFAULT '',
		rating REAL DEFAULT 0, total_reviews INTEGER DEFAULT 0, total_orders INTEGER DEFAULT 0,
		accepting_orders INTEGER DEFAULT 1, paused_until DATETIME,
		created_at DATETIME, updated_at DATETIME
	)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE orders (
		id TEXT PRIMARY KEY, order_number TEXT, customer_id TEXT, chef_id TEXT,
		status TEXT DEFAULT 'pending', payment_status TEXT DEFAULT 'pending',
		payment_method TEXT DEFAULT '', fulfillment_type TEXT DEFAULT '',
		subtotal REAL DEFAULT 0, tax REAL DEFAULT 0, total REAL DEFAULT 0,
		currency TEXT DEFAULT 'INR',
		created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
	)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE order_items (
		id TEXT PRIMARY KEY, order_id TEXT, menu_item_id TEXT, name TEXT DEFAULT '',
		quantity INTEGER DEFAULT 1, price REAL DEFAULT 0,
		created_at DATETIME, updated_at DATETIME
	)`).Error)
	// Reverse-link tables that classify an order's source (#435). An order is a
	// meal-plan / subscription / group order when its id appears as order_id here.
	// deleted_at is included because the subscription/group models are soft-delete
	// (GORM adds `deleted_at IS NULL` to their subqueries); prod AutoMigrate has it.
	require.NoError(t, db.Exec(`CREATE TABLE meal_plan_days (
		id TEXT PRIMARY KEY, order_id TEXT, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
	)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE meal_subscription_fulfillments (
		id TEXT PRIMARY KEY, order_id TEXT, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
	)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE group_orders (
		id TEXT PRIMARY KEY, order_id TEXT, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
	)`).Error)

	prev := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = prev })
	return db
}

// seedVisChef creates a chef user + profile and returns (chefUserID, chefID).
func seedVisChef(t *testing.T, db *gorm.DB) (uuid.UUID, uuid.UUID) {
	t.Helper()
	userID, chefID := uuid.New(), uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO users (id, email, role) VALUES (?, ?, 'chef')`,
		userID.String(), userID.String()[:8]+"@chef.test").Error)
	require.NoError(t, db.Exec(`INSERT INTO chef_profiles (id, user_id, business_name) VALUES (?, ?, 'Vis Kitchen')`,
		chefID.String(), userID.String()).Error)
	return userID, chefID
}

func seedVisOrder(t *testing.T, db *gorm.DB, chefID uuid.UUID, status, payStatus string) {
	t.Helper()
	// Anchor created_at to IST noon *today* — the SAME IST basis the dashboard's
	// "today" boundary uses (services.CapacityDay = IST midnight). Bare time.Now()
	// is UTC/local in CI, and sqlite compares the naive datetime strings without
	// timezone normalization: when CI runs after IST midnight (18:30–24:00 UTC), a
	// UTC-stamped created_at sorts on the *previous* calendar day and drops out of
	// `created_at >= todayStart`, making todayOrders 0. IST noon has 12h of margin on
	// each side of the boundary, so the count is stable in every runner timezone.
	// (Production Postgres compares timestamptz as instants and is unaffected.)
	createdAt := services.CapacityDay(time.Now()).Add(12 * time.Hour)
	require.NoError(t, db.Exec(`INSERT INTO orders (id, order_number, customer_id, chef_id, status, payment_status, total, created_at)
		VALUES (?, ?, ?, ?, ?, ?, 100, ?)`,
		uuid.NewString(), "ORD-"+uuid.NewString()[:6], uuid.NewString(), chefID.String(),
		status, payStatus, createdAt).Error)
}

// seedVisOrderID is seedVisOrder but returns the new order id so a source
// reverse-link can be attached.
func seedVisOrderID(t *testing.T, db *gorm.DB, chefID uuid.UUID, status, payStatus string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	createdAt := services.CapacityDay(time.Now()).Add(12 * time.Hour)
	require.NoError(t, db.Exec(`INSERT INTO orders (id, order_number, customer_id, chef_id, status, payment_status, total, created_at)
		VALUES (?, ?, ?, ?, ?, ?, 100, ?)`,
		id.String(), "ORD-"+uuid.NewString()[:6], uuid.NewString(), chefID.String(),
		status, payStatus, createdAt).Error)
	return id
}

func linkOrderSource(t *testing.T, db *gorm.DB, table string, orderID uuid.UUID) {
	t.Helper()
	require.NoError(t, db.Exec(`INSERT INTO `+table+` (id, order_id) VALUES (?, ?)`,
		uuid.NewString(), orderID.String()).Error)
}

// orderByNumberSource pulls the `source` field for each order in a /chef/orders
// response, keyed by order id.
func sourcesByID(t *testing.T, body map[string]any) map[string]string {
	t.Helper()
	out := map[string]string{}
	orders, _ := body["orders"].([]any)
	for _, o := range orders {
		m, _ := o.(map[string]any)
		id, _ := m["id"].(string)
		src, _ := m["source"].(string)
		out[id] = src
	}
	return out
}

func chefVisRouter(userID uuid.UUID) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("userID", userID) })
	h := NewChefHandler()
	r.GET("/chef/orders", h.GetChefOrders)
	r.GET("/chef/dashboard", h.GetChefDashboard)
	return r
}

func chefVisGET(t *testing.T, r *gin.Engine, path string) map[string]any {
	t.Helper()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	return body
}

// The reported bug: 3 orders created today, none paid → the header said
// "3 Orders" while the New/History tabs were empty. Every chef surface must
// exclude unpaid rows, so the counters and the tabs always agree.
func TestChefDashboardAndQueueAgree_UnpaidOrdersInvisible(t *testing.T) {
	db := setupChefVisDB(t)
	userID, chefID := seedVisChef(t, db)
	// Three rows created today: one abandoned checkout, one paid awaiting
	// acceptance, one paid + delivered.
	seedVisOrder(t, db, chefID, "pending", "pending")     // unpaid — invisible
	seedVisOrder(t, db, chefID, "pending", "completed")   // the real new order
	seedVisOrder(t, db, chefID, "delivered", "completed") // history

	r := chefVisRouter(userID)

	// Orders tab — New: exactly the paid pending order.
	newTab := chefVisGET(t, r, "/chef/orders?status=pending")
	require.Len(t, newTab["orders"], 1, "queue shows only PAID pending orders")
	require.EqualValues(t, 1, newTab["total"])

	// Orders — unfiltered (history view source): both paid orders, never the unpaid.
	all := chefVisGET(t, r, "/chef/orders")
	require.Len(t, all["orders"], 2)
	require.EqualValues(t, 2, all["total"])

	// Dashboard counters MUST match what the tabs can show.
	dash := chefVisGET(t, r, "/chef/dashboard")
	require.EqualValues(t, 2, dash["todayOrders"], "header count must equal what the orders tab shows")
	require.EqualValues(t, 2, dash["weekOrders"])
	require.EqualValues(t, 1, dash["pendingOrders"], "badge must equal the New tab length")
	require.Len(t, dash["recentOrders"], 2)
}

// An abandoned-checkout-only day reads as a genuinely quiet day everywhere.
func TestChefDashboard_AllUnpaidMeansZeroEverywhere(t *testing.T) {
	db := setupChefVisDB(t)
	userID, chefID := seedVisChef(t, db)
	for range 3 {
		seedVisOrder(t, db, chefID, "pending", "pending")
	}

	r := chefVisRouter(userID)
	dash := chefVisGET(t, r, "/chef/dashboard")
	require.EqualValues(t, 0, dash["todayOrders"], "unpaid rows must not inflate the header")
	require.EqualValues(t, 0, dash["pendingOrders"])
	require.EqualValues(t, 0, dash["weekOrders"])

	newTab := chefVisGET(t, r, "/chef/orders?status=pending")
	require.Len(t, newTab["orders"], 0)
	require.EqualValues(t, 0, newTab["total"])
}

// Refunded orders stay visible in history (and its counters) — the chef needs
// the paper trail — while failed payments behave like pending: invisible.
func TestChefOrders_RefundedVisibleFailedInvisible(t *testing.T) {
	db := setupChefVisDB(t)
	userID, chefID := seedVisChef(t, db)
	seedVisOrder(t, db, chefID, "cancelled", "refunded")
	seedVisOrder(t, db, chefID, "pending", "failed")

	r := chefVisRouter(userID)
	all := chefVisGET(t, r, "/chef/orders")
	require.Len(t, all["orders"], 1)
	require.EqualValues(t, 1, all["total"])
}

// #435: an escrow-off meal-plan/subscription DAY order is PaymentPending but is a
// confirmed plan's day (linked via the reverse table), not an abandoned checkout
// — so the chef MUST see it, and it carries a `source` tag. A plain à-la-carte
// pending row (no link) stays invisible.
func TestChefOrders_EscrowOffPlanDayVisibleWithSource(t *testing.T) {
	db := setupChefVisDB(t)
	userID, chefID := seedVisChef(t, db)

	seedVisOrder(t, db, chefID, "pending", "pending") // abandoned à-la-carte — invisible
	alacarte := seedVisOrderID(t, db, chefID, "pending", "completed")
	planDay := seedVisOrderID(t, db, chefID, "pending", "pending") // escrow-off, unpaid
	linkOrderSource(t, db, "meal_plan_days", planDay)

	r := chefVisRouter(userID)
	all := chefVisGET(t, r, "/chef/orders")
	require.Len(t, all["orders"], 2, "the paid à-la-carte + the plan-day order are visible; the abandoned one is not")
	require.EqualValues(t, 2, all["total"])

	src := sourcesByID(t, all)
	require.Equal(t, "meal_plan", src[planDay.String()], "escrow-off plan day order tagged meal_plan")
	require.Equal(t, "alacarte", src[alacarte.String()], "plain order tagged alacarte")

	// The dashboard counters use the same scope, so the plan day order counts too.
	dash := chefVisGET(t, r, "/chef/dashboard")
	require.EqualValues(t, 2, dash["pendingOrders"], "both pending orders (à-la-carte paid + plan day) are in the New badge")
}

// #435: group + subscription orders classify to their source. Group orders are
// already paid (visible); a subscription day order may be escrow-off pending but
// visible via its link.
func TestChefOrders_GroupAndSubscriptionSources(t *testing.T) {
	db := setupChefVisDB(t)
	userID, chefID := seedVisChef(t, db)

	// Group + subscription day orders are always created PaymentCompleted (the
	// group's consolidated payment / the subscription cycle charge), so both are
	// visible via the payment filter; the reverse link only classifies the source.
	group := seedVisOrderID(t, db, chefID, "pending", "completed")
	linkOrderSource(t, db, "group_orders", group)
	sub := seedVisOrderID(t, db, chefID, "pending", "completed")
	linkOrderSource(t, db, "meal_subscription_fulfillments", sub)

	r := chefVisRouter(userID)
	all := chefVisGET(t, r, "/chef/orders")
	src := sourcesByID(t, all)
	require.Equal(t, "group", src[group.String()])
	require.Equal(t, "subscription", src[sub.String()])
}
