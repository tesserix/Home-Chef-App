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

const payoutOrdersDDL = `CREATE TABLE orders (delivery_address_line1_enc text DEFAULT '', delivery_address_line2_enc text DEFAULT '', id TEXT PRIMARY KEY, order_number TEXT DEFAULT '',
	customer_id TEXT, chef_id TEXT, status TEXT, payment_status TEXT DEFAULT 'completed', razorpay_order_id TEXT DEFAULT '', total REAL DEFAULT 0,
	subtotal REAL DEFAULT 0, tax REAL DEFAULT 0, chef_tip REAL DEFAULT 0,
	chef_funded_discount REAL DEFAULT 0, commission_rate REAL DEFAULT 0,
	payout_hold_status TEXT DEFAULT '', customer_confirmed_at DATETIME, delivered_at DATETIME,
	payout_settled_at DATETIME, payout_settle_attempts INTEGER DEFAULT 0,
	refunded_at DATETIME, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME)`

const payoutDaysDDL = `CREATE TABLE meal_plan_days (id TEXT PRIMARY KEY, meal_plan_id TEXT, order_id TEXT,
	status TEXT, payout_transfer_id TEXT DEFAULT '', price REAL DEFAULT 0, commission_rate REAL DEFAULT 0,
	payout_hold_status TEXT DEFAULT '', customer_confirmed_at DATETIME, delivered_at DATETIME,
	payout_settled_at DATETIME, payout_settle_attempts INTEGER DEFAULT 0,
	date DATETIME, created_at DATETIME, updated_at DATETIME)`

const payoutPlansDDL = `CREATE TABLE meal_plans (id TEXT PRIMARY KEY, meal_plan_number TEXT DEFAULT '',
	customer_id TEXT, chef_id TEXT, status TEXT, subtotal REAL DEFAULT 0, tax REAL DEFAULT 0)`

const payoutGroupOrdersDDL = `CREATE TABLE group_orders (delivery_address_line1_enc text DEFAULT '', delivery_address_line2_enc text DEFAULT '', id TEXT PRIMARY KEY, host_id TEXT, chef_id TEXT,
	order_id TEXT, status TEXT, payout_transfer_id TEXT DEFAULT '', payout_hold_status TEXT DEFAULT '',
	customer_confirmed_at DATETIME, delivered_at DATETIME, payout_settled_at DATETIME,
	payout_settle_attempts INTEGER DEFAULT 0, subtotal REAL DEFAULT 0, tax REAL DEFAULT 0, commission_rate REAL DEFAULT 0,
	currency TEXT DEFAULT 'INR', created_at DATETIME, updated_at DATETIME)`

const payoutOutboxDDL = `CREATE TABLE outbox_events (id TEXT DEFAULT '00000000-0000-0000-0000-000000000000',
	subject TEXT, msg_id TEXT, aggregate_type TEXT, aggregate_id TEXT, payload TEXT,
	status TEXT, attempts INTEGER DEFAULT 0, last_error TEXT, next_retry_at DATETIME,
	created_at DATETIME, updated_at DATETIME, published_at DATETIME)`

const payoutAuditDDL = `CREATE TABLE audit_logs (id TEXT DEFAULT '00000000-0000-0000-0000-000000000000',
	user_id TEXT, action TEXT, entity_type TEXT, entity_id TEXT, old_value TEXT, new_value TEXT,
	ip_address TEXT, user_agent TEXT, correlation_id TEXT, created_at DATETIME)`

// payoutChefProfilesDDL backs the blocked-chefs list + the automation switch
// (#747). Only the columns those two handlers and the shared seedChef helper
// (chef_dpdp_test.go) touch — not the full models.ChefProfile column set,
// since nothing here does a full-struct gorm Save().
const payoutChefProfilesDDL = `CREATE TABLE chef_profiles (
	id TEXT PRIMARY KEY, user_id TEXT, business_name TEXT DEFAULT '',
	description TEXT DEFAULT '', accepting_orders INTEGER DEFAULT 1,
	razorpay_settlement_status TEXT DEFAULT '', razorpay_settlement_requirements TEXT DEFAULT '',
	payout_auto_release TEXT DEFAULT '', created_at DATETIME, updated_at DATETIME)`

func setupPayoutHandlerDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: glogger.Default.LogMode(glogger.Silent)})
	require.NoError(t, err)
	for _, s := range []string{
		payoutOrdersDDL, payoutDaysDDL, payoutPlansDDL, payoutGroupOrdersDDL, payoutOutboxDDL, payoutAuditDDL,
		payoutChefProfilesDDL,
		`CREATE TABLE platform_settings (id TEXT PRIMARY KEY, key TEXT, value TEXT, type TEXT, updated_at DATETIME)`,
		// order_issues backs the pending-queue open-issue flag + release guard (#457).
		`CREATE TABLE order_issues (id TEXT PRIMARY KEY, order_id TEXT, meal_plan_day_id TEXT, status TEXT, created_at DATETIME, updated_at DATETIME)`,
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
	r.GET("/admin/payouts/blocked-chefs", h.GetBlockedChefs)
	r.PUT("/admin/chefs/:id/payout-automation", h.SetPayoutAutomation)
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

// TestBulkRelease_LabelsRealFailureNotSkipped — a genuine (non-sentinel) release
// error must land in `failed` (a reconcile candidate), NOT be mislabeled a benign
// `skipped` (#462). A missing meal-plan-day makes ReleaseHold return a wrapped
// (non-ErrHoldNotEligible) error.
func TestBulkRelease_LabelsRealFailureNotSkipped(t *testing.T) {
	db := setupPayoutHandlerDB(t)
	eligible := seedHandlerOrder(t, db, models.PayoutHoldReleaseEligible)
	alreadyDone := seedHandlerOrder(t, db, models.PayoutHoldReleased)
	missingDay := uuid.New()

	body := map[string]any{"items": []map[string]string{
		{"aggType": "order", "id": eligible.String()},
		{"aggType": "order", "id": alreadyDone.String()},
		{"aggType": "meal-plan-day", "id": missingDay.String()},
	}}
	w := doJSON(t, payoutRouter(uuid.New()), http.MethodPost, "/admin/payouts/release-bulk", body)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var resp struct {
		Released int              `json:"released"`
		Skipped  []map[string]any `json:"skipped"`
		Failed   []map[string]any `json:"failed"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, 1, resp.Released, "the eligible row advanced")
	require.Len(t, resp.Skipped, 1, "the already-released row is a benign skip")
	require.Len(t, resp.Failed, 1, "the genuine error is reported as failed, not skipped")
	require.Equal(t, missingDay.String(), resp.Failed[0]["id"])
}

// TestBulkRelease_CapsBatchSize — an oversized batch is rejected (400), not
// silently processed/truncated (#462).
func TestBulkRelease_CapsBatchSize(t *testing.T) {
	db := setupPayoutHandlerDB(t)
	_ = db
	items := make([]map[string]string, 0, 501)
	for i := 0; i < 501; i++ {
		items = append(items, map[string]string{"aggType": "order", "id": uuid.NewString()})
	}
	w := doJSON(t, payoutRouter(uuid.New()), http.MethodPost, "/admin/payouts/release-bulk", map[string]any{"items": items})
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
}

// #747 — the admin controls over payout automation.
//
// seedChef here is the existing helper from chef_dpdp_test.go
// (func seedChef(t, db, userID, business) uuid.UUID) — there is no
// `newAdminTestRouter`/`seedChef(t, db) *models.ChefProfile` pair in this
// package, so these tests are built on setupPayoutHandlerDB + payoutRouter
// (now carrying the two new routes) and the real seedChef signature.

func TestSetPayoutAutomation_RejectsAnUnknownValue(t *testing.T) {
	// Only the three legal values may be stored. Anything else is read back as
	// "follow the default", which would silently re-enable a chef an admin
	// deliberately suspended.
	db := setupPayoutHandlerDB(t)
	chefID := seedChef(t, db, uuid.New(), "Test Kitchen")

	w := doJSON(t, payoutRouter(uuid.New()), http.MethodPut,
		"/admin/chefs/"+chefID.String()+"/payout-automation", map[string]any{"value": "enabled"})
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())

	var reloaded models.ChefProfile
	require.NoError(t, db.First(&reloaded, "id = ?", chefID.String()).Error)
	require.Equal(t, "", reloaded.PayoutAutoRelease, "a rejected value must not be persisted")
}

func TestSetPayoutAutomation_StoresALegalValue(t *testing.T) {
	db := setupPayoutHandlerDB(t)
	chefID := seedChef(t, db, uuid.New(), "Test Kitchen")

	w := doJSON(t, payoutRouter(uuid.New()), http.MethodPut,
		"/admin/chefs/"+chefID.String()+"/payout-automation", map[string]any{"value": "off"})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var reloaded models.ChefProfile
	require.NoError(t, db.First(&reloaded, "id = ?", chefID.String()).Error)
	require.Equal(t, "off", reloaded.PayoutAutoRelease)
}

func TestSetPayoutAutomation_WritesAnAuditEntry(t *testing.T) {
	// Suspending a chef acts on someone else's money; "who did this" must be
	// answerable months later. The audit entry must capture old→new state and the actor.
	db := setupPayoutHandlerDB(t)
	actor := uuid.New()
	chefID := seedChef(t, db, uuid.New(), "Test Kitchen")

	w := doJSON(t, payoutRouter(actor), http.MethodPut,
		"/admin/chefs/"+chefID.String()+"/payout-automation", map[string]any{"value": "off"})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var audit models.AuditLog
	require.NoError(t, db.Where("action = ? AND entity_id = ?", "chef.payout.automation", chefID.String()).
		First(&audit).Error, "exactly one audit entry")
	require.Equal(t, actor, *audit.UserID, "acting user is recorded")
	require.Contains(t, audit.OldValue, `"payoutAutoRelease":""`, "old value is empty string")
	require.Contains(t, audit.NewValue, `"payoutAutoRelease":"off"`, "new value is off")
}

func TestSetPayoutAutomation_RejectsWhitespaceValue(t *testing.T) {
	// A value that is only whitespace ("  ") should be rejected like an invalid
	// value — trimmed, treated as empty string, and validated. If not trimmed
	// before validation, a future change adding TrimSpace could silently accept
	// it and reactivate a suspended chef.
	db := setupPayoutHandlerDB(t)
	chefID := seedChef(t, db, uuid.New(), "Test Kitchen")

	w := doJSON(t, payoutRouter(uuid.New()), http.MethodPut,
		"/admin/chefs/"+chefID.String()+"/payout-automation", map[string]any{"value": "  "})
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())

	// Reload the chef and confirm nothing was persisted
	var reloaded models.ChefProfile
	require.NoError(t, db.First(&reloaded, "id = ?", chefID.String()).Error)
	require.Equal(t, "", reloaded.PayoutAutoRelease, "whitespace value must not be persisted")

	// Confirm no audit entry was written for this failed request
	var count int64
	require.NoError(t, db.Model(&models.AuditLog{}).
		Where("action = ? AND entity_id = ?", "chef.payout.automation", chefID.String()).
		Count(&count).Error)
	require.Equal(t, int64(0), count, "no audit entry on validation failure")
}

func TestBlockedChefs_ListsNeedsClarificationWithRequirements(t *testing.T) {
	// Status alone tells an admin nothing about what to fix, so the
	// requirements have to travel with it.
	db := setupPayoutHandlerDB(t)
	chefID := seedChef(t, db, uuid.New(), "Test Kitchen")
	chef := models.ChefProfile{ID: chefID}
	require.NoError(t, db.Model(&chef).Updates(map[string]any{
		"razorpay_settlement_status":       "needs_clarification",
		"razorpay_settlement_requirements": `[{"field_reference":"settlements.ifsc_code"}]`,
	}).Error)

	w := doJSON(t, payoutRouter(uuid.New()), http.MethodGet, "/admin/payouts/blocked-chefs", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var body struct {
		Chefs []struct {
			SettlementStatus string `json:"settlementStatus"`
			Requirements     string `json:"requirements"`
		} `json:"chefs"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Len(t, body.Chefs, 1)
	require.Contains(t, body.Chefs[0].Requirements, "ifsc_code", "must say what to fix")
}

func TestBlockedChefs_ExcludesAnActivatedChef(t *testing.T) {
	db := setupPayoutHandlerDB(t)
	chefID := seedChef(t, db, uuid.New(), "Test Kitchen")
	chef := models.ChefProfile{ID: chefID}
	require.NoError(t, db.Model(&chef).Update("razorpay_settlement_status", "activated").Error)

	w := doJSON(t, payoutRouter(uuid.New()), http.MethodGet, "/admin/payouts/blocked-chefs", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var body struct {
		Chefs []json.RawMessage `json:"chefs"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Len(t, body.Chefs, 0, "a healthy chef is not a blockage")
}
