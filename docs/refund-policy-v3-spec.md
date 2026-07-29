# Refund policy v3

Status: **IMPLEMENTED** (#834). Supersedes the tier model *and* the §1 "FIRM RULE" of
`meal-plan-refund-flow-design.md` — that document describes v2 and is kept only as the
history of what this replaced.

Gated by `MEALPLAN_REFUND_FLOW_V2_ENABLED` (the flag v2 shipped behind, `true` in prod) plus
the runtime policy below.

## Confirmed decisions

1. **The refund base is `food − platform commission + GST + delivery`.** This changes exactly
   one term from v2 (which never refunded GST): the tax now comes back. The platform **keeps
   its commission** on a cancelled day.
   *There is no separate customer-facing "platform fee" line on a meal plan:* the platform's
   take is the commission withheld from the **chef** inside the food price, so retaining it
   means refunding food NET of that commission.

   Worked example — a ₹100 day, 8% GST, ₹10 delivery, 15% commission:

   | | Charged | Refunded at 100% |
   |---|---|---|
   | Food | ₹100.00 | ₹85.00 (less ₹15 commission) |
   | GST | ₹8.00 | ₹8.00 |
   | Delivery | ₹10.00 | ₹10.00 |
   | **Total** | **₹118.00** | **₹103.00** |
2. **GST is refunded, and a credit note is issued for it** — shipped together, because a
   refund that returns tax without a note creates a filing discrepancy on day one.
3. **The chef chooses the amount; the server enforces the floor.** Web and mobile call the
   same endpoint, so a client-side constraint is no constraint.
4. **Chef-cancel penalty is in scope** — a percentage levy, netted off the next settlement,
   shown as a statement line, with a customer-facing explanation.

## Target policy — customer cancels

| Lead before cook-start | Refund | Decided by |
|---|---|---|
| **> 12h** | **100%, automatic** | nobody — instant, unchanged from v2 |
| 12h – 6h | floor 75% | chef sets 75–100% |
| 6h – 2h | floor 50% | chef sets 50–100% |
| < 2h | floor 0% | chef may still grant up to 100% |

**Why >12h stayed automatic** (this differs from the first draft, which put a 75% floor on
it): the 12h cutoff exists *because prep has not started*, so withholding 25% there is a
penalty with no cost behind it — the part a customer would dispute and win. It would also add
an approval step to the most common and most benign cancellation, where chefs either
rubber-stamp 100% (friction, no change) or habitually take a free 25% (customers worse off
than before). The tiering intent is preserved; it begins where prep begins.

This is a **policy call, not a technical constraint**: the whole table is configuration
(below). Setting the top band to `{minLeadHours: 12, floorPercent: 75, autoApprove: false}`
implements the original model with no code change.

## Target policy — chef cancels

| Lead | Customer refund | Chef penalty |
|---|---|---|
| > 4h | 100% incl. GST + fees (already the behaviour) | none |
| < 4h | 100% incl. GST + fees | **6%** of the order, deducted from the next settlement |

Two guards, both configurable and both deliberate:

- **Grace** — the first N cancellations in a rolling window are exempt (default: 1 per 30
  days). A genuine emergency is not fraud, and auto-fining it with no recourse costs chefs
  faster than the levy recovers.
- **Waiver** — an admin can cancel any pending levy, with the reason recorded.

An order with no scheduled service time is treated as **zero lead**: it is on-demand, already
accepted, and being cooked now.

## Configuration

Everything lives in the `platform_policy` `PlatformSettings` blob
(`services/platform_policy.go`) and is admin-tunable at runtime:

```jsonc
{
  "mealPlanRefundTiers": [
    { "minLeadHours": 12, "floorPercent": 100, "autoApprove": true },
    { "minLeadHours": 6,  "floorPercent": 75 },
    { "minLeadHours": 2,  "floorPercent": 50 },
    { "minLeadHours": 0,  "floorPercent": 0  }
  ],
  "chefCancelPenaltyEnabled": true,
  "chefCancelPenaltyPercent": 6,
  "chefCancelPenaltyLeadHours": 4,
  "chefCancelPenaltyGraceCount": 1,
  "chefCancelPenaltyGraceDays": 30
}
```

Bands are matched highest-lead-first and a boundary belongs to the **higher** band (exactly
12h out is still automatic). A configured table that is empty or wholly invalid falls back to
the default; `autoApprove` below 100% is demoted to a chef decision, since resolving a partial
refund with nobody having agreed to it is not something the system should be able to do.

## The floor is pinned, not recomputed

Raising a request stamps `meal_plan_days.refund_floor_percent` with the band that applied **at
that moment**. The chef's decision is validated against that stored value, never against a
fresh clock reading — otherwise a chef could shrink their own obligation simply by sitting on
the decision until the meal was imminent.

A pre-v3 day carries no pinned floor and falls back to 0: it keeps the freedom it was created
under rather than being retroactively bound by a rule nobody told the chef about.

## What changed in the code

| Area | Change |
|---|---|
| Base | `MealPlanRefundAmount(plan, day, percent)` off `mealPlanDayRefundBase` (food − commission + GST + delivery). **Not** `perDayGross` — that divides delivery by `len(plan.Days)` and silently collapses to the bare food price for the narrow `Select`s every refund handler uses. |
| Proportion | `MealPlanDay.RefundPercent` (0–100) is authoritative. `ChefRefundChoice` stays as a coarse label (`full`/`half`/`none`/`partial`) for pre-v3 readers. |
| Tiers | `services/meal_plan_refund_tiers.go` — `ResolveMealPlanRefundTier(leadHours)`. |
| Floor | `ChefDecideMealPlanRefund` rejects below-floor with `ErrRefundBelowFloor` → HTTP 422. |
| Credit notes | `models.CreditNote` + `services/credit_note.go`. Issued **inside** the refund tx, so a note that cannot be written rolls the refund back. Idempotent on `source_key`. Covers both GST-returning paths: the v3 cancellation executor **and** `RefundDay` (declined / undelivered / failed days), which has always refunded the full gross — wiring it here closes that pre-existing gap rather than leaving half the tax-refunding paths unnoted. |
| Penalty | `models.ChefPenalty` + `services/chef_penalty.go`. Levied from `handlers/chef_order_cancel.go`, deducted in `services/statement.go`, surfaced at `/admin/chef-penalties` and `/chef/penalties`. |
| Statement | `WeeklyStatement.PenaltyDeductions` — the levy's invoice line. `NetPayout` is after it. |

## API compatibility

`POST /chef/meal-plan-days/:dayId/refund-decision` accepts the v3 `{"percent": N}` and the
pre-v3 `{"choice": "full|half|none"}`. A legacy choice maps to 100/50/0 and is then subject to
the **same** floor check — so an un-updated app asking for "none" on a 75%-floor day is
rejected rather than silently under-refunding. An unrecognised choice is rejected outright
rather than read as 0%.

## Rollout notes

- The base change is live the moment the code deploys (the flag was already on). Every
  meal-plan refund grows by the day's GST — on the worked example, ₹95 under v2 becomes ₹103.
  The platform keeps its commission but is now out of pocket the tax it returns (offset by the
  credit note) plus the delivery. Finance should expect that delta.
- The penalty defaults to **enabled**. Set `chefCancelPenaltyEnabled: false` in
  `platform_policy` to ship the refund change without the levy.
- Old vendor builds keep working (see API compatibility) but cannot refund between the fixed
  Full/Half/None points until they are updated.
