package services

// meal_plan_refund_tiers_test.go — the lead-time tier table (#834 item 4).
//
// Every boundary is table-tested from BOTH sides, because each one moves money: a day that
// resolves at 12.001h auto-refunds 100% with no chef step, and the same day at 11.999h can be
// refunded as little as 75%.

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// withTiers installs a tier table for the duration of a test. Writes the policy cache
// directly (same package) so no DB is needed.
func withTiers(t *testing.T, tiers []MealPlanRefundTier) {
	t.Helper()
	p := DefaultPlatformPolicy()
	p.MealPlanRefundTiers = tiers
	withPlatformPolicy(t, p)
}

// withPlatformPolicy installs a whole policy for the duration of a test.
func withPlatformPolicy(t *testing.T, p PlatformPolicy) {
	t.Helper()
	platformPolicyMu.Lock()
	prev := platformPolicyCache
	platformPolicyCache = &p
	platformPolicyFetchedAt = time.Now()
	platformPolicyMu.Unlock()
	t.Cleanup(func() {
		platformPolicyMu.Lock()
		platformPolicyCache = prev
		platformPolicyMu.Unlock()
	})
}

func TestResolveMealPlanRefundTier_DefaultBoundaries(t *testing.T) {
	withTiers(t, DefaultMealPlanRefundTiers())

	cases := []struct {
		name        string
		leadHours   float64
		wantFloor   int
		wantAuto    bool
		description string
	}{
		{"well clear of prep", 48, 100, true, "nothing has been bought yet"},
		{"just over 12h", 12.01, 100, true, "still the auto band"},
		{"exactly 12h", 12, 100, true, "the boundary belongs to the higher band"},
		{"just under 12h", 11.99, 75, false, "prep may have started — chef decides, floor 75%"},
		{"7h", 7, 75, false, ""},
		{"exactly 6h", 6, 75, false, "boundary belongs to the higher band"},
		{"just under 6h", 5.99, 50, false, ""},
		{"exactly 2h", 2, 50, false, "boundary belongs to the higher band"},
		{"just under 2h", 1.99, 0, false, "chef may grant, but owes nothing"},
		{"at cook-start", 0, 0, false, ""},
		{"after cook-start", -3, 0, false, "a cancellation past the start still resolves"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ResolveMealPlanRefundTier(tc.leadHours)
			require.Equal(t, tc.wantFloor, got.FloorPercent, tc.description)
			require.Equal(t, tc.wantAuto, got.AutoApprove, tc.description)
		})
	}
}

// The policy is CONFIGURATION: switching the >12h band to a chef-decided 75% floor — the
// alternative model the issue describes — must need no code change.
func TestResolveMealPlanRefundTier_ConfiguredAlternativePolicy(t *testing.T) {
	withTiers(t, []MealPlanRefundTier{
		{MinLeadHours: 12, FloorPercent: 75},
		{MinLeadHours: 6, FloorPercent: 50},
		{MinLeadHours: 2, FloorPercent: 0},
	})
	top := ResolveMealPlanRefundTier(48)
	require.Equal(t, 75, top.FloorPercent)
	require.False(t, top.AutoApprove, "the alternative policy routes even early cancels to the chef")

	// The catch-all is synthesised so a table that stops at 2h still prices a 1h cancellation.
	require.Equal(t, 0, ResolveMealPlanRefundTier(1).FloorPercent)
}

func TestNormalizeRefundTiers(t *testing.T) {
	t.Run("empty falls back to the default", func(t *testing.T) {
		require.Equal(t, DefaultMealPlanRefundTiers(), normalizeRefundTiers(nil))
	})

	t.Run("wholly invalid falls back to the default", func(t *testing.T) {
		require.Equal(t, DefaultMealPlanRefundTiers(), normalizeRefundTiers([]MealPlanRefundTier{
			{MinLeadHours: -1, FloorPercent: 50},
			{MinLeadHours: 4, FloorPercent: 140},
		}))
	})

	t.Run("sorts highest-lead-first regardless of input order", func(t *testing.T) {
		got := normalizeRefundTiers([]MealPlanRefundTier{
			{MinLeadHours: 2, FloorPercent: 50},
			{MinLeadHours: 12, FloorPercent: 100, AutoApprove: true},
			{MinLeadHours: 0, FloorPercent: 0},
		})
		require.Equal(t, []float64{12, 2, 0}, []float64{got[0].MinLeadHours, got[1].MinLeadHours, got[2].MinLeadHours})
	})

	t.Run("auto-approve below 100% is demoted to a chef decision", func(t *testing.T) {
		got := normalizeRefundTiers([]MealPlanRefundTier{{MinLeadHours: 12, FloorPercent: 80, AutoApprove: true}})
		require.False(t, got[0].AutoApprove,
			"auto-resolving below 100% would agree a partial refund with nobody having agreed to it")
	})

	t.Run("a table with no zero band gets a catch-all", func(t *testing.T) {
		got := normalizeRefundTiers([]MealPlanRefundTier{{MinLeadHours: 6, FloorPercent: 75}})
		require.Equal(t, 0.0, got[len(got)-1].MinLeadHours)
	})
}
