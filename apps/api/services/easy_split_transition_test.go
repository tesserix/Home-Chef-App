package services

// easy_split_transition_test.go — #1083. Cashfree verifies a bank account
// asynchronously, and the answer arrives twice: once on a webhook, once on the
// next reconcile tick. Both must land on the same state, and the chef must be
// told once.

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	glogger "gorm.io/gorm/logger"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

// The chef row as the sweep sees it: hand DDL, because the model carries
// postgres-only column defaults that sqlite cannot parse.
const transitionChefDDL = `CREATE TABLE chef_profiles (
	id TEXT PRIMARY KEY, user_id TEXT, business_name TEXT DEFAULT '', mode TEXT DEFAULT 'live',
	payout_method TEXT DEFAULT '', pan_number TEXT DEFAULT '',
	cashfree_vendor_id TEXT DEFAULT '', cashfree_vendor_status TEXT DEFAULT '',
	cashfree_test_vendor_id TEXT DEFAULT '', cashfree_test_vendor_status TEXT DEFAULT '',
	created_at DATETIME, updated_at DATETIME, deleted_at DATETIME)`

const transitionAuditDDL = `CREATE TABLE audit_logs (
	id TEXT PRIMARY KEY, user_id TEXT, action TEXT, entity_type TEXT, entity_id TEXT,
	old_value TEXT DEFAULT '', new_value TEXT DEFAULT '', ip_address TEXT DEFAULT '',
	user_agent TEXT DEFAULT '', correlation_id TEXT DEFAULT '', created_at DATETIME)`

func setupTransitionDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=private"), &gorm.Config{
		Logger: glogger.Default.LogMode(glogger.Silent),
	})
	require.NoError(t, err)
	require.NoError(t, db.Exec(transitionChefDDL).Error)
	require.NoError(t, db.Exec(transitionAuditDDL).Error)

	orig := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = orig })
	return db
}

func seedTransitionChef(t *testing.T, db *gorm.DB, status string) *models.ChefProfile {
	t.Helper()
	chef := &models.ChefProfile{
		ID: uuid.New(), UserID: uuid.New(), PayoutMethod: "bank_transfer",
		CashfreeVendorID: "hc_1", CashfreeVendorStatus: status,
	}
	require.NoError(t, db.Exec(
		`INSERT INTO chef_profiles (id, user_id, payout_method, cashfree_vendor_id, cashfree_vendor_status, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		chef.ID.String(), chef.UserID.String(), chef.PayoutMethod,
		chef.CashfreeVendorID, status, time.Now().Add(-time.Hour)).Error)
	return chef
}

func storedStatus(t *testing.T, db *gorm.DB, id uuid.UUID) string {
	t.Helper()
	var got string
	require.NoError(t, db.Raw(`SELECT cashfree_vendor_status FROM chef_profiles WHERE id = ?`, id.String()).Scan(&got).Error)
	return got
}

func auditCount(t *testing.T, db *gorm.DB) int64 {
	t.Helper()
	var n int64
	require.NoError(t, db.Raw(`SELECT COUNT(*) FROM audit_logs`).Scan(&n).Error)
	return n
}

func TestApplyEasySplitVendorStatus_RecordsARealTransition(t *testing.T) {
	db := setupTransitionDB(t)
	chef := seedTransitionChef(t, db, CashfreeVendorInBankValidation)

	changed := ApplyEasySplitVendorStatus(db, chef, "hc_1", CashfreeVendorActive)

	require.True(t, changed)
	require.Equal(t, CashfreeVendorActive, storedStatus(t, db, chef.ID))
	require.Equal(t, CashfreeVendorActive, chef.CashfreeVendorStatus, "the caller's copy must not go stale")
	require.EqualValues(t, 1, auditCount(t, db), "every transition is audited")
}

// A tick that finds nothing new must write nothing and tell nobody — otherwise
// a chef gets "payouts are now active" every 30 minutes, forever.
func TestApplyEasySplitVendorStatus_UnchangedStatusIsInert(t *testing.T) {
	db := setupTransitionDB(t)
	chef := seedTransitionChef(t, db, CashfreeVendorActive)

	require.False(t, ApplyEasySplitVendorStatus(db, chef, "hc_1", CashfreeVendorActive))
	require.Zero(t, auditCount(t, db))
}

// Cashfree's casing is not a contract; a re-cased echo is not a transition.
func TestApplyEasySplitVendorStatus_IgnoresCasingAndPadding(t *testing.T) {
	db := setupTransitionDB(t)
	chef := seedTransitionChef(t, db, CashfreeVendorActive)

	require.False(t, ApplyEasySplitVendorStatus(db, chef, "hc_1", "  active "))
	require.Equal(t, CashfreeVendorActive, storedStatus(t, db, chef.ID))
}

// The webhook and the cron both report the same transition. The write is
// conditional on the old value, so the second caller sees no change and the
// chef is notified once.
func TestApplyEasySplitVendorStatus_DoubleDeliveryChangesStateOnce(t *testing.T) {
	db := setupTransitionDB(t)
	fromWebhook := seedTransitionChef(t, db, CashfreeVendorInBankValidation)
	fromCron := *fromWebhook // a second read of the same row, pre-transition

	require.True(t, ApplyEasySplitVendorStatus(db, fromWebhook, "hc_1", CashfreeVendorActive))
	require.False(t, ApplyEasySplitVendorStatus(db, &fromCron, "hc_1", CashfreeVendorActive),
		"the loser of the race must not notify the chef a second time")
	require.EqualValues(t, 1, auditCount(t, db))
}

// An empty status is what a parse failure or a truncated payload looks like.
// Treating it as a transition would mark a verified chef unpayable.
func TestApplyEasySplitVendorStatus_RefusesAnEmptyStatus(t *testing.T) {
	db := setupTransitionDB(t)
	chef := seedTransitionChef(t, db, CashfreeVendorActive)

	require.False(t, ApplyEasySplitVendorStatus(db, chef, "hc_1", ""))
	require.Equal(t, CashfreeVendorActive, storedStatus(t, db, chef.ID))
}

// A vendor id arriving for the first time is recorded even when the status
// itself is unchanged — otherwise the chef stays unlinked and the sweep keeps
// re-registering them.
func TestApplyEasySplitVendorStatus_BackfillsAMissingVendorID(t *testing.T) {
	db := setupTransitionDB(t)
	chef := seedTransitionChef(t, db, CashfreeVendorInBeneCreation)
	require.NoError(t, db.Exec(`UPDATE chef_profiles SET cashfree_vendor_id = '' WHERE id = ?`, chef.ID.String()).Error)
	chef.CashfreeVendorID = ""

	require.True(t, ApplyEasySplitVendorStatus(db, chef, "hc_1", CashfreeVendorInBeneCreation))
	require.Equal(t, "hc_1", chef.CashfreeVendorID)
}

// What the chef is told, decided separately from delivery so it can be asserted.
func TestEasySplitStatusNotice_TellsTheChefOnlyWhatChangesTheirDay(t *testing.T) {
	active, ok := easySplitStatusNotice(CashfreeVendorInBankValidation, CashfreeVendorActive, "")
	require.True(t, ok)
	require.Equal(t, "easy_split_payouts_active", active.Kind)
	require.NotEmpty(t, active.Title)
	require.NotContains(t, active.Body, "_", "gateway vocabulary must not reach a chef")

	blocked, ok := easySplitStatusNotice(CashfreeVendorInBankValidation, CashfreeVendorBlocked, "")
	require.True(t, ok)
	require.Equal(t, "easy_split_payouts_failed", blocked.Kind)
	require.NotContains(t, blocked.Body, CashfreeVendorBlocked)

	// Progress between two pending states is not news.
	_, ok = easySplitStatusNotice(CashfreeVendorInBeneCreation, CashfreeVendorInBankValidation, "")
	require.False(t, ok)
}

// A failed bank validation is a refusal, not a stage: a chef told "pending"
// waits for something that is never coming.
func TestEasySplitStatusNotice_TreatsAFailedValidationAsAFailure(t *testing.T) {
	notice, ok := easySplitStatusNotice(CashfreeVendorInBankValidation, CashfreeVendorBankValidationFailed, "")
	require.True(t, ok)
	require.Equal(t, "easy_split_payouts_failed", notice.Kind)

	require.Equal(t, PayoutRegistrationFailed, PayoutRegistrationFor(&models.ChefProfile{
		CashfreeVendorID: "hc_1", CashfreeVendorStatus: CashfreeVendorBankValidationFailed,
	}).State)
	require.False(t, EasySplitNeedsReconcile(CashfreeVendorBankValidationFailed, time.Now()),
		"a refusal will not resolve itself — polling it burns a call per tick")
}

// Cashfree's remark is the only thing that says WHICH detail was wrong, so it
// is worth passing on — but only when it reads as a sentence. A status code
// echoed at a chef tells them nothing and reads like a fault in our app.
func TestEasySplitStatusNotice_PassesOnAReadableReasonOnly(t *testing.T) {
	readable, ok := easySplitStatusNotice("", CashfreeVendorBankValidationFailed,
		"The name on the account does not match the PAN")
	require.True(t, ok)
	require.Contains(t, readable.Body, "does not match the PAN")

	coded, ok := easySplitStatusNotice("", CashfreeVendorBankValidationFailed, "BENE_NAME_MISMATCH")
	require.True(t, ok)
	require.NotContains(t, coded.Body, "BENE_NAME_MISMATCH")
}

// A registration that has been dead for months costs a Cashfree call per tick
// and will never resolve on its own.
func TestEasySplitNeedsReconcile_SkipsTerminalAndLongDeadRegistrations(t *testing.T) {
	fresh := time.Now().Add(-time.Hour)
	stale := time.Now().Add(-90 * 24 * time.Hour)

	require.True(t, EasySplitNeedsReconcile(CashfreeVendorInBankValidation, fresh))
	require.True(t, EasySplitNeedsReconcile("", fresh), "never registered — the sweep is the retry")
	require.False(t, EasySplitNeedsReconcile(CashfreeVendorActive, fresh), "already payable")
	require.False(t, EasySplitNeedsReconcile(CashfreeVendorDeleted, fresh), "terminal")
	require.False(t, EasySplitNeedsReconcile(CashfreeVendorInBankValidation, stale), "dead registration")
}
