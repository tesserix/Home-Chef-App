package handlers

// approval_remind_test.go — #697. The chef-facing bump endpoint.
//
// models/approval_reminder_test.go pins the CADENCE; these pin the ENDPOINT:
// the cooldown is actually enforced, escalation is stamped once, a decided
// request refuses, and a chef can only bump their own requests.

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
	glogger "gorm.io/gorm/logger"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

func setupRemindDB(t *testing.T) (*gorm.DB, uuid.UUID, uuid.UUID) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: glogger.Default.LogMode(glogger.Silent)})
	require.NoError(t, err)
	for _, s := range []string{
		`CREATE TABLE chef_profiles (address_line1_enc text DEFAULT '', address_line2_enc text DEFAULT '', id text PRIMARY KEY, user_id text, business_name text, is_active integer DEFAULT 1)`,
		`CREATE TABLE approval_requests (
			id text PRIMARY KEY, type text, status text, priority text DEFAULT 'normal',
			chef_id text, partner_id text, submitted_by_id text, reviewed_by_id text,
			entity_type text, entity_id text, title text, description text,
			submitted_data text, admin_notes text, reviewed_at datetime, expires_at datetime,
			reminder_count integer NOT NULL DEFAULT 0, last_reminded_at datetime, escalated_at datetime,
			created_at datetime, updated_at datetime)`,
		`CREATE TABLE users (email_enc text DEFAULT '', email_bidx text DEFAULT '', first_name_enc text DEFAULT '', last_name_enc text DEFAULT '', phone_enc text DEFAULT '', phone_bidx text DEFAULT '', id text PRIMARY KEY, email text, first_name text, last_name text, role text)`,
		`CREATE TABLE notifications (id text PRIMARY KEY, user_id text, type text, title text,
			message text, is_read integer DEFAULT 0, created_at datetime, updated_at datetime)`,
		outboxDDL, auditDDL,
	} {
		require.NoError(t, db.Exec(s).Error)
	}

	userID, chefID := uuid.New(), uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO chef_profiles (id, user_id, business_name, is_active) VALUES (?,?,?,1)`,
		chefID.String(), userID.String(), "Dum Alooo Kitchen").Error)

	orig := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = orig })
	return db, userID, chefID
}

// seedApproval inserts a request created `age` ago with `count` prior bumps.
// lastRemindedAgo < 0 means never reminded.
func seedApproval(t *testing.T, db *gorm.DB, chefID uuid.UUID, status string, age time.Duration, count int, lastRemindedAgo time.Duration) uuid.UUID {
	t.Helper()
	id := uuid.New()
	created := time.Now().Add(-age)
	var last any
	if lastRemindedAgo >= 0 {
		last = time.Now().Add(-lastRemindedAgo)
	}
	require.NoError(t, db.Exec(`INSERT INTO approval_requests
		(id, type, status, priority, chef_id, submitted_by_id, entity_type, entity_id, title,
		 reminder_count, last_reminded_at, created_at, updated_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		id.String(), "menu_item_new", status, "normal", chefID.String(), uuid.NewString(),
		"menu_item", uuid.NewString(), "New Menu Item: Dal",
		count, last, created, created).Error)
	return id
}

func remind(t *testing.T, userID, approvalID uuid.UUID) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("userID", userID); c.Next() })
	r.POST("/chef/admin-requests/:id/remind", (&ApprovalHandler{}).RemindApprovalRequest)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/chef/admin-requests/"+approvalID.String()+"/remind", nil))
	return w
}

func loadApproval(t *testing.T, db *gorm.DB, id uuid.UUID) models.ApprovalRequest {
	t.Helper()
	var a models.ApprovalRequest
	require.NoError(t, db.Where("id = ?", id).First(&a).Error)
	return a
}

func TestRemind_RefusedInsideTheCooldown(t *testing.T) {
	db, userID, chefID := setupRemindDB(t)
	id := seedApproval(t, db, chefID, "pending", 2*time.Hour, 0, -1)

	w := remind(t, userID, id)
	require.Equal(t, http.StatusTooManyRequests, w.Code,
		"2h old — the chef must wait a day before the first bump")

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.NotEmpty(t, body["nextRemindAt"], "tell the client when it unlocks so it can resync its countdown")
	require.Equal(t, 0, loadApproval(t, db, id).ReminderCount, "a refused bump must not count")
}

func TestRemind_AllowedAfterTwentyFourHours(t *testing.T) {
	db, userID, chefID := setupRemindDB(t)
	id := seedApproval(t, db, chefID, "pending", 25*time.Hour, 0, -1)

	require.Equal(t, http.StatusOK, remind(t, userID, id).Code)

	a := loadApproval(t, db, id)
	require.Equal(t, 1, a.ReminderCount)
	require.NotNil(t, a.LastRemindedAt)
	require.Nil(t, a.EscalatedAt, "one bump is a nudge, not an escalation")
	require.False(t, a.IsEscalated())
}

// The bump that crosses the threshold escalates.
func TestRemind_ThirdBumpEscalates(t *testing.T) {
	db, userID, chefID := setupRemindDB(t)
	// 2 prior bumps, last one 25h ago → the 3rd is due.
	id := seedApproval(t, db, chefID, "pending", 72*time.Hour, 2, 25*time.Hour)

	w := remind(t, userID, id)
	require.Equal(t, http.StatusOK, w.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, true, body["escalated"])

	a := loadApproval(t, db, id)
	require.Equal(t, 3, a.ReminderCount)
	require.NotNil(t, a.EscalatedAt, "3 bumps means it belongs in the admin escalation panel")
}

// escalated_at is the audit fact "escalated at T" and the admin panel triages on
// it. Re-stamping on every later bump would keep moving T and make the oldest
// escalation look new.
func TestRemind_EscalatedAtIsStampedOnce(t *testing.T) {
	db, userID, chefID := setupRemindDB(t)
	id := seedApproval(t, db, chefID, "pending", 96*time.Hour, 2, 25*time.Hour)

	require.Equal(t, http.StatusOK, remind(t, userID, id).Code)
	first := loadApproval(t, db, id).EscalatedAt
	require.NotNil(t, first)

	// A 4th bump, 7h later — past the tightened 6h cadence.
	seven := time.Now().Add(-7 * time.Hour)
	require.NoError(t, db.Exec(`UPDATE approval_requests SET last_reminded_at = ? WHERE id = ?`,
		seven, id.String()).Error)

	require.Equal(t, http.StatusOK, remind(t, userID, id).Code)
	a := loadApproval(t, db, id)
	require.Equal(t, 4, a.ReminderCount)
	require.WithinDuration(t, *first, *a.EscalatedAt, time.Second,
		"escalated_at must not move on later bumps — it is when it BECAME escalated")
}

// Once escalated the cadence tightens to 6h.
func TestRemind_EscalatedCadenceIsSixHours(t *testing.T) {
	db, userID, chefID := setupRemindDB(t)
	id := seedApproval(t, db, chefID, "pending", 96*time.Hour, 3, 5*time.Hour)

	require.Equal(t, http.StatusTooManyRequests, remind(t, userID, id).Code,
		"5h since the last bump — still inside the 6h escalated cooldown")

	require.NoError(t, db.Exec(`UPDATE approval_requests SET last_reminded_at = ? WHERE id = ?`,
		time.Now().Add(-7*time.Hour), id.String()).Error)
	require.Equal(t, http.StatusOK, remind(t, userID, id).Code,
		"7h > 6h — allowed, without waiting another full day")
}

func TestRemind_DecidedRequestIsRefused(t *testing.T) {
	db, userID, chefID := setupRemindDB(t)
	for _, s := range []string{"approved", "rejected", "cancelled"} {
		id := seedApproval(t, db, chefID, s, 30*24*time.Hour, 0, -1)
		require.Equal(t, http.StatusConflict, remind(t, userID, id).Code,
			"%s: bumping a decided request blocks nobody", s)
		require.Equal(t, 0, loadApproval(t, db, id).ReminderCount)
	}
}

// A chef must not be able to bump someone else's request.
func TestRemind_OtherChefsRequestIsNotFound(t *testing.T) {
	db, userID, _ := setupRemindDB(t)
	otherChef := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO chef_profiles (id, user_id, business_name, is_active) VALUES (?,?,?,1)`,
		otherChef.String(), uuid.NewString(), "Someone Else").Error)
	id := seedApproval(t, db, otherChef, "pending", 48*time.Hour, 0, -1)

	require.Equal(t, http.StatusNotFound, remind(t, userID, id).Code)
	require.Equal(t, 0, loadApproval(t, db, id).ReminderCount)
}

// A double-tap must not burn two of the three bumps that lead to escalation, or
// fire two admin pings. The second is inside the cooldown the first just started.
func TestRemind_DoubleTapCountsOnce(t *testing.T) {
	db, userID, chefID := setupRemindDB(t)
	id := seedApproval(t, db, chefID, "pending", 25*time.Hour, 0, -1)

	require.Equal(t, http.StatusOK, remind(t, userID, id).Code)
	require.Equal(t, http.StatusTooManyRequests, remind(t, userID, id).Code,
		"the first bump started the cooldown — an impatient second tap must bounce off it")
	require.Equal(t, 1, loadApproval(t, db, id).ReminderCount)
}

// The bump's whole purpose is that someone finds out, so the admin event is
// staged in the SAME transaction — it cannot be lost if the process dies.
func TestRemind_StagesTheAdminEventTransactionally(t *testing.T) {
	db, userID, chefID := setupRemindDB(t)
	id := seedApproval(t, db, chefID, "pending", 25*time.Hour, 0, -1)

	require.Equal(t, http.StatusOK, remind(t, userID, id).Code)

	var n int64
	require.NoError(t, db.Raw(
		`SELECT count(*) FROM outbox_events WHERE subject = ?`, "approvals.reminded",
	).Scan(&n).Error)
	require.Equal(t, int64(1), n, "the reminder event must be staged in the outbox, not published best-effort")
}
