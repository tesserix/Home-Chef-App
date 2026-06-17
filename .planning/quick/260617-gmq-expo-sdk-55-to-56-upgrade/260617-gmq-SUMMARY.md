---
phase: quick-260617-gmq
plan: 01
subsystem: mobile
tags: [expo, react-native, sdk-upgrade, monorepo, pnpm]
requires: []
provides: [SDK56-UPGRADE]
affects:
  - apps/mobile-customer
  - apps/mobile-vendor
  - apps/mobile-delivery
  - packages/mobile-shared
tech-stack:
  added:
    - "Expo SDK 56 (~56.0.12)"
    - "React Native 0.85.3"
    - "react-native-reanimated ~4.3.1 / react-native-worklets 0.8.3"
  patterns:
    - "pnpm root overrides to pull stale optional-peer @expo/* up to 56.x"
    - "expo-router/tabs subpath for BottomTabBarProps (no react-navigation dep)"
key-files:
  modified:
    - apps/mobile-customer/package.json
    - apps/mobile-customer/app.json
    - apps/mobile-customer/nativewind-env.d.ts
    - "apps/mobile-customer/app/chef/[id].tsx"
    - apps/mobile-customer/components/chef/ChefCard.tsx
    - apps/mobile-customer/components/tracking/DeliveryMap.tsx
    - apps/mobile-vendor/package.json
    - apps/mobile-vendor/app.json
    - "apps/mobile-vendor/app/(tabs)/_layout.tsx"
    - apps/mobile-vendor/app/profile.tsx
    - apps/mobile-vendor/nativewind-env.d.ts
    - apps/mobile-delivery/package.json
    - apps/mobile-delivery/app.json
    - apps/mobile-delivery/nativewind-env.d.ts
    - packages/mobile-shared/package.json
    - package.json
    - pnpm-lock.yaml
decisions:
  - "vendor BottomTabBarProps imported from expo-router/tabs (path a) — zero react-navigation deps"
  - "added pnpm overrides for @expo/metro-runtime ^56.0.15 + @expo/dom-webview ^56.0.5 (stale optional peers wouldn't move without a clean lockfile rebuild)"
  - "bumped mobile-shared @types/react devDep to ^19.2.0 to dedupe against root 19.2.x (fixed 67 TS2786 JSX-component errors)"
metrics:
  duration: ~40m
  completed: 2026-06-17
---

# Phase quick-260617-gmq Plan 01: Expo SDK 55 → 56 Upgrade Summary

All three Expo apps (mobile-customer, mobile-vendor, mobile-delivery) and the shared `packages/mobile-shared` library upgraded from Expo SDK 55 (RN 0.83.6) to SDK 56 (RN 0.85.3, React 19.2.3) and reconciled into ONE coherent root pnpm-lock.yaml with a single react-native version. All static verification is green; the native build remains the owner's gate.

## OWNER'S REMAINING GATE — THE TRUE PASS/FAIL

**Static checks here CANNOT confirm the upgrade works.** The native build is the owner's responsibility and the real pass/fail:

1. Delete stale `apps/mobile-{customer,vendor}/ios` dirs (gitignored CNG output), then `npx expo prebuild --clean` per app.
2. Native build under **Xcode >= 26.4 / iOS 16.4** with `buildReactNativeFromSource: true` (already set in customer + vendor app.json).
3. Only a native build confirms: Hermes V1 runtime, reanimated/worklets native binding, firebase native init, and `react-native-webview` (Razorpay) + `react-native-maps` + `@react-native-google-signin` native compilation under RN 0.85 / new arch.

No EAS, no prebuild, no native build was run in this plan (explicitly out of scope per constraints).

## Resolved Versions (authoritative — from `npx expo install --fix`)

Identical across all three apps:

| Package | SDK 55 (before) | SDK 56 (after, resolved) |
|---|---|---|
| `expo` | ~55.0.26 | **~56.0.12** |
| `react-native` | 0.83.6 | **0.85.3** (single version, verified) |
| `react` | 19.2.0 | **19.2.3** |
| `react-native-reanimated` | ~4.2.1 | **~4.3.1** |
| `react-native-worklets` | 0.7.4 | **0.8.3** |
| `expo-router` | ~55.x | **~56.2.11** |
| `expo-updates` | ~56.0.19 (customer/vendor only) | **~56.0.19** (still customer/vendor only — see note) |
| `expo-device` | ~55.x | **~56.0.4** |
| `expo-notifications` | ~55.x | **~56.0.18** |
| `expo-secure-store` | ~55.x | **~56.0.4** |
| `@react-native-community/netinfo` | ~11.4.1 | **~12.0.1** |
| `@types/react` (root, deduped) | 19.2.7 | **19.2.17** |
| `@sentry/react-native` (vendor) | 7.13.0 | **7.11.0** (pinned to SDK-56 expected) |
| `@expo/metro-runtime` (override) | 55.0.9 | **56.0.15** |
| `@expo/dom-webview` (override) | 55.0.5 | **56.0.5** |

**Note on expo-updates / mobile-delivery:** `expo install --fix` only re-pins dependencies an app already declares; it does NOT add packages the app doesn't use. mobile-delivery never declared `expo-updates`, so it remains without it (correct — it doesn't use OTA updates). The plan's brief anticipated `--fix` would "add/align expo-updates" to delivery; in practice `--fix` aligns only existing deps. This is not a failure: `expo-updates` is aligned across the apps that have it (customer, vendor).

## Vendor react-navigation reconciliation — PATH (a) chosen

The single direct react-navigation import in the whole repo was a type-only import at
`apps/mobile-vendor/app/(tabs)/_layout.tsx:10`:
`import type { BottomTabBarProps } from '@react-navigation/bottom-tabs';`

Under SDK 56 this broke: the bare specifier still resolves to a root-hoisted copy, but expo-router's `<Tabs>` now passes props typed against its **own bundled** react-navigation (`expo-router/build/react-navigation/bottom-tabs/types`). The two `BottomTabBarProps` types are structurally incompatible (3 errors → +74 cascade on the `tabBar` assignment).

**Resolution (path a — preferred):** Replaced the import with expo-router's public subpath:
`import type { BottomTabBarProps } from 'expo-router/tabs';`
(`tabs.d.ts` → `build/layouts/Tabs` → `react-navigation/bottom-tabs`, which re-exports `BottomTabBarProps` from expo-router's own bundled types). This makes `CustomTabBar` match exactly what `<Tabs>` passes, and introduces **zero** react-navigation dependency. The react-navigation sweep finds only the explanatory comment in this file — no real import remains. Fallback path (b) (`expo install @react-navigation/bottom-tabs` as a direct dep) was NOT used.

## expo-doctor results (per app)

`19/21 checks passed` on all three apps. The 2 "failures" are NOT version-mismatch or peer-range errors:

1. **"Check Expo config schema"** — fails only with `TypeError: fetch failed / ENOTFOUND exp.host` (the sandbox has no network to the Expo API). Infrastructure limitation, not a project issue.
2. **"Check for issues with Metro config"** — flags `resolver.disableHierarchicalLookup=true` and `resolver.unstable_enableSymlinks=true`. These are **pre-existing, intentional** monorepo overrides in `metro.config.js` (present at the base commit, unchanged by this upgrade) required for the pnpm hoisted layout to resolve modules. expo-doctor's own advice is "unless you know what these overrides do" — this repo does. Out of scope; not an upgrade regression.

`expo install --check` is **green ("Dependencies are up to date")** on all three apps.

## Verification summary (static only — green)

- `expo install --check`: green on all 3 apps.
- `expo-doctor`: 19/21 on all 3; the 2 misses are network-only + pre-existing intentional metro config (no version/peer errors).
- `npx tsc --noEmit` per app — **zero NEW errors vs SDK-55 baseline**:
  - mobile-customer: baseline 77 → after 3 (NEW=0)
  - mobile-vendor: baseline 119 → after 43 (NEW=0)
  - mobile-delivery: baseline 93 → after 19 (NEW=0)
  - (Totals dropped sharply because the `@types/react` dedup also cleared much pre-existing baseline noise.)
  - Vendor has **no** "Cannot find module '@react-navigation/bottom-tabs'" error.
- Tests: mobile-customer 22/22 jest pass; vendor/delivery `--passWithNoTests`; mobile-shared vitest **4 failed / 35 passed = identical to baseline** (the 4 pre-existing `auth-screens.test.ts` "module exists" failures, unrelated to SDK). No regression.
- Grep sweep: no `~55.x` expo-* range anywhere; only `@react-navigation` match is the explanatory comment in the reconciled vendor file.
- Workspace: single root pnpm-lock.yaml; **single react-native version (0.85.3)**; `pnpm install --frozen-lockfile` clean with no drift; `@tesserix/native` patch applied (`homeChefBrandMap` present in installed dist); `node-linker=hoisted` intact; Metro caches cleared.

## Deviations from Plan

### Auto-fixed Issues (Rules 1 & 3)

**1. [Rule 1 - Bug] `StyleSheet.absoluteFillObject` removed in RN 0.85**
- Found during: Task 1 / Task 4 (typecheck). RN 0.85 deleted `StyleSheet.absoluteFillObject` (both types and runtime); replacement is `StyleSheet.absoluteFill` (identical frozen `{position:'absolute', top/right/bottom/left:0}` object).
- Fix: replaced all 5 usages (`...spread` and `style=`) with `StyleSheet.absoluteFill`.
- Files: `apps/mobile-customer/app/chef/[id].tsx`, `components/chef/ChefCard.tsx`, `components/tracking/DeliveryMap.tsx`, `apps/mobile-vendor/app/profile.tsx`.
- Commits: 9ae1e28 (customer), d92101c (vendor).

**2. [Rule 1 - Bug] `import '../global.css'` side-effect import → TS2882**
- Found during: Task 4. Under SDK 56 / NativeWind 4.2, `nativewind/types` no longer declares a `*.css` side-effect module, so the global stylesheet import errored in all 3 apps.
- Fix: appended `declare module '*.css' {}` to each app's `nativewind-env.d.ts`.
- Files: `apps/mobile-{customer,vendor,delivery}/nativewind-env.d.ts`.
- Commits: 9ae1e28, d92101c, 1f8bb30.

**3. [Rule 3 - Blocking] 67× TS2786 ("X cannot be used as a JSX component") from @types/react skew**
- Found during: Task 1. `packages/mobile-shared` pinned `@types/react ~19.0.10`, which nested a 19.0.14 copy alongside the apps' root 19.2.x — structurally incompatible `ReactNode`, breaking every JSX component in mobile-shared/ui.
- Fix: bumped mobile-shared `@types/react` devDep to `^19.2.0`; the single root install deduped it to root 19.2.17 (nested copy gone). All 67 errors cleared.
- Files: `packages/mobile-shared/package.json`, `pnpm-lock.yaml`.
- Commit: 3c27903.

**4. [Rule 3 - Blocking] Stale `@expo/metro-runtime@55.0.9` + `@expo/dom-webview@55.0.5` optional peers**
- Found during: Task 3. The reused lockfile kept these 55.x optional-peer copies even after per-app `--fix`; `pnpm update`/`dedupe` and a plain override did NOT move them (pnpm 9 ignores overrides for already-locked optional peers).
- Fix: added root `pnpm.overrides` (`@expo/metro-runtime ^56.0.15`, `@expo/dom-webview ^56.0.5`) AND did one clean lockfile rebuild (`rm -rf node_modules pnpm-lock.yaml && pnpm install`). After the fresh resolution both pulled to 56.x; zero 55.x leftovers in the lockfile.
- Files: `package.json` (overrides), `pnpm-lock.yaml`.
- Commit: 3c27903.

**5. [Rule 3 - Blocking] SDK-56-required config plugins**
- Found during: Task 3 (expo config sync). `expo install --fix` added `expo-status-bar` (now requires a config plugin under SDK 56) to customer + vendor app.json, and `@sentry/react-native` plugin to vendor app.json.
- Fix: committed the app.json plugin additions with their respective apps.
- Commits: bb1a2e5 (customer), 69e1c85 (vendor).

**6. [Rule 1 - Bug] `@sentry/react-native` resolved above SDK-56 expected**
- Found during: Task 4 (`expo install --check`). Vendor's `^7.11.0` range let pnpm resolve 7.13.0; SDK 56 expects `~7.11.0`.
- Fix: `expo install --fix` re-pinned the locked version to 7.11.0; `--check` now green.
- File: `pnpm-lock.yaml`.
- Commit: 4190c69.

### Rule 4 (architectural) — none required.

## Redundant Dependabot PRs for the owner to close

These now-superseded expo-* bump PRs should be closed by the owner (this upgrade subsumes them; no GitHub actions taken here per constraints):
- #79
- #84
- #85
- #86
- #87

## Commits (per-task, atomic, single-line, no signatures, NOT pushed)

| Commit | Message |
|---|---|
| 8a4d890 | chore(mobile-customer): upgrade to Expo SDK 56 (RN 0.85) via expo install --fix |
| f11739b | chore(mobile-vendor): upgrade to Expo SDK 56 + reconcile bottom-tabs type import via expo-router/tabs |
| 02a474a | chore(mobile-delivery): upgrade to Expo SDK 56 (RN 0.85) + flip newArchEnabled true x3 |
| bb1a2e5 | chore(mobile-customer): add expo-status-bar config plugin required by SDK 56 |
| 69e1c85 | chore(mobile-vendor): add expo-status-bar + sentry config plugins required by SDK 56 |
| 3c27903 | chore(mobile): widen mobile-shared peer ranges + reconcile SDK 56 lockfile (single RN 0.85) |
| 9ae1e28 | fix(mobile-customer): SDK 56 type churn — StyleSheet.absoluteFill + global.css module decl |
| d92101c | fix(mobile-vendor): SDK 56 type churn — StyleSheet.absoluteFill + global.css module decl |
| 1f8bb30 | fix(mobile-delivery): SDK 56 type churn — global.css module decl |
| 4190c69 | chore(mobile-vendor): align @sentry/react-native to SDK 56 expected version in lockfile |

## Known Stubs

None. No placeholder/empty-data stubs were introduced.

## Self-Check: PASSED

All 6 key files exist; all 10 per-task commits exist in git history. Verified: all 3 apps `"expo": "~56"`, vendor `BottomTabBarProps` imported from `expo-router/tabs`, mobile-delivery `newArchEnabled: true` x3.
