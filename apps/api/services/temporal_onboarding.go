package services

import (
	"context"
	"log"

	"github.com/google/uuid"

	"github.com/homechef/api/config"
	"github.com/homechef/api/database"
	apitemporal "github.com/homechef/api/temporal"
	"github.com/homechef/api/temporal/workflows"
)

// temporal_onboarding.go — start of the durable chef-onboarding activation
// workflow (#126) + the activity implementation. Gated behind
// OnboardingWorkflowEnabled (default OFF): when off the approval handler runs the
// activation inline (unchanged); when on it hands activation to the durable
// workflow. The activation op is the same idempotent ActivateChefOnboarding in
// both paths, so there's one source of truth and no duplication.

// OnboardingWorkflowActive reports whether onboarding activation should run via
// the durable workflow (Temporal up + flag on).
func OnboardingWorkflowActive() bool {
	return temporalRT != nil && config.AppConfig != nil && config.AppConfig.OnboardingWorkflowEnabled
}

// StartOnboardingActivation durably activates an approved chef onboarding.
// Idempotent on the approval-keyed workflow id. No-op when the workflow is
// disabled — callers fall back to the inline activation.
func StartOnboardingActivation(approvalID uuid.UUID) {
	if !OnboardingWorkflowActive() {
		return
	}
	id := "homechef:onboarding:" + approvalID.String()
	if _, err := temporalRT.Start(context.Background(), apitemporal.TaskQueueOnboarding, id, workflows.OnboardingActivationWorkflow, workflows.OnboardingActivationInput{ApprovalID: approvalID}); err != nil {
		log.Printf("onboarding workflow: start failed for approval %s: %v", approvalID, err)
	}
}

// ActivateChefOnboardingFromActivity is the activity transport — wired onto
// workflows.ActivateChefFunc by the worker.
func ActivateChefOnboardingFromActivity(_ context.Context, approvalID uuid.UUID) error {
	return ActivateChefOnboarding(database.DB, approvalID)
}
