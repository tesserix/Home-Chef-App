---
phase: 260801-wwx-extract-a-shared-usereducedmotion-hook-a
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - packages/mobile-shared/src/ui/useReducedMotion.ts
  - packages/mobile-shared/src/ui/index.ts
  - packages/mobile-shared/src/__tests__/useReducedMotion.test.ts
  - packages/mobile-shared/package.json
  - packages/mobile-shared/src/ui/SheetBase.tsx
  - packages/mobile-shared/src/ui/UndoSnackbar.tsx
  - packages/mobile-shared/src/ui/Toast.tsx
  - packages/mobile-shared/src/ui/Skeleton.tsx
  - packages/mobile-shared/src/ui/OnboardingScaffold.tsx
  - apps/mobile-customer/components/chef/MenuCategorySheet.tsx
autonomous: true
requirements: [GH-881-AC3, GH-881-AC4]

must_haves:
  truths:
    - "SheetBase no longer keeps its own AccessibilityInfo subscription — it consumes the shared hook (issue #881 AC4)"
    - "MenuCategorySheet's drag-release settle-back snaps instantly instead of animating when OS Reduce Motion is on (issue #881 AC3)"
    - "All five pre-existing duplicate AccessibilityInfo reduce-motion blocks (SheetBase, UndoSnackbar, Toast, Skeleton, OnboardingScaffold) are gone, replaced by the shared hook"
    - "The shared hook has its own vitest coverage; the mobile-shared suite's failure count does not grow past the 4 pre-existing failures"
  artifacts:
    - path: "packages/mobile-shared/src/ui/useReducedMotion.ts"
      provides: "useReducedMotion(): boolean hook — reads AccessibilityInfo.isReduceMotionEnabled() once, subscribes to reduceMotionChanged"
      min_lines: 15
    - path: "packages/mobile-shared/src/__tests__/useReducedMotion.test.ts"
      provides: "vitest coverage: default false, resolves true, live update, unsubscribe on unmount"
      min_lines: 20
    - path: "packages/mobile-shared/src/ui/index.ts"
      provides: "barrel export of the new hook"
      contains: "useReducedMotion"
  key_links:
    - from: "apps/mobile-customer/components/chef/MenuCategorySheet.tsx"
      to: "packages/mobile-shared/src/ui/useReducedMotion.ts"
      via: "useSheetDrag({ translateY, onDismiss, reduceMotion })"
      pattern: "reduceMotion"
    - from: "packages/mobile-shared/src/ui/SheetBase.tsx"
      to: "packages/mobile-shared/src/ui/useReducedMotion.ts"
      via: "import { useReducedMotion } from './useReducedMotion'"
      pattern: "useReducedMotion"
---

<objective>
Extract the AccessibilityInfo-based "OS Reduce Motion" read+subscribe pattern — currently duplicated verbatim across five files in `packages/mobile-shared/src/ui/` — into a single shared `useReducedMotion()` hook, repoint all five existing consumers at it, and use the hook to fix the actual reported bug: `MenuCategorySheet`'s bespoke drag-to-dismiss sheet never read Reduce Motion at all, so its settle-back (and its Modal's own open/close chrome) always animated regardless of the OS setting.

Purpose: Closes GitHub issue #881 acceptance criteria 3 (MenuCategorySheet honors reduce-motion) and 4 (SheetBase uses the shared hook, not its own copy), and removes duplicated logic per the repo's small-files/DRY conventions.

Output: `packages/mobile-shared/src/ui/useReducedMotion.ts` (new hook + test), five repointed consumers, one bug fix in `apps/mobile-customer`.
</objective>

<execution_context>
@$HOME/.claude/get-shit-done/workflows/execute-plan.md
@$HOME/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/STATE.md
@./CLAUDE.md

<interfaces>
<!-- Canonical pattern being extracted, verbatim from SheetBase.tsx lines 91, 101-115. -->
<!-- This exact shape (state + one-shot read + changed-listener + cleanup) is what -->
<!-- useReducedMotion.ts must reproduce, so every consumer's runtime behavior is -->
<!-- unchanged except for OnboardingScaffold (see Task 2). -->

```tsx
const [reduceMotion, setReduceMotion] = useState(false);

useEffect(() => {
  let alive = true;
  AccessibilityInfo.isReduceMotionEnabled()
    .then((enabled) => {
      if (alive) setReduceMotion(enabled);
    })
    .catch(() => {});
  const sub = AccessibilityInfo.addEventListener('reduceMotionChanged', (enabled) => {
    setReduceMotion(enabled);
  });
  return () => {
    alive = false;
    sub.remove();
  };
}, []);
```

From `useSheetDrag.ts` — the consumer contract `MenuCategorySheet` must satisfy:
```typescript
export interface UseSheetDragOptions {
  translateY: Animated.Value;
  onDismiss: () => void;
  /** When true, skip the spring-back animation (snap instead). */
  reduceMotion?: boolean;
  enabled?: boolean;
}
export function useSheetDrag(options: UseSheetDragOptions): { panHandlers: PanResponderInstance['panHandlers'] };
```
`useSheetDrag` already branches correctly on `reduceMotion` (see `settleBack` in `useSheetDrag.ts`) — `MenuCategorySheet` is simply never passing it today.

From `packages/mobile-shared/src/ui/index.ts` — barrel to extend:
```typescript
export { SheetBase, type SheetBaseProps } from './SheetBase';
export { useSheetDrag, type UseSheetDragOptions } from './useSheetDrag';
export { UndoSnackbarProvider, useUndoSnackbar } from './UndoSnackbar';
export { OnboardingScaffold } from './OnboardingScaffold';
```
</interfaces>
</context>

<tasks>

<task type="auto" tdd="true">
  <name>Task 1: Extract useReducedMotion hook with vitest coverage</name>
  <files>packages/mobile-shared/src/ui/useReducedMotion.ts, packages/mobile-shared/src/ui/index.ts, packages/mobile-shared/src/__tests__/useReducedMotion.test.ts, packages/mobile-shared/package.json</files>
  <behavior>
    - Test 1: returns `false` synchronously on first render, before the async `isReduceMotionEnabled()` check resolves
    - Test 2: flips to `true` once `AccessibilityInfo.isReduceMotionEnabled()` resolves `true`
    - Test 3: updates live from `false` to `true` when the `reduceMotionChanged` listener callback fires with `true`
    - Test 4: calls `subscription.remove()` when the consuming component unmounts
  </behavior>
  <action>
    Create `packages/mobile-shared/src/ui/useReducedMotion.ts` exporting `useReducedMotion(): boolean`, reproducing the canonical block from `<interfaces>` above verbatim (same `useState(false)` + `useEffect` with `isReduceMotionEnabled().then(...).catch(() => {})` + `addEventListener('reduceMotionChanged', ...)` + cleanup that calls `sub.remove()`). No new behavior — this is a pure extraction. Add a short header comment noting it replaces five duplicated copies (GitHub #881 AC4).

    Add `export { useReducedMotion } from './useReducedMotion';` to `packages/mobile-shared/src/ui/index.ts`.

    Add `"react-test-renderer": "^19.2.3"` and `"@types/react-test-renderer": "^19.1.0"` to `packages/mobile-shared/package.json` devDependencies — these exact versions already resolve in `pnpm-lock.yaml` (used transitively elsewhere in the monorepo) and the repo's `node-linker=hoisted` setting (`.npmrc`) means both are already physically resolvable from `packages/mobile-shared` without a new download. Run `pnpm install` at the repo root to register the explicit dependency in the lockfile; if that is blocked (no network), skip it and note the gap in the summary — `npx vitest run` will still resolve both packages correctly via the hoisted `node_modules` layout either way.

    Write `packages/mobile-shared/src/__tests__/useReducedMotion.test.ts` (plain `.ts`, no JSX — use `createElement` from `react` so no `jsx` compiler settings are needed). Pattern:
    - `vi.mock('react-native', () => ({ AccessibilityInfo: { isReduceMotionEnabled: vi.fn(), addEventListener: vi.fn() } }))` local to this file (does not touch the shared `src/__mocks__/react-native.ts`, which has no `AccessibilityInfo` today and is used by unrelated screen tests).
    - A tiny harness function component: `function Harness({ onValue }: { onValue: (v: boolean) => void }) { onValue(useReducedMotion()); return null; }`.
    - Use `create` and `act` imported from `react-test-renderer` to mount the harness and drive effects/microtasks (`await act(async () => { ...; await Promise.resolve(); })`), asserting on the value captured by `onValue`.
    - For the unmount test, capture the `remove` mock passed back from `addEventListener`'s mocked return value and assert it was called after `renderer.unmount()`.
  </action>
  <verify>
    <automated>cd packages/mobile-shared && npx vitest run src/__tests__/useReducedMotion.test.ts</automated>
  </verify>
  <done>`useReducedMotion.ts` exists and is exported from `index.ts`; all 4 new tests pass; `react-test-renderer` + `@types/react-test-renderer` are declared in `package.json` devDependencies.</done>
</task>

<task type="auto">
  <name>Task 2: Repoint the five duplicate AccessibilityInfo blocks onto the shared hook</name>
  <files>packages/mobile-shared/src/ui/SheetBase.tsx, packages/mobile-shared/src/ui/UndoSnackbar.tsx, packages/mobile-shared/src/ui/Toast.tsx, packages/mobile-shared/src/ui/Skeleton.tsx, packages/mobile-shared/src/ui/OnboardingScaffold.tsx</files>
  <action>
    In `SheetBase.tsx`, `UndoSnackbar.tsx`, `Toast.tsx`, and `Skeleton.tsx`: delete the local `const [reduceMotion, setReduceMotion] = useState(false);` line and its paired `useEffect` (the exact block shown in `<interfaces>`), replace with `const reduceMotion = useReducedMotion();`, and add `import { useReducedMotion } from './useReducedMotion';`. Remove `AccessibilityInfo` from each file's `react-native` import list (grep confirms it has no other use in any of these four files). In `SheetBase.tsx`, `UndoSnackbar.tsx`, and `Toast.tsx`, leave `useState`/`useEffect` in the React import untouched — each still uses them for other state (`mounted`/`visible` in SheetBase, `current` in UndoSnackbar/Toast, timeout cleanup effects). In `Skeleton.tsx` specifically, drop `useState` from the React import too — after this change nothing else in the file calls `useState` (the shimmer `opacity` value is a `useRef`, and the only other hook is the animation-loop `useEffect`, which stays).

    In `OnboardingScaffold.tsx`: replace the `const reduceMotion = useRef(false);` plus its one-shot `useEffect` (the block reading `AccessibilityInfo.isReduceMotionEnabled().then((on) => { reduceMotion.current = on; })`) with `const reduceMotion = useReducedMotion();` and `import { useReducedMotion } from './useReducedMotion';`. Remove `AccessibilityInfo` from the file's `react-native` import (no other use). Update the progress-fill `useEffect` immediately below: change `if (reduceMotion.current) {` to `if (reduceMotion) {`, and add `reduceMotion` to that effect's dependency array (`[percent, fill]` → `[percent, fill, reduceMotion]`). Add a one-line comment noting this is a deliberate behavior improvement over the old ref-based version: the fill animation now correctly reacts if Reduce Motion changes mid-session, which the ref (read once, never causing a re-render) silently ignored.

    Per issue #881 AC4, `SheetBase` MUST end up on the shared hook — do not skip it.
  </action>
  <verify>
    <automated>cd packages/mobile-shared && grep -rn "AccessibilityInfo" src/ui/SheetBase.tsx src/ui/UndoSnackbar.tsx src/ui/Toast.tsx src/ui/Skeleton.tsx src/ui/OnboardingScaffold.tsx | grep -v '^#' | wc -l | grep -qx 0 && echo "OK: no AccessibilityInfo left in the five consumers"</automated>
  </verify>
  <done>All five files import and call `useReducedMotion()`; none imports `AccessibilityInfo` directly; `npx tsc --noEmit` in `packages/mobile-shared`'s consuming apps shows no new errors (see plan-level verification).</done>
</task>

<task type="auto">
  <name>Task 3: Fix MenuCategorySheet — thread reduceMotion into useSheetDrag and the Modal</name>
  <files>apps/mobile-customer/components/chef/MenuCategorySheet.tsx</files>
  <action>
    Add `useReducedMotion` to the existing `import { useSheetDrag } from '@homechef/mobile-shared/ui';` line (becomes `import { useReducedMotion, useSheetDrag } from '@homechef/mobile-shared/ui';`). Call `const reduceMotion = useReducedMotion();` alongside the existing `translateY` ref. Pass it through: `useSheetDrag({ translateY, onDismiss: onClose, reduceMotion })` — `useSheetDrag` already branches its `settleBack` on this flag (see `<interfaces>`); `MenuCategorySheet` was simply never supplying it, which is the root cause of issue #881 AC3.

    Also change `<Modal visible={visible} transparent animationType="slide" onRequestClose={onClose}>` to `animationType={reduceMotion ? 'none' : 'slide'}` so the Modal's own native open/close chrome (owned by React Native, independent of the `translateY` drag transform) honors the setting too — this is safe against the "Reset on OPEN, not on close" `useEffect` at the top of the component (it only resets the drag gesture's `translateY` Animated.Value when `visible` becomes `true`, and has no relationship to the Modal's `animationType` prop). Add a one-line comment on the `animationType` line explaining the conditional.
  </action>
  <verify>
    <automated>cd apps/mobile-customer && npx tsc --noEmit 2>&1 | tee /tmp/mc-tsc.txt; grep -c "error TS" /tmp/mc-tsc.txt | grep -qx 0 && echo "OK: 0 tsc errors (matches known baseline)"</automated>
  </verify>
  <done>`MenuCategorySheet` calls `useReducedMotion()`, passes `reduceMotion` into `useSheetDrag`, and its `Modal`'s `animationType` is conditional on it. `npx tsc --noEmit` in `apps/mobile-customer` shows the same error count as the pre-change baseline (re-measure — do not trust the stated 0 without checking).</done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|--------------|
| None crossed | This change reads a device-local OS accessibility setting (`AccessibilityInfo.isReduceMotionEnabled`) and touches no network input, user input, or persisted data. |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|------------------|
| T-260801wwx-01 | Tampering | `useReducedMotion.ts` | accept | Pure read of a device-local accessibility API; no external input to tamper with. |
| T-260801wwx-SC | Tampering | `react-test-renderer` / `@types/react-test-renderer` devDependency add | accept | Official React org package (not a third-party/ASSUMED package); versions already resolved and pinned in the existing `pnpm-lock.yaml` via other workspace members' dependency trees — no new, unvetted code is introduced, only an explicit `devDependencies` declaration for an already-present, already-hoisted package. No blocking checkpoint needed. |

</threat_model>

<verification>
1. `cd packages/mobile-shared && npx vitest run` — assert total failing tests stays at the 4 known pre-existing failures (`auth-screens` x3, `resolve-auth-error` x1) and does not grow; the 4 new `useReducedMotion` tests pass.
2. `cd packages/mobile-shared && npx tsc --noEmit` — no new errors.
3. `cd apps/mobile-customer && npx tsc --noEmit` — error count matches the pre-change baseline (re-measure before/after in the same session, do not trust the stated "0" blindly).
4. `cd apps/mobile-vendor && npx tsc --noEmit` — error count matches the pre-change baseline (OnboardingScaffold is vendor-exclusive).
5. `grep -rn "AccessibilityInfo" packages/mobile-shared/src/ui/*.tsx` — only `useReducedMotion.ts` itself should reference it (all five prior duplicates gone).
6. `grep -n "reduceMotion" apps/mobile-customer/components/chef/MenuCategorySheet.tsx` — confirms the fix threaded through.
</verification>

<success_criteria>
- `useReducedMotion()` exists in `packages/mobile-shared/src/ui/`, is exported from the barrel, and has its own passing vitest suite.
- `SheetBase`, `UndoSnackbar`, `Toast`, `Skeleton`, and `OnboardingScaffold` all consume the shared hook; none keeps its own `AccessibilityInfo` subscription.
- `MenuCategorySheet` now honors OS Reduce Motion for both its drag-release settle-back and its own Modal open/close animation.
- No typecheck regression in `packages/mobile-shared`, `apps/mobile-customer`, or `apps/mobile-vendor`.
- `packages/mobile-shared`'s vitest failure count does not exceed the 4 pre-existing failures.
</success_criteria>

<output>
Create `.planning/quick/260801-wwx-extract-a-shared-usereducedmotion-hook-a/260801-wwx-SUMMARY.md` when done
</output>
