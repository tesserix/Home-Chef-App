package workflows

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
)

// success: the workflow runs the send activity exactly once and completes.
func TestNotificationWorkflow_Success(t *testing.T) {
	orig := SendFunc
	t.Cleanup(func() { SendFunc = orig })
	calls := 0
	SendFunc = func(context.Context, NotificationInput) error { calls++; return nil }

	var ts testsuite.WorkflowTestSuite
	env := ts.NewTestWorkflowEnvironment()
	env.RegisterActivity(SendNotificationActivity)

	env.ExecuteWorkflow(NotificationWorkflow, NotificationInput{
		Channel: "email", Recipient: "chef@example.com", Subject: "New order", Body: "x",
	})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	require.Equal(t, 1, calls)
}

// transient failures are retried, then the workflow succeeds — proving durability
// (a flaky send no longer loses the message the way a goroutine would).
func TestNotificationWorkflow_RetriesThenSucceeds(t *testing.T) {
	orig := SendFunc
	t.Cleanup(func() { SendFunc = orig })
	attempts := 0
	SendFunc = func(context.Context, NotificationInput) error {
		attempts++
		if attempts < 3 {
			return errors.New("transient transport error")
		}
		return nil
	}

	var ts testsuite.WorkflowTestSuite
	env := ts.NewTestWorkflowEnvironment()
	env.RegisterActivity(SendNotificationActivity)

	env.ExecuteWorkflow(NotificationWorkflow, NotificationInput{Channel: "push", Recipient: "device-token"})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	require.GreaterOrEqual(t, attempts, 3)
}
