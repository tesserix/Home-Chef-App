---
phase: quick-260802-j8e
plan: 01
subsystem: mobile-customer / mobile-shared
tags: [plans-tab, subscriptions, ux-copy, tdd]
requires: []
provides:
  - subscriptionRowSummary (packages/mobile-shared/src/utils/subscription-summary-line.ts)
affects:
  - apps/mobile-customer/components/meal-plan/SubscriptionsSummary.tsx
tech-stack:
  added: []
  patterns: [pure-formatter-extraction, TDD RED-GREEN]
key-files:
  created:
    - packages/mobile-shared/src/utils/subscription-summary-line.ts
    - packages/mobile-shared/src/__tests__/subscription-summary-line.test.ts
  modified:
    - packages/mobile-shared/src/utils/index.ts
    - apps/mobile-customer/components/meal-plan/SubscriptionsSummary.tsx
decisions:
  - "!picked branch reads 'Tiffin · no meals scheduled yet' (per plan wording) rather than a generic empty state, since this is the default state every current subscriber sees (trialing status, auto-activate off, daily-order generator gated on active)"
  - "Accessibility label strips the leading 'Tiffin · ' prefix before interpolating into 'Manage your tiffin subscription — …' to avoid 'subscription: Tiffin · …' redundancy"
  - "Icon swapped UtensilsCrossed -> CalendarCheck, matching Dock.tsx plans tab icon and ActivePlanChip precedent"
metrics:
  duration: "~25 min"
  completed: 2026-08-02
---

# Quick Task 260802-j8e: Fix #907 — Plans-tab subscription row self-identifies in every state

One-liner: Extracted `subscriptionRowSummary()` so all five row states (loading/error/no-meals/today/next) are prefixed with "Tiffin ·", and swapped the row icon from `UtensilsCrossed` to `CalendarCheck` to match existing subscription iconography.

## What Was Built

GH #907 closed the last gap in #900/#903: the Plans-tab subscription row's `summaryLine()` only prefixed two of its four branches with "Tiffin", so the `isLoading`/`isError`/`!picked` states rendered as bare, unbranded text next to a cutlery icon — indistinguishable from a generic empty-state banner. Because new subscriptions land `trialing` and only `active` subscriptions generate fulfillment days, the `!picked` branch (`pickSubscriptionMealDay` returns `null`) is the state **every current subscriber sees today**, not an edge case.

### Task 1 (TDD): `subscriptionRowSummary()` in `packages/mobile-shared`

- New pure function `subscriptionRowSummary<T extends SubscriptionMealDayLike>(isLoading, isError, picked): string` in `packages/mobile-shared/src/utils/subscription-summary-line.ts`, generic over the same `SubscriptionMealDayLike`/`PickedSubscriptionMealDay` types `pickSubscriptionMealDay` already exports.
- All five branches now start with `'Tiffin'`:
  - loading: `'Tiffin · loading…'`
  - error: `"Tiffin · couldn't load — tap to view"` (keeps the tap-to-retry affordance)
  - no meals scheduled: `'Tiffin · no meals scheduled yet'` (exact wording from plan scope — this is the default state, not a generic empty banner)
  - today: `'Tiffin · today: Lunch'` (unchanged shape)
  - next: `'Tiffin · next: Wed dinner'` (unchanged shape, ported `shortWeekday`/`slotLabel` helpers verbatim — no change to IST arithmetic)
- Followed the RED→GREEN TDD cycle strictly: test file committed first against a nonexistent module (confirmed `Cannot find module` failure), then the implementation + barrel export committed together to turn it green.
- `export * from './subscription-summary-line';` appended to `packages/mobile-shared/src/utils/index.ts` (insertion-order style, matching existing file).
- Test file `packages/mobile-shared/src/__tests__/subscription-summary-line.test.ts` pins the actual #907 defect directly: one dedicated test asserts every one of the five branches starts with `'Tiffin'` in a single loop, not just a re-assertion of one string.

### Task 2: Wire `SubscriptionsSummary.tsx`

- Row now imports and calls `subscriptionRowSummary(isLoading, isError, picked)` from `@homechef/mobile-shared/utils`; the local `summaryLine`/`slotLabel`/`shortWeekday` functions and the now-unused `istCalendarDate` import were deleted.
- Icon swapped `UtensilsCrossed` → `CalendarCheck` (same size, color, `iconWrap` circle — no style changes), matching `Dock.tsx`'s Plans-tab icon and `ActivePlanChip.tsx`'s existing precedent for "this is your subscription, not a promotion."
- Accessibility label fixed to avoid the "…subscription: Tiffin · …" duplication once every branch carries the prefix: `` `Manage your tiffin subscription — ${summary.replace(/^Tiffin · /, '')}` ``, so a screen reader now hears "Manage your tiffin subscription — no meals scheduled yet" instead of the redundant compound.
- Top-level `errorRow` block (lines 34-53, "Couldn't load your tiffin subscription") was left untouched per plan scope — it already self-identifies.
- `MealPlanList.tsx`, `pickSubscriptionMealDay`, and chef-name enrichment were explicitly not touched — chef-name display remains a noted follow-up (`MealSubscription` only carries `chefId`, would need a new query).

## Verification

- `packages/mobile-shared`: `npx vitest run` → 124 tests, 120 pass / 4 fail. The 4 failures are the pre-existing baseline (`auth-screens.test.ts` x3, `resolve-auth-error.test.ts` x1 collection error) — confirmed identical before and after this change; failure count did not grow. 6 new tests added, all passing.
- `apps/mobile-customer`: `npx tsc --noEmit` → 0 errors (baseline measured at the start of this task was also 0, not the 1 documented in the plan's must_haves — re-measured per critical-repo-rules instruction rather than trusting the stale note; either way, zero new errors introduced).
- TDD gate sequence confirmed in git log: `test(260802-j8e): add failing test...` → `feat(260802-j8e): extract subscriptionRowSummary...` → `feat(260802-j8e): wire SubscriptionsSummary...`.
- Manual read-through: all five `subscriptionRowSummary` branches produce a string starting with "Tiffin" when rendered next to the `CalendarCheck` icon.
- Not device-verified on an emulator/simulator — this is a text/icon change with full unit-test coverage of the pure formatter; no new async/data-fetching behavior was introduced.

## Deviations from Plan

None — plan executed exactly as written, including the exact wording specified for the `!picked` branch and the accessibility-label fix called out in the quality bar.

## TDD Gate Compliance

RED gate: `6ab5404e test(260802-j8e): add failing test for subscriptionRowSummary Tiffin-prefix regression (#907)` — verified failing (`Cannot find module`) before commit.
GREEN gate: `956ea800 feat(260802-j8e): extract subscriptionRowSummary with Tiffin prefix in every state (#907)` — verified passing (6/6) before commit.
Both gates present and correctly ordered.

## Commits

- `6ab5404e` — test(260802-j8e): add failing test for subscriptionRowSummary Tiffin-prefix regression (#907)
- `956ea800` — feat(260802-j8e): extract subscriptionRowSummary with Tiffin prefix in every state (#907)
- `74ddd65d` — feat(260802-j8e): wire SubscriptionsSummary row to subscriptionRowSummary and CalendarCheck icon (#907)

## Follow-ups (explicitly out of scope, noted per plan constraints)

- Chef-name enrichment on the row is not implemented — `MealSubscription` only carries `chefId`, not a chef name; would need a new query/join. Noted in code context, not built.
- Not device-verified on an emulator — recommend a quick on-device pass to confirm the row reads calmly next to the meal-plan cards (quality bar: "keep it quiet, not louder than the meal-plan cards").

## Self-Check: PASSED

- FOUND: packages/mobile-shared/src/utils/subscription-summary-line.ts
- FOUND: packages/mobile-shared/src/__tests__/subscription-summary-line.test.ts
- FOUND: .planning/quick/260802-j8e-fix-907-plans-tab-subscription-row-self-/260802-j8e-SUMMARY.md
- FOUND commit: 6ab5404e
- FOUND commit: 956ea800
- FOUND commit: 74ddd65d
