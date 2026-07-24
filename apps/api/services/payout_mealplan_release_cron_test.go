package services

// payout_mealplan_release_cron_test.go — the per-day tiffin auto-release sweep
// (services/payout_mealplan_release_cron.go). Pure DB logic on the shared
// hand-DDL'd sqlite harness (setupHoldDB); escrow flags are OFF (flagsOff) so
// ReleaseHold's money seam is a no-op and every transition is a plain DB advance
// — no Razorpay needed. The clock is injected, so maturation is exact.

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

// mplrSeedDay inserts a meal_plans row + one meal_plan_days row (order_id NULL, so
// no refund/dispute cross-guard can block a clean release) and returns the day id.
// Nil time pointers are stored as SQL NULL.
func mplrSeedDay(t *testing.T, db *gorm.DB, hold models.PayoutHoldStatus, confirmedAt, deliveredAt, settledAt *time.Time) uuid.UUID {
	t.Helper()
	dayID, planID := uuid.New(), uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO meal_plans (id, customer_id, chef_id, status) VALUES (?,?,?,?)`,
		planID.String(), uuid.NewString(), uuid.NewString(), "active").Error)
	require.NoError(t, db.Exec(`INSERT INTO meal_plan_days
		(id, meal_plan_id, order_id, status, payout_hold_status, customer_confirmed_at, delivered_at, payout_settled_at)
		VALUES (?,?,?,?,?,?,?,?)`,
		dayID.String(), planID.String(), nil, "delivered", string(hold), confirmedAt, deliveredAt, settledAt).Error)
	return dayID
}

// mplrDaySettledAt returns the day's payout_settled_at (nil when unset).
func mplrDaySettledAt(t *testing.T, db *gorm.DB, id uuid.UUID) *time.Time {
	t.Helper()
	var d models.MealPlanDay
	require.NoError(t, db.First(&d, "id = ?", id).Error)
	return d.PayoutSettledAt
}

// TestMealPlanDayReleaseSweep covers the maturation gate across the four states a
// day can be in when the sweep runs, on a fixed clock.
func TestMealPlanDayReleaseSweep(t *testing.T) {
	now := time.Date(2026, 7, 24, 12, 0, 0, 0, time.UTC)
	confirmed25h := now.Add(-25 * time.Hour) // matured (past the 24h default)
	confirmed1h := now.Add(-1 * time.Hour)   // still maturing
	deliveredRecent := now.Add(-2 * time.Hour)
	alreadySettled := now.Add(-10 * time.Hour)

	cases := []struct {
		name        string
		hold        models.PayoutHoldStatus
		confirmedAt *time.Time
		deliveredAt *time.Time
		settledAt   *time.Time
		wantCount   int
		wantHold    models.PayoutHoldStatus
		wantSettled string // "set" | "nil" | "unchanged"
	}{
		{
			// (a) matured & eligible → RELEASED + settled stamped.
			name:        "matured_release_eligible_releases",
			hold:        models.PayoutHoldReleaseEligible,
			confirmedAt: &confirmed25h,
			wantCount:   1,
			wantHold:    models.PayoutHoldReleased,
			wantSettled: "set",
		},
		{
			// (b) confirmed only 1h ago → still maturing, untouched.
			name:        "immature_release_eligible_not_released",
			hold:        models.PayoutHoldReleaseEligible,
			confirmedAt: &confirmed1h,
			wantCount:   0,
			wantHold:    models.PayoutHoldReleaseEligible,
			wantSettled: "nil",
		},
		{
			// (c) still awaiting confirmation (delivered recently) → not this
			// sweep's stage; never touched.
			name:        "awaiting_confirmation_untouched",
			hold:        models.PayoutHoldAwaitingConfirmation,
			deliveredAt: &deliveredRecent,
			wantCount:   0,
			wantHold:    models.PayoutHoldAwaitingConfirmation,
			wantSettled: "nil",
		},
		{
			// (d) matured & eligible but already settled → excluded, not re-touched.
			name:        "already_settled_not_retouched",
			hold:        models.PayoutHoldReleaseEligible,
			confirmedAt: &confirmed25h,
			settledAt:   &alreadySettled,
			wantCount:   0,
			wantHold:    models.PayoutHoldReleaseEligible,
			wantSettled: "unchanged",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			flagsOff(t)
			db := setupHoldDB(t)
			id := mplrSeedDay(t, db, tc.hold, tc.confirmedAt, tc.deliveredAt, tc.settledAt)

			require.Equal(t, tc.wantCount, sweepMaturedMealPlanDays(db, now))
			require.Equal(t, tc.wantHold, loadDayHold(t, db, id))

			settled := mplrDaySettledAt(t, db, id)
			switch tc.wantSettled {
			case "set":
				require.NotNil(t, settled, "payout_settled_at must be stamped after release")
			case "nil":
				require.Nil(t, settled, "payout_settled_at must stay NULL")
			case "unchanged":
				require.NotNil(t, settled)
				require.WithinDuration(t, *tc.settledAt, *settled, time.Second, "settled stamp must not be overwritten")
			}
		})
	}
}

// TestMealPlanDayReleaseSweep_DisabledReturnsZero — an explicit "false" opt-out
// short-circuits the sweep before any release.
func TestMealPlanDayReleaseSweep_DisabledReturnsZero(t *testing.T) {
	flagsOff(t)
	db := setupHoldDB(t)
	setSetting(t, db, "payout.mealplan_auto_release_enabled", "false")

	now := time.Date(2026, 7, 24, 12, 0, 0, 0, time.UTC)
	confirmed25h := now.Add(-25 * time.Hour)
	id := mplrSeedDay(t, db, models.PayoutHoldReleaseEligible, &confirmed25h, nil, nil)

	require.Equal(t, 0, sweepMaturedMealPlanDays(db, now))
	require.Equal(t, models.PayoutHoldReleaseEligible, loadDayHold(t, db, id), "disabled sweep must not release")
	require.Nil(t, mplrDaySettledAt(t, db, id))
}

// TestMealPlanAutoReleaseEnabled — opt-OUT semantics: unset defaults true; only an
// explicit false/0/off (case-insensitive) turns it off; anything else is on.
func TestMealPlanAutoReleaseEnabled(t *testing.T) {
	cases := []struct {
		name string
		set  bool
		val  string
		want bool
	}{
		{"unset_defaults_true", false, "", true},
		{"empty_defaults_true", true, "", true},
		{"false_disables", true, "false", false},
		{"upper_false_disables", true, "FALSE", false},
		{"zero_disables", true, "0", false},
		{"off_disables", true, "off", false},
		{"mixed_case_off_disables", true, "Off", false},
		{"true_enabled", true, "true", true},
		{"on_enabled", true, "on", true},
		{"other_value_enabled", true, "yes", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db := setupHoldDB(t)
			if tc.set {
				setSetting(t, db, "payout.mealplan_auto_release_enabled", tc.val)
			}
			require.Equal(t, tc.want, mealPlanAutoReleaseEnabled(db))
		})
	}
}

// TestMealPlanReleaseMaturation — defaults to 24h when unset/unparseable/<=0 and
// honours an explicit whole-hours override.
func TestMealPlanReleaseMaturation(t *testing.T) {
	t.Run("unset_defaults_24h", func(t *testing.T) {
		db := setupHoldDB(t)
		require.Equal(t, 24*time.Hour, mealPlanReleaseMaturation(db))
	})
	t.Run("explicit_48h_honoured", func(t *testing.T) {
		db := setupHoldDB(t)
		setSetting(t, db, "payout.mealplan_maturation_hours", "48")
		require.Equal(t, 48*time.Hour, mealPlanReleaseMaturation(db))
	})
	t.Run("zero_falls_back_to_default", func(t *testing.T) {
		db := setupHoldDB(t)
		setSetting(t, db, "payout.mealplan_maturation_hours", "0")
		require.Equal(t, 24*time.Hour, mealPlanReleaseMaturation(db))
	})
	t.Run("unparseable_falls_back_to_default", func(t *testing.T) {
		db := setupHoldDB(t)
		setSetting(t, db, "payout.mealplan_maturation_hours", "notanumber")
		require.Equal(t, 24*time.Hour, mealPlanReleaseMaturation(db))
	})
}
