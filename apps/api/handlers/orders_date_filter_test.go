package handlers

// orders_date_filter_test.go — GET /v1/orders date windowing.
//
// The support assistant's list_recent_orders tool has always advertised
// "orders placed in the last N days" and sent `since_days`, but the handler
// read only status/page/limit and dropped it. Otto therefore narrated a 14-day
// window while listing every order the customer had ever placed — and a
// customer asking "what did I order on the 12th" could not be answered at all.
//
// These pin the three windows the assistant relies on. Self-contained in-memory
// SQLite; not parallel (shares the global DB).

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
)

func setupOrderDateDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TABLE orders (mode text DEFAULT 'live', test_session_id text, cloned_from_id text, delivery_address_line1_enc text DEFAULT '', delivery_address_line2_enc text DEFAULT '',
		id TEXT PRIMARY KEY, order_number TEXT, customer_id TEXT, chef_id TEXT,
		status TEXT DEFAULT 'delivered', payment_status TEXT DEFAULT 'completed',
		payment_method TEXT DEFAULT '', fulfillment_type TEXT DEFAULT '',
		subtotal REAL DEFAULT 0, tax REAL DEFAULT 0, total REAL DEFAULT 0,
		currency TEXT DEFAULT 'INR',
		created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
	)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE order_items (mode text DEFAULT 'live', test_session_id text, cloned_from_id text,
		id TEXT PRIMARY KEY, order_id TEXT, menu_item_id TEXT, name TEXT DEFAULT '',
		quantity INTEGER DEFAULT 1, price REAL DEFAULT 0,
		created_at DATETIME, updated_at DATETIME
	)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE chef_profiles (mode text DEFAULT 'live',
		id TEXT PRIMARY KEY, user_id TEXT, business_name TEXT DEFAULT '',
		created_at DATETIME, updated_at DATETIME
	)`).Error)
	prev := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = prev })
	return db
}

func seedDatedOrder(t *testing.T, db *gorm.DB, customerID uuid.UUID, number string, at time.Time) {
	t.Helper()
	require.NoError(t, db.Exec(
		`INSERT INTO orders (id, order_number, customer_id, chef_id, created_at) VALUES (?,?,?,?,?)`,
		uuid.New().String(), number, customerID.String(), uuid.New().String(), at,
	).Error)
}

func orderDateRouter(userID uuid.UUID) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("userID", userID) })
	r.GET("/v1/orders", NewOrderHandler().GetOrders)
	return r
}

func orderNumbersFor(t *testing.T, r *gin.Engine, path string) []string {
	t.Helper()
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var body struct {
		Data []struct {
			OrderNumber string `json:"orderNumber"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	out := make([]string, 0, len(body.Data))
	for _, o := range body.Data {
		out = append(out, o.OrderNumber)
	}
	return out
}

func TestGetOrdersSinceDaysWindow(t *testing.T) {
	db := setupOrderDateDB(t)
	customer := uuid.New()
	now := time.Now()
	seedDatedOrder(t, db, customer, "RECENT", now.AddDate(0, 0, -2))
	seedDatedOrder(t, db, customer, "EDGE", now.AddDate(0, 0, -13))
	seedDatedOrder(t, db, customer, "OLD", now.AddDate(0, 0, -90))

	r := orderDateRouter(customer)

	got := orderNumbersFor(t, r, "/v1/orders?since_days=14&limit=50")
	require.ElementsMatch(t, []string{"RECENT", "EDGE"}, got,
		"since_days must exclude the 90-day-old order — this is the window Otto narrates")

	// Without the param nothing is filtered, so the old behaviour is preserved
	// for every existing caller.
	require.Len(t, orderNumbersFor(t, r, "/v1/orders?limit=50"), 3)
}

func TestGetOrdersExplicitDateRangeIsInclusive(t *testing.T) {
	db := setupOrderDateDB(t)
	customer := uuid.New()
	// Mid-afternoon, so an end date handled as "midnight at the start of the day"
	// would wrongly drop it — which is what "what did I order on the 12th" hits.
	seedDatedOrder(t, db, customer, "ON_THE_DAY", time.Date(2026, 7, 12, 14, 30, 0, 0, time.UTC))
	seedDatedOrder(t, db, customer, "DAY_BEFORE", time.Date(2026, 7, 11, 23, 59, 0, 0, time.UTC))
	seedDatedOrder(t, db, customer, "DAY_AFTER", time.Date(2026, 7, 13, 0, 30, 0, 0, time.UTC))

	r := orderDateRouter(customer)

	got := orderNumbersFor(t, r, "/v1/orders?from=2026-07-12&to=2026-07-12&limit=50")
	require.Equal(t, []string{"ON_THE_DAY"}, got,
		"a single-day range must cover the whole day, not just its first instant")

	got = orderNumbersFor(t, r, "/v1/orders?from=2026-07-12&limit=50")
	require.ElementsMatch(t, []string{"ON_THE_DAY", "DAY_AFTER"}, got)
}

// A malformed date must not silently empty the list — it is ignored, so the
// customer still sees their orders rather than a blank screen.
func TestGetOrdersIgnoresUnparseableDates(t *testing.T) {
	db := setupOrderDateDB(t)
	customer := uuid.New()
	seedDatedOrder(t, db, customer, "ONLY", time.Now().AddDate(0, 0, -1))

	r := orderDateRouter(customer)
	require.Len(t, orderNumbersFor(t, r, "/v1/orders?from=yesterday&to=soon&since_days=abc&limit=50"), 1)
}
