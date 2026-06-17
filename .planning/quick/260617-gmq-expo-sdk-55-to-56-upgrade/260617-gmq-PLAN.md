---
phase: quick-260617-gmq
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - apps/mobile-customer/package.json
  - apps/mobile-vendor/package.json
  - apps/mobile-vendor/app/(tabs)/_layout.tsx
  - apps/mobile-delivery/package.json
  - apps/mobile-delivery/app.json
  - packages/mobile-shared/package.json
  - pnpm-lock.yaml
autonomous: true
requirements: [SDK56-UPGRADE]

must_haves:
  truths:
    - "All 3 apps + mobile-shared resolve a single coherent Expo SDK 56 / RN 0.85 dependency graph (no two react-native versions)"
    - "`npx expo install --check` passes (green) in each of the 3 apps"
    - "`npx expo-doctor` reports no version-mismatch / peer-range errors in each app (informational warnings about unmaintained third-party native modules are acceptable)"
    - "No ~55.x expo-* range remains in any app package.json"
    - "mobile-shared peerDependencies are widened (permissive ranges) so the apps' SDK-56 expo-* versions satisfy them"
    - "mobile-delivery app.json newArchEnabled is true (honesty no-op) — all 3 occurrences"
    - "vendor (tabs)/_layout.tsx react-navigation type import resolves under SDK 56 (replaced with an expo-router/react-native equivalent, or @react-navigation/bottom-tabs added as a direct dep) — `npx tsc --noEmit` shows no 'Cannot find module @react-navigation/bottom-tabs' error"
    - "`pnpm install` resolves cleanly into ONE root pnpm-lock.yaml; @tesserix__native.patch still applies; node-linker=hoisted unchanged"
    - "Typecheck per app shows no NEW errors versus the SDK-55 baseline captured at start"
    - "OWNER GATE (NOT verified here): `expo prebuild --clean` + native build under Xcode >=26.4 / iOS 16.4 with buildReactNativeFromSource:true — the true pass/fail. Static checks cannot catch Hermes V1, reanimated/worklets native binding, firebase/webview/maps/google-signin native compilation."
  artifacts:
    - path: "apps/mobile-customer/package.json"
      provides: "expo ~56.x, react-native 0.85.x, reanimated/worklets bumped via --fix"
      contains: '"expo": "~56'
    - path: "apps/mobile-vendor/package.json"
      provides: "expo ~56.x SDK 56 deps via --fix (+ @react-navigation/bottom-tabs as direct dep IF that reconciliation path is chosen)"
      contains: '"expo": "~56'
    - path: "apps/mobile-vendor/app/(tabs)/_layout.tsx"
      provides: "react-navigation type import reconciled for SDK 56 (BottomTabBarProps replaced or backed by a direct dep)"
    - path: "apps/mobile-delivery/package.json"
      provides: "expo ~56.x SDK 56 deps via --fix (was the un-bumped app; gains expo-updates alignment)"
      contains: '"expo": "~56'
    - path: "apps/mobile-delivery/app.json"
      provides: "newArchEnabled: true (no-op honesty flip)"
      contains: '"newArchEnabled": true'
    - path: "packages/mobile-shared/package.json"
      provides: "widened expo-* peerDependencies compatible with SDK 56"
    - path: "pnpm-lock.yaml"
      provides: "single coherent reconciled lockfile for the whole workspace"
  key_links:
    - from: "apps/mobile-*/package.json"
      to: "pnpm-lock.yaml"
      via: "single root pnpm install after per-app expo install --fix"
      pattern: "react-native.*0\\.85"
    - from: "packages/mobile-shared/package.json peerDependencies"
      to: "apps SDK-56 expo-* versions"
      via: "permissive peer ranges (no exact pins)"
      pattern: "expo-(device|notifications|secure-store)"
    - from: "apps/mobile-vendor/app/(tabs)/_layout.tsx"
      to: "@react-navigation/bottom-tabs (or expo-router/RN equivalent)"
      via: "type-only import of BottomTabBarProps, reconciled because expo-router drops react-navigation under SDK 56"
      pattern: "BottomTabBarProps"
---

<objective>
Upgrade all three Expo apps (`mobile-customer`, `mobile-vendor`, `mobile-delivery`) and the shared `packages/mobile-shared` library from Expo SDK 55 to SDK 56 in the pnpm monorepo, reconciling everything into ONE coherent root lockfile. This completes the existing partial bump (`expo-updates` is already at ~56 in customer + vendor).

IMPORTANT — this is NOT a stay-on-RN-0.83 consistency nudge. Per RESEARCH, SDK 56 ships React Native 0.85 (a two-minor RN jump) and React 19.2 (unchanged). Treat as a real SDK upgrade. Versions are resolved by `npx expo install --fix` (the authoritative source), NOT by hardcoded numbers.

Purpose: One coherent SDK 56 dependency graph across all mobile workspaces, ready for the owner's native build gate.
Output: Updated package.json (3 apps + shared), reconciled vendor `_layout.tsx` react-navigation import, flipped mobile-delivery app.json, single reconciled pnpm-lock.yaml, atomic commits per app + shared.

Scope guards (from constraints):
- Static verification ONLY — NO EAS, NO native build, NO prebuild. The native build is the OWNER's final gate.
- Do NOT push. Do NOT update ROADMAP.md. Do NOT touch GitHub PRs.
- Single-line commits, no signatures (per CLAUDE.md + global git prefs).
</objective>

<execution_context>
@$HOME/.claude/get-shit-done/workflows/execute-plan.md
@$HOME/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/quick/260617-gmq-expo-sdk-55-to-56-upgrade/260617-gmq-RESEARCH.md
@.planning/STATE.md

# Verified facts (grep, this session):
# - node-linker=hoisted in .npmrc — MUST stay (RN codegen depends on it)
# - patches/@tesserix__native.patch — must re-apply after install
# - root package name "homechef"; workspace filters: @homechef/mobile-customer, @homechef/mobile-vendor, @homechef/mobile-delivery
# - Current (SDK 55): expo ~55.0.26, react-native 0.83.6, react 19.2.0, reanimated ~4.2.1, worklets 0.7.4
# - expo-updates already ~56.0.19 in customer + vendor; mobile-delivery has NO expo-updates entry
# - mobile-shared peerDependencies are STALE: expo-device ~7.1.4, expo-notifications ~0.30.2, expo-secure-store ~55.0.11 (widen these)
# - mobile-delivery app.json newArchEnabled:false at 3 locations (lines 10, 88, 91) — flip to true
# - babel.config.js uses babel-preset-expo (worklets plugin bundled in preset) — do NOT add reanimated/plugin manually
# - All 3 apps use the standard JS `Tabs` from expo-router (no NativeTabs). Apps directory layout is `app/` ONLY — there is NO `src/` dir under apps/mobile-* (verified `ls`).
# - react-navigation imports: there is exactly ONE direct import — a TYPE-ONLY import in vendor `app/(tabs)/_layout.tsx:10`:
#     `import type { BottomTabBarProps } from '@react-navigation/bottom-tabs';` (used to type `CustomTabBar({state,descriptors,navigation}: BottomTabBarProps)` at line 31).
#     It is NOT a direct dependency — today it resolves transitively via expo-router. SDK 56 expo-router DROPS its react-navigation dependency, so this import will become a "Cannot find module" tsc error and MUST be reconciled (see Task 1).
#     customer + delivery: 0 direct @react-navigation imports (verified).
# - no @expo/vector-icons src usage, no expo-file-system copy/move — most breaking changes don't apply
# - customer uses react-native-webview directly for Razorpay (real native module, keep it) — native-build gate exposure only
# - .metro-cache/ exists in customer + vendor; stale apps/mobile-{customer,vendor}/ios dirs are gitignored CNG output
# - No `typecheck` npm script exists — typecheck = `npx tsc --noEmit` per app
# - Apps test = `jest --passWithNoTests`; mobile-shared test = `vitest run`
</context>

<tasks>

<task type="auto">
  <name>Task 0: Capture baselines, then upgrade mobile-customer to SDK 56</name>
  <files>apps/mobile-customer/package.json</files>
  <action>
    FIRST, capture baselines (needed by later tasks to prove "no NEW errors"):
    1. `node -v` — confirm Node >= 20.19.4 (repo is Node 22). If lower, STOP and report.
    2. Confirm clean working tree relevant to mobile (`git status --short apps/mobile-* packages/mobile-shared`). Untracked `app.json` at repo root from gitStatus is unrelated — ignore.
    3. Capture a typecheck baseline for each app BEFORE any change. From each app dir run `npx tsc --noEmit 2>&1 | tee /tmp/gmq-tsc-baseline-{app}.txt; echo EXIT=$?`. Record the error count per app (this is the SDK-55 baseline). Do the same intent for tests: `pnpm --filter @homechef/{app} test 2>&1 | tail -20` to know the pre-upgrade state. Save these counts in the SUMMARY.

    THEN upgrade mobile-customer. From apps/mobile-customer:
    - `npx expo install expo@^56.0.0`
    - `npx expo install --fix` (this rewrites every expo-* + RN-ecosystem dep — expo, react-native, reanimated, worklets, expo-updates, etc. — to the SDK-56-pinned ranges; --fix is the authoritative resolver, do NOT hand-edit version numbers)

    Sanity-check the diff against RESEARCH section 1 (expo ~56.x, RN 0.85.x, react 19.2 unchanged, reanimated ~4.3.x, worklets ~0.8.x). If --fix produces something wildly different, note it but trust --fix.

    Do NOT run a workspace `pnpm install` yet — that single reconciliation happens in Task 3. (Per-app `expo install` may write partial lockfile state; that is expected and reconciled later.)
    Do NOT touch babel.config.js (worklets plugin is bundled in babel-preset-expo). Do NOT add @react-navigation or @expo/vector-icons unless a later check demands it.

    If network is unavailable for `expo install`, STOP and report — the upgrade cannot proceed offline.
  </action>
  <verify>
    <automated>cd apps/mobile-customer && grep -E '"expo":\s*"~56' package.json && grep -E '"react-native":\s*"0\.85' package.json && ! grep -E '"expo[^"]*":\s*"~?55\.' package.json && echo OK</automated>
  </verify>
  <done>mobile-customer/package.json shows expo ~56.x, react-native 0.85.x, no ~55.x expo-* ranges remain. Baselines captured to /tmp.</done>
</task>

<task type="auto">
  <name>Task 1: Upgrade mobile-vendor to SDK 56 + reconcile react-navigation type import</name>
  <files>apps/mobile-vendor/package.json, apps/mobile-vendor/app/(tabs)/_layout.tsx</files>
  <action>
    From apps/mobile-vendor:
    - `npx expo install expo@^56.0.0`
    - `npx expo install --fix`

    Same rules as Task 0: --fix is authoritative; sanity-check diff vs RESEARCH section 1; do NOT hand-edit versions; do NOT run workspace pnpm install yet (reconciled in Task 3); do NOT touch babel.config.js; keep `"jsEngine": "hermes"` as-is. Vendor carries expo-file-system + expo-image-manipulator but has no copy/move calls (RESEARCH verified) — no code change needed.

    THEN reconcile the react-navigation type import (REQUIRED — this is the one direct react-navigation import in the whole repo and it WILL break under SDK 56):
    `apps/mobile-vendor/app/(tabs)/_layout.tsx:10` has `import type { BottomTabBarProps } from '@react-navigation/bottom-tabs';`, used at line 31 to type the custom tab bar: `function CustomTabBar({ state, descriptors, navigation }: BottomTabBarProps)`. Today this type resolves transitively via expo-router; under SDK 56 expo-router drops its react-navigation dependency, so the bare specifier becomes a "Cannot find module '@react-navigation/bottom-tabs'" tsc error.

    Pick whichever of these two reconciliation paths resolves CLEANLY under SDK 56 (verify with `npx tsc --noEmit` after, see verify block) — do NOT do both:
      (a) PREFERRED — replace the `BottomTabBarProps` import with the expo-router / react-native-provided equivalent type if one exists and gives the same `{ state, descriptors, navigation }` shape (e.g. an expo-router tab-bar props type, or hand-type the three props against the types expo-router re-exports). Remove the `@react-navigation/bottom-tabs` import line entirely if this path resolves.
      (b) FALLBACK — if no clean equivalent type exists, make the dependency explicit: from apps/mobile-vendor run `npx expo install @react-navigation/bottom-tabs` so it becomes a direct dep that resolves independently of expo-router's internals. Keep the import as-is.

    The executor must confirm the chosen path makes the import resolve (no "Cannot find module" error) before moving on, and record which path was taken in the SUMMARY. If path (b), the new `@react-navigation/bottom-tabs` dep is reconciled into the lockfile by Task 3's single root pnpm install (do NOT run workspace pnpm install here).
  </action>
  <verify>
    <automated>cd apps/mobile-vendor && grep -E '"expo":\s*"~56' package.json && grep -E '"react-native":\s*"0\.85' package.json && ! grep -E '"expo[^"]*":\s*"~?55\.' package.json && echo VERSIONS_OK; cd apps/mobile-vendor && npx tsc --noEmit 2>&1 | tee /tmp/gmq-tsc-vendor-reconcile.txt | grep -F "@react-navigation/bottom-tabs" && { echo "FATAL: bottom-tabs import unresolved"; exit 1; } || echo BOTTOM_TABS_RESOLVED</automated>
  </verify>
  <done>mobile-vendor/package.json shows expo ~56.x, react-native 0.85.x, no ~55.x expo-* ranges remain. The `BottomTabBarProps` import in `app/(tabs)/_layout.tsx` resolves under SDK 56 (replaced with an equivalent, or `@react-navigation/bottom-tabs` added as a direct dep) — tsc reports no "Cannot find module @react-navigation/bottom-tabs" error. Chosen path recorded for SUMMARY.</done>
</task>

<task type="auto">
  <name>Task 2: Upgrade mobile-delivery to SDK 56 + flip newArchEnabled</name>
  <files>apps/mobile-delivery/package.json, apps/mobile-delivery/app.json</files>
  <action>
    From apps/mobile-delivery:
    - `npx expo install expo@^56.0.0`
    - `npx expo install --fix`

    Note: mobile-delivery is the app WITHOUT a prior expo-updates bump — `--fix` will add/align expo-updates to SDK 56 along with everything else. Same rules: --fix authoritative, no hand-edits, no workspace pnpm install yet, no babel changes.

    THEN edit apps/mobile-delivery/app.json: change `newArchEnabled` from `false` to `true` at ALL 3 occurrences (lines ~10, ~88, ~91 — top-level + both platform blocks). Per RESEARCH this is a no-op in RN 0.85 (legacy arch is already removed/ignored), done for honesty/consistency with customer + vendor. Use Edit per occurrence; verify count.
  </action>
  <verify>
    <automated>cd apps/mobile-delivery && grep -E '"expo":\s*"~56' package.json && grep -E '"react-native":\s*"0\.85' package.json && ! grep -E '"expo[^"]*":\s*"~?55\.' package.json && [ "$(grep -c '"newArchEnabled": true' app.json)" -eq 3 ] && [ "$(grep -c '"newArchEnabled": false' app.json)" -eq 0 ] && echo OK</automated>
  </verify>
  <done>mobile-delivery/package.json on SDK 56 (incl. expo-updates aligned); app.json newArchEnabled true x3, false x0.</done>
</task>

<task type="auto">
  <name>Task 3: Widen mobile-shared peer ranges + single root pnpm install (reconcile lockfile)</name>
  <files>packages/mobile-shared/package.json, pnpm-lock.yaml</files>
  <action>
    1. Widen the STALE expo-* peerDependencies in packages/mobile-shared/package.json so the apps' new SDK-56 versions satisfy them. Current stale values: `expo-device ~7.1.4`, `expo-notifications ~0.30.2`, `expo-secure-store ~55.0.11`. Peers must be PERMISSIVE (not exact pins). Set them to the SDK-56-compatible ranges that the apps now resolve to — read the new app package.json values written by --fix and use a matching caret/tilde, OR use a permissive range. Keep `react ^19.0.0`, `axios ^1.13.0`, `@tesserix/native ^1.0.0`, `zustand ^5.0.2`, `@react-native-community/netinfo` aligned to whatever --fix wrote in the apps. Do NOT introduce `catalog:` refs (repo has no pnpm catalog).

    2. Run ONE root `pnpm install` (repo root) to reconcile the entire workspace into a single coherent pnpm-lock.yaml. Per RESEARCH this single final reconciliation (after all per-app --fix, and after any direct @react-navigation/bottom-tabs added in Task 1 path b) is what avoids a mixed-version lockfile.

    3. Confirm node-linker=hoisted is still in .npmrc (do NOT change it).

    4. Confirm the @tesserix/native patch re-applied (pnpm reports patched dependency on install). If `pnpm install` ERRORS on the patch failing to apply, STOP and report — the patch base changed and needs regeneration (out of scope to silently force).

    5. Clear stale Metro cache so the next dev/build picks up new transforms: `rm -rf apps/mobile-customer/.metro-cache apps/mobile-vendor/.metro-cache apps/mobile-delivery/.metro-cache` (delivery may not have one — fine).

    Do NOT delete the stale apps/mobile-{customer,vendor}/ios dirs here — that is part of the owner's prebuild gate, not static verification.
  </action>
  <verify>
    <automated>pnpm install --frozen-lockfile=false >/tmp/gmq-install.txt 2>&1; tail -5 /tmp/gmq-install.txt; grep -q 'node-linker=hoisted' .npmrc && echo NPMRC_OK; pnpm why react-native 2>/dev/null | grep -Eo 'react-native [0-9]+\.[0-9]+\.[0-9]+' | sort -u | tee /tmp/gmq-rn-versions.txt; [ "$(grep -Eo '0\.[0-9]+\.[0-9]+' /tmp/gmq-rn-versions.txt | sort -u | wc -l | tr -d ' ')" = "1" ] && echo SINGLE_RN_VERSION</automated>
  </verify>
  <done>pnpm install resolves cleanly; node-linker=hoisted intact; @tesserix/native patch applied; `pnpm why react-native` shows exactly ONE version; mobile-shared peers widened; Metro caches cleared.</done>
</task>

<task type="auto">
  <name>Task 4: Static verification sweep (expo-doctor, --check, typecheck vs baseline, grep)</name>
  <files>(verification only — no file changes)</files>
  <action>
    Run the full static verification (NO EAS, NO native build, NO prebuild). For EACH app (mobile-customer, mobile-vendor, mobile-delivery), from the app dir:
    1. `npx expo install --check` — must report versions are correct (green / no out-of-range deps).
    2. `npx expo-doctor` — record output. Version-mismatch / peer-range ERRORS must be zero. Known-acceptable (do NOT fail on these): informational warnings about unmaintained or untested-on-new-arch third-party native modules (e.g. react-native-maps), and pnpm peer-range warnings for @tesserix/native / firebase whose declared peers lag.
    3. `npx tsc --noEmit 2>&1 | tee /tmp/gmq-tsc-after-{app}.txt` — compare error count to /tmp/gmq-tsc-baseline-{app}.txt from Task 0. PASS = no NEW errors vs baseline (pre-existing baseline errors are acceptable). The vendor run MUST NOT contain any "Cannot find module '@react-navigation/bottom-tabs'" error (Task 1 reconciled it). If new errors appear, triage: are they SDK-56 type-only churn (e.g. @types/react-native) that's trivially fixable, or real breakages? Fix trivial type churn; report anything structural.

    Then workspace-wide:
    4. Tests: `pnpm --filter @homechef/mobile-customer test`, same for vendor + delivery (jest --passWithNoTests), and `pnpm --filter @homechef/mobile-shared test` (vitest run) — preset must resolve under SDK 56; compare to Task 0 pre-state.
    5. Grep sweep — confirm NO ~55.x expo-* range remains anywhere (see verify block; assertion fails loudly on any match).
    6. react-navigation sweep — confirm the ONLY remaining `@react-navigation` reference (if any) is the reconciled vendor import backed by a direct dep, NOT a bare transitive import. The verify block searches `app/` (the real source dir — there is NO `apps/mobile-*/src`) plus `packages/mobile-shared/src`, prints every match, and asserts the only allowed match is the reconciled vendor line. If Task 1 chose the "replace the type" path, expect ZERO matches.

    Record all results for the SUMMARY. Do NOT run `expo start`, `expo prebuild`, EAS, or any native build — explicitly out of scope.
  </action>
  <verify>
    <automated>
set -e
fail=0
# 1) expo install --check per app, fatal on failure
for a in mobile-customer mobile-vendor mobile-delivery; do
  if (cd "apps/$a" && npx expo install --check >"/tmp/gmq-check-$a.txt" 2>&1); then :; else echo "FATAL: expo install --check failed for $a"; fail=1; fi
done
# 2) No ~55.x expo-* range anywhere — capture matches, assert empty, fatal on hit
m55=$(grep -rEn '"expo[^"]*":[[:space:]]*"~?55\.' apps/mobile-customer/package.json apps/mobile-vendor/package.json apps/mobile-delivery/package.json packages/mobile-shared/package.json || true)
if [ -n "$m55" ]; then echo "FATAL: ~55.x expo-* range remains:"; echo "$m55"; fail=1; else echo NO_55_REMAINS; fi
# 3) react-navigation sweep — search app/ (NOT src/, which does not exist) + mobile-shared/src; print matches; fatal on any UNexpected bare import
rnav=$(grep -rn "@react-navigation" \
  apps/mobile-customer/app \
  apps/mobile-vendor/app \
  apps/mobile-delivery/app \
  packages/mobile-shared/src 2>/dev/null || true)
echo "react-navigation matches:"; echo "$rnav"
# Allowed iff Task1 path(b): the single reconciled vendor _layout import backed by a direct dep. Path(a) => zero matches.
unexpected=$(printf '%s\n' "$rnav" | grep -v "apps/mobile-vendor/app/(tabs)/_layout.tsx" | grep "@react-navigation" || true)
if [ -n "$unexpected" ]; then echo "FATAL: unexpected @react-navigation import(s):"; echo "$unexpected"; fail=1; fi
# 4) Vendor tsc must not have the unresolved-module error
if grep -F "Cannot find module '@react-navigation/bottom-tabs'" /tmp/gmq-tsc-after-mobile-vendor.txt 2>/dev/null; then echo "FATAL: vendor bottom-tabs import unresolved"; fail=1; else echo VENDOR_IMPORT_RESOLVED; fi
[ "$fail" -eq 0 ] && echo ALL_CHECKS_GREEN || { echo CHECKS_FAILED; exit 1; }
    </automated>
  </verify>
  <done>All 3 apps: `expo install --check` green, expo-doctor no version/peer ERRORS, typecheck has no NEW errors vs baseline (and vendor has NO unresolved @react-navigation/bottom-tabs error), tests resolve under SDK 56. No ~55.x expo-* remains. The only @react-navigation reference is the reconciled vendor import (or zero if the type was replaced). Results recorded for SUMMARY.</done>
</task>

</tasks>

<verification>
Static only (per constraints — NO EAS, NO native build, NO prebuild):
- Per app: `npx expo install --check` green; `npx expo-doctor` no version/peer ERRORS; `npx tsc --noEmit` no NEW errors vs SDK-55 baseline.
- Vendor specifically: the `BottomTabBarProps` react-navigation type import in `app/(tabs)/_layout.tsx` resolves under SDK 56 (replaced or backed by a direct dep) — no "Cannot find module" tsc error.
- Workspace: `pnpm install` clean into ONE lockfile; `pnpm why react-native` shows exactly one version; @tesserix/native patch applied; node-linker=hoisted intact.
- Grep: no ~55.x expo-* range anywhere (assertion fails loudly on match); only @react-navigation reference is the reconciled vendor import.
- Tests: jest (apps) + vitest (mobile-shared) preset resolves under SDK 56.

OWNER'S FINAL GATE (NOT done in this plan — must be stated in SUMMARY as remaining):
- Delete stale apps/mobile-{customer,vendor}/ios dirs, then `npx expo prebuild --clean` per app.
- Native build under Xcode >= 26.4 / iOS 16.4, with buildReactNativeFromSource:true (already set in customer + vendor app.json).
- Confirms what static checks CANNOT: Hermes V1 runtime, reanimated/worklets native binding, firebase native init, react-native-webview (Razorpay) + react-native-maps + google-signin native compilation under RN 0.85.
</verification>

<success_criteria>
- 3 apps + mobile-shared on Expo SDK 56 / RN 0.85 in one coherent lockfile (single RN version).
- vendor `(tabs)/_layout.tsx` react-navigation type import reconciled (resolves under SDK 56).
- mobile-delivery newArchEnabled true x3; expo-updates aligned across all apps.
- mobile-shared peer ranges widened, no longer stale.
- All static verification green (see verification block); no NEW typecheck errors vs baseline.
- Atomic commits: one per app + one for shared/lockfile reconciliation. Single-line messages, no signatures. NOT pushed.
- SUMMARY explicitly states: (a) the owner's native build gate remains the true pass/fail, (b) which reconciliation path was taken for the vendor react-navigation import, and (c) redundant dependabot expo-* PRs #79/#84/#85/#86/#87 for the owner to close.
</success_criteria>

<output>
After completion, create `.planning/quick/260617-gmq-expo-sdk-55-to-56-upgrade/260617-gmq-SUMMARY.md`.
Commit atomically (one commit per app + one for shared/lockfile). Do NOT push. Do NOT update ROADMAP.md. Do NOT touch GitHub PRs — list #79/#84/#85/#86/#87 in the SUMMARY for the owner to close.
</output>
</content>
</invoke>
