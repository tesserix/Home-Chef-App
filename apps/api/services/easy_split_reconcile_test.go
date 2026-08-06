package services

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/homechef/api/models"
)

// #1029 — nothing retried a failed Easy Split registration, so a chef could sit
// un-payable (and un-tippable) forever. These pin WHICH chefs the reconcile
// sweep picks up; the Cashfree calls themselves need a live gateway, so the
// selection query is what is asserted here.

func setupEasySplitReconcileDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TABLE chef_profiles (
		id text PRIMARY KEY, user_id text, mode text DEFAULT 'live',
		business_name text DEFAULT '', pan_number text DEFAULT '',
		payout_method text DEFAULT '',
		cashfree_vendor_id text DEFAULT '', cashfree_vendor_status text DEFAULT '',
		razorpay_account_id text DEFAULT '', deleted_at datetime
	)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE users (id text PRIMARY KEY, email text, phone text, deleted_at datetime)`).Error)
	return db
}

func insertChef(t *testing.T, db *gorm.DB, payoutMethod, vendorID, vendorStatus string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	require.NoError(t, db.Exec(
		`INSERT INTO chef_profiles (id, user_id, mode, payout_method, cashfree_vendor_id, cashfree_vendor_status) VALUES (?,?,?,?,?,?)`,
		id.String(), uuid.New().String(), "live", payoutMethod, vendorID, vendorStatus).Error)
	return id
}

// pendingChefs runs the sweep's own selection criteria, so the test pins the
// query rather than a copy of it.
func pendingChefs(t *testing.T, db *gorm.DB) []models.ChefProfile {
	t.Helper()
	var chefs []models.ChefProfile
	require.NoError(t, db.
		Where("COALESCE(payout_method, '') <> ''").
		Where("COALESCE(cashfree_vendor_status, '') <> ?", CashfreeVendorActive).
		Find(&chefs).Error)
	return chefs
}

func TestEasySplitReconcile_PicksUpChefsStuckWithoutAnActiveVendor(t *testing.T) {
	db := setupEasySplitReconcileDB(t)

	// The #1029 production state: payout details saved, registration never landed.
	stuck := insertChef(t, db, "bank", "", "")
	// Registered but still verifying at Cashfree — needs a status re-read.
	verifying := insertChef(t, db, "bank", "vend_2", "PENDING")
	// Already done — must not be swept, or every pass burns a Cashfree call.
	insertChef(t, db, "bank", "vend_3", CashfreeVendorActive)
	// No payout destination at all — nothing to register.
	insertChef(t, db, "", "", "")

	got := pendingChefs(t, db)
	ids := map[uuid.UUID]bool{}
	for _, c := range got {
		ids[c.ID] = true
	}
	require.Len(t, got, 2, "only the two incomplete chefs are candidates")
	require.True(t, ids[stuck], "a chef whose registration never landed must be retried")
	require.True(t, ids[verifying], "a vendor still verifying must be re-read")
}

func TestEasySplitReconcile_ActiveIsMatchedCaseInsensitively(t *testing.T) {
	db := setupEasySplitReconcileDB(t)
	// Cashfree returns "ACTIVE"; the column is whatever was persisted. A chef
	// stored lowercase must not be swept forever.
	insertChef(t, db, "bank", "vend_1", "ACTIVE")
	require.Empty(t, pendingChefs(t, db))
}

func TestReconcileEasySplitVendors_GuardsAgainstNoWork(t *testing.T) {
	db := setupEasySplitReconcileDB(t)
	// No candidates → no attempts, no panic.
	activated, attempted := ReconcileEasySplitVendors(context.Background(), db, 50)
	require.Zero(t, activated)
	require.Zero(t, attempted)

	// Defensive: a nil db or a non-positive limit must be inert rather than
	// panicking inside a cron goroutine.
	a, b := ReconcileEasySplitVendors(context.Background(), nil, 50)
	require.Zero(t, a)
	require.Zero(t, b)
	a, b = ReconcileEasySplitVendors(context.Background(), db, 0)
	require.Zero(t, a)
	require.Zero(t, b)
}
