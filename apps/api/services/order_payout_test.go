package services

// order_payout_test.go — settle/claw-back gating (#123). Live transfer
// release/reversal is gated by ORDER_PAYOUT_AUTO_RELEASE_ENABLED; with it off
// (the default), both are complete no-ops that never touch the gateway or the DB
// — so the saga's settle + refund-reversal are safe before sandbox validation.

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/homechef/api/config"
)

func TestPayoutMovement_NoOpWhenDisabled(t *testing.T) {
	orig := config.AppConfig
	t.Cleanup(func() { config.AppConfig = orig })
	config.AppConfig = &config.Config{OrderPayoutAutoReleaseEnabled: false}

	// Both gate on the flag before any DB/gateway call, so they're safe no-ops
	// even with no DB configured.
	require.NoError(t, ReleaseOrderPayouts(uuid.New()))
	require.NoError(t, ReverseOrderPayouts(uuid.New()))
}
