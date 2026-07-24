package services

import (
	"testing"

	"github.com/google/uuid"
)

// TestStartConfirmReceiptFlow_NoOpWhenDisabled verifies the producer is a safe
// no-op in the default unit-test state: temporalRT is nil (never set via
// SetTemporalRuntime) and config.AppConfig.ConfirmReceiptFlowEnabled defaults
// false, so confirmFlowActive() is false and none of these calls should touch
// Temporal or panic.
func TestStartConfirmReceiptFlow_NoOpWhenDisabled(t *testing.T) {
	StartConfirmReceiptFlow(uuid.New())
	SignalOrderConfirmedFlow(uuid.New())
	SignalOrderDisputedFlow(uuid.New())
}
