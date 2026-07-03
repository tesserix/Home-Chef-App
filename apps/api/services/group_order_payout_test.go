package services

// group_order_payout_test.go — the group/office order payout hold aggregate (#456).
// A group order is a first-class payout-hold aggregate: delivery parks the group
// hold in awaiting_customer_confirmation (never releasing money), the host confirms
// to reach release_eligible, and the admin queue / sweep / reconcile drive it exactly
// like a regular order or meal-plan day. Runs on the same in-memory sqlite harness as
// payout_hold_test.go (setupHoldDB) / payout_release_test.go (setupReleaseDB), which
// now hand-DDL the group_orders table (gen_random_uuid() can't run on sqlite).

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/homechef/api/config"
	"github.com/homechef/api/models"
)

// seedGroupOrder inserts a placed group order in the given hold state with a held
// Route transfer and a linked consolidated order id (razorpay-less). Returns the
// group order id and its consolidated order id.
func seedGroupOrder(t *testing.T, db *gorm.DB, hold models.PayoutHoldStatus) (uuid.UUID, uuid.UUID) {
	t.Helper()
	gid, oid, host, chef := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO group_orders
		(id, host_id, chef_id, order_id, status, payout_transfer_id, payout_hold_status, subtotal, tax, currency)
		VALUES (?,?,?,?,?,?,?,?,?,?)`,
		gid.String(), host.String(), chef.String(), oid.String(), string(models.GroupOrderPlaced),
		"trf_grp_"+gid.String()[:8], string(hold), 200.0, 20.0, "INR").Error)
	return gid, oid
}

// loadGroupOrder reads a group order row.
func loadGroupOrder(t *testing.T, db *gorm.DB, id uuid.UUID) models.GroupOrder {
	t.Helper()
	var g models.GroupOrder
	require.NoError(t, db.First(&g, "id = ?", id).Error)
	return g
}

// TestMarkGroupOrderDelivered_ParksHold_NoRelease — delivery parks the group hold in
// awaiting_customer_confirmation + stamps delivered_at and releases NOTHING, with the
// money flag BOTH off and on. GetRazorpay() is nil in tests, so any real release would
// error; we assert the hold parks cleanly and never releases.
func TestMarkGroupOrderDelivered_ParksHold_NoRelease(t *testing.T) {
	for _, flagOn := range []bool{false, true} {
		flagOn := flagOn
		t.Run(map[bool]string{false: "flag-off", true: "flag-on"}[flagOn], func(t *testing.T) {
			saved := config.AppConfig
			t.Cleanup(func() { config.AppConfig = saved })
			config.AppConfig = &config.Config{OrderPayoutAutoReleaseEnabled: flagOn}

			db := setupHoldDB(t)
			gid, oid := seedGroupOrder(t, db, models.PayoutHoldNone)

			withSweepDB(t, db, func() { MarkGroupOrderDelivered(oid) })

			g := loadGroupOrder(t, db, gid)
			require.Equal(t, models.GroupOrderDelivered, g.Status, "group marked delivered")
			require.Equal(t, models.PayoutHoldAwaitingConfirmation, g.PayoutHoldStatus, "delivery parks the hold, never releases")
			require.NotNil(t, g.DeliveredAt, "delivered_at stamped")
		})
	}
}

// TestConfirmGroupOrderHold_ReleaseEligible — host confirm on an awaiting group hold
// advances it to release_eligible, stamps customer_confirmed_at, emits exactly one
// release-eligible event; a re-confirm is an idempotent no-op (no double emit).
func TestConfirmGroupOrderHold_ReleaseEligible(t *testing.T) {
	db := setupHoldDB(t)
	gid, _ := seedGroupOrder(t, db, models.PayoutHoldAwaitingConfirmation)

	g := loadGroupOrder(t, db, gid)
	status, err := ConfirmGroupOrderHold(db, &g)
	require.NoError(t, err)
	require.Equal(t, models.PayoutHoldReleaseEligible, status)

	got := loadGroupOrder(t, db, gid)
	require.Equal(t, models.PayoutHoldReleaseEligible, got.PayoutHoldStatus)
	require.NotNil(t, got.CustomerConfirmedAt)
	require.Equal(t, 1, countOutbox(t, db, SubjectHoldReleaseEligible))

	again := loadGroupOrder(t, db, gid)
	status2, err := ConfirmGroupOrderHold(db, &again)
	require.NoError(t, err)
	require.Equal(t, models.PayoutHoldReleaseEligible, status2)
	require.Equal(t, 1, countOutbox(t, db, SubjectHoldReleaseEligible), "re-confirm must not double-emit")
}

// TestConfirmGroupOrderHold_Disputed — an open pending OrderIssue on the consolidated
// order forces disputed, never release_eligible.
func TestConfirmGroupOrderHold_Disputed(t *testing.T) {
	db := setupHoldDB(t)
	gid, oid := seedGroupOrder(t, db, models.PayoutHoldAwaitingConfirmation)
	require.NoError(t, db.Exec(`INSERT INTO order_issues (id, order_id, status) VALUES (?,?,?)`,
		uuid.NewString(), oid.String(), string(models.IssuePending)).Error)

	g := loadGroupOrder(t, db, gid)
	status, err := ConfirmGroupOrderHold(db, &g)
	require.NoError(t, err)
	require.Equal(t, models.PayoutHoldDisputed, status)
	require.Equal(t, models.PayoutHoldDisputed, loadGroupOrder(t, db, gid).PayoutHoldStatus)
	require.Equal(t, 1, countOutbox(t, db, SubjectHoldDisputed))
	require.Equal(t, 0, countOutbox(t, db, SubjectHoldReleaseEligible))
}

// TestReleaseHold_GroupOrder — ReleaseHold("group-order", id) on a release_eligible
// group hold advances it to released; a re-release is ErrHoldNotEligible (409). Flag
// off ⇒ state-only; flag on with GetRazorpay()==nil ⇒ seam reached, no crash.
func TestReleaseHold_GroupOrder(t *testing.T) {
	saved := config.AppConfig
	t.Cleanup(func() { config.AppConfig = saved })
	config.AppConfig = &config.Config{OrderPayoutAutoReleaseEnabled: true}

	db := setupReleaseDB(t)
	gid, _ := seedGroupOrder(t, db, models.PayoutHoldReleaseEligible)

	require.NoError(t, ReleaseHold(db, "group-order", gid))
	require.Equal(t, models.PayoutHoldReleased, loadGroupOrder(t, db, gid).PayoutHoldStatus)
	require.Equal(t, 1, countOutbox(t, db, SubjectHoldReleased))

	err := ReleaseHold(db, "group-order", gid)
	require.ErrorIs(t, err, ErrHoldNotEligible)
	require.Equal(t, 1, countOutbox(t, db, SubjectHoldReleased), "no double-emit")
}

// TestListPendingPayouts_IncludesGroupOrders — a release_eligible group hold surfaces
// in the admin pending queue with AggType "group-order" and a non-empty, human-scannable
// context (never a raw UUID).
func TestListPendingPayouts_IncludesGroupOrders(t *testing.T) {
	db := setupReleaseDB(t)
	gid, _ := seedGroupOrder(t, db, models.PayoutHoldReleaseEligible)

	rows, err := ListPendingPayouts(db, PendingFilter{})
	require.NoError(t, err)

	var found *PendingPayout
	for i := range rows {
		if rows[i].ID == gid {
			found = &rows[i]
		}
	}
	require.NotNil(t, found, "group hold surfaced in the pending queue")
	require.Equal(t, "group-order", found.AggType)
	require.Equal(t, 220.0, found.Amount, "amount = subtotal + tax (chef slice)")
	require.NotEmpty(t, found.Context, "context must be non-empty")
	require.NotEqual(t, gid.String(), found.Context, "context must not be a raw UUID")
}

// TestSweepAndReconcile_CoverGroupOrders — a stale awaiting group hold is auto-confirmed
// by the sweep; a released-but-unsettled group hold (flag on) is re-driven by reconcile.
func TestSweepAndReconcile_CoverGroupOrders(t *testing.T) {
	saved := config.AppConfig
	t.Cleanup(func() { config.AppConfig = saved })
	config.AppConfig = &config.Config{OrderPayoutAutoReleaseEnabled: true}

	db := setupHoldDB(t)

	// A stale awaiting group hold (delivered 30h ago) → auto-confirm to release_eligible.
	stale := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO group_orders
		(id, host_id, chef_id, order_id, status, payout_transfer_id, payout_hold_status, delivered_at, subtotal, tax)
		VALUES (?,?,?,?,?,?,?,?,?,?)`,
		stale.String(), uuid.NewString(), uuid.NewString(), uuid.NewString(), string(models.GroupOrderDelivered),
		"trf_grp_stale", string(models.PayoutHoldAwaitingConfirmation), time.Now().Add(-30*time.Hour), 100.0, 10.0).Error)

	withSweepDB(t, db, func() { runPayoutAutoConfirmScan(context.Background()) })
	require.Equal(t, models.PayoutHoldReleaseEligible, loadGroupOrder(t, db, stale).PayoutHoldStatus,
		"stale awaiting group hold auto-confirmed")

	// A released-but-unsettled group hold (with a transfer id) → reconcile stamps settled.
	drift := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO group_orders
		(id, host_id, chef_id, order_id, status, payout_transfer_id, payout_hold_status, subtotal, tax)
		VALUES (?,?,?,?,?,?,?,?,?)`,
		drift.String(), uuid.NewString(), uuid.NewString(), uuid.NewString(), string(models.GroupOrderDelivered),
		"trf_grp_drift", string(models.PayoutHoldReleased), 100.0, 10.0).Error)

	withSweepDB(t, db, func() { runPayoutReconcileScan(context.Background()) })
	require.NotNil(t, loadGroupOrder(t, db, drift).PayoutSettledAt, "released+unsettled group hold re-driven + stamped")
}

// TestNoPhantomConsolidatedOrderHold (Q1 regression) — delivering a group order parks
// ONLY the group hold; the consolidated order (no razorpay_order_id) stays PayoutHoldNone
// and never enters the pending queue, locking the no-phantom-double-hold invariant.
func TestNoPhantomConsolidatedOrderHold(t *testing.T) {
	saved := config.AppConfig
	t.Cleanup(func() { config.AppConfig = saved })
	config.AppConfig = &config.Config{}

	db := setupHoldDB(t)
	gid, oid := seedGroupOrder(t, db, models.PayoutHoldNone)
	// The consolidated order carries NO razorpay_order_id (participants pay their own).
	require.NoError(t, db.Exec(`INSERT INTO orders (id, customer_id, status, razorpay_order_id, payout_hold_status) VALUES (?,?,?,?,?)`,
		oid.String(), uuid.NewString(), "delivered", "", "").Error)

	withSweepDB(t, db, func() { MarkGroupOrderDelivered(oid) })

	require.Equal(t, models.PayoutHoldAwaitingConfirmation, loadGroupOrder(t, db, gid).PayoutHoldStatus,
		"group hold advances")
	require.Equal(t, models.PayoutHoldNone, loadOrder(t, db, oid).PayoutHoldStatus,
		"consolidated order stays PayoutHoldNone (no phantom double-hold)")
}
