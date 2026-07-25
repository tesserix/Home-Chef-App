package handlers

// approval_bulk_test.go — bulk approval + the menu-visibility side effect it drives. Reuses the
// sqlite harness from approval_remind_test.go (setupRemindDB / loadApproval / outboxDDL).

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

// seedMenuApproval inserts a menu_item_new approval whose entity_id points at a specific menu item.
func seedMenuApproval(t *testing.T, db *gorm.DB, chefID, entityID uuid.UUID, status string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO approval_requests
		(id, type, status, priority, chef_id, submitted_by_id, entity_type, entity_id, title, created_at, updated_at)
		VALUES (?,?,?,?,?,?,?,?,?,datetime('now'),datetime('now'))`,
		id.String(), "menu_item_new", status, "normal", chefID.String(), uuid.NewString(),
		"menu_item", entityID.String(), "New Menu Item: Dish").Error)
	return id
}

func menuApproved(t *testing.T, db *gorm.DB, id uuid.UUID) bool {
	t.Helper()
	var approved bool
	require.NoError(t, db.Raw(`SELECT is_approved FROM menu_items WHERE id = ?`, id.String()).Scan(&approved).Error)
	return approved
}

func bulkApprove(t *testing.T, adminID uuid.UUID, ids []string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("userID", adminID); c.Next() })
	r.POST("/admin/approvals/bulk-approve", (&ApprovalHandler{}).BulkApproveRequests)
	body, _ := json.Marshal(map[string]any{"ids": ids})
	req := httptest.NewRequest(http.MethodPost, "/admin/approvals/bulk-approve", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// Bulk-approving menu items flips each to is_approved=true (making it visible to customers), and a
// row that's already decided fails independently without blocking the others.
func TestBulkApprove_FlipsMenuVisibility(t *testing.T) {
	db, _, chefID := setupRemindDB(t)
	// deleted_at is required: models.MenuItem is soft-deleted, so GORM's Update adds
	// "AND deleted_at IS NULL" — without the column the side-effect update errors silently.
	require.NoError(t, db.Exec(`CREATE TABLE menu_items (mode text DEFAULT 'live', test_session_id text, cloned_from_id text, id text PRIMARY KEY, chef_id text, name text,
		is_available integer DEFAULT 1, is_approved integer DEFAULT 0, created_at datetime, updated_at datetime, deleted_at datetime)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE approval_request_histories (id text PRIMARY KEY, approval_id text,
		from_status text, to_status text, changed_by_id text, notes text, created_at datetime, updated_at datetime)`).Error)

	item1, item2 := uuid.New(), uuid.New()
	for _, it := range []uuid.UUID{item1, item2} {
		require.NoError(t, db.Exec(`INSERT INTO menu_items (id, chef_id, name, is_available, is_approved) VALUES (?,?,?,1,0)`,
			it.String(), chefID.String(), "Dish").Error)
	}
	ap1 := seedMenuApproval(t, db, chefID, item1, "pending")
	ap2 := seedMenuApproval(t, db, chefID, item2, "pending")
	apDone := seedMenuApproval(t, db, chefID, uuid.New(), "approved") // already decided → fails

	w := bulkApprove(t, uuid.New(), []string{ap1.String(), ap2.String(), apDone.String()})
	require.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Approved int `json:"approved"`
		Failed   int `json:"failed"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, 2, resp.Approved, "two pending items approved")
	require.Equal(t, 1, resp.Failed, "the already-approved row fails independently")

	// The two pending items are now visible to customers; their approvals are approved.
	require.True(t, menuApproved(t, db, item1))
	require.True(t, menuApproved(t, db, item2))
	require.Equal(t, models.ApprovalApproved, loadApproval(t, db, ap1).Status)
	require.Equal(t, models.ApprovalApproved, loadApproval(t, db, ap2).Status)
}

// An empty id list is rejected (nothing to do).
func TestBulkApprove_EmptyRejected(t *testing.T) {
	setupRemindDB(t)
	w := bulkApprove(t, uuid.New(), []string{})
	require.Equal(t, http.StatusBadRequest, w.Code)
}
