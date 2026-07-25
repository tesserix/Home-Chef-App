package workflows

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
)

// resetProbes wires the transport seams to counters and returns them.
func resetProbes(env interface {
	RegisterActivity(any)
}) (notices, applied, cancels *int) {
	n, a, c := 0, 0, 0
	env.RegisterActivity(MFAResetNoticeActivity)
	env.RegisterActivity(MFAResetApplyActivity)
	env.RegisterActivity(MFAResetCancelledActivity)
	MFAResetNoticeFunc = func(_ context.Context, _ uuid.UUID, _ int) error { n++; return nil }
	MFAResetApplyFunc = func(_ context.Context, _ uuid.UUID) error { a++; return nil }
	MFAResetCancelledFunc = func(_ context.Context, _ uuid.UUID) error { c++; return nil }
	return &n, &a, &c
}

const testHoldSeconds = 24 * 60 * 60

func TestAdminMFAReset_NoResponse_AppliesAfterHold(t *testing.T) {
	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()
	notices, applied, cancels := resetProbes(env)

	env.ExecuteWorkflow(AdminMFAResetWorkflow, AdminMFAResetInput{
		UserID: uuid.New(), RequestedByAdminID: uuid.New(), HoldSeconds: testHoldSeconds,
	})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	// The user is warned before the hold, not after — otherwise the window is
	// useless to them.
	require.Equal(t, 1, *notices)
	require.Equal(t, 1, *applied)
	require.Equal(t, 0, *cancels)
}

func TestAdminMFAReset_UserCancels_NothingIsApplied(t *testing.T) {
	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()
	notices, applied, cancels := resetProbes(env)

	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(SignalMFAResetCancelled, nil)
	}, time.Hour)

	env.ExecuteWorkflow(AdminMFAResetWorkflow, AdminMFAResetInput{
		UserID: uuid.New(), RequestedByAdminID: uuid.New(), HoldSeconds: testHoldSeconds,
	})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	require.Equal(t, 1, *notices)
	require.Equal(t, 0, *applied, "a cancelled reset must never be applied")
	require.Equal(t, 1, *cancels)
}

func TestAdminMFAReset_SecondAdminExpedites(t *testing.T) {
	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()
	_, applied, cancels := resetProbes(env)

	requester := uuid.New()
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(SignalMFAResetExpedited, ExpediteSignal{ApprovedByAdminID: uuid.New()})
	}, time.Minute)

	env.ExecuteWorkflow(AdminMFAResetWorkflow, AdminMFAResetInput{
		UserID: uuid.New(), RequestedByAdminID: requester, HoldSeconds: testHoldSeconds,
	})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	require.Equal(t, 1, *applied)
	require.Equal(t, 0, *cancels)
}

// The two-person rule. If the requesting admin could expedite their own request,
// the hold would be decorative — a single compromised admin account could clear
// someone's second factor instantly, which is the exact attack this defends.
func TestAdminMFAReset_RequesterCannotExpediteOwnRequest(t *testing.T) {
	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()
	_, applied, _ := resetProbes(env)

	requester := uuid.New()
	// Same admin tries to self-approve one minute in.
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(SignalMFAResetExpedited, ExpediteSignal{ApprovedByAdminID: requester})
	}, time.Minute)
	// ...and the user cancels well before the hold would have elapsed. If the
	// self-expedite had been honoured, the reset would already be applied and
	// this cancel would arrive too late.
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(SignalMFAResetCancelled, nil)
	}, 2*time.Hour)

	env.ExecuteWorkflow(AdminMFAResetWorkflow, AdminMFAResetInput{
		UserID: uuid.New(), RequestedByAdminID: requester, HoldSeconds: testHoldSeconds,
	})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	require.Equal(t, 0, *applied, "self-approval must not satisfy the two-person rule")
}

// An empty approver must not count either — a signal sent with no payload would
// otherwise deserialize to the zero UUID and skip the hold.
func TestAdminMFAReset_EmptyApproverIsIgnored(t *testing.T) {
	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()
	_, applied, cancels := resetProbes(env)

	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(SignalMFAResetExpedited, ExpediteSignal{})
	}, time.Minute)
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(SignalMFAResetCancelled, nil)
	}, 2*time.Hour)

	env.ExecuteWorkflow(AdminMFAResetWorkflow, AdminMFAResetInput{
		UserID: uuid.New(), RequestedByAdminID: uuid.New(), HoldSeconds: testHoldSeconds,
	})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	require.Equal(t, 0, *applied)
	require.Equal(t, 1, *cancels)
}
