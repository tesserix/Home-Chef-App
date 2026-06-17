package handlers

// onboarding_approval_test.go — backend verification for issue #62 (onboarding +
// auth E2E), the "admin approval → chef goes live" and "renew expired doc →
// re-approved → back online" slices. The full on-device signup/social-login and
// session-resilience checks are manual; the force-upgrade gate (a #62 criterion)
// is already covered by middleware/version_check_test.go. What we pin down here
// is the admin-approval state machine that flips a chef live and the renewal
// path that lifts the FSSAI lockout (feeds #32).
//
// Self-contained in-memory SQLite with distinct helper names. Not parallel.

import (
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
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
)

func setupApprovalDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	require.NoError(t, err)

	// users carries gorm.DeletedAt → needs the column.
	require.NoError(t, db.Exec(`CREATE TABLE users (
		id TEXT PRIMARY KEY, email TEXT, role TEXT DEFAULT 'customer', is_active INTEGER DEFAULT 1,
		created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
	)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE chef_profiles (
		id TEXT PRIMARY KEY, user_id TEXT, business_name TEXT DEFAULT '',
		is_verified INTEGER DEFAULT 0, verified_at DATETIME, is_active INTEGER DEFAULT 0,
		payout_country TEXT DEFAULT 'IN', fssai_override_until DATETIME,
		created_at DATETIME, updated_at DATETIME
	)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE chef_documents (
		id TEXT PRIMARY KEY, chef_id TEXT, type TEXT, status TEXT, expiry_date DATETIME,
		created_at DATETIME, updated_at DATETIME
	)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE approval_requests (
		id TEXT PRIMARY KEY, type TEXT, status TEXT DEFAULT 'pending',
		chef_id TEXT, partner_id TEXT, submitted_by_id TEXT, reviewed_by_id TEXT,
		entity_type TEXT DEFAULT '', entity_id TEXT, title TEXT DEFAULT '',
		admin_notes TEXT DEFAULT '', submitted_data TEXT DEFAULT '', reviewed_at DATETIME,
		created_at DATETIME, updated_at DATETIME
	)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE approval_request_histories (
		id TEXT, approval_id TEXT, from_status TEXT, to_status TEXT, changed_by_id TEXT,
		notes TEXT DEFAULT '', created_at DATETIME
	)`).Error)

	prev := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = prev })
	return db
}

func apprUser(t *testing.T, db *gorm.DB, role string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO users (id, email, role, is_active, created_at, updated_at) VALUES (?, ?, ?, 1, ?, ?)`,
		id.String(), id.String()+"@t.com", role, time.Now(), time.Now()).Error)
	return id
}

func apprChef(t *testing.T, db *gorm.DB, userID uuid.UUID, verified bool) uuid.UUID {
	t.Helper()
	id := uuid.New()
	v := 0
	if verified {
		v = 1
	}
	require.NoError(t, db.Exec(
		`INSERT INTO chef_profiles (id, user_id, business_name, is_verified, is_active, payout_country, created_at, updated_at)
		 VALUES (?, ?, 'Test Kitchen', ?, ?, 'IN', ?, ?)`,
		id.String(), userID.String(), v, v, time.Now(), time.Now()).Error)
	return id
}

func apprDoc(t *testing.T, db *gorm.DB, chefID uuid.UUID, status models.DocumentStatus, expiry *time.Time) {
	t.Helper()
	var exp any
	if expiry != nil {
		exp = *expiry
	}
	require.NoError(t, db.Exec(
		`INSERT INTO chef_documents (id, chef_id, type, status, expiry_date, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		uuid.New().String(), chefID.String(), string(models.DocFSSAILicense), string(status), exp, time.Now(), time.Now()).Error)
}

func apprRequest(t *testing.T, db *gorm.DB, chefID, submitter uuid.UUID, typ models.ApprovalRequestType, status models.ApprovalRequestStatus) uuid.UUID {
	t.Helper()
	id := uuid.New()
	require.NoError(t, db.Exec(
		`INSERT INTO approval_requests (id, type, status, chef_id, submitted_by_id, title, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, 'Kitchen Onboarding: Test', ?, ?)`,
		id.String(), string(typ), string(status), chefID.String(), submitter.String(), time.Now(), time.Now()).Error)
	return id
}

func callApprove(adminID, approvalID uuid.UUID) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("userID", adminID); c.Next() })
	r.PUT("/admin/approvals/:id/approve", NewApprovalHandler().ApproveRequest)
	req := httptest.NewRequest(http.MethodPut, "/admin/approvals/"+approvalID.String()+"/approve", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestApproveRequest_NotFound_404(t *testing.T) {
	setupApprovalDB(t)
	w := callApprove(uuid.New(), uuid.New())
	if w.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d (%s)", w.Code, w.Body.String())
	}
}

func TestApproveRequest_AlreadyApproved_400(t *testing.T) {
	db := setupApprovalDB(t)
	chefUser := apprUser(t, db, "customer")
	chef := apprChef(t, db, chefUser, false)
	appr := apprRequest(t, db, chef, chefUser, models.ApprovalKitchenOnboarding, models.ApprovalApproved)
	w := callApprove(apprUser(t, db, "admin"), appr)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400 (cannot approve already-approved), got %d (%s)", w.Code, w.Body.String())
	}
}

func TestApproveRequest_KitchenOnboarding_GoesLive(t *testing.T) {
	db := setupApprovalDB(t)
	chefUser := apprUser(t, db, "customer")
	chef := apprChef(t, db, chefUser, false)
	appr := apprRequest(t, db, chef, chefUser, models.ApprovalKitchenOnboarding, models.ApprovalPending)

	w := callApprove(apprUser(t, db, "admin"), appr)
	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d (%s)", w.Code, w.Body.String())
	}

	var isVerified, isActive int
	db.Raw(`SELECT is_verified FROM chef_profiles WHERE id = ?`, chef.String()).Scan(&isVerified)
	db.Raw(`SELECT is_active FROM chef_profiles WHERE id = ?`, chef.String()).Scan(&isActive)
	if isVerified != 1 || isActive != 1 {
		t.Fatalf("chef must go live: is_verified=%d is_active=%d", isVerified, isActive)
	}
	var role string
	db.Raw(`SELECT role FROM users WHERE id = ?`, chefUser.String()).Scan(&role)
	if role != string(models.RoleChef) {
		t.Fatalf("user role must flip to chef, got %q", role)
	}
	var status string
	db.Raw(`SELECT status FROM approval_requests WHERE id = ?`, appr.String()).Scan(&status)
	if status != string(models.ApprovalApproved) {
		t.Fatalf("approval status must be approved, got %q", status)
	}
	var histCount int
	db.Raw(`SELECT COUNT(*) FROM approval_request_histories WHERE approval_id = ?`, appr.String()).Scan(&histCount)
	if histCount != 1 {
		t.Fatalf("expected 1 audit-history row, got %d", histCount)
	}
}

func TestApproveRequest_DocumentVerification_VerifiesPendingDocs(t *testing.T) {
	db := setupApprovalDB(t)
	chefUser := apprUser(t, db, "chef")
	chef := apprChef(t, db, chefUser, true)
	future := time.Now().AddDate(0, 0, 365)
	apprDoc(t, db, chef, models.DocStatusPending, &future)
	appr := apprRequest(t, db, chef, chefUser, models.ApprovalDocumentVerification, models.ApprovalPending)

	w := callApprove(apprUser(t, db, "admin"), appr)
	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d (%s)", w.Code, w.Body.String())
	}
	var pendingLeft int
	db.Raw(`SELECT COUNT(*) FROM chef_documents WHERE chef_id = ? AND status = ?`, chef.String(), string(models.DocStatusPending)).Scan(&pendingLeft)
	if pendingLeft != 0 {
		t.Fatalf("pending docs must be verified on approval, %d still pending", pendingLeft)
	}
}

// TestApproveRequest_RenewalLiftsFSSAILock ties #62 (renew expired doc → re-approved)
// to the #32 lockout: an India chef with a lapsed verified FSSAI is locked; a
// pending renewal with a future expiry, once approved, must lift the lock.
func TestApproveRequest_RenewalLiftsFSSAILock(t *testing.T) {
	db := setupApprovalDB(t)
	chefUser := apprUser(t, db, "chef")
	chef := apprChef(t, db, chefUser, true)
	past := time.Now().AddDate(0, 0, -10)
	future := time.Now().AddDate(0, 0, 365)
	apprDoc(t, db, chef, models.DocStatusVerified, &past)  // lapsed licence → locked
	apprDoc(t, db, chef, models.DocStatusPending, &future) // the renewal awaiting approval
	appr := apprRequest(t, db, chef, chefUser, models.ApprovalDocumentVerification, models.ApprovalPending)

	// Sanity: chef is locked before approval (latest VERIFIED expiry is in the past).
	var beforeChef models.ChefProfile
	require.NoError(t, db.First(&beforeChef, "id = ?", chef).Error)
	if !services.IsChefFSSAIExpired(&beforeChef) {
		t.Fatal("precondition: chef should be FSSAI-locked before renewal is approved")
	}

	w := callApprove(apprUser(t, db, "admin"), appr)
	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d (%s)", w.Code, w.Body.String())
	}

	var afterChef models.ChefProfile
	require.NoError(t, db.First(&afterChef, "id = ?", chef).Error)
	if services.IsChefFSSAIExpired(&afterChef) {
		t.Fatal("approving the renewal must lift the FSSAI lockout")
	}
}
