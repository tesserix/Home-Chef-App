# Refund policy v3 — implementation spec

Status: SPEC ONLY. Nothing below is implemented. Supersedes the tier model in
`meal-plan-refund-flow-design.md` (that doc's §1 "FIRM RULE" is deliberately reversed here).

## Confirmed decisions

1. **GST is refunded**, and GST credit notes must be issued so filings stay correct.
   This reverses the documented firm rule that GST is never refunded.
2. **Chef-cancel penalty is in scope** — 6% levy, auto-deducted from the chef's next
   payout, shown as an invoice line, with a customer-facing explanation.

## BLOCKING — resolve before writing code

Both change who is out of pocket. Do not guess.

- **Platform fee: refunded or retained?**
  First statement: "75% + the gst ... − the platform fee" (retained).
  Later statement: "gst and platform fees are included" (refunded).
  These are opposite. Pick one.
- **Who chooses the adjustable amount above the floor?**
  First statement: chef approves and sets it. Later statement: customer selects 80/100%.
  A customer always picks 100%, which makes a floor meaningless — so "customer selects"
  probably means the chef picks within a floor, or the customer picks a *destination*
  (wallet vs source, which already exists). Confirm.

## Target policy

### Customer cancels remaining plan
Request routes to the chef, who approves and sets the amount subject to a floor by lead time:

| Lead time before cook-start | Floor | Chef may set |
|---|---|---|
| > 12h | 75% | 75–100% |
| ≤ 6h  | 50% | 50–100% |
| ≤ 2h  | 0%  | 0% unless chef judges it genuine and approves |

Refund base = food + GST (+ platform fee — SEE BLOCKING).

### Chef cancels
| Lead time | Customer refund | Chef penalty |
|---|---|---|
| > 12h | 100% incl. GST + platform fee | none |
| < 4h  | 100% incl. GST + platform fee | **6%**, auto-deducted from next payout, invoice line, customer notified |

## Gap against what exists

- `models.RefundProportion` is a fixed enum `{full, half, none}` — cannot express a
  floor or an adjustable percentage. Needs to become a bounded numeric.
- **No 12h/6h/2h tier logic exists.** Only one cutoff:
  `platform_settings` → `mealplan.refund_prep_cutoff_hours` (default 12).
  Follow that config-driven pattern for the new tiers; do not hardcode.
- `> 12h` currently **auto-approves 100% food, no chef step**. New policy adds a chef
  step and can *reduce* that to 75% — a customer-facing downgrade. Confirm intended.
- Refund base excludes GST/platform/delivery today (`refundableFood()`), and every
  refund path computes off it. Changing the base touches all of them.
- **No penalty mechanism of any kind exists.** `services/payout_recovery.go` is unrelated
  (it recovers failed payouts, it does not levy). Needs: penalty ledger entry, next-payout
  deduction, invoice line, customer notification.
- No GST credit-note issuance exists anywhere.

## Key files

- `apps/api/handlers/meal_plan_refund_v2.go` — chef decision endpoint
- `apps/api/services/meal_plan_refund_v2_flow.go` — state machine
- `apps/api/services/meal_plan_refund_v2_transitions.go` — transitions
- `apps/api/models/meal_plan.go` — `RefundProportion`, `MealPlanRefundStage`
- `apps/api/services/meal_plan_escrow.go` — `MealPlanRefundAmount`, refund base
- Chef-cancel full-refund path: see `project_chef_cancel_accountability` memory
- Flag: `MEALPLAN_REFUND_FLOW_V2_ENABLED=true` in prod (v2 IS live)

## Testing

Every branch moves money — table-test each tier boundary (just-over / just-under 12h,
6h, 2h), the floor enforcement, GST inclusion, penalty arithmetic, and idempotency on
double-submit. The existing suites in `handlers/meal_plan_refund_v2*_test.go` are the
pattern to follow.
