package workflows

// admin_mfa_reset.go — the durable hold on an admin-initiated two-factor reset.
//
// Clearing someone's second factor is the most powerful thing support can do to
// an account: afterwards, the password alone is enough. So an admin request does
// not apply immediately. The user is told at once, a durable timer runs, and only
// then is the reset applied — unless the user says "this wasn't me".
//
// That window is the entire point. A compromised admin account cannot silently
// disarm a user's 2FA and walk in; the legitimate owner gets a chance to stop it.
//
// The escape hatch is a second admin: support who have verified someone in person
// should not have to make them wait a day, but one admin acting alone cannot skip
// the hold.

import (
	"context"
	"time"

	"github.com/google/uuid"
	"go.temporal.io/sdk/workflow"

	apitemporal "github.com/homechef/api/temporal"
)

// Signals the flow listens for.
const (
	// SignalMFAResetCancelled is the user pressing "this wasn't me".
	SignalMFAResetCancelled = "mfa.reset.cancelled"
	// SignalMFAResetExpedited carries a second admin's ID, applying the reset
	// immediately. One admin cannot expedite their own request.
	SignalMFAResetExpedited = "mfa.reset.expedited"
)

// AdminMFAResetInput starts the hold for one user.
type AdminMFAResetInput struct {
	UserID uuid.UUID
	// RequestedByAdminID is recorded so the expedite signal can be checked
	// against it — a second admin must be a *different* admin.
	RequestedByAdminID uuid.UUID
	// HoldSeconds is read from settings at start time and passed in, so a
	// running workflow stays deterministic across a config change.
	HoldSeconds int
}

// ExpediteSignal is the payload of SignalMFAResetExpedited.
type ExpediteSignal struct {
	ApprovedByAdminID uuid.UUID
}

// MFAResetNoticeInput is the notice activity's argument.
type MFAResetNoticeInput struct {
	UserID      uuid.UUID
	HoldSeconds int
}

// Transport seams — the worker wires these to services.*. Nil in unit tests
// unless the test overrides them.
var (
	// MFAResetNoticeFunc warns the user that a reset was requested and how to
	// stop it. Sent BEFORE the hold, which is what makes the hold useful.
	MFAResetNoticeFunc func(ctx context.Context, userID uuid.UUID, holdSeconds int) error
	// MFAResetApplyFunc clears the user's second factor.
	MFAResetApplyFunc func(ctx context.Context, userID uuid.UUID) error
	// MFAResetCancelledFunc confirms to the user that the reset was stopped.
	MFAResetCancelledFunc func(ctx context.Context, userID uuid.UUID) error
)

// MFAResetNoticeActivity tells the user a reset is pending.
func MFAResetNoticeActivity(ctx context.Context, in MFAResetNoticeInput) error {
	if MFAResetNoticeFunc == nil {
		return nil
	}
	return MFAResetNoticeFunc(ctx, in.UserID, in.HoldSeconds)
}

// MFAResetApplyActivity clears the second factor.
func MFAResetApplyActivity(ctx context.Context, userID uuid.UUID) error {
	if MFAResetApplyFunc == nil {
		return nil
	}
	return MFAResetApplyFunc(ctx, userID)
}

// MFAResetCancelledActivity confirms the reset was stopped.
func MFAResetCancelledActivity(ctx context.Context, userID uuid.UUID) error {
	if MFAResetCancelledFunc == nil {
		return nil
	}
	return MFAResetCancelledFunc(ctx, userID)
}

// AdminMFAResetWorkflow notifies the user, holds for HoldSeconds, then applies
// the reset — unless cancelled by the user or expedited by a second admin.
func AdminMFAResetWorkflow(ctx workflow.Context, in AdminMFAResetInput) error {
	actx := apitemporal.Activities(ctx, 30*time.Second)

	// Warn first. A hold the user is never told about protects nobody, so this
	// runs before the timer rather than alongside it.
	_ = workflow.ExecuteActivity(actx, MFAResetNoticeActivity, MFAResetNoticeInput{
		UserID: in.UserID, HoldSeconds: in.HoldSeconds,
	}).Get(ctx, nil)

	cancelCh := workflow.GetSignalChannel(ctx, SignalMFAResetCancelled)
	expediteCh := workflow.GetSignalChannel(ctx, SignalMFAResetExpedited)

	cancelled := false
	expedited := false

	// Loop rather than a single select: an expedite from the REQUESTING admin
	// must be ignored, and ignoring it has to leave the hold still running
	// rather than falling through to apply.
	for !cancelled && !expedited {
		timerFired := false
		sel := workflow.NewSelector(ctx)
		sel.AddReceive(cancelCh, func(c workflow.ReceiveChannel, _ bool) {
			c.Receive(ctx, nil)
			cancelled = true
		})
		sel.AddReceive(expediteCh, func(c workflow.ReceiveChannel, _ bool) {
			var sig ExpediteSignal
			c.Receive(ctx, &sig)
			// Two-person rule. A single admin cannot approve their own request,
			// which is exactly the case the hold exists to defend against.
			if sig.ApprovedByAdminID != uuid.Nil && sig.ApprovedByAdminID != in.RequestedByAdminID {
				expedited = true
			}
		})
		sel.AddFuture(workflow.NewTimer(ctx, time.Duration(in.HoldSeconds)*time.Second), func(workflow.Future) {
			timerFired = true
		})
		sel.Select(ctx)

		if timerFired {
			break
		}
	}

	if cancelled {
		return workflow.ExecuteActivity(actx, MFAResetCancelledActivity, in.UserID).Get(ctx, nil)
	}
	return workflow.ExecuteActivity(actx, MFAResetApplyActivity, in.UserID).Get(ctx, nil)
}
