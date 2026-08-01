---
quick_id: 260801-wwx
slug: extract-a-shared-usereducedmotion-hook-a
issue: 881
branch: worktree-agent-a120a385559f1ed03
subsystem: ui
tags: [react-native, accessibility, vitest, react-test-renderer, mobile-shared]
key-files:
  created:
    - packages/mobile-shared/src/ui/useReducedMotion.ts
    - packages/mobile-shared/src/__tests__/useReducedMotion.test.ts
  modified:
    - packages/mobile-shared/src/ui/index.ts
    - packages/mobile-shared/package.json
    - packages/mobile-shared/src/ui/SheetBase.tsx
    - packages/mobile-shared/src/ui/UndoSnackbar.tsx
    - packages/mobile-shared/src/ui/Toast.tsx
    - packages/mobile-shared/src/ui/Skeleton.tsx
    - packages/mobile-shared/src/ui/OnboardingScaffold.tsx
    - apps/mobile-customer/components/chef/MenuCategorySheet.tsx
requirements-completed: [GH-881-AC3, GH-881-AC4]
completed: 2026-08-01
---

# Quick Task 260801-wwx: Extract a shared useReducedMotion hook Summary

**Extracted the five copy-pasted `AccessibilityInfo`-based "OS Reduce Motion" read+subscribe blocks into one shared `useReducedMotion()` hook, repointed all five consumers at it, and threaded the missing `reduceMotion` flag into `MenuCategorySheet`'s drag-release and Modal animation — closing GH #881 AC3 and AC4.**

## Accomplishments

- New `packages/mobile-shared/src/ui/useReducedMotion.ts` — pure extraction of the canonical block (one-shot `AccessibilityInfo.isReduceMotionEnabled()` read + `reduceMotionChanged` listener + cleanup), exported from the `ui` barrel.
- 4 new vitest cases in `src/__tests__/useReducedMotion.test.ts` using `react-test-renderer` (`create`/`act`) and a locally-scoped `vi.mock('react-native', ...)` (kept separate from the shared `src/__mocks__/react-native.ts`, which has no `AccessibilityInfo` and is used by unrelated screen tests): default-false-on-first-render, resolves-true, live-update-via-listener, and unsubscribe-on-unmount.
- `SheetBase`, `UndoSnackbar`, `Toast`, `Skeleton`, and `OnboardingScaffold` all now call `useReducedMotion()` instead of keeping their own `AccessibilityInfo` subscription. `OnboardingScaffold` additionally gained a real behavior fix: its old `useRef`-based one-shot read never re-rendered the progress-fill effect when Reduce Motion changed mid-session; the shared hook (backed by `useState`) does.
- `MenuCategorySheet` (the actual bug in #881 AC3) now calls `useReducedMotion()`, passes `reduceMotion` into `useSheetDrag` (which already branched on it — the component was just never supplying the flag), and its `Modal`'s `animationType` is `'none'` instead of `'slide'` when Reduce Motion is on.
- Added `react-test-renderer` (`^19.2.3`) and `@types/react-test-renderer` (`^19.1.0`) to `packages/mobile-shared/package.json` devDependencies.

## Task Commits

1. **Task 1: Extract useReducedMotion hook with vitest coverage** - `4acf614c` (feat)
2. **Task 2: Repoint the five duplicate AccessibilityInfo blocks onto the shared hook** - `a3142846` (refactor)
3. **Task 3: Fix MenuCategorySheet — thread reduceMotion into useSheetDrag and the Modal** - `5235400f` (fix)

## Files Created/Modified

- `packages/mobile-shared/src/ui/useReducedMotion.ts` (new) - the shared hook.
- `packages/mobile-shared/src/__tests__/useReducedMotion.test.ts` (new) - 4 vitest cases.
- `packages/mobile-shared/src/ui/index.ts` - barrel export added.
- `packages/mobile-shared/package.json` - `react-test-renderer` + `@types/react-test-renderer` devDeps.
- `packages/mobile-shared/src/ui/SheetBase.tsx` - repointed; `useState`/`useEffect` for `mounted`/`visible` untouched.
- `packages/mobile-shared/src/ui/UndoSnackbar.tsx` - repointed; other `useState`/`useEffect` usage untouched.
- `packages/mobile-shared/src/ui/Toast.tsx` - repointed; other `useState`/`useEffect` usage untouched.
- `packages/mobile-shared/src/ui/Skeleton.tsx` - repointed; `useState` dropped entirely from the React import (nothing else in the file used it).
- `packages/mobile-shared/src/ui/OnboardingScaffold.tsx` - repointed; `useRef`-based flag replaced with the hook's boolean, `reduceMotion` added to the progress-fill effect's dependency array.
- `apps/mobile-customer/components/chef/MenuCategorySheet.tsx` - `reduceMotion` threaded into `useSheetDrag` and the `Modal`'s `animationType`.

## Decisions Made

- Followed the plan's exact extraction shape and consumer-by-consumer edit list — no architectural deviation.
- Left the `IS_REACT_ACT_ENVIRONMENT` global unset in the new test file (tried setting it to silence a cosmetic React warning, but doing so surfaced a stricter act-timing warning on two subtests; reverted to the plan's original approach, which passes cleanly with only the harmless "environment not configured" stderr noise — same tradeoff pattern the plan anticipated by not requiring the flag).

## Deviations from Plan

None — plan executed exactly as written. `react-test-renderer` and `@types/react-test-renderer` were already present in the hoisted `node_modules` layout after a `pnpm install --frozen-lockfile --offline` (no lockfile change, no network fetch), exactly as the plan's threat-model entry `T-260801wwx-SC` predicted.

## Issues Encountered

- **This worktree had no `node_modules` at all** (worktrees don't inherit the main checkout's install). Ran `pnpm install --frozen-lockfile --offline` at the repo root — resolved entirely from the local pnpm content-addressable store (`~/Library/pnpm/store/v3`), zero downloads, zero lockfile changes (`git status` on `pnpm-lock.yaml` confirmed clean before and after). This was necessary to run any of the plan's `tsc`/`vitest` verification commands, not a deviation from the plan's code changes.
- `vi.mock('react-native', factory)` initially threw a hoisting `ReferenceError` because the factory referenced top-level `vi.fn()` variables declared outside `vi.hoisted()`. Fixed by wrapping the two mock functions in `vi.hoisted(() => ({...}))` — a mechanical vitest-hoisting fix, not a deviation from the test's intended behavior (Rule 3).

## Verification

- `cd packages/mobile-shared && npx vitest run` — 97 passed, 4 failed (the known pre-existing `auth-screens` x3 + `resolve-auth-error` x1; unchanged from the pre-change baseline of 93 passed / 4 failed). The 4 new `useReducedMotion` tests are in the 97.
- `cd packages/mobile-shared && npx tsc --noEmit` — 2 errors, both pre-existing `OfflineBanner.tsx` `className` overload errors unrelated to this change (baseline: 2).
- `cd apps/mobile-customer && npx tsc --noEmit` — 0 errors (baseline re-measured this session: 0; differs from an older HANDOFF note of "1 pre-existing" — re-measured per plan instruction rather than trusted).
- `cd apps/mobile-vendor && npx tsc --noEmit` — 0 errors (baseline re-measured this session: 0; older HANDOFF note said "2 pre-existing").
- `grep -rn "AccessibilityInfo" packages/mobile-shared/src/ui/*.tsx` — no matches (all five duplicates gone; only `useReducedMotion.ts` itself, a `.ts` file, still references it).
- `grep -n "reduceMotion" apps/mobile-customer/components/chef/MenuCategorySheet.tsx` — 3 matches confirming the fix threaded through (`useReducedMotion()` call, passed into `useSheetDrag`, used in `animationType`).

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Issue #881 AC3 and AC4 are closed by this change.
- Remaining #881 scope (per the 2026-08-01 handoff) is unrelated to this task: drag-threshold tuning by hand-feel and iOS on-device verification of the drag gesture itself (this task only fixed the reduce-motion wiring, not the drag mechanics).
- No blockers for follow-up work.

---
*Quick task: 260801-wwx*
*Completed: 2026-08-01*

## Self-Check: PASSED

All 10 created/modified files confirmed tracked via `git ls-files`. All 3 task commits (`4acf614c`, `a3142846`, `5235400f`) confirmed present via `git log --oneline --all`.
