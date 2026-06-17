package workflows

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
)

func TestDeliveryWorkflow_Success(t *testing.T) {
	orig := DispatchFunc
	t.Cleanup(func() { DispatchFunc = orig })
	calls := 0
	DispatchFunc = func(context.Context, uuid.UUID) error { calls++; return nil }

	var ts testsuite.WorkflowTestSuite
	env := ts.NewTestWorkflowEnvironment()
	env.RegisterActivity(DispatchDeliveryActivity)

	env.ExecuteWorkflow(DeliveryWorkflow, uuid.New())

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	require.Equal(t, 1, calls)
}

// A flaky provider booking is retried until it succeeds — the durability that
// the fire-and-forget dispatch goroutine cannot provide.
func TestDeliveryWorkflow_RetriesThenSucceeds(t *testing.T) {
	orig := DispatchFunc
	t.Cleanup(func() { DispatchFunc = orig })
	attempts := 0
	DispatchFunc = func(context.Context, uuid.UUID) error {
		attempts++
		if attempts < 3 {
			return errors.New("provider 503")
		}
		return nil
	}

	var ts testsuite.WorkflowTestSuite
	env := ts.NewTestWorkflowEnvironment()
	env.RegisterActivity(DispatchDeliveryActivity)

	env.ExecuteWorkflow(DeliveryWorkflow, uuid.New())

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	require.GreaterOrEqual(t, attempts, 3)
}
