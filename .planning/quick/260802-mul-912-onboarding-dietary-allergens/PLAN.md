---
id: 260802-mul
slug: 912-onboarding-dietary-allergens
issue: 912
date: 2026-08-02
mode: quick
branch: fix/912-onboarding-dietary
worktree: ../hc-wt-912
---

# GH #912 — onboarding never asks for allergens or diet

## Premise, verified against main (not taken from the issue text)

| Claim | Verified | Evidence |
|---|---|---|
| Onboarding collects cuisines only | TRUE | `apps/mobile-customer/app/(onboarding)/preferences.tsx` — `CUISINE_OPTIONS` is the only list; the POST body at :50-70 sends `cuisinePreferences` and no dietary field |
| Diet/allergens are collected elsewhere | TRUE | `app/profile/preferences.tsx:86` saves `{ dietaryPreferences, foodAllergies }` |
| Conflict machinery is gated on having a profile | TRUE | `hooks/useDietaryConflicts.ts` → `hasDietaryProfile` |
| **Backend change needed** | **FALSE — this is frontend-only** | `apps/api/handlers/customer.go:217-218` already binds `dietaryPreferences` + `foodAllergies`; :292-297 persist them to `dietary_preferences` / `food_allergies`. The server already accepts what the client never sends — the same dropped-field shape as #901. |
| Shared option lists exist | TRUE | `packages/mobile-shared/src/dietary/index.ts` — `DIET_OPTIONS` (10), `ALLERGEN_OPTIONS` (14), already imported by `app/profile/preferences.tsx:7` |

**Scope consequence: no Go change, no API change, no migration.** One screen plus the draft store.

## Decisions

- **Same screen, still "Step 3 of 3".** The issue prescribes extending the existing preferences step. Adding a 4th step would churn the progress bar in `user-info`/`address` and lengthen a flow we want people to finish.
- **Skippable — no validation gate on the CTA.** A hard gate on a safety question produces garbage answers (issue's own reasoning). `Finish Setup` stays enabled with nothing selected, and the whole existing flow is unchanged for a user who selects nothing.
- **Full lists, not a shortlist.** Parity with `app/profile/preferences.tsx`, and a truncated allergen list is a safety regression. All 14.
- **Draft store, not local `useState`.** Cuisines already live in `useCustomerOnboardingStore`; the store survives a mid-onboarding app restart. Local state would silently lose an allergen answer that a cuisine answer would survive.

## Tasks

### T1 — Draft store carries the two new fields
`apps/mobile-customer/store/onboarding-store.ts`
- Add `dietaryPreferences: string[]` and `foodAllergies: string[]`, both defaulting to `[]`.
- **`reset()` must clear them** — it is what stops a previous account's answers leaking into a re-onboard on the same device (the existing `draft.reset()` call at `preferences.tsx:74` relies on this).
- Match the existing field/update conventions in that file exactly; do not restructure it.

### T2 — Two chip sections on the onboarding preferences screen
`apps/mobile-customer/app/(onboarding)/preferences.tsx`
- Import `DIET_OPTIONS`, `ALLERGEN_OPTIONS` from `@homechef/mobile-shared/dietary`. Do **not** redeclare either list.
- Add a diet section and an allergen section below the cuisine chips, above the CTA.
- **Reuse the existing chip markup pattern verbatim** (:127-153): `Pressable` with `accessibilityRole="checkbox"`, `accessibilityState={{ checked }}`, `android_ripple={{ color: CHIP_RIPPLE }}`, and the inner-`View` styling. The options are `{value,label}` objects, not strings — key on `value`, render `label`, store `value`.
  - **iOS trap (see `feedback_ios_pressable_array_style`):** keep layout classes on the inner `View`, function-style children returning a single `View`, exactly as the cuisine chips do.
- Copy: the current heading "What do you love to eat?" no longer covers the screen. Give each section its own small heading + one line of helper text. The allergen helper must make it clear this drives warnings, and that it is optional and changeable later in Profile → Food preferences. Non-alarmist — no red, no warning iconography; these are the same neutral chips as everywhere else.
- Send both arrays in the existing `api.post('/v1/customer/onboarding/complete', …)` body alongside `cuisinePreferences`. Field names must be exactly `dietaryPreferences` and `foodAllergies` (the Go binding tags).

### T3 — Test the part that can be tested
`packages/mobile-shared` vitest is the only working mobile test bed; the screen itself is not renderable there (no `@testing-library/react`, and the lockfile cannot be regenerated — #897).
- If T1/T2 produce any pure logic worth pinning, extract it to `packages/mobile-shared/src/utils/` and test it there.
- **Do not add any dependency** to `packages/mobile-shared/package.json` or touch `pnpm-lock.yaml`. CI runs `--frozen-lockfile`; #881 and #909 both nearly broke on this.
- If there is no pure logic to extract, say so in the summary rather than inventing a test seam. Chip toggling and a POST body are not it.

## Verification

- `npx tsc --noEmit -p apps/mobile-customer` — compare against a stashed baseline; **1 pre-existing error** (`lib/payment.ts`, expo-router path typing). 1 before, 1 after.
- `pnpm --filter @homechef/mobile-shared test` — baseline is **126 pass / 4 pre-existing fail** (`auth-screens` ×3, `resolve-auth-error` ×1).
- `git diff --stat` on `package.json` and `pnpm-lock.yaml` must be **empty**.
- No file under `apps/api/` may appear in the diff.

## Out of scope

- Backfilling the question for existing users who already onboarded — separate issue if wanted.
- The vendor app.
- Any change to `useDietaryConflicts` or the surfaces #901 wired up.

## Known limitation to state honestly in the summary

Not device-verified unless an emulator run happens — this is a screen change on a flow that only runs once per account, so exercising it needs a fresh signup.
