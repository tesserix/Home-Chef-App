package workflows

// wallet_payment_test.go — pins the WalletPaymentWorkflow compensation logic: a mixed payment
// captures the hold when the external leg succeeds, releases it when the external leg fails or
// times out, and a wallet-only payment captures immediately.

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
)

// wireHoldCounters registers the hold activities and points the seams at counters.
func wireHoldCounters(env *testsuite.TestWorkflowEnvironment) (placed, captured, released *int) {
	p, c, r := new(int), new(int), new(int)
	env.RegisterActivity(PlaceWalletHoldActivity)
	env.RegisterActivity(CaptureWalletHoldActivity)
	env.RegisterActivity(ReleaseWalletHoldActivity)
	PlaceWalletHoldFunc = func(_ context.Context, _ WalletHoldActivityInput) error { *p++; return nil }
	CaptureWalletHoldFunc = func(_ context.Context, _ WalletHoldRefInput) error { *c++; return nil }
	ReleaseWalletHoldFunc = func(_ context.Context, _ WalletHoldRefInput) error { *r++; return nil }
	return p, c, r
}

// External leg succeeds → the wallet hold is captured, never released.
func TestWalletPaymentWorkflow_ExternalCaptured_CapturesHold(t *testing.T) {
	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()
	placed, captured, released := wireHoldCounters(env)

	env.RegisterDelayedCallback(func() { env.SignalWorkflow(SignalWalletPaymentCaptured, nil) }, 1*time.Minute)
	env.ExecuteWorkflow(WalletPaymentWorkflow, WalletPaymentInput{
		OrderID: uuid.New(), UserID: uuid.New(),
		WalletAmountMinor: 30000, ExternalAmountMinor: 55000, ExternalTimeoutSeconds: 900,
	})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	require.Equal(t, 1, *placed)
	require.Equal(t, 1, *captured)
	require.Equal(t, 0, *released)
}

// External leg fails → the wallet hold is released (compensation), never captured.
func TestWalletPaymentWorkflow_ExternalFailed_ReleasesHold(t *testing.T) {
	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()
	placed, captured, released := wireHoldCounters(env)

	env.RegisterDelayedCallback(func() { env.SignalWorkflow(SignalWalletPaymentFailed, nil) }, 1*time.Minute)
	env.ExecuteWorkflow(WalletPaymentWorkflow, WalletPaymentInput{
		OrderID: uuid.New(), UserID: uuid.New(),
		WalletAmountMinor: 30000, ExternalAmountMinor: 55000, ExternalTimeoutSeconds: 900,
	})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	require.Equal(t, 1, *placed)
	require.Equal(t, 0, *captured)
	require.Equal(t, 1, *released)
}

// No external signal within the timeout → the hold is released, so the customer's money is
// never stranded by a payment the gateway silently dropped.
func TestWalletPaymentWorkflow_ExternalTimeout_ReleasesHold(t *testing.T) {
	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()
	placed, captured, released := wireHoldCounters(env)

	env.ExecuteWorkflow(WalletPaymentWorkflow, WalletPaymentInput{
		OrderID: uuid.New(), UserID: uuid.New(),
		WalletAmountMinor: 30000, ExternalAmountMinor: 55000, ExternalTimeoutSeconds: 900,
	})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	require.Equal(t, 1, *placed)
	require.Equal(t, 0, *captured)
	require.Equal(t, 1, *released)
}

// Wallet-only (no external leg) captures immediately — the one-tap, no-gateway fast path.
func TestWalletPaymentWorkflow_WalletOnly_CapturesImmediately(t *testing.T) {
	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()
	placed, captured, released := wireHoldCounters(env)

	env.ExecuteWorkflow(WalletPaymentWorkflow, WalletPaymentInput{
		OrderID: uuid.New(), UserID: uuid.New(),
		WalletAmountMinor: 30000, ExternalAmountMinor: 0, ExternalTimeoutSeconds: 900,
	})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	require.Equal(t, 1, *placed)
	require.Equal(t, 1, *captured)
	require.Equal(t, 0, *released)
}
