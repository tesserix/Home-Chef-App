package services

// order_release_easy_split_test.go — #1086. The Route transfer layer that used to
// run beside the split in releaseMoney is gone. This is the proof the release path
// itself survived: an admin release of a delivered Cashfree order must still reach
// Cashfree and pay the chef, not quietly settle the hold having moved nothing.

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/homechef/api/config"
	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

// setupHoldSplitDB is the hold-machinery harness plus the columns and chef table
// the split reads, so one release exercises both halves of the seam.
func setupHoldSplitDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := setupReleaseDB(t)
	for _, s := range []string{
		`ALTER TABLE orders ADD COLUMN gateway_split_paise INTEGER DEFAULT 0`,
		`ALTER TABLE orders ADD COLUMN wallet_applied REAL DEFAULT 0`,
		`ALTER TABLE orders ADD COLUMN loyalty_applied REAL DEFAULT 0`,
		`CREATE TABLE chef_documents (id TEXT PRIMARY KEY, chef_id TEXT, type TEXT,
			status TEXT DEFAULT 'approved', expiry_date DATETIME,
			created_at DATETIME, updated_at DATETIME, deleted_at DATETIME)`,
		releaseChefDDL,
	} {
		require.NoError(t, db.Exec(s).Error)
	}
	require.NoError(t, db.Exec(`INSERT INTO platform_settings (id, key, value) VALUES (?,?,?)`,
		uuid.NewString(), SettingEasySplitEnabled, "true").Error)

	orig := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = orig })
	return db
}

// seedCashfreeOrderHold writes a delivered, Cashfree-captured order sitting in
// release_eligible, with a registered vendor and inside the split window.
func seedCashfreeOrderHold(t *testing.T, db *gorm.DB) uuid.UUID {
	t.Helper()
	id, chef := uuid.New(), uuid.New()
	require.NoError(t, db.Exec(
		`INSERT INTO chef_profiles (id, user_id, payout_country, cashfree_vendor_id, cashfree_vendor_status)
		 VALUES (?,?,?,?,?)`,
		chef.String(), uuid.NewString(), "IN", "vendor_"+chef.String()[:8], CashfreeVendorActive).Error)
	// Two hours old: past Cashfree's sync delay, inside the split window.
	now := time.Now().Add(-2 * time.Hour)
	require.NoError(t, db.Exec(`INSERT INTO orders
		(id, order_number, customer_id, chef_id, status, gateway_order_id, payment_provider,
		 total, subtotal, commission_rate, payout_hold_status, delivered_at, created_at, updated_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		id.String(), "ORD-"+id.String()[:8], uuid.NewString(), chef.String(), "delivered",
		"cf-"+id.String()[:8], models.PaymentProviderCashfree,
		1000.0, 1000.0, 20.0, string(models.PayoutHoldReleaseEligible), now, now, now).Error)
	return id
}

// The release actuator still pays the chef: Cashfree is called and the split is
// stamped, which is also what keeps the order off the weekly statement.
func TestReleaseHold_DeliveredCashfreeOrder_StillSplitsAtTheGateway(t *testing.T) {
	saved := config.AppConfig
	t.Cleanup(func() { config.AppConfig = saved })
	config.AppConfig = &config.Config{OrderPayoutAutoReleaseEnabled: true}

	db := setupHoldSplitDB(t)
	calls := withSplitGateway(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"split_status":"SUCCESS"}`))
	})
	id := seedCashfreeOrderHold(t, db)

	require.NoError(t, ReleaseHold(db, "order", id))

	require.Equal(t, models.PayoutHoldReleased, loadOrder(t, db, id).PayoutHoldStatus)
	require.Equal(t, 1, *calls, "the release called Cashfree exactly once")
	require.Positive(t, splitPaiseOf(t, db, id), "the chef's share is stamped on the order")
}

// The other rail: an order Cashfree will not split settles on the statement path,
// so nothing is stamped — and the hold still releases rather than stranding.
func TestReleaseHold_UnregisteredVendor_FallsBackToTheStatementPath(t *testing.T) {
	saved := config.AppConfig
	t.Cleanup(func() { config.AppConfig = saved })
	config.AppConfig = &config.Config{OrderPayoutAutoReleaseEnabled: true}

	db := setupHoldSplitDB(t)
	calls := withSplitGateway(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	id := seedCashfreeOrderHold(t, db)
	require.NoError(t, db.Exec(`UPDATE chef_profiles SET cashfree_vendor_id = '', cashfree_vendor_status = ''`).Error)

	require.NoError(t, ReleaseHold(db, "order", id))

	require.Equal(t, models.PayoutHoldReleased, loadOrder(t, db, id).PayoutHoldStatus)
	require.Equal(t, 0, *calls, "no split attempted for a chef Cashfree does not know")
	require.Zero(t, splitPaiseOf(t, db, id), "unstamped — the weekly statement pays this one")
}
