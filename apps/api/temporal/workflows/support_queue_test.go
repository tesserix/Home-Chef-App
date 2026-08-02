package workflows

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
)

// captureNotices swaps the notify transport for a recorder.
func captureNotices(t *testing.T) *[]SupportQueueNotice {
	t.Helper()
	orig := SupportQueueNotifyFunc
	t.Cleanup(func() { SupportQueueNotifyFunc = orig })
	var mu sync.Mutex
	notices := &[]SupportQueueNotice{}
	SupportQueueNotifyFunc = func(_ context.Context, n SupportQueueNotice) error {
		mu.Lock()
		defer mu.Unlock()
		*notices = append(*notices, n)
		return nil
	}
	return notices
}

func newSupportQueueEnv() (*testsuite.TestWorkflowEnvironment, SupportQueueInput) {
	var ts testsuite.WorkflowTestSuite
	env := ts.NewTestWorkflowEnvironment()
	env.RegisterActivity(SupportQueueNotifyActivity)
	return env, SupportQueueInput{
		ConversationID: "conv-1",
		TenantID:       "homechef",
		CaseID:         "CS-260802-0001",
		CustomerName:   "Asha",
		IntakeReason:   "refund",
	}
}

// accepted quickly: exactly one "waiting" notice, no reminders.
func TestSupportQueueWorkflow_AcceptedBeforeReminder(t *testing.T) {
	notices := captureNotices(t)
	env, in := newSupportQueueEnv()

	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(SupportQueueSignalName, SupportQueueSignal{Event: "accepted"})
	}, time.Minute)

	env.ExecuteWorkflow(SupportQueueWorkflow, in)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	require.Len(t, *notices, 1)
	require.Equal(t, SupportNoticeWaiting, (*notices)[0].Kind)
}

// nobody accepts: waiting → reminder (3m) → unattended (10m), and at the
// timeout the chat is converted into a ticket so the customer is not left
// waiting on a human who never arrives.
func TestSupportQueueWorkflow_TimesOutIntoATicket(t *testing.T) {
	notices := captureNotices(t)
	origTicket := SupportQueueTicketFunc
	t.Cleanup(func() { SupportQueueTicketFunc = origTicket })
	var raised []SupportQueueTicketInput
	SupportQueueTicketFunc = func(_ context.Context, in SupportQueueTicketInput) (SupportQueueTicketResult, error) {
		raised = append(raised, in)
		return SupportQueueTicketResult{TicketNumber: "TKT-TEST-0001", Created: true}, nil
	}

	env, in := newSupportQueueEnv()
	in.Escalated = true
	in.CustomerEmail = "chef@example.com"
	env.RegisterActivity(SupportQueueTicketActivity)

	env.ExecuteWorkflow(SupportQueueWorkflow, in)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	kinds := make([]string, 0, len(*notices))
	for _, n := range *notices {
		kinds = append(kinds, n.Kind)
	}
	require.Equal(t, []string{SupportNoticeEscalated, SupportNoticeReminder, SupportNoticeUnattended}, kinds)

	require.Len(t, raised, 1, "exactly one ticket should be raised at the timeout")
	require.Equal(t, in.ConversationID, raised[0].ConversationID)
	require.Equal(t, "chef@example.com", raised[0].CustomerEmail)
	require.GreaterOrEqual(t, raised[0].WaitedSeconds, int(supportQueueTicketTimeout.Seconds()))
}

// accepted inside the window: no ticket is raised at all.
func TestSupportQueueWorkflow_AcceptedRaisesNoTicket(t *testing.T) {
	captureNotices(t)
	origTicket := SupportQueueTicketFunc
	t.Cleanup(func() { SupportQueueTicketFunc = origTicket })
	tickets := 0
	SupportQueueTicketFunc = func(context.Context, SupportQueueTicketInput) (SupportQueueTicketResult, error) {
		tickets++
		return SupportQueueTicketResult{}, nil
	}

	env, in := newSupportQueueEnv()
	env.RegisterActivity(SupportQueueTicketActivity)
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(SupportQueueSignalName, SupportQueueSignal{Event: "accepted"})
	}, 5*time.Minute)

	env.ExecuteWorkflow(SupportQueueWorkflow, in)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	require.Zero(t, tickets, "a chat answered in time must not raise a ticket")
}

// an AI handoff mid-thread re-notifies without resetting the SLA clock,
// and a close stops everything.
func TestSupportQueueWorkflow_EscalatedSignalThenClosed(t *testing.T) {
	notices := captureNotices(t)
	env, in := newSupportQueueEnv()

	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(SupportQueueSignalName, SupportQueueSignal{Event: "escalated"})
	}, time.Minute)
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(SupportQueueSignalName, SupportQueueSignal{Event: "closed"})
	}, 2*time.Minute)

	env.ExecuteWorkflow(SupportQueueWorkflow, in)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	kinds := make([]string, 0, len(*notices))
	for _, n := range *notices {
		kinds = append(kinds, n.Kind)
	}
	require.Equal(t, []string{SupportNoticeWaiting, SupportNoticeEscalated}, kinds)
}
