package workflows

import (
	"context"
	"time"

	"github.com/google/uuid"
	apitemporal "github.com/homechef/api/temporal"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/workflow"
)

// onboarding.go — durable chef-onboarding activation (epic #116, sub-issue #126).
//
// When an admin approves a kitchen onboarding, the activation (verify + activate
// the chef profile, promote the user to the chef role) was a sequence of
// non-transactional, error-ignored writes in the HTTP handler: a crash mid-way
// left the chef approved-but-not-activated with no retry. This workflow runs the
// activation as a single retried, idempotent activity so it always completes,
// and is observable in the Temporal UI. The workflow ID is keyed on the approval
// (homechef:onboarding:<approvalID>) so it never runs twice.

// OnboardingActivationInput starts the activation for one approved onboarding.
type OnboardingActivationInput struct {
	ApprovalID uuid.UUID `json:"approvalId"`
}

// ActivateChefFunc is the pluggable activation call; the worker wires it to
// services.ActivateChefOnboarding. Default no-op so the workflow is runnable
// before integration. Mirrors the DispatchFunc pattern.
var ActivateChefFunc = func(_ context.Context, _ uuid.UUID) error { return nil }

// ActivateChefOnboardingActivity applies the chef-onboarding activation. Retried
// by the workflow's retry policy; the underlying op is idempotent.
func ActivateChefOnboardingActivity(ctx context.Context, approvalID uuid.UUID) error {
	activity.GetLogger(ctx).Info("activate chef onboarding", "approval", approvalID.String())
	return ActivateChefFunc(ctx, approvalID)
}

// OnboardingActivationWorkflow durably activates an approved chef onboarding.
func OnboardingActivationWorkflow(ctx workflow.Context, in OnboardingActivationInput) error {
	ctx = apitemporal.Activities(ctx, 2*time.Minute)
	return workflow.ExecuteActivity(ctx, ActivateChefOnboardingActivity, in.ApprovalID).Get(ctx, nil)
}
