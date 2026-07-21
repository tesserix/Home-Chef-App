package handlers

// chef_dashboard_active_test.go — #695. The dashboard's `activeOrders` is the
// chef's kitchen queue and must never hide live work.
//
// The bug: the app derived in-flight orders by filtering `recentOrders`, which is
// the 10 newest orders ACROSS ALL STATUSES. Accept an order, take ten more, and
// the first fell out of that window — and since there was no Active tab and
// History excludes accepted/preparing/ready, it left the app entirely. Food on
// the stove, no ticket. These tests pin that `activeOrders` is scoped by STATUS,
// not by recency, so no volume of newer orders can push live work out of it.

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// setupDashboardDB is setupChefOrderDB plus the tables the DASHBOARD reads that
// the order-status harness doesn't need. chefVisibleOrders subqueries
// meal_plan_days (#435: a confirmed plan's day reaches the chef even while
// payment_status is pending), so without that table every dashboard query errors
// and silently returns an empty list — which looks exactly like "no orders".
// Added here rather than in the shared helper to avoid changing what other suites
// exercise.
func setupDashboardDB(t *testing.T) (*gorm.DB, uuid.UUID, uuid.UUID) {
	t.Helper()
	db, userID, chefID := setupChefOrderDB(t)
	require.NoError(t, db.Exec(
		`CREATE TABLE meal_plan_days (id text PRIMARY KEY, order_id text, status text, deleted_at datetime)`,
	).Error)
	// The active query subquery-excludes orders under an open delivery-failure
	// review (#393). Without this table that subquery errors and the whole
	// dashboard query silently returns an empty queue — hiding live work.
	require.NoError(t, db.Exec(
		`CREATE TABLE order_issues (id text PRIMARY KEY, order_id text, reason text, status text)`,
	).Error)
	return db, userID, chefID
}

// seedOpenDeliveryFailure marks orderID as under an open delivery-failure review
// (#393): a pending `delivery_failed` issue. The chef has closed it off; an admin
// now decides the money.
func seedOpenDeliveryFailure(t *testing.T, db *gorm.DB, orderID uuid.UUID) {
	t.Helper()
	require.NoError(t, db.Exec(
		`INSERT INTO order_issues (id, order_id, reason, status) VALUES (?,?,?,?)`,
		uuid.New().String(), orderID.String(), "delivery_failed", "pending").Error)
}

// seedOrderAt inserts an order with an explicit status/fulfilment/created_at so a
// test can build a precise recency ordering.
func seedOrderAt(t *testing.T, db *gorm.DB, chefID uuid.UUID, status, ft string, createdAt time.Time) uuid.UUID {
	t.Helper()
	orderID, custID := uuid.New(), uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO orders
		(id, order_number, customer_id, chef_id, status, payment_status, fulfillment_type,
		 subtotal, total, currency, created_at, updated_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`,
		orderID.String(), "ORD-"+orderID.String()[:8], custID.String(), chefID.String(),
		status, "completed", ft, 100.0, 100.0, "INR", createdAt, createdAt).Error)
	return orderID
}

func getDashboard(t *testing.T, userID uuid.UUID) map[string]any {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("userID", userID); c.Next() })
	r.GET("/chef/dashboard", (&ChefHandler{}).GetChefDashboard)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/chef/dashboard", nil))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	return body
}

func activeIDs(t *testing.T, body map[string]any) []string {
	t.Helper()
	rows, _ := body["activeOrders"].([]any)
	ids := make([]string, 0, len(rows))
	for _, r := range rows {
		m, ok := r.(map[string]any)
		require.True(t, ok)
		ids = append(ids, m["id"].(string))
	}
	return ids
}

// THE bug, reproduced: an order the chef is cooking must survive any number of
// newer orders. Under the old Limit(10)-and-filter approach this order was
// invisible everywhere in the app.
func TestDashboard_ActiveOrder_SurvivesTenNewerOrders(t *testing.T) {
	db, userID, chefID := setupDashboardDB(t)

	base := time.Now().Add(-3 * time.Hour)
	cooking := seedOrderAt(t, db, chefID, "preparing", "delivery", base)

	// Ten newer orders — exactly the rush that used to evict it.
	for i := 1; i <= 10; i++ {
		seedOrderAt(t, db, chefID, "pending", "delivery", base.Add(time.Duration(i)*time.Minute))
	}

	ids := activeIDs(t, getDashboard(t, userID))
	require.Contains(t, ids, cooking.String(),
		"an order being cooked fell out of the kitchen queue because 10 newer orders arrived — "+
			"this is the bug: the chef has food on the stove and no ticket for it anywhere in the app")
	require.Len(t, ids, 1, "pending orders are not kitchen work — they belong to the accept queue")
}

// The queue is a WORK LIST: the order closest to breaching must be at the top.
// recentOrders is DESC because it answers a different question.
func TestDashboard_ActiveOrders_AreOldestFirst(t *testing.T) {
	db, userID, chefID := setupDashboardDB(t)

	base := time.Now().Add(-2 * time.Hour)
	oldest := seedOrderAt(t, db, chefID, "preparing", "delivery", base)
	middle := seedOrderAt(t, db, chefID, "accepted", "delivery", base.Add(30*time.Minute))
	newest := seedOrderAt(t, db, chefID, "ready", "delivery", base.Add(60*time.Minute))

	require.Equal(t,
		[]string{oldest.String(), middle.String(), newest.String()},
		activeIDs(t, getDashboard(t, userID)),
		"newest-first is backwards for a kitchen — the most urgent order would sit at the bottom")
}

// picked_up means opposite things depending on who is carrying the food. For
// chef_delivery the CHEF is out delivering — still their job. Before #695 this
// order was in neither the in-flight set nor History: invisible.
func TestDashboard_ChefDeliveryPickedUp_StaysInTheQueue(t *testing.T) {
	db, userID, chefID := setupDashboardDB(t)

	base := time.Now().Add(-time.Hour)
	chefDriving := seedOrderAt(t, db, chefID, "picked_up", "chef_delivery", base)
	riderHasIt := seedOrderAt(t, db, chefID, "picked_up", "delivery", base)

	ids := activeIDs(t, getDashboard(t, userID))
	require.Contains(t, ids, chefDriving.String(),
		"the chef is out delivering this — it is still their job and must stay on the queue")
	require.NotContains(t, ids, riderHasIt.String(),
		"a 3PL rider has this one; it is out of the chef's hands and belongs in History")
}

// A chef_delivery order the chef reported "couldn't deliver" (#393) stays
// picked_up while an admin rules on the money — but it is out of the chef's
// hands, so it must leave the kitchen queue (and route to History) rather than
// linger as live work the chef can no longer act on.
func TestDashboard_OpenDeliveryFailure_DropsFromQueue(t *testing.T) {
	db, userID, chefID := setupDashboardDB(t)

	base := time.Now().Add(-time.Hour)
	stillDriving := seedOrderAt(t, db, chefID, "picked_up", "chef_delivery", base)
	reportedFailed := seedOrderAt(t, db, chefID, "picked_up", "chef_delivery", base.Add(time.Minute))
	seedOpenDeliveryFailure(t, db, reportedFailed)

	ids := activeIDs(t, getDashboard(t, userID))
	require.Contains(t, ids, stillDriving.String(),
		"a chef still out delivering must stay on the queue")
	require.NotContains(t, ids, reportedFailed.String(),
		"an order under an open delivery-failure review is an admin's call now — it must leave the chef's queue")
}

// Terminal orders must not clutter the queue.
func TestDashboard_TerminalOrders_AreNotQueued(t *testing.T) {
	db, userID, chefID := setupDashboardDB(t)

	base := time.Now().Add(-time.Hour)
	for _, s := range []string{"delivered", "cancelled", "rejected", "pending"} {
		seedOrderAt(t, db, chefID, s, "delivery", base)
	}
	live := seedOrderAt(t, db, chefID, "accepted", "delivery", base)

	require.Equal(t, []string{live.String()}, activeIDs(t, getDashboard(t, userID)),
		"only live kitchen work belongs in the queue")
}
