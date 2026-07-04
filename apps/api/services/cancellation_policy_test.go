package services

// cancellation_policy_test.go — the auto-fast-path classifier + runtime tier
// config resolution (#476).

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/homechef/api/models"
)

func TestClassifyCancellation(t *testing.T) {
	cases := []struct {
		status models.OrderStatus
		want   CancellationPath
	}{
		{models.OrderStatusPending, CancelPathFullRefund},    // chef not engaged
		{models.OrderStatusRejected, CancelPathFullRefund},   // chef declined
		{models.OrderStatusAccepted, CancelPathVendorReview}, // vendor decides tier
		{models.OrderStatusPreparing, CancelPathVendorReview},
		{models.OrderStatusReady, CancelPathNotAllowed}, // made → not cancellable
		{models.OrderStatusDelivering, CancelPathNotAllowed},
		{models.OrderStatusDelivered, CancelPathNotAllowed},
	}
	for _, tc := range cases {
		t.Run(string(tc.status), func(t *testing.T) {
			path, msg := ClassifyCancellation(tc.status)
			require.Equal(t, tc.want, path)
			if tc.want == CancelPathNotAllowed {
				require.NotEmpty(t, msg)
			}
		})
	}
}

func setupCancelCfgDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TABLE platform_settings (id text PRIMARY KEY, key text, value text, type text, updated_by text, updated_at datetime)`).Error)
	return db
}

func TestResolveCancellationTiers_Defaults(t *testing.T) {
	db := setupCancelCfgDB(t)
	tiers, window := ResolveCancellationTiers(db)
	require.Equal(t, DefaultCancellationTiers(), tiers, "no rows → defaults")
	require.Equal(t, 15, window)
}

func TestResolveCancellationTiers_Overrides(t *testing.T) {
	db := setupCancelCfgDB(t)
	set := func(k, v string) {
		require.NoError(t, db.Exec(`INSERT INTO platform_settings (id, key, value) VALUES (?,?,?)`, uuid.NewString(), k, v).Error)
	}
	set("cancel.refund.not_started_pct", "80")
	set("cancel.refund.materials_pct", "50")
	set("cancel.refund.in_prep_pct", "10")
	set("cancel.vendor_response_minutes", "20")
	set("cancel.refund.not_started_pct_bad", "notanint") // ignored

	tiers, window := ResolveCancellationTiers(db)
	require.Equal(t, 80, tiers.NotStartedPct)
	require.Equal(t, 50, tiers.MaterialsPct)
	require.Equal(t, 10, tiers.InPrepPct)
	require.Equal(t, 0, tiers.ReadyPct, "unset key keeps its default")
	require.Equal(t, 20, window)
}
