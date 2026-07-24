package workflows

// confirm_receipt_test.go — pins the ConfirmReceiptWorkflow reminder loop:
// up to MaxReminders timer-driven reminders, auto-confirm on exhaustion, and
// early exit on an order.confirmed/order.disputed signal.

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
)

func TestConfirmReceiptWorkflow_NoAction_AutoConfirms(t *testing.T) {
	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()

	reminders := 0
	autoConfirmed := 0
	env.RegisterActivity(ReminderActivity)
	env.RegisterActivity(AutoConfirmActivity)
	ConfirmReminderFunc = func(_ context.Context, _ uuid.UUID, _ int) error { reminders++; return nil }
	AutoConfirmFunc = func(_ context.Context, _ uuid.UUID) error { autoConfirmed++; return nil }

	env.ExecuteWorkflow(ConfirmReceiptWorkflow, ConfirmReceiptInput{
		OrderID: uuid.New(), ReminderIntervalSeconds: 600, MaxReminders: 3,
	})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	require.Equal(t, 3, reminders)
	require.Equal(t, 1, autoConfirmed)
}

func TestConfirmReceiptWorkflow_ConfirmedSignal_StopsEarly(t *testing.T) {
	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()

	reminders := 0
	autoConfirmed := 0
	env.RegisterActivity(ReminderActivity)
	env.RegisterActivity(AutoConfirmActivity)
	ConfirmReminderFunc = func(_ context.Context, _ uuid.UUID, _ int) error { reminders++; return nil }
	AutoConfirmFunc = func(_ context.Context, _ uuid.UUID) error { autoConfirmed++; return nil }

	// Fire the confirmed signal ~15 min in (after the first reminder at +10m).
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(SignalOrderConfirmed, nil)
	}, 15*time.Minute)

	env.ExecuteWorkflow(ConfirmReceiptWorkflow, ConfirmReceiptInput{
		OrderID: uuid.New(), ReminderIntervalSeconds: 600, MaxReminders: 3,
	})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	require.Equal(t, 1, reminders)
	require.Equal(t, 0, autoConfirmed)
}
