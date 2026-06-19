package workflows

// onboarding_test.go — the durable chef-onboarding activation (#126). Pins that
// the workflow runs the activation activity, and that a transient failure is
// retried by the default policy (the whole point — a crash mid-activation must
// not leave a chef approved-but-inactive).

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
)

func TestOnboardingActivationWorkflow_Success(t *testing.T) {
	orig := ActivateChefFunc
	t.Cleanup(func() { ActivateChefFunc = orig })
	calls := 0
	ActivateChefFunc = func(context.Context, uuid.UUID) error { calls++; return nil }

	var ts testsuite.WorkflowTestSuite
	env := ts.NewTestWorkflowEnvironment()
	env.RegisterActivity(ActivateChefOnboardingActivity)
	env.ExecuteWorkflow(OnboardingActivationWorkflow, OnboardingActivationInput{ApprovalID: uuid.New()})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	require.Equal(t, 1, calls)
}

func TestOnboardingActivationWorkflow_RetriesUntilSuccess(t *testing.T) {
	orig := ActivateChefFunc
	t.Cleanup(func() { ActivateChefFunc = orig })
	attempts := 0
	ActivateChefFunc = func(context.Context, uuid.UUID) error {
		attempts++
		if attempts < 3 {
			return errors.New("transient db error")
		}
		return nil
	}

	var ts testsuite.WorkflowTestSuite
	env := ts.NewTestWorkflowEnvironment()
	env.RegisterActivity(ActivateChefOnboardingActivity)
	env.ExecuteWorkflow(OnboardingActivationWorkflow, OnboardingActivationInput{ApprovalID: uuid.New()})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	require.GreaterOrEqual(t, attempts, 3)
}
