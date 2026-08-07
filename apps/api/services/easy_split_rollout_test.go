package services

// easy_split_rollout_test.go — #1084. Easy Split changes where a chef's money
// goes at the moment of capture, so it cannot be switched on for everyone at
// once. The per-chef override is what makes a first live chef possible, and
// what makes a chef reversible without touching the platform flag.

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/homechef/api/models"
)

// The whole point of the tri-state: force-on splits under a global off, and
// force-off never splits under a global on.
func TestEasySplitEnabledForChef_TriStateAgainstTheGlobalFlag(t *testing.T) {
	db := setupEasySplitDB(t)

	cases := []struct {
		mode   string
		global string
		want   bool
	}{
		{"", "false", false},
		{"", "true", true},
		{PayoutAutoOn, "false", true},
		{PayoutAutoOn, "true", true},
		{PayoutAutoOff, "false", false},
		{PayoutAutoOff, "true", false},
	}
	for _, tc := range cases {
		setEasySplitSetting(t, db, SettingEasySplitEnabled, tc.global)
		chef := &models.ChefProfile{EasySplitMode: tc.mode}
		require.Equal(t, tc.want, EasySplitEnabledForChef(db, chef),
			"mode %q under global %s", tc.mode, tc.global)
	}
}

// A chef with no override behaves exactly as before this issue existed.
func TestEasySplitEnabledForChef_DefaultsToTheGlobalFlag(t *testing.T) {
	db := setupEasySplitDB(t)
	setEasySplitSetting(t, db, SettingEasySplitEnabled, "true")

	require.True(t, EasySplitEnabledForChef(db, nil))
	require.True(t, EasySplitEnabledForChef(db, &models.ChefProfile{}))
	// An unrecognised value is not a licence to move money — it inherits.
	require.True(t, EasySplitEnabledForChef(db, &models.ChefProfile{EasySplitMode: "maybe"}))
}

func TestBuildOrderSplit_HonoursThePerChefOverride(t *testing.T) {
	db := setupEasySplitDB(t)
	order := easySplitOrder()
	capture := ToPaise(order.Total)

	// Global off, chef forced on — the rollout case.
	order.Chef.EasySplitMode = PayoutAutoOn
	require.NotNil(t, BuildOrderSplit(db, order, capture, 0))

	// Global on, chef forced off — the rollback case.
	setEasySplitSetting(t, db, SettingEasySplitEnabled, "true")
	order.Chef.EasySplitMode = PayoutAutoOff
	require.Nil(t, BuildOrderSplit(db, order, capture, 0))
}

// Force-on grants candidacy only. Every guard that protects the money still
// runs on top of it, exactly as it does for a globally enabled chef.
func TestBuildOrderSplit_ForceOnStillObeysEveryGuard(t *testing.T) {
	db := setupEasySplitDB(t)
	capture := ToPaise(easySplitOrder().Total)

	unverified := easySplitOrder()
	unverified.Chef.EasySplitMode = PayoutAutoOn
	unverified.Chef.CashfreeVendorStatus = CashfreeVendorInBankValidation
	require.Nil(t, BuildOrderSplit(db, unverified, capture, 0), "unverified vendor")

	credited := easySplitOrder()
	credited.Chef.EasySplitMode = PayoutAutoOn
	require.Nil(t, BuildOrderSplit(db, credited, capture-5000, 5000), "credit-funded order")

	lapsed := easySplitOrder()
	lapsed.Chef.EasySplitMode = PayoutAutoOn
	require.NoError(t, db.Exec(
		`INSERT INTO chef_documents (id, chef_id, type, status, expiry_date) VALUES (?, ?, ?, ?, ?)`,
		uuid.NewString(), lapsed.Chef.ID.String(), string(models.DocFSSAILicense),
		string(models.DocStatusVerified), time.Now().Add(-24*time.Hour)).Error)
	require.Nil(t, BuildOrderSplit(db, lapsed, capture, 0), "lapsed FSSAI")
}

// "Why did this order not split?" has to be answerable from what we recorded,
// not by re-running the decision against state that has since moved on.
func TestBuildOrderSplitWithReason_NamesTheGuardThatRefused(t *testing.T) {
	db := setupEasySplitDB(t)
	capture := ToPaise(easySplitOrder().Total)

	off := easySplitOrder()
	_, reason := BuildOrderSplitWithReason(db, off, capture, 0)
	require.Equal(t, EasySplitSkipDisabled, reason)

	setEasySplitSetting(t, db, SettingEasySplitEnabled, "true")

	credited := easySplitOrder()
	_, reason = BuildOrderSplitWithReason(db, credited, capture-5000, 5000)
	require.Equal(t, EasySplitSkipCreditFunded, reason)

	unverified := easySplitOrder()
	unverified.Chef.CashfreeVendorStatus = CashfreeVendorBlocked
	_, reason = BuildOrderSplitWithReason(db, unverified, capture, 0)
	require.Equal(t, EasySplitSkipVendorNotActive, reason)

	ok := easySplitOrder()
	split, reason := BuildOrderSplitWithReason(db, ok, capture, 0)
	require.NotNil(t, split)
	require.Empty(t, reason)
}

// The rollout switch has to reach the rail, not just the builder: releasing a
// forced-on chef's order splits it while the platform flag is still off.
func TestReleaseOrderSplit_SplitsAForcedOnChefWhileTheGlobalFlagIsOff(t *testing.T) {
	db := setupReleaseSplitDB(t)
	setEasySplitSetting(t, db, SettingEasySplitEnabled, "false")
	order := seedReleaseOrder(t, db, time.Now().Add(-2*time.Hour))
	require.NoError(t, db.Exec(`UPDATE chef_profiles SET easy_split_mode = ? WHERE id = ?`,
		PayoutAutoOn, order.Chef.ID.String()).Error)
	calls := withSplitGateway(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"status":"OK"}`))
	})

	split, err := ReleaseOrderSplit(db, order.ID, time.Now())

	require.NoError(t, err)
	require.True(t, split)
	require.Equal(t, 1, *calls)
}

// And a forced-off chef stays on the payout rail with the flag on.
func TestReleaseOrderSplit_LeavesAForcedOffChefOnThePayoutRail(t *testing.T) {
	db := setupReleaseSplitDB(t)
	order := seedReleaseOrder(t, db, time.Now().Add(-2*time.Hour))
	require.NoError(t, db.Exec(`UPDATE chef_profiles SET easy_split_mode = ? WHERE id = ?`,
		PayoutAutoOff, order.Chef.ID.String()).Error)
	calls := withSplitGateway(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"status":"OK"}`))
	})

	split, err := ReleaseOrderSplit(db, order.ID, time.Now())

	require.NoError(t, err)
	require.False(t, split)
	require.Zero(t, *calls)
}

// An order that took the payout rail must say why, in stored data. Without it
// the answer only exists for as long as the chef's state stays unchanged.
func TestReleaseOrderSplit_RecordsWhyTheOrderWasNotSplit(t *testing.T) {
	db := setupReleaseSplitDB(t)
	require.NoError(t, db.Exec(transitionAuditDDL).Error)
	order := seedReleaseOrder(t, db, time.Now().Add(-2*time.Hour))
	require.NoError(t, db.Exec(`UPDATE chef_profiles SET cashfree_vendor_status = ? WHERE id = ?`,
		CashfreeVendorInBankValidation, order.Chef.ID.String()).Error)
	withSplitGateway(t, func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(`{}`)) })

	split, err := ReleaseOrderSplit(db, order.ID, time.Now())
	require.NoError(t, err)
	require.False(t, split)

	var newValue string
	require.NoError(t, db.Raw(
		`SELECT new_value FROM audit_logs WHERE entity_id = ? AND action = 'order.payout.easy_split_skipped'`,
		order.ID.String()).Scan(&newValue).Error)
	require.Contains(t, newValue, EasySplitSkipVendorNotActive)
}
