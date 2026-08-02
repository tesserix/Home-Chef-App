---
phase: 260802-gnt-fix-904-tiffin-subscribe-500-empty-strin
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - apps/api/handlers/meal_subscription.go
  - apps/api/handlers/meal_subscription_edit.go
  - apps/api/handlers/meal_subscription_test.go
  - apps/mobile-customer/app/meal-subscription/[chefId].tsx
autonomous: true
requirements: [GH-904]

must_haves:
  truths:
    - "Subscribing to a tiffin plan from the app (no per-day overrides — the subscribe screen has no per-day picker) succeeds with HTTP 201, not 500"
    - "Editing a subscription to clear all per-day overrides succeeds, not 500 — the edit path shares the same bug"
    - "normaliseDayVariants never returns a string that fails json.Valid, for any input (no overrides, all entries filtered out, or a genuine happy-path override)"
    - "If subscription create or update ever fails at the DB layer again, the server log carries enough context (customer id, chef id or subscription id, and the underlying error) to diagnose without an A/B test against production"
    - "If subscribe still fails for any reason, the customer sees a human-readable message, never the raw axios 'Request failed with status code 500' string"
  artifacts:
    - path: "apps/api/handlers/meal_subscription.go"
      provides: "normaliseDayVariants returns \"{}\" (valid JSON) instead of \"\" for its three no-override paths; Subscribe logs the discarded database.DB.Create error with customer/chef context before responding 500"
    - path: "apps/api/handlers/meal_subscription_edit.go"
      provides: "UpdateSubscription logs the discarded res.Error with subscription/customer context before responding 500"
    - path: "apps/api/handlers/meal_subscription_test.go"
      provides: "TestNormaliseDayVariants — asserts json.Valid(result) for the no-overrides case, the all-filtered-out case, and the happy-path case (the property the Postgres jsonb column actually enforces, not a brittle == \"{}\" check)"
      min_lines: 20
    - path: "apps/mobile-customer/app/meal-subscription/[chefId].tsx"
      provides: "Subscribe's onError uses friendlyErrorMessage(e, ...) instead of e.message, per the repo-wide no-raw-error-codes rule"
  key_links:
    - from: "apps/api/handlers/meal_subscription.go (Subscribe)"
      to: "normaliseDayVariants"
      via: "DayVariants: normaliseDayVariants(req.DayVariants, req.Days) assigned into models.MealSubscription.DayVariants (gorm:\"type:jsonb\")"
      pattern: "normaliseDayVariants\\(req\\.DayVariants, req\\.Days\\)"
    - from: "apps/api/handlers/meal_subscription_edit.go (UpdateSubscription)"
      to: "normaliseDayVariants"
      via: "\"day_variants\": normaliseDayVariants(req.DayVariants, req.Days) in the GORM Updates map"
      pattern: "normaliseDayVariants\\(req\\.DayVariants, req\\.Days\\)"
    - from: "apps/mobile-customer/app/meal-subscription/[chefId].tsx"
      to: "apps/mobile-customer/lib/errors.ts"
      via: "import { friendlyErrorMessage } from '../../lib/errors'"
      pattern: "friendlyErrorMessage"
---

<objective>
Fix GitHub #904: subscribing to a tiffin plan returns HTTP 500 in production for every real customer request. Root cause is proven (see below), not inferred: `normaliseDayVariants` returns the Go zero value `""` when a customer has no per-day overrides (i.e. every subscribe from the app, since the subscribe screen has no per-day picker), that empty string is assigned to `MealSubscription.DayVariants` (`gorm:"type:jsonb"`), and Postgres rejects `''` as invalid JSON — so `database.DB.Create` fails on literally every real subscribe. The identical bug exists on the edit path. The DB error that would have revealed this was silently discarded on both paths, and the customer saw the raw axios error string instead of a human message.

Purpose: unblock the core tiffin-subscription flow (every subscribe in production is currently broken) and remove the two conditions (discarded error, raw error surfaced to the user) that made this take an A/B test against production to diagnose.

Output: `normaliseDayVariants` always returns valid JSON; both the create and edit paths log DB failures with diagnosable context; the subscribe screen shows a friendly message on failure.
</objective>

<execution_context>
@$HOME/.claude/get-shit-done/workflows/execute-plan.md
@$HOME/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/STATE.md
@./CLAUDE.md

<interfaces>
<!-- Current normaliseDayVariants (apps/api/handlers/meal_subscription.go:192-224) — -->
<!-- the function this plan changes. All three `return ""` sites become `return "{}"`. -->
```go
// normaliseDayVariants keeps only well-formed entries: a day the customer
// actually subscribed to, and a variant the chef actually offers. Anything else
// is dropped rather than stored, so VariantForDay never has to defend against
// junk and the plan default cleanly covers the gap. Returns "" when nothing
// survives, which is the "no override" state.
func normaliseDayVariants(in map[string]string, days []int64) string {
	if len(in) == 0 {
		return ""
	}
	allowed := make(map[string]bool, len(days))
	for _, d := range days {
		allowed[strconv.FormatInt(d, 10)] = true
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		if !allowed[k] {
			continue
		}
		mv := models.MealVariant(v)
		if mv != models.MealVariantVeg && mv != models.MealVariantNonVeg {
			continue
		}
		out[k] = string(mv)
	}
	if len(out) == 0 {
		return ""
	}
	b, err := json.Marshal(out)
	if err != nil {
		return ""
	}
	return string(b)
}
```

<!-- Subscribe's discarded error (apps/api/handlers/meal_subscription.go:299-302) — -->
<!-- userID and chefID are already in scope at this point in Subscribe. -->
```go
	if err := database.DB.Create(&sub).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create subscription"})
		return
	}
```

<!-- UpdateSubscription's discarded error (apps/api/handlers/meal_subscription_edit.go:118-124) — -->
<!-- id (the subscription id, a uuid.UUID) and userID are already in scope. -->
<!-- meal_subscription_edit.go does NOT currently import "log" — add it. -->
```go
	res := database.DB.Model(&models.MealSubscription{}).
		Where("id = ? AND customer_id = ? AND status = ?", id, userID, sub.Status).
		Updates(updates)
	if res.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update subscription"})
		return
	}
```

<!-- Existing log.Printf convention in this exact file (meal_subscription.go:420), -->
<!-- "log" is already imported — match this style, don't invent a new logger. -->
```go
		log.Printf("winback: offer on meal-sub cancel failed for user=%s: %v", userID, werr)
```

<!-- The ONLY reader of DayVariants (apps/api/models/meal_subscription.go:154-169) — -->
<!-- proves "{}" is a safe, behavior-preserving replacement for "": it unmarshals to -->
<!-- an empty map, so the lookup below simply misses and falls through to Variant, -->
<!-- identical to today's "" short-circuit. Do not modify this file. -->
```go
func (s *MealSubscription) VariantForDay(day int) MealVariant {
	if s.DayVariants != "" {
		var m map[string]string
		if err := json.Unmarshal([]byte(s.DayVariants), &m); err == nil {
			if v, ok := m[strconv.Itoa(day)]; ok {
				if mv := MealVariant(v); mv == MealVariantVeg || mv == MealVariantNonVeg {
					return mv
				}
			}
		}
	}
	if s.Variant == MealVariantVeg || s.Variant == MealVariantNonVeg {
		return s.Variant
	}
	return MealVariantVeg
}
```

<!-- Existing handlers-package test style using testify, for reference -->
<!-- (apps/api/services/meal_subscription_test.go:1-20, same repo, testify already a dependency): -->
```go
package services

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/homechef/api/models"
)

func TestComputeMealCycleAmount(t *testing.T) {
	assert.Equal(t, 1240.0, ComputeMealCycleAmount(120, 2, 5, models.MealCadenceWeekly, 40))
}
```

<!-- The exact leak point in the mobile app (apps/mobile-customer/app/meal-subscription/[chefId].tsx:82-94) — -->
<!-- subscribe.mutate's onError. useSubscribeMeal() (hooks/useMealSubscription.ts:74-76) types -->
<!-- this as useMutation<{subscription: MealSubscription}, Error, MealSelection>, so `e` is `Error`. -->
```typescript
  function onSubscribe() {
    if (!selectionValid) return;
    subscribe.mutate(
      { chefId: chefId!, slots, days, variant, cadence },
      {
        onSuccess: () =>
          showAlert('Subscription created', 'Your daily tiffin is set up. Manage it under My Subscriptions.', [
            { text: 'View', onPress: () => router.replace('/subscriptions' as never) },
          ]),
        onError: (e) => showAlert('Could not subscribe', e.message || 'Please try again.'),
      },
    );
  }
```

<!-- friendlyErrorMessage signature (apps/mobile-customer/lib/errors.ts:21-24) — the -->
<!-- established helper; the repo-wide rule (no raw error codes to users) requires using -->
<!-- this here, matching the pattern already used at app/subscriptions.tsx:190-202 -->
<!-- (`friendlyErrorMessage(err, '<fallback>')` passed straight into showAlert's message arg). -->
```typescript
export function friendlyErrorMessage(
  error: unknown,
  fallback = 'Something went wrong. Please try again.',
): string
```
</interfaces>
</context>

<tasks>

<task type="auto" tdd="true">
  <name>Task 1: Fix the invalid-JSON DayVariants bug and stop discarding the DB error</name>
  <files>apps/api/handlers/meal_subscription.go, apps/api/handlers/meal_subscription_edit.go, apps/api/handlers/meal_subscription_test.go</files>
  <behavior>
    - Test 1 (no overrides — the real-world case, every current app request): `normaliseDayVariants(nil, []int64{1,2,3,4,5})` and `normaliseDayVariants(map[string]string{}, []int64{1,2,3,4,5})` — assert `json.Valid([]byte(result))` is true. This is currently FALSE (returns `""`) — this is the case that 500s in production today.
    - Test 2 (all entries filtered out — an invalid day and an invalid variant, nothing survives): `normaliseDayVariants(map[string]string{"9": "veg", "1": "extra-spicy"}, []int64{1,2,3,4,5})` (day `9` isn't in `days`; variant `extra-spicy` isn't veg/nonveg) — assert `json.Valid([]byte(result))` is true. Also currently FALSE.
    - Test 3 (happy path — genuine overrides survive): `normaliseDayVariants(map[string]string{"1": "veg", "6": "nonveg"}, []int64{1,2,3,4,5,6})` — assert `json.Valid([]byte(result))` is true AND `json.Unmarshal` into `map[string]string` equals `{"1": "veg", "6": "nonveg"}` (this path already works today — pins it against regression).
    Assert the `json.Valid` property directly (not `result == "{}"`), because that is the invariant the Postgres `jsonb` column actually enforces and the one a future refactor could silently break again.
  </behavior>
  <action>
    In `apps/api/handlers/meal_subscription.go`, change all three `return ""` statements inside `normaliseDayVariants` (the `len(in) == 0` guard, the `len(out) == 0` guard, and the `json.Marshal` error branch) to `return "{}"`. Update the function's doc comment: replace "Returns "" when nothing survives, which is the "no override" state." with "Returns "{}" (valid empty JSON, never the Go zero value "") when nothing survives — the DayVariants column is jsonb and Postgres rejects '' as invalid JSON (#904)."

    In the same file's `Subscribe` handler, replace the discarded-error block at the `database.DB.Create(&sub).Error` check with a version that logs before responding, matching the existing `log.Printf` convention already used at line ~420 in this file: `log.Printf("meal-subscription: create failed for customer=%s chef=%s: %v", userID, chefID, err)` then the existing `c.JSON(http.StatusInternalServerError, ...)` / `return`. `userID` and `chefID` are already in scope at that point in `Subscribe`.

    In `apps/api/handlers/meal_subscription_edit.go`, add `"log"` to the import block (it is not currently imported). Replace the discarded-error block at the `res.Error != nil` check in `UpdateSubscription` with a version that logs first: `log.Printf("meal-subscription: update failed for subscription=%s customer=%s: %v", id, userID, res.Error)` then the existing `c.JSON(http.StatusInternalServerError, ...)` / `return`. `id` (the subscription uuid) and `userID` are already in scope.

    Create `apps/api/handlers/meal_subscription_test.go` (package `handlers`, importing `encoding/json`, `testing`, `github.com/stretchr/testify/assert`) with `TestNormaliseDayVariants` covering the three cases in `<behavior>` as `t.Run` subtests, asserting `json.Valid` (and the unmarshalled equality for the happy path) via `assert`. Add a short header comment: this test pins #904 — the property the database actually enforces on this column is JSON validity, not string equality with any particular sentinel value.
  </action>
  <verify>
    <automated>cd apps/api && gofmt -l handlers/meal_subscription.go handlers/meal_subscription_edit.go handlers/meal_subscription_test.go; go build ./...; go test ./handlers/ -run TestNormaliseDayVariants -v</automated>
  </verify>
  <done>`gofmt -l` prints nothing for the three touched files; `go build ./...` succeeds; `TestNormaliseDayVariants` passes all three subtests; `normaliseDayVariants` no longer returns `""` from any branch; `Subscribe` and `UpdateSubscription` each log the underlying DB error (with customer/chef or subscription/customer context) before responding 500.</done>
</task>

<task type="auto">
  <name>Task 2: Stop showing the raw axios error on subscribe failure</name>
  <files>apps/mobile-customer/app/meal-subscription/[chefId].tsx</files>
  <action>
    Add `import { friendlyErrorMessage } from '../../lib/errors';` to the import block in `apps/mobile-customer/app/meal-subscription/[chefId].tsx` (alongside the existing `useAlert` import from `'@homechef/mobile-shared/ui'`).

    Change the `onError` handler inside `onSubscribe()` from `onError: (e) => showAlert('Could not subscribe', e.message || 'Please try again.')` to `onError: (e) => showAlert('Could not subscribe', friendlyErrorMessage(e, 'Could not subscribe. Please try again.'))`, matching the established call pattern at `apps/mobile-customer/app/subscriptions.tsx:190-202`. Do not touch the `onSuccess` branch, `usePreviewMealPrice`'s `onError` (which intentionally has no user-facing message — it just clears the price preview), or any other line in this file.
  </action>
  <verify>
    <automated>cd apps/mobile-customer && npx tsc --noEmit 2>&1 | grep -c "error TS"</automated>
  </verify>
  <done>Output of the verify command is `1` (the pre-existing `lib/payment.ts` baseline error — re-measure at execution time, do not trust this blindly); `friendlyErrorMessage` is imported and used in the subscribe screen's `onError`; a failed subscribe shows a human-readable message, never the raw axios status-code string.</done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|--------------|
| Customer app -> `POST /v1/meal-subscriptions` (Subscribe) | Existing authenticated write endpoint; this plan changes what gets stored (never `""`) and what gets logged on failure, not what is accepted or who can call it. |
| Customer app -> `PUT /v1/meal-subscriptions/:id` (UpdateSubscription) | Same — existing authenticated, ownership-scoped write endpoint (`customer_id = ?` in the WHERE clause, unchanged). |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-------------------|
| T-260802-gnt-01 | Information Disclosure | New `log.Printf` in `Subscribe`/`UpdateSubscription` | accept | Logs go to server-side stdout only (customer id, chef/subscription id, error text) — never returned in the HTTP response, which keeps its existing generic `"Failed to create/update subscription"` message. Matches the existing `log.Printf` pattern already used elsewhere in this same file for the win-back offer failure. |
| T-260802-gnt-02 | Tampering | `normaliseDayVariants` JSON output stored into a `jsonb` column | mitigate | Fixed by this plan: every return path now produces valid JSON (`"{}"` instead of `""`), verified by `json.Valid` in `TestNormaliseDayVariants`, so the column can never receive a value Postgres would reject or that `VariantForDay`'s `json.Unmarshal` could choke on. |
| T-260802-gnt-03 | Information Disclosure | Raw axios error string shown to the customer | mitigate | Fixed by this plan: `onError` now routes through `friendlyErrorMessage`, which strips transport-noise strings (status codes, network errors) before anything reaches the UI, per the existing repo-wide helper and convention. |

No package installs in this plan — Package Legitimacy Gate not applicable.
</threat_model>

<verification>
1. `cd apps/api && go build ./...` — succeeds.
2. `cd apps/api && gofmt -l handlers/meal_subscription.go handlers/meal_subscription_edit.go handlers/meal_subscription_test.go` — no output.
3. `cd apps/api && go test ./handlers/ -run TestNormaliseDayVariants -v` — all subtests pass, including the two that were failing against the pre-fix behavior (no-overrides, all-filtered-out).
4. `cd apps/mobile-customer && npx tsc --noEmit 2>&1 | grep -c "error TS"` — equals the pre-existing baseline (1, `lib/payment.ts`; re-measure, don't trust blindly).
5. `grep -n 'return ""' apps/api/handlers/meal_subscription.go` — zero matches inside `normaliseDayVariants` (confirms all three sites were changed).
6. `grep -n 'log.Printf' apps/api/handlers/meal_subscription_edit.go` — at least one match (confirms the previously-silent edit-path failure now logs).
7. `grep -n 'friendlyErrorMessage' "apps/mobile-customer/app/meal-subscription/[chefId].tsx"` — confirms the import and usage.
8. Do not run `npx prettier` — no config in this repo; it rewrites whole files to double quotes.
</verification>

<success_criteria>
- Subscribing to a tiffin plan from the app (no `dayVariants`, exactly what every real request sends) returns HTTP 201, not 500.
- Editing a subscription to clear all per-day overrides succeeds, not 500.
- `normaliseDayVariants` provably never returns invalid JSON, for any input — pinned by `TestNormaliseDayVariants`.
- A future DB failure on either the create or edit path leaves a log line with enough context (customer/chef or subscription/customer id, plus the underlying error) to diagnose without reproducing against production.
- A failed subscribe shows a friendly message, never the raw axios "Request failed with status code 500" string.
- No TypeScript or Go regression versus the measured baselines (1 pre-existing `apps/mobile-customer` tsc error; clean `go build`).
</success_criteria>

<output>
Create `.planning/quick/260802-gnt-fix-904-tiffin-subscribe-500-empty-strin/260802-gnt-SUMMARY.md` when done
</output>
