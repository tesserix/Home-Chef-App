package services

// easy_split_vendor_partition_test.go — #1145. Cashfree sandbox and production
// vendor IDs are separate namespaces, so a registration made while a kitchen is
// being used as a sandbox must not be able to become its live payout identity.

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

const partitionChefDDL = `CREATE TABLE chef_profiles (
	id TEXT PRIMARY KEY, user_id TEXT, business_name TEXT DEFAULT '', mode TEXT DEFAULT 'live',
	payout_method TEXT DEFAULT '', pan_number TEXT DEFAULT '',
	cashfree_vendor_id TEXT DEFAULT '', cashfree_vendor_status TEXT DEFAULT '',
	cashfree_test_vendor_id TEXT DEFAULT '', cashfree_test_vendor_status TEXT DEFAULT '',
	created_at DATETIME, updated_at DATETIME, deleted_at DATETIME)`

func setupPartitionDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=private"), &gorm.Config{
		Logger: glogger.Default.LogMode(glogger.Silent),
	})
	require.NoError(t, err)
	require.NoError(t, db.Exec(partitionChefDDL).Error)
	require.NoError(t, db.Exec(transitionAuditDDL).Error)
	require.NoError(t, db.Exec(`CREATE TABLE users (
		id TEXT PRIMARY KEY, email TEXT DEFAULT '', created_at DATETIME,
		updated_at DATETIME, deleted_at DATETIME)`).Error)

	orig := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = orig })
	return db
}

// seedPartitionChef writes a kitchen with a live registration already in place.
func seedPartitionChef(t *testing.T, db *gorm.DB, mode string) *models.ChefProfile {
	t.Helper()
	chef := &models.ChefProfile{
		ID: uuid.New(), UserID: uuid.New(), Mode: mode, PayoutMethod: "bank_transfer",
		CashfreeVendorID: "hc_live_vendor", CashfreeVendorStatus: CashfreeVendorActive,
	}
	require.NoError(t, db.Exec(
		`INSERT INTO chef_profiles (id, user_id, mode, payout_method, cashfree_vendor_id, cashfree_vendor_status, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		chef.ID.String(), chef.UserID.String(), mode, chef.PayoutMethod,
		chef.CashfreeVendorID, chef.CashfreeVendorStatus, time.Now().Add(-time.Hour)).Error)
	return chef
}

func reloadPartitionChef(t *testing.T, db *gorm.DB, id uuid.UUID) *models.ChefProfile {
	t.Helper()
	var got models.ChefProfile
	require.NoError(t, db.First(&got, "id = ?", id).Error)
	return &got
}

// The bug this issue exists for: a sandbox registration must never become the
// identity real customers' money is split to.
func TestTestModeRegistrationDoesNotClobberTheLiveVendor(t *testing.T) {
	db := setupPartitionDB(t)
	chef := seedPartitionChef(t, db, models.ChefModeTest)

	changed := ApplyEasySplitVendorStatus(db, chef, "hc_sandbox_vendor", CashfreeVendorActive)
	require.True(t, changed, "a first sandbox registration is a change")

	got := reloadPartitionChef(t, db, chef.ID)
	require.Equal(t, "hc_live_vendor", got.CashfreeVendorID,
		"the live vendor id must survive a sandbox registration")
	require.Equal(t, CashfreeVendorActive, got.CashfreeVendorStatus)
	require.Equal(t, "hc_sandbox_vendor", got.CashfreeTestVendorID)
	require.Equal(t, CashfreeVendorActive, got.CashfreeTestVendorStatus)
}

func TestLiveModeRegistrationWritesTheLivePairOnly(t *testing.T) {
	db := setupPartitionDB(t)
	chef := seedPartitionChef(t, db, models.ChefModeLive)

	require.True(t, ApplyEasySplitVendorStatus(db, chef, "hc_live_vendor", CashfreeVendorBankValidationFailed))

	got := reloadPartitionChef(t, db, chef.ID)
	require.Equal(t, CashfreeVendorBankValidationFailed, got.CashfreeVendorStatus)
	require.Empty(t, got.CashfreeTestVendorID, "a live registration must not touch the test pair")
	require.Empty(t, got.CashfreeTestVendorStatus)
}

func TestVendorIdentityReadsThePairForTheChefsOwnMode(t *testing.T) {
	chef := &models.ChefProfile{
		CashfreeVendorID: "hc_live_vendor", CashfreeVendorStatus: CashfreeVendorActive,
		CashfreeTestVendorID: "hc_sandbox_vendor", CashfreeTestVendorStatus: CashfreeVendorBankValidationFailed,
	}

	chef.Mode = models.ChefModeLive
	require.Equal(t, "hc_live_vendor", chef.VendorID())
	require.Equal(t, CashfreeVendorActive, chef.VendorStatus())

	chef.Mode = models.ChefModeTest
	require.Equal(t, "hc_sandbox_vendor", chef.VendorID())
	require.Equal(t, CashfreeVendorBankValidationFailed, chef.VendorStatus())
}

// An unreadable mode must resolve to live, matching NormalizeMode everywhere
// else — a kitchen is a real one unless something says otherwise.
func TestUnknownModeReadsTheLivePair(t *testing.T) {
	chef := &models.ChefProfile{
		Mode:             "  ",
		CashfreeVendorID: "hc_live_vendor", CashfreeVendorStatus: CashfreeVendorActive,
		CashfreeTestVendorID: "hc_sandbox_vendor", CashfreeTestVendorStatus: CashfreeVendorActive,
	}
	require.Equal(t, "hc_live_vendor", chef.VendorID())
}

// The reconcile sweep must judge each chef by the registration of the partition
// it is in, or a kitchen borrowed as a sandbox is picked by its live status and
// its pending sandbox registration never completes.
func TestReconcileCandidatesJudgeEachChefByItsOwnPartition(t *testing.T) {
	db := setupPartitionDB(t)

	// Live-ACTIVE, so nothing to do on the live partition — but its sandbox
	// registration is still pending and is the one that matters right now.
	sandboxing := seedPartitionChef(t, db, models.ChefModeTest)
	require.NoError(t, db.Exec(
		`UPDATE chef_profiles SET cashfree_test_vendor_id = '', cashfree_test_vendor_status = '' WHERE id = ?`,
		sandboxing.ID.String()).Error)

	settled := seedPartitionChef(t, db, models.ChefModeLive)

	got, err := easySplitReconcileCandidates(db, 10)
	require.NoError(t, err)

	ids := make([]string, 0, len(got))
	for _, c := range got {
		ids = append(ids, c.ID.String())
	}
	require.Contains(t, ids, sandboxing.ID.String(), "a pending sandbox registration must still be swept")
	require.NotContains(t, ids, settled.ID.String(), "an ACTIVE live registration has nothing to reconcile")
}

// A kitchen whose sandbox registration is verified is still unpayable on its
// live partition, and the split builder must say so.
func TestSplitIsRefusedWhenOnlyTheOtherModesVendorIsActive(t *testing.T) {
	chef := &models.ChefProfile{
		Mode:             models.ChefModeLive,
		EasySplitMode:    "on",
		CashfreeVendorID: "", CashfreeVendorStatus: "",
		CashfreeTestVendorID: "hc_sandbox_vendor", CashfreeTestVendorStatus: CashfreeVendorActive,
	}
	require.Equal(t, "", chef.VendorID(), "a live kitchen must not inherit the sandbox vendor")
	require.NotEqual(t, CashfreeVendorActive, chef.VendorStatus())
}
