package payouts

// approve.go — the batch auto-approval decision.
//
// PrepareStatementBatch always creates the batch; this decides whether it may
// skip the human approval step and land pre-approved. Pure by construction,
// like the release governor: the same inputs re-derive the same verdict, so an
// audit row recording the reasons stays explainable months later.

// HoldReason is why a batch was queued for manual approval rather than
// auto-approved. Persisted in audit logs, so these string values are stable.
type HoldReason string

const (
	// HoldAutomationOff — auto-disburse is off globally or for this chef.
	HoldAutomationOff HoldReason = "automation_off"
	// HoldFirstDisbursement — no batch has ever been paid to this destination;
	// the first payout to a new bank account always gets human eyes.
	HoldFirstDisbursement HoldReason = "first_disbursement"
	// HoldAboveAutoCap — the amount exceeds the auto-approval cap.
	HoldAboveAutoCap HoldReason = "above_auto_cap"
	// HoldCapUnreadable — the cap setting exists but cannot be parsed. A typo
	// in money config must fail closed to manual review, never open.
	HoldCapUnreadable HoldReason = "cap_unreadable"
)

// AutoApproveInput is everything the decision may consider.
type AutoApproveInput struct {
	// AutomationEnabled is the resolved per-chef verdict (chef tri-state,
	// falling back to the global flag).
	AutomationEnabled bool

	// DestinationPaidBefore is whether any batch has reached paid against the
	// same payout method. False forces the first disbursement through review.
	DestinationPaidBefore bool

	// AmountMinor is this batch's value. AutoCapMinor is the ceiling above
	// which a human looks; zero disables the check. CapUnreadable marks a
	// present-but-unparseable cap setting.
	AmountMinor   int64
	AutoCapMinor  int64
	CapUnreadable bool
}

// AutoApproveDecision is the outcome for one batch.
type AutoApproveDecision struct {
	Approve bool
	Reasons []HoldReason
}

// DecideAutoApprove evaluates every guardrail without short-circuiting, so the
// audit trail shows the whole picture rather than the first objection.
func DecideAutoApprove(in AutoApproveInput) AutoApproveDecision {
	var reasons []HoldReason

	if !in.AutomationEnabled {
		reasons = append(reasons, HoldAutomationOff)
	}
	if !in.DestinationPaidBefore {
		reasons = append(reasons, HoldFirstDisbursement)
	}
	if in.CapUnreadable {
		reasons = append(reasons, HoldCapUnreadable)
	} else if in.AutoCapMinor > 0 && in.AmountMinor > in.AutoCapMinor {
		reasons = append(reasons, HoldAboveAutoCap)
	}

	return AutoApproveDecision{Approve: len(reasons) == 0, Reasons: reasons}
}
