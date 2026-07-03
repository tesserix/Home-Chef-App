package handlers

// admin_payout_test.go — the admin payout release queue endpoints (#388). Mirrors
// chef_delivered_gate_test.go: in-memory sqlite, hand-DDL'd tables (the models'
// gen_random_uuid() default can't create on sqlite), swap database.DB, restore in
// t.Cleanup. Handler methods are bound directly on a bare gin router — bffAuth +
// RequireAdmin are asserted by the route-wiring check, not re-exercised per test.
//
// Flags default OFF, so every release/withhold/reverse here is a DB-only state
// advance with no money moved (GetRazorpay() is nil without Secret Manager).

import (
	"bytes"
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
	glogger "gorm.io/gorm/logger"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

const payoutOrdersDDL = `CREATE TABLE orders (id TEXT PRIMARY KEY, order_number TEXT DEFAULT '',
	customer_id TEXT, chef_id TEXT, status TEXT, razorpay_order_id TEXT DEFAULT '', total REAL DEFAULT 0,
	payout_hold_status TEXT DEFAULT '', customer_confirmed_at DATETIME, delivered_at DATETIME,
	payout_settled_at DATETIME, payout_settle_attempts INTEGER DEFAULT 0,
	refunded_at DATETIME, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME)`

const payoutDaysDDL = `CREATE TABLE meal_plan_days (id TEXT PRIMARY KEY, meal_plan_id TEXT, order_id TEXT,
	status TEXT, payout_transfer_id TEXT DEFAULT '', price REAL DEFAULT 0,
	payout_hold_status TEXT DEFAULT '', customer_confirmed_at DATETIME, delivered_at DATETIME,
	payout_settled_at DATETIME, payout_settle_attempts INTEGER DEFAULT 0,
	date DATETIME, created_at DATETIME, updated_at DATETIME)`

const payoutPlansDDL = `CREATE TABLE meal_plans (id TEXT PRIMARY KEY, meal_plan_number TEXT DEFAULT '',
	customer_id TEXT, chef_id TEXT, status TEXT)`

const payoutGroupOrdersDDL = `CREATE TABLE group_orders (id TEXT PRIMARY KEY, host_id TEXT, chef_id TEXT,
	order_id TEXT, status TEXT, payout_transfer_id TEXT DEFAULT '', payout_hold_status TEXT DEFAULT '',
	customer_confirmed_at DATETIME, delivered_at DATETIME, payout_settled_at DATETIME,
	payout_settle_attempts INTEGER DEFAULT 0, subtotal REAL DEFAULT 0, tax REAL DEFAULT 0,
	currency TEXT DEFAULT 'INR', created_at DATETIME, updated_at DATETIME)`

const payoutOutboxDDL = `CREATE TABLE outbox_events (id TEXT DEFAULT '00000000-0000-0000-0000-000000000000',
	subject TEXT, msg_id TEXT, aggregate_type TEXT, aggregate_id TEXT, payload TEXT,
	status TEXT, attempts INTEGER DEFAULT 0, last_error TEXT, next_retry_at DATETIME,
	created_at DATETIME, updated_at DATETIME, published_at DATETIME)`

const payoutAuditDDL = `CREATE TABLE audit_logs (id TEXT DEFAULT '00000000-0000-0000-0000-000000000000',
	user_id TEXT, action TEXT, entity_type TEXT, entity_id TEXT, old_value TEXT, new_value TEXT,
	ip_address TEXT, user_agent TEXT, correlation_id TEXT, created_at DATETIME)`

func setupPayoutHandlerDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: glogger.Default.LogMode(glogger.Silent)})
	require.NoError(t, err)
	for _, s := range []string{
		payoutOrdersDDL, payoutDaysDDL, payoutPlansDDL, payoutGroupOrdersDDL, payoutOutboxDDL, payoutAuditDDL,
		`CREATE TABLE platform_settings (id TEXT PRIMARY KEY, key TEXT, value TEXT, type TEXT, updated_at DATETIME)`,
		// order_issues backs the pending-queue open-issue flag + release guard (#457).
		`CREATE TABLE order_issues (id TEXT PRIMARY KEY, order_id TEXT, status TEXT, created_at DATETIME, updated_at DATETIME)`,
	} {
		require.NoError(t, db.Exec(s).Error)
	}
	orig := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = orig })
	return db
}

func seedHandlerOrder(t *testing.T, db *gorm.DB, hold models.PayoutHoldStatus) uuid.UUID {
	t.Helper()
	id, chef := uuid.New(), uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO orders
		(id, order_number, customer_id, chef_id, status, razorpay_order_id, total, payout_hold_status, delivered_at)
		VALUES (?,?,?,?,?,?,?,?,?)`,
		id.String(), "ORD-"+id.String()[:8], uuid.NewString(), chef.String(),
		"delivered", "order_rzp_123", 250.0, string(hold), time.Now().Add(-30*time.Hour)).Error)
	return id
}

// payoutRouter binds the admin payout handler methods on a bare router (no
// bffAuth/RequireAdmin — those live on the group and are asserted separately). A
// fake userID is set so LogAudit records an actor.
func payoutRouter(userID uuid.UUID) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("userID", userID); c.Next() })
	h := NewAdminPayoutHandler()
	r.GET("/admin/payouts/pending", h.GetPendingPayouts)
	r.POST("/admin/payouts/:aggType/:id/release", h.ReleasePayout)
	r.POST("/admin/payouts/:aggType/:id/withhold", h.WithholdPayout)
	r.POST("/admin/payouts/:aggType/:id/reverse", h.ReversePayout)
	r.POST("/admin/payouts/release-bulk", h.BulkReleasePayouts)
	return r
}

func doJSON(t *testing.T, r *gin.Engine, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
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

func TestGetPendingPayouts_200(t *testing.T) {
	db := setupPayoutHandlerDB(t)
	seedHandlerOrder(t, db, models.PayoutHoldReleaseEligible)
	seedHandlerOrder(t, db, models.PayoutHoldAwaitingConfirmation)

	w := doJSON(t, payoutRouter(uuid.New()), http.MethodGet, "/admin/payouts/pending", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var body struct {
		Payouts []map[string]any `json:"payouts"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Len(t, body.Payouts, 1, "only the release_eligible row")
}

func TestReleasePayout_Conflict(t *testing.T) {
	db := setupPayoutHandlerDB(t)
	id := seedHandlerOrder(t, db, models.PayoutHoldReleased)

	w := doJSON(t, payoutRouter(uuid.New()), http.MethodPost, "/admin/payouts/order/"+id.String()+"/release", nil)
	require.Equal(t, http.StatusConflict, w.Code, w.Body.String())

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.NotEmpty(t, body["error"])
}

func TestWithholdPayout_MissingReason_400(t *testing.T) {
	db := setupPayoutHandlerDB(t)
	id := seedHandlerOrder(t, db, models.PayoutHoldReleaseEligible)

	w := doJSON(t, payoutRouter(uuid.New()), http.MethodPost,
		"/admin/payouts/order/"+id.String()+"/withhold", map[string]any{"reason": "  "})
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
}

func TestReversePayout_MissingReason_400(t *testing.T) {
	db := setupPayoutHandlerDB(t)
	id := seedHandlerOrder(t, db, models.PayoutHoldReleased)

	w := doJSON(t, payoutRouter(uuid.New()), http.MethodPost,
		"/admin/payouts/order/"+id.String()+"/reverse", map[string]any{"reason": ""})
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
}

func TestReleasePayout_WritesAudit(t *testing.T) {
	db := setupPayoutHandlerDB(t)
	actor := uuid.New()
	id := seedHandlerOrder(t, db, models.PayoutHoldReleaseEligible)

	w := doJSON(t, payoutRouter(actor), http.MethodPost, "/admin/payouts/order/"+id.String()+"/release", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var n int
	require.NoError(t, db.Raw(
		`SELECT count(*) FROM audit_logs WHERE entity_id = ? AND action = 'payout.released'`,
		id.String()).Scan(&n).Error)
	require.Equal(t, 1, n, "exactly one payout.released audit row")
}

func TestBulkRelease_SkipsIneligible(t *testing.T) {
	db := setupPayoutHandlerDB(t)
	eligible := seedHandlerOrder(t, db, models.PayoutHoldReleaseEligible)
	alreadyDone := seedHandlerOrder(t, db, models.PayoutHoldReleased)

	body := map[string]any{"items": []map[string]string{
		{"aggType": "order", "id": eligible.String()},
		{"aggType": "order", "id": alreadyDone.String()},
	}}
	w := doJSON(t, payoutRouter(uuid.New()), http.MethodPost, "/admin/payouts/release-bulk", body)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var resp struct {
		Released int              `json:"released"`
		Skipped  []map[string]any `json:"skipped"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, 1, resp.Released, "only the eligible row advanced")
	require.Len(t, resp.Skipped, 1, "the already-released row skipped")

	var status string
	require.NoError(t, db.Raw(`SELECT payout_hold_status FROM orders WHERE id = ?`, eligible.String()).Scan(&status).Error)
	require.Equal(t, string(models.PayoutHoldReleased), status)
}
