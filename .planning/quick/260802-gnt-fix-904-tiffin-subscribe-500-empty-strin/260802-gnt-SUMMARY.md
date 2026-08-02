---
phase: 260802-gnt-fix-904-tiffin-subscribe-500-empty-strin
plan: 01
subsystem: api
tags: [go, gorm, postgres, jsonb, tdd, mobile-customer, error-handling]

requires: []
provides:
  - "normaliseDayVariants (apps/api/handlers/meal_subscription.go) always returns valid JSON — never the Go zero value \"\" — for every input"
  - "Subscribe and UpdateSubscription both log the discarded DB error (with customer/chef or subscription/customer context) before responding 500"
  - "Customer app subscribe screen routes DB/network failures through friendlyErrorMessage instead of surfacing the raw axios error string"
affects: [meal-subscription, tiffin-billing]

tech-stack:
  added: []
  patterns:
    - "jsonb column invariant tests assert json.Valid(result), not string equality with a sentinel — pins the property Postgres actually enforces"

key-files:
  created:
    - apps/api/handlers/meal_subscription_test.go
  modified:
    - apps/api/handlers/meal_subscription.go
    - apps/api/handlers/meal_subscription_edit.go
    - "apps/mobile-customer/app/meal-subscription/[chefId].tsx"

key-decisions:
  - "Fixed sentinel is \"{}\" not \"\" — verified as behavior-preserving by inspecting VariantForDay (the only reader), which treats \"\" and \"{}\" identically (both fall through to Variant)"
  - "Test asserts json.Valid([]byte(result)), not result == \"{}\", per the plan's correctness bar — pins the actual DB-enforced invariant so a future refactor to a different valid-JSON sentinel can't silently regress"

requirements-completed: [GH-904]

duration: 6min
completed: 2026-08-02
---

# Quick Task 260802-gnt: Fix #904 tiffin subscribe 500 Summary

**Fixed the P0 bug blocking every tiffin subscribe in production: `normaliseDayVariants` returned Go's zero-value `""` for the no-override case (every real app request), which Postgres rejects as invalid JSON for the `jsonb` `DayVariants` column, causing `database.DB.Create` to fail on literally every subscribe — plus the DB error that would have revealed this was silently discarded on both create and edit paths, and the customer saw the raw axios error string.**

## Performance

- **Duration:** 6 min (12:04 → 12:10)
- **Tasks:** 2 completed
- **Files modified:** 4 (1 created, 3 modified)

## Accomplishments

- `normaliseDayVariants` now returns `"{}"` from all three code paths that used to return `""` — proven correct by `VariantForDay` (the only reader), which treats an empty map identically to the old `""` short-circuit, so this is behavior-preserving, not just JSON-valid
- `TestNormaliseDayVariants` (4 subtests, TDD RED→GREEN) pins `json.Valid` for the no-overrides, all-filtered-out, and happy-path cases — the two failure cases were confirmed FAILING against the pre-fix code before the fix was applied
- Both `Subscribe` (create) and `UpdateSubscription` (edit) now log the previously-discarded DB error with diagnosable context (customer/chef id or subscription/customer id) before responding 500, matching the existing `log.Printf` convention already used elsewhere in `meal_subscription.go`
- Customer app's subscribe screen `onError` now routes through `friendlyErrorMessage` instead of the raw `e.message`, matching the established pattern at `app/subscriptions.tsx`

## Task Commits

1. **Task 1 (RED): pin #904 with a failing test** - `cf97cd16` (test) — confirmed 2 of 4 subtests failed against pre-fix code (`normaliseDayVariants` returning `""`), exactly the two cases that 500 in production
2. **Task 1 (GREEN): fix the invalid-JSON bug and stop discarding DB errors** - `e2142643` (fix) — all 3 `return ""` sites → `return "{}"`; `Subscribe` and `UpdateSubscription` now `log.Printf` the discarded error before responding 500
3. **Task 2: stop showing the raw axios error on subscribe failure** - `676446db` (fix)

## Files Created/Modified

- `apps/api/handlers/meal_subscription_test.go` - New `TestNormaliseDayVariants`, 4 subtests asserting `json.Valid` (no-overrides nil, no-overrides empty map, all-filtered-out, happy-path with unmarshal equality check)
- `apps/api/handlers/meal_subscription.go` - `normaliseDayVariants`'s three `return ""` → `return "{}"` + doc comment update; `Subscribe`'s `database.DB.Create` error branch now logs before responding
- `apps/api/handlers/meal_subscription_edit.go` - Added `"log"` import; `UpdateSubscription`'s `res.Error` branch now logs before responding
- `apps/mobile-customer/app/meal-subscription/[chefId].tsx` - Imported `friendlyErrorMessage`; `onSubscribe`'s `onError` now calls `friendlyErrorMessage(e, 'Could not subscribe. Please try again.')` instead of `e.message || 'Please try again.'`

## Decisions Made

- Kept the sentinel value as `"{}"` (empty JSON object) rather than any other valid-JSON placeholder, because it's what the plan's interface excerpt already proved safe against the only reader (`VariantForDay`) — no exploration needed, followed as specified.
- Test asserts `json.Valid([]byte(result))` rather than `result == "{}"`, per the plan's correctness bar — this is the actual invariant the Postgres `jsonb` column enforces, and it survives a future refactor to a different sentinel.

## Deviations from Plan

None - plan executed exactly as written. One gofmt quirk was corrected inline (not a deviation from the plan's intent): `gofmt -w` on the updated doc comment in `meal_subscription.go` converted a straight double-apostrophe (`''`, meaning "empty string" in prose) into a single curly closing quote character, changing the comment's meaning nonsensically. Reworded the comment to say "the empty string" instead of using `''`, re-ran `gofmt -l` clean, and re-verified the build/tests — this is line-level prose polish within Task 1's own file, not a scope change.

## Issues Encountered

- The plan's documented tsc baseline (`1 pre-existing error in lib/payment.ts`) did not hold at execution time — a fresh `npx tsc --noEmit` measured **0** errors both before and after Task 2's change. Per the plan's own instruction ("re-measure rather than trusting it blindly"), this was treated as the authoritative baseline; Task 2 introduced zero new errors (0 → 0), which meets the "no TypeScript regression" success criterion regardless of which baseline number was expected.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- The core tiffin-subscribe flow (`POST /meal-subscriptions` with no `dayVariants`, i.e. every real app request) is unblocked — `normaliseDayVariants` can never again produce a value Postgres rejects.
- Any future DB failure on the create or edit path will now be diagnosable from server logs alone (customer/chef or subscription/customer id + underlying error), removing the need for another production A/B test.
- Not device/production-verified in this session (backend-only fix requires a deploy + a live subscribe attempt to close the loop; no infra changes were in scope here).

---
*Phase: 260802-gnt-fix-904-tiffin-subscribe-500-empty-strin*
*Completed: 2026-08-02*

## Self-Check: PASSED

All created/modified files confirmed present on disk (`apps/api/handlers/meal_subscription.go`, `apps/api/handlers/meal_subscription_edit.go`, `apps/api/handlers/meal_subscription_test.go`, `apps/mobile-customer/app/meal-subscription/[chefId].tsx`, this SUMMARY.md). All 3 task commits confirmed in `git log` (`cf97cd16`, `e2142643`, `676446db`).
