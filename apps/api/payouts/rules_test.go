package payouts

import (
	"context"
	"testing"
)

func staticRule(id string, res RuleResult) Rule {
	return RuleFunc{RuleID: id, Fn: func(context.Context, EvalContext) RuleResult { return res }}
}

func TestChainReturnsTheMostSevereDecision(t *testing.T) {
	cases := []struct {
		name  string
		chain Chain
		want  Decision
	}{
		{"all allow", Chain{
			staticRule("a", Allow("a")),
			staticRule("b", Allow("b")),
		}, DecisionAllow},
		{"one queue", Chain{
			staticRule("a", Allow("a")),
			staticRule("b", Queue("b", "amount_above_cap", "")),
		}, DecisionQueue},
		{"block beats queue regardless of order", Chain{
			staticRule("a", Block("a", "method_unverified", "")),
			staticRule("b", Queue("b", "amount_above_cap", "")),
		}, DecisionBlock},
		{"queue then block", Chain{
			staticRule("a", Queue("a", "amount_above_cap", "")),
			staticRule("b", Block("b", "payee_suspended", "")),
		}, DecisionBlock},
		{"empty chain allows", Chain{}, DecisionAllow},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.chain.Evaluate(context.Background(), EvalContext{})
			if got.Decision != tc.want {
				t.Fatalf("Decision = %s, want %s", got.Decision, tc.want)
			}
		})
	}
}

func TestChainDoesNotShortCircuit(t *testing.T) {
	// A payee held up by three problems should learn all three at once rather
	// than one failed batch at a time, and #747 needs the full set to rank
	// causes. So every rule runs even after the first block.
	evaluated := map[string]bool{}
	mark := func(id string, res RuleResult) Rule {
		return RuleFunc{RuleID: id, Fn: func(context.Context, EvalContext) RuleResult {
			evaluated[id] = true
			return res
		}}
	}
	chain := Chain{
		mark("first", Block("first", "payee_suspended", "")),
		mark("second", Block("second", "method_unverified", "")),
		mark("third", Queue("third", "amount_above_cap", "")),
	}
	res := chain.Evaluate(context.Background(), EvalContext{})

	for _, id := range []string{"first", "second", "third"} {
		if !evaluated[id] {
			t.Fatalf("rule %q was skipped; the chain must not short-circuit", id)
		}
	}
	if len(res.Results) != 3 {
		t.Fatalf("got %d results, want 3", len(res.Results))
	}
	if len(res.Blocking()) != 3 {
		t.Fatalf("got %d blocking results, want 3", len(res.Blocking()))
	}
}

func TestBlockingIsSortedMostSevereFirst(t *testing.T) {
	chain := Chain{
		staticRule("a", Allow("a")),
		staticRule("b", Queue("b", "amount_above_cap", "")),
		staticRule("c", Block("c", "payee_suspended", "")),
		staticRule("d", Queue("d", "balance_unknown", "")),
	}
	res := chain.Evaluate(context.Background(), EvalContext{})

	blocking := res.Blocking()
	if len(blocking) != 3 {
		t.Fatalf("got %d blocking, want 3 (allow excluded)", len(blocking))
	}
	if blocking[0].Decision != DecisionBlock {
		t.Fatalf("most severe must lead, got %s", blocking[0].Decision)
	}
	// Stable within a severity, so the admin queue reads in rule order.
	if blocking[1].RuleID != "b" || blocking[2].RuleID != "d" {
		t.Fatalf("queue results must keep chain order, got %s then %s", blocking[1].RuleID, blocking[2].RuleID)
	}

	codes := res.ReasonCodes()
	if len(codes) != 3 || codes[0] != "payee_suspended" {
		t.Fatalf("ReasonCodes = %v, want payee_suspended first", codes)
	}
}

func TestRuleIDIsBackfilledFromTheRule(t *testing.T) {
	// A rule that forgets to stamp its own ID on the result must still be
	// attributable — an unattributed block is unactionable in the admin queue.
	chain := Chain{
		RuleFunc{RuleID: "forgetful", Fn: func(context.Context, EvalContext) RuleResult {
			return RuleResult{Decision: DecisionBlock, ReasonCode: "some_reason"}
		}},
	}
	res := chain.Evaluate(context.Background(), EvalContext{})
	if res.Results[0].RuleID != "forgetful" {
		t.Fatalf("RuleID = %q, want it backfilled to %q", res.Results[0].RuleID, "forgetful")
	}
}

func TestReasonCodesSkipsEmptyCodes(t *testing.T) {
	chain := Chain{staticRule("a", RuleResult{Decision: DecisionQueue})}
	res := chain.Evaluate(context.Background(), EvalContext{})
	if len(res.ReasonCodes()) != 0 {
		t.Fatalf("an empty reason code must not surface as a blank chip: %v", res.ReasonCodes())
	}
}

func TestDecisionSeverityOrdering(t *testing.T) {
	if !(DecisionAllow < DecisionQueue && DecisionQueue < DecisionBlock) {
		t.Fatal("Decision constants must be ordered by severity; Chain.Evaluate relies on it")
	}
}
