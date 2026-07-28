package workflows

// pickup_test.go — pins PickupReadyWorkflow: the immediate ready notice, the
// timer-driven reminder loop, the chef escalation when nobody collects, and
// early exit on a collected/cancelled signal.

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
)

// pickupSpy wires all three activity seams to counters and returns them.
func pickupSpy(env interface {
	RegisterActivity(any)
}) (notices, reminders, escalations *int) {
	notices, reminders, escalations = new(int), new(int), new(int)
	env.RegisterActivity(PickupReadyNoticeActivity)
	env.RegisterActivity(PickupReminderActivity)
	env.RegisterActivity(PickupUncollectedActivity)
	PickupReadyNoticeFunc = func(context.Context, uuid.UUID) error { *notices++; return nil }
	PickupReminderFunc = func(context.Context, uuid.UUID, int) error { *reminders++; return nil }
	PickupUncollectedFunc = func(context.Context, uuid.UUID) error { *escalations++; return nil }
	return
}

func TestPickupReadyWorkflow_NeverCollected_EscalatesToChef(t *testing.T) {
	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()
	notices, reminders, escalations := pickupSpy(env)

	env.ExecuteWorkflow(PickupReadyWorkflow, PickupReadyInput{
		OrderID: uuid.New(), ReminderIntervalSeconds: 1200, MaxReminders: 3,
	})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	require.Equal(t, 1, *notices, "the ready notice fires exactly once")
	require.Equal(t, 3, *reminders)
	// The chef is the terminal recipient: nobody can auto-complete an order that
	// nobody demonstrably collected, so a person has to decide.
	require.Equal(t, 1, *escalations)
}

func TestPickupReadyWorkflow_ReadyNoticeFiresBeforeAnyReminder(t *testing.T) {
	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()
	env.RegisterActivity(PickupReadyNoticeActivity)
	env.RegisterActivity(PickupReminderActivity)
	env.RegisterActivity(PickupUncollectedActivity)

	// The whole point of the flow is that the customer is told immediately —
	// making them wait out a 20-minute reminder interval to learn their food is
	// ready would be worse than the generic push it replaces.
	var order []string
	PickupReadyNoticeFunc = func(context.Context, uuid.UUID) error {
		order = append(order, "notice")
		return nil
	}
	PickupReminderFunc = func(context.Context, uuid.UUID, int) error {
		order = append(order, "reminder")
		return nil
	}
	PickupUncollectedFunc = func(context.Context, uuid.UUID) error {
		order = append(order, "escalation")
		return nil
	}

	env.ExecuteWorkflow(PickupReadyWorkflow, PickupReadyInput{
		OrderID: uuid.New(), ReminderIntervalSeconds: 1200, MaxReminders: 1,
	})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	require.Equal(t, []string{"notice", "reminder", "escalation"}, order)
}

func TestPickupReadyWorkflow_CollectedSignal_StopsEarly(t *testing.T) {
	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()
	notices, reminders, escalations := pickupSpy(env)

	// Collected ~25 min in: after the first reminder (+20m), before the second.
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(SignalOrderCollected, nil)
	}, 25*time.Minute)

	env.ExecuteWorkflow(PickupReadyWorkflow, PickupReadyInput{
		OrderID: uuid.New(), ReminderIntervalSeconds: 1200, MaxReminders: 3,
	})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	require.Equal(t, 1, *notices)
	require.Equal(t, 1, *reminders, "no further nagging once the food is collected")
	require.Equal(t, 0, *escalations, "the chef is never told about a collected order")
}

func TestPickupReadyWorkflow_CancelledSignal_StopsEarly(t *testing.T) {
	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()
	notices, reminders, escalations := pickupSpy(env)

	// Cancelled before the first reminder interval elapses.
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(SignalPickupCancelled, nil)
	}, 5*time.Minute)

	env.ExecuteWorkflow(PickupReadyWorkflow, PickupReadyInput{
		OrderID: uuid.New(), ReminderIntervalSeconds: 1200, MaxReminders: 3,
	})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	require.Equal(t, 1, *notices)
	require.Equal(t, 0, *reminders, "a cancelled order is never chased")
	require.Equal(t, 0, *escalations)
}

func TestPickupReadyWorkflow_CollectedDuringGraceWindow_SkipsEscalation(t *testing.T) {
	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()
	notices, reminders, escalations := pickupSpy(env)

	// The final interval after the last reminder is a grace window — the customer
	// acting on that last nudge must not still trigger the chef escalation.
	// With 2 reminders at 20m intervals, the last fires at +40m and the escalation
	// would otherwise land at +60m.
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(SignalOrderCollected, nil)
	}, 50*time.Minute)

	env.ExecuteWorkflow(PickupReadyWorkflow, PickupReadyInput{
		OrderID: uuid.New(), ReminderIntervalSeconds: 1200, MaxReminders: 2,
	})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	require.Equal(t, 1, *notices)
	require.Equal(t, 2, *reminders)
	require.Equal(t, 0, *escalations)
}

func TestPickupReadyWorkflow_FailedNotice_DoesNotAbandonTheFlow(t *testing.T) {
	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()
	env.RegisterActivity(PickupReadyNoticeActivity)
	env.RegisterActivity(PickupReminderActivity)
	env.RegisterActivity(PickupUncollectedActivity)

	reminders, escalations := 0, 0
	// A dead push token or a NATS blip must not cost the customer their reminders
	// or the chef their escalation — the notice is best-effort by design.
	PickupReadyNoticeFunc = func(context.Context, uuid.UUID) error {
		return errors.New("push token expired")
	}
	PickupReminderFunc = func(context.Context, uuid.UUID, int) error { reminders++; return nil }
	PickupUncollectedFunc = func(context.Context, uuid.UUID) error { escalations++; return nil }

	env.ExecuteWorkflow(PickupReadyWorkflow, PickupReadyInput{
		OrderID: uuid.New(), ReminderIntervalSeconds: 1200, MaxReminders: 2,
	})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	require.Equal(t, 2, reminders)
	require.Equal(t, 1, escalations)
}

func TestPickupReadyWorkflow_ZeroReminders_StillAnnouncesAndEscalates(t *testing.T) {
	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()
	notices, reminders, escalations := pickupSpy(env)

	// A misconfigured MaxReminders of 0 must not skip the ready notice — that is
	// the one message the customer genuinely needs.
	env.ExecuteWorkflow(PickupReadyWorkflow, PickupReadyInput{
		OrderID: uuid.New(), ReminderIntervalSeconds: 1200, MaxReminders: 0,
	})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	require.Equal(t, 1, *notices)
	require.Equal(t, 0, *reminders)
	require.Equal(t, 1, *escalations)
}
