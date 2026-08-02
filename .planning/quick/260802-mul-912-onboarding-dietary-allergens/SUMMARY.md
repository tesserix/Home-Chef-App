---
id: 260802-mul
slug: 912-onboarding-dietary-allergens
issue: 912
date: 2026-08-02
status: complete
branch: fix/912-onboarding-dietary
commits: [d8d4f64c, b473e0b2]
---

# GH #912 — onboarding now asks for diet and allergens

## What shipped

Two chip sections (`DIET_OPTIONS` ×10, `ALLERGEN_OPTIONS` ×14, both imported from
`@homechef/mobile-shared/dietary` — neither list redeclared) on the existing
step-3 onboarding preferences screen, writing through the draft store into the
`dietaryPreferences` / `foodAllergies` fields the POST body now carries.

Skippable by design: the CTA stays enabled with nothing selected, so the flow is
byte-for-byte unchanged in behaviour for a user who answers nothing.

## The finding that shrank the work

**No backend change was needed.** `apps/api/handlers/customer.go:217-218` already
binds `dietaryPreferences` and `foodAllergies`, and :292-297 already persist them.
The server has been accepting these fields all along; the client simply never sent
them — the same dropped-field shape as #901's `WeeklyMenuItem`, found the same way
(read the Go model before scoping the API work).

## Effect

Every conflict surface is gated on `hasDietaryProfile`. Before this, that gate was
false for anyone who never found Profile → Food preferences, so #41's à-la-carte
badges and all of #901's recurring-meal warnings were inert for them. New signups
now arrive with the profile populated and the safety net live.

## Verification — measured, not assumed

| Check | Baseline | After |
|---|---|---|
| `tsc --noEmit -p apps/mobile-customer` | 1 (`lib/payment.ts:114`, expo-router path typing) | 1, identical |
| `pnpm --filter @homechef/mobile-shared test` | 126 pass / 4 fail | 126 pass / 4 fail, identical |
| `package.json` + `pnpm-lock.yaml` diff | — | empty |
| `apps/api/` in diff | — | absent |

## Two corrections to project lore, worth keeping

1. **A fresh-worktree `tsc` run on `apps/mobile-customer` is quietly weaker than a
   dev machine's.** `experiments.typedRoutes: true` generates `.expo/types/router.d.ts`,
   which is gitignored — absent, `Href` degrades to `string` and the known
   `payment.ts` error *disappears*. A fresh worktree reported 0 errors, not 1. It
   would also not have caught a bad `router.push()` path. Baseline was only
   reproducible after copying that generated file in from the main tree.
2. **The "4 pre-existing failures" attribution in the handoffs is wrong.** All four
   failing *tests* are in `auth-screens.test.ts`. `resolve-auth-error.test.ts` fails
   at *collection* (`SecureStore.AFTER_FIRST_UNLOCK` undefined) so it is the second
   failed *file* but contributes 0 failed tests — not "auth-screens ×3 +
   resolve-auth-error ×1".

## No test added, deliberately

The only pure logic produced is `toggleValue(list, value)`, a two-line immutable
toggle whose test would restate its implementation. The pattern does appear at
**9 sites** across the three mobile apps, so the duplication is real — but
extracting it to `mobile-shared/utils` and migrating those sites is a cross-app
refactor, not part of a dietary-question fix, and doing one site would leave a
half-migration. Worth its own issue.

## Deviations

- **Allergen chips are neutral coral, not the destructive tint
  `app/profile/preferences.tsx` uses for the same list.** Plan instruction
  ("no red, no warning iconography" during signup) beat strict cross-screen
  parity. The two screens now render the same allergen list in different
  colours — deliberate, not an oversight, and worth a second opinion.
- Chip markup extracted to a local `OptionChips` component rather than pasted
  twice. The body is the existing markup verbatim (ripple, `accessibilityRole`,
  function-style children, layout classes on the inner `View`); the
  already-device-verified cuisine block was left inline and untouched.

## Not done

- **Not device-verified.** This screen runs once per account, so exercising it
  needs a fresh signup on a simulator. The 14 allergen chips include long labels
  ("Shellfish (crustaceans)") and their wrapping has not been seen rendered —
  the most likely thing to need a visual tweak.
- **Existing users are not backfilled.** #912 is fixed for new signups only.
  Anyone who already onboarded still has to find Profile → Food preferences.
  Separate issue if wanted.
