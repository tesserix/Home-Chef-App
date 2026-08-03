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

// #956: cancelling an order left the flow running to MaxReminders and then
// calling AutoConfirmActivity. It must end early, and — unlike the exhaustion
// path — must NOT auto-confirm: a cancelled order is the one thing that must
// never be confirmed on the customer's behalf.
func TestConfirmReceiptWorkflow_CancelledSignal_StopsWithoutAutoConfirming(t *testing.T) {
	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()

	reminders := 0
	autoConfirmed := 0
	env.RegisterActivity(ReminderActivity)
	env.RegisterActivity(AutoConfirmActivity)
	ConfirmReminderFunc = func(_ context.Context, _ uuid.UUID, _ int) error { reminders++; return nil }
	AutoConfirmFunc = func(_ context.Context, _ uuid.UUID) error { autoConfirmed++; return nil }

	// Cancelled ~15 min in, after the first reminder at +10m.
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(SignalConfirmCancelled, nil)
	}, 15*time.Minute)

	env.ExecuteWorkflow(ConfirmReceiptWorkflow, ConfirmReceiptInput{
		OrderID: uuid.New(), ReminderIntervalSeconds: 600, MaxReminders: 3,
	})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	require.Equal(t, 1, reminders, "the loop stops at the cancellation instead of running to MaxReminders")
	require.Equal(t, 0, autoConfirmed, "a cancelled order must never be auto-confirmed")
}

// A cancellation arriving after the LAST reminder is not seen by the selector —
// the loop has already exited. The post-loop drain is what stops it falling
// through to auto-confirm.
func TestConfirmReceiptWorkflow_CancelledAfterFinalReminder_DoesNotAutoConfirm(t *testing.T) {
	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()

	reminders := 0
	autoConfirmed := 0
	env.RegisterActivity(ReminderActivity)
	env.RegisterActivity(AutoConfirmActivity)
	ConfirmReminderFunc = func(_ context.Context, _ uuid.UUID, _ int) error { reminders++; return nil }
	AutoConfirmFunc = func(_ context.Context, _ uuid.UUID) error { autoConfirmed++; return nil }

	// Reminders land at +10m, +20m, +30m; auto-confirm follows the third. Signal
	// just past the last timer so the selector has already gone.
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(SignalConfirmCancelled, nil)
	}, 30*time.Minute)

	env.ExecuteWorkflow(ConfirmReceiptWorkflow, ConfirmReceiptInput{
		OrderID: uuid.New(), ReminderIntervalSeconds: 600, MaxReminders: 3,
	})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	require.Equal(t, 3, reminders, "all three reminders ran, so the loop had exited before the signal")
	require.Equal(t, 0, autoConfirmed, "the post-loop drain catches a late cancellation")
}
