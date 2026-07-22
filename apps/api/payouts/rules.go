package payouts

import (
	"context"
	"sort"
	"time"

	"github.com/google/uuid"
)

// Decision is a rule's verdict. The ordering of the constants is the severity
// ordering used when combining results — a chain returns its most severe.
type Decision int

const (
	// DecisionAllow — this rule sees no reason not to pay.
	DecisionAllow Decision = iota
	// DecisionQueue — do not pay automatically; a human must look.
	DecisionQueue
	// DecisionBlock — do not pay at all until the underlying condition changes.
	DecisionBlock
)

func (d Decision) String() string {
	switch d {
	case DecisionAllow:
		return "allow"
	case DecisionQueue:
		return "queue"
	case DecisionBlock:
		return "block"
	default:
		return "unknown"
	}
}

// RuleResult is one rule's verdict with a stable, machine-readable reason.
//
// ReasonCode is part of the engine's contract: it is persisted on the batch,
// shown in the admin queue, and grouped in the failure analytics of #747.
// Codes must stay stable across releases even when wording changes — Detail
// carries the human wording.
type RuleResult struct {
	RuleID     string   `json:"ruleId"`
	Decision   Decision `json:"decision"`
	ReasonCode string   `json:"reasonCode,omitempty"`
	Detail     string   `json:"detail,omitempty"`
}

// Allow is the conventional pass result for a rule.
func Allow(ruleID string) RuleResult {
	return RuleResult{RuleID: ruleID, Decision: DecisionAllow}
}

// Queue sends the subject to manual review.
func Queue(ruleID, reasonCode, detail string) RuleResult {
	return RuleResult{RuleID: ruleID, Decision: DecisionQueue, ReasonCode: reasonCode, Detail: detail}
}

// Block refuses payment outright.
func Block(ruleID, reasonCode, detail string) RuleResult {
	return RuleResult{RuleID: ruleID, Decision: DecisionBlock, ReasonCode: reasonCode, Detail: detail}
}

// EvalContext is everything a rule may consider. It is deliberately a value
// type of engine-owned data: a rule cannot reach into the database, so rule
// evaluation is pure, deterministic and table-testable.
type EvalContext struct {
	// Now is the evaluation clock. Never call time.Now() inside a rule —
	// guardrails are re-evaluated at build and again at execution, and both
	// passes must be reproducible from a recorded context.
	Now time.Time

	Payee   PayeeProfile
	Balance Balance

	// Method is the payout destination under consideration.
	Method *PayoutMethodView

	// Amount is the net the batch proposes to pay.
	Amount Money

	// Settings are the runtime-tunable limits (PlatformSettings), passed in
	// rather than read, so a rule cannot drift from the values the batch was
	// built with.
	Settings Limits

	// ProviderBalance is what the rail reports it can pay from, when known.
	// Nil means "not checked" and is itself a reason to queue rather than pay.
	ProviderBalance *Money
}

// PayoutMethodView is the read-only projection of a payout destination that
// rules are allowed to see. It carries no account number — the engine never
// handles secret material, which lives in Secret Manager (#740).
type PayoutMethodView struct {
	ID          uuid.UUID
	Type        string // bank_account | vpa
	Verified    bool
	NameMatched bool
	// ActivatedAt is when the change cooldown expires. A method added or
	// edited pays out nothing until then — the control that makes account
	// takeover survivable.
	ActivatedAt *time.Time
	Disabled    bool
}

// Limits are the runtime knobs a rule may consult.
type Limits struct {
	MinPayout          Money
	PerPayeeDailyCap   Money
	PlatformDailyCap   Money
	MaturationDuration time.Duration
	MethodCooldown     time.Duration
	// AutoApproveCap bounds what risk_tiered mode will pay without a human.
	AutoApproveCap Money
}

// Rule is one guardrail. Implementations live in this package (engine rules)
// or in a PayeeAdapter (domain rules such as FSSAI expiry).
type Rule interface {
	// ID is stable and unique; it appears in audit rows and analytics.
	ID() string
	Evaluate(ctx context.Context, ec EvalContext) RuleResult
}

// RuleFunc adapts a function to the Rule interface.
type RuleFunc struct {
	RuleID string
	Fn     func(ctx context.Context, ec EvalContext) RuleResult
}

func (r RuleFunc) ID() string { return r.RuleID }
func (r RuleFunc) Evaluate(ctx context.Context, ec EvalContext) RuleResult {
	return r.Fn(ctx, ec)
}

// ChainResult is the combined verdict of a rule chain.
type ChainResult struct {
	Decision Decision     `json:"decision"`
	Results  []RuleResult `json:"results"`
}

// Blocking returns the results that were not an outright allow, most severe
// first, so the admin queue can lead with the real reason.
func (c ChainResult) Blocking() []RuleResult {
	out := make([]RuleResult, 0, len(c.Results))
	for _, r := range c.Results {
		if r.Decision != DecisionAllow {
			out = append(out, r)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Decision > out[j].Decision })
	return out
}

// ReasonCodes lists the non-allow reason codes, most severe first.
func (c ChainResult) ReasonCodes() []string {
	blocking := c.Blocking()
	codes := make([]string, 0, len(blocking))
	for _, r := range blocking {
		if r.ReasonCode != "" {
			codes = append(codes, r.ReasonCode)
		}
	}
	return codes
}

// Chain is an ordered set of rules evaluated together.
type Chain []Rule

// Evaluate runs every rule and returns the most severe decision.
//
// It deliberately does not short-circuit on the first block. A payee held up
// by three separate problems should learn about all three at once rather than
// discovering them one failed batch at a time, and the analytics in #747 need
// the full set to rank causes.
func (c Chain) Evaluate(ctx context.Context, ec EvalContext) ChainResult {
	res := ChainResult{Decision: DecisionAllow, Results: make([]RuleResult, 0, len(c))}
	for _, rule := range c {
		r := rule.Evaluate(ctx, ec)
		if r.RuleID == "" {
			r.RuleID = rule.ID()
		}
		res.Results = append(res.Results, r)
		if r.Decision > res.Decision {
			res.Decision = r.Decision
		}
	}
	return res
}
