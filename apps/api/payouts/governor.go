package payouts

import "time"

// governor.go — the per-order release decision (#741).
//
// Route creates each order's payee transfers held at checkout, so a payout is
// not a batch to build but a hold to clear. This decides whether one order's
// transfers may be released.
//
// Pure by construction: no database, no gateway, no clock of its own. A block
// recorded against an order can therefore be re-derived from the same inputs
// months later, which is what makes the admin queue auditable.

// BlockReason is why a release was withheld. Persisted and grouped in
// analytics, so these string values are stable API.
type BlockReason string

const (
	// BlockNotMatured — still inside the maturation window.
	BlockNotMatured BlockReason = "not_matured"
	// BlockAutomationOff — automation is off globally or for this chef.
	BlockAutomationOff BlockReason = "automation_off"
	// BlockSettlementNotActivated — the chef's linked account has no working
	// settlement destination, so a release would strand the money.
	BlockSettlementNotActivated BlockReason = "settlement_not_activated"
	// BlockRefundOpen — a refund, chargeback or cancellation is in flight.
	BlockRefundOpen BlockReason = "refund_open"
	// BlockRecoveryBalance — the chef owes the platform.
	BlockRecoveryBalance BlockReason = "recovery_balance"
	// BlockNewChefRamp — inside the new-chef review period.
	BlockNewChefRamp BlockReason = "new_chef_ramp"
	// BlockAboveReviewThreshold — order value warrants human eyes.
	BlockAboveReviewThreshold BlockReason = "above_review_threshold"
)

// Overridable reports whether an admin may release despite this reason.
//
// The two hard blocks are not judgement calls: releasing to an unactivated
// account strands the money in a balance with no destination, and releasing
// against an open refund can settle to the chef's bank before the claw-back
// lands.
func (r BlockReason) Overridable() bool {
	switch r {
	case BlockSettlementNotActivated, BlockRefundOpen:
		return false
	default:
		return true
	}
}

// ReleaseInput is everything the decision may consider.
type ReleaseInput struct {
	Now         time.Time
	DeliveredAt time.Time
	Maturation  time.Duration

	AutomationEnabled   bool
	SettlementActivated bool
	RefundOpen          bool

	// RecoveryBalance is what the chef owes the platform, zero when clear.
	RecoveryBalance Money

	// DeliveredOrderCount is the chef's lifetime delivered orders; RampOrders
	// is the review period, and zero disables the check.
	DeliveredOrderCount int
	RampOrders          int

	// OrderTotal is this order's value; ReviewAbove is the threshold above
	// which a human looks, and zero disables the check.
	OrderTotal  Money
	ReviewAbove Money
}

// ReleaseDecision is the outcome for one order.
type ReleaseDecision struct {
	Release bool
	Reasons []BlockReason
}

// Blocked reports whether anything held this order back.
func (d ReleaseDecision) Blocked() bool { return len(d.Reasons) > 0 }

// DecideRelease evaluates every guardrail without short-circuiting, so an
// admin sees the whole picture rather than fixing one problem and immediately
// hitting the next.
func DecideRelease(in ReleaseInput) ReleaseDecision {
	var reasons []BlockReason

	if in.Now.Before(in.DeliveredAt.Add(in.Maturation)) {
		reasons = append(reasons, BlockNotMatured)
	}
	if !in.AutomationEnabled {
		reasons = append(reasons, BlockAutomationOff)
	}
	if !in.SettlementActivated {
		reasons = append(reasons, BlockSettlementNotActivated)
	}
	if in.RefundOpen {
		reasons = append(reasons, BlockRefundOpen)
	}
	if in.RecoveryBalance.IsPositive() {
		reasons = append(reasons, BlockRecoveryBalance)
	}
	if in.RampOrders > 0 && in.DeliveredOrderCount < in.RampOrders {
		reasons = append(reasons, BlockNewChefRamp)
	}
	if !in.ReviewAbove.IsZero() {
		if cmp, err := in.OrderTotal.Cmp(in.ReviewAbove); err == nil && cmp > 0 {
			reasons = append(reasons, BlockAboveReviewThreshold)
		}
	}

	return ReleaseDecision{Release: len(reasons) == 0, Reasons: reasons}
}
