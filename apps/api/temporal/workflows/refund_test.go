package workflows

// refund_test.go — pins the durable deferred chef-cancel gateway-refund
// retry flow: the workflow calls the gateway activity then persists the
// resulting id, and — the whole point of this workflow existing — a
// transient gateway failure is retried until it eventually succeeds, with
// persist called exactly once carrying the FINAL (successful) refund id.

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
)

// withRefundFlowStubs swaps the activity transports for counting stubs and
// restores them after the test.
func withRefundFlowStubs(t *testing.T) {
	t.Helper()
	gf, pf := GatewayRefundFunc, PersistRefundIDFunc
	t.Cleanup(func() { GatewayRefundFunc, PersistRefundIDFunc = gf, pf })
}

func newRefundFlowEnv() *testsuite.TestWorkflowEnvironment {
	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()
	env.RegisterActivity(GatewayRefundActivity)
	env.RegisterActivity(PersistRefundIDActivity)
	return env
}

// TestDeferredRefundWorkflow_HappyPath — the gateway succeeds on the first
// attempt: it's called once, and the resulting refund id is persisted once.
func TestDeferredRefundWorkflow_HappyPath(t *testing.T) {
	withRefundFlowStubs(t)
	env := newRefundFlowEnv()

	gatewayCalls := 0
	var persistedID string
	persistCalls := 0
	GatewayRefundFunc = func(_ context.Context, _ uuid.UUID, paymentID string, amountPaise int) (string, error) {
		gatewayCalls++
		require.Equal(t, "pay_abc123", paymentID)
		require.Equal(t, 50000, amountPaise)
		return "rfnd_immediate", nil
	}
	PersistRefundIDFunc = func(_ context.Context, _ uuid.UUID, refundID string) error {
		persistCalls++
		persistedID = refundID
		return nil
	}

	orderID := uuid.New()
	env.ExecuteWorkflow(DeferredRefundWorkflow, DeferredRefundInput{
		OrderID: orderID, PaymentID: "pay_abc123", AmountPaise: 50000,
	})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	require.Equal(t, 1, gatewayCalls, "the gateway activity must run exactly once when it succeeds immediately")
	require.Equal(t, 1, persistCalls, "the persist activity must run exactly once")
	require.Equal(t, "rfnd_immediate", persistedID)
}

// TestDeferredRefundWorkflow_RetriesUntilSuccess — the whole reason this
// workflow exists: a gateway that fails transiently must be retried by
// Temporal's backoff (not given up on), and once it finally succeeds, the
// FINAL refund id — not any of the failed attempts' (nonexistent) ids — is
// what gets persisted, exactly once.
func TestDeferredRefundWorkflow_RetriesUntilSuccess(t *testing.T) {
	withRefundFlowStubs(t)
	env := newRefundFlowEnv()

	gatewayCalls := 0
	const failuresBeforeSuccess = 3
	var persistedID string
	persistCalls := 0
	GatewayRefundFunc = func(_ context.Context, _ uuid.UUID, _ string, _ int) (string, error) {
		gatewayCalls++
		if gatewayCalls <= failuresBeforeSuccess {
			return "", errors.New("gateway temporarily unavailable")
		}
		return "rfnd_after_retries", nil
	}
	PersistRefundIDFunc = func(_ context.Context, _ uuid.UUID, refundID string) error {
		persistCalls++
		persistedID = refundID
		return nil
	}

	env.ExecuteWorkflow(DeferredRefundWorkflow, DeferredRefundInput{
		OrderID: uuid.New(), PaymentID: "pay_xyz", AmountPaise: 12345,
	})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	require.Equal(t, failuresBeforeSuccess+1, gatewayCalls, "the activity must be retried until it succeeds")
	require.Equal(t, 1, persistCalls, "persist must run exactly once, after the gateway finally succeeds")
	require.Equal(t, "rfnd_after_retries", persistedID, "only the FINAL successful refund id is persisted")
}

// TestDeferredRefundWorkflow_NothingToRefund — GatewayRefundFunc returning ""
// (no error) means there was nothing to refund; the workflow must complete
// cleanly without ever calling persist.
func TestDeferredRefundWorkflow_NothingToRefund(t *testing.T) {
	withRefundFlowStubs(t)
	env := newRefundFlowEnv()

	persistCalls := 0
	GatewayRefundFunc = func(_ context.Context, _ uuid.UUID, _ string, _ int) (string, error) { return "", nil }
	PersistRefundIDFunc = func(_ context.Context, _ uuid.UUID, _ string) error { persistCalls++; return nil }

	env.ExecuteWorkflow(DeferredRefundWorkflow, DeferredRefundInput{
		OrderID: uuid.New(), PaymentID: "pay_none", AmountPaise: 0,
	})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	require.Equal(t, 0, persistCalls, "nothing to persist when the gateway activity reports no refund id")
}
