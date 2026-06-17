# Expo SDK 55 → 56 Upgrade — Research

**Researched:** 2026-06-17
**Scope:** apps/mobile-customer, apps/mobile-vendor, apps/mobile-delivery + packages/mobile-shared (pnpm monorepo, single root lockfile, CNG, Node 22)
**Confidence:** HIGH on versions/process (official sources reachable), MEDIUM on per-module breaking-change impact (verified against this codebase by grep)

> ⚠️ **Headline correction to the task brief:** SDK 56 is **NOT** a stay-on-RN-0.83 patch bump. SDK 56 ships **React Native 0.85** and **React 19.2**. That is a **two-minor RN jump** (0.83 → 0.84 → 0.85) with real native + import breaking changes, plus an **Xcode 26.4 / iOS 16.4 minimum**. Treat this as a real SDK upgrade, not a version-consistency nudge. `[CITED: expo.dev/changelog/sdk-56]` `[CITED: reactnative.dev/blog/2025/12/10/react-native-0.83]`

---

## 1. Target Versions (SDK 56)

**Authoritative resolver — do NOT hardcode these, let Expo pin them:**
```sh
# per app, from each app dir:
npx expo install expo@^56.0.0
npx expo install --fix      # rewrites every expo-* + RN-ecosystem dep to the SDK-56-pinned range
```
`expo install --fix` reads the bundled SDK→package version map and is the single source of truth. The values below are for sanity-checking the diff, not for hand-editing package.json.

| Package | SDK 55 (current) | SDK 56 target | Source |
|---|---|---|---|
| `expo` | ~55.0.26 | **~56.0.x** (exact patch via `--fix`) | `[CITED: expo.dev/changelog/sdk-56]` |
| `react-native` | 0.83.6 | **0.85.x** | `[CITED: expo.dev/changelog/sdk-56]` |
| `react` | 19.2.0 | **19.2** (unchanged) | `[CITED: expo.dev/changelog/sdk-56]` |
| `react-native-reanimated` | ~4.2.1 | **~4.3.1** | `[CITED: WebSearch verified vs Reanimated peer rules]` MEDIUM |
| `react-native-worklets` | 0.7.4 | **~0.8.3** | `[CITED: WebSearch]` MEDIUM — Reanimated 4.3+ requires worklets as separate peer |
| `expo-updates` | ~56.0.19 (already bumped) | aligns to SDK 56 via `--fix` | `[VERIFIED: package.json]` — this is the partial bump that motivated the upgrade |

**If network is unavailable:** the table above is for reference only. The authoritative answer is always whatever `npx expo install --fix` writes after `expo@^56` is installed. Do not freeze versions from this doc.

---

## 2. Breaking Changes That Touch THIS Codebase

Verified by grep against `apps/mobile-*/{app,src}` and `packages/mobile-shared/src`.

| Change | Applies here? | Action |
|---|---|---|
| **Native-tabs API** (`Icon/Label/Badge` → `NativeTabs.Trigger.*`) | **NO** | All 3 apps use the standard JS `Tabs` from `expo-router` (`app/(tabs)/_layout.tsx`). The new `NativeTabs` API is opt-in and unused. `[VERIFIED: grep]` |
| **expo-router drops react-navigation dep** — direct `@react-navigation/*` imports break; codemod provided | **NO direct imports found** (count: 0). Run codemod anyway as a safety net. | `npx expo install --fix` then run the Router codemod from repo root per [docs.expo.dev/router/migrate/sdk-55-to-56](https://docs.expo.dev/router/migrate/sdk-55-to-56/). `[VERIFIED: grep]` |
| **`@expo/vector-icons` no longer a transitive dep of `expo`** | **No src usage** found (only metro-cache/build artifacts). Confirm icons come from `lucide-react-native` (they do). | None expected. If `expo-doctor`/build complains, add `@expo/vector-icons` explicitly. `[VERIFIED: grep]` |
| **`@expo/dom-webview` is the new default WebView; `react-native-webview` no longer needed for DOM components** | **PARTIAL** — customer app uses `react-native-webview` directly for the in-app Razorpay checkout (`app/checkout.tsx`, `app/payment/checkout.tsx`), NOT as a DOM component. | Keep `react-native-webview` as an explicit dep (it is). `--fix` will bump it. No code change; this is a real native module, not a DOM webview. `[VERIFIED: grep + MEMORY: in-app Razorpay]` |
| **`expo/fetch` is now default `globalThis.fetch`** | App uses `axios`, not global fetch directly. Low risk. Opt-out flag exists (`EXPO_PUBLIC_USE_RN_FETCH=1`) if a regression appears. | Watch for fetch-based regressions in `react-query`/firebase at runtime; not statically detectable. `[CITED: sdk-56]` |
| **`expo-file-system` `copy()`/`move()` now async** (use `copySync`/`moveSync` for old behavior) | **No src usage of copy/move** found. vendor has `expo-file-system` + `expo-image-manipulator` but no copy/move calls. | None expected; re-verify after `--fix` if cropper/upload misbehaves. `[VERIFIED: grep]` |
| **Reanimated 4.3 + worklets 0.8** | YES — all 3 apps. Babel uses `babel-preset-expo` (worklets plugin is bundled inside the preset; no separate `react-native-reanimated/plugin` entry in babel.config.js — correct for SDK 54+). | After `--fix`, clear Metro cache and verify worklets plugin still loads via preset. Do NOT manually add the reanimated babel plugin. `[VERIFIED: babel.config.js]` |
| **Legacy architecture fully gone; `newArchEnabled=false` is IGNORED in RN 0.85** | **mobile-delivery has `newArchEnabled: false`** in app.json — this is **already a no-op** (legacy arch was removed back in SDK 55 / RN 0.82). | Flip mobile-delivery `app.json` → `newArchEnabled: true` (3 occurrences) for honesty; behavior is unchanged either way. `[VERIFIED: grep app.json]` `[CITED: github.com/reactwg/react-native-new-architecture #290]` |
| **Hermes V1 default** (opt out via `expo-build-properties.useHermesV1`) | All apps use Hermes; vendor explicitly `"jsEngine": "hermes"`. Performance change, not breaking. | None; flag for owner's on-device smoke test. `[CITED: sdk-56]` |
| **iOS 16.4 min, Xcode 26.4 min** | YES — affects the owner's native build gate. | Owner must build with Xcode ≥26.4. Pairs with existing `buildReactNativeFromSource:true` (already set in customer + vendor app.json for the firebase/Xcode-26 precompile fix). `[CITED: sdk-56]` `[VERIFIED: grep app.json + MEMORY]` |
| **Node ≥ 20.19.4** | Repo is Node 22. | None. `[CITED: sdk-56]` |

**expo-notifications / expo-secure-store / expo-device / expo-constants:** no SDK-56-specific breaking changes called out in the changelog; they get routine version bumps via `--fix`. `[CITED: sdk-56]` (LOW confidence on "no breaking change" — changelog didn't enumerate them individually; treat as ASSUMED-safe, verify with expo-doctor.)

---

## 3. pnpm Monorepo Upgrade Procedure (single root lockfile)

**Constraints in play:** one root `pnpm-lock.yaml`; `.npmrc` has `node-linker=hoisted` (required for RN codegen); root `pnpm.overrides` + `patchedDependencies` (`@tesserix/native`); `packages/mobile-shared` is a `workspace:*` lib carrying expo-* in **peerDependencies**.

**Recommended order of operations:**

```sh
# 0. Clean branch + snapshot lockfile for easy diff/revert
git checkout -b chore/expo-sdk-56

# 1. Upgrade expo in each app FIRST (don't touch pnpm-lock manually)
cd apps/mobile-customer && npx expo install expo@^56.0.0 && npx expo install --fix && cd ../..
cd apps/mobile-vendor   && npx expo install expo@^56.0.0 && npx expo install --fix && cd ../..
cd apps/mobile-delivery && npx expo install expo@^56.0.0 && npx expo install --fix && cd ../..

# 2. Align the shared lib's expo-* PEER ranges by hand (peers aren't touched by --fix).
#    Edit packages/mobile-shared/package.json peerDependencies to widen for SDK 56:
#      expo-device, expo-notifications, expo-secure-store, react  -> match the new app ranges
#    (peerDependencies should be permissive ranges, not exact pins — see gotcha below)

# 3. ONE root install to reconcile the whole workspace into a single coherent lockfile
pnpm install

# 4. Run the expo-router codemod from repo root (safety net even though 0 direct imports)
#    See docs.expo.dev/router/migrate/sdk-55-to-56

# 5. Verify (section 4)
```

**Why this order:** `expo install --fix` edits each app's `package.json` ranges but its own `pnpm install` step in a hoisted workspace can partially write the shared lockfile. Running `--fix` per app and then **one** root `pnpm install` lets pnpm resolve the whole graph once, avoiding the half-resolved lockfile that causes mixed versions. `[ASSUMED]` based on pnpm workspace + hoisted-linker behavior — validate the lockfile diff is coherent (no two RN versions).

**Gotchas:**
- **`--save` / catalog:** This repo does **not** use a pnpm catalog. Do not introduce `catalog:` refs during the upgrade — `expo install` writes plain ranges and won't understand catalog entries. Keep it simple.
- **`mobile-shared` peers, not deps:** Its expo-* are in `peerDependencies` (currently with stale/incorrect ranges like `expo-device ~7.1.4`, `expo-notifications ~0.30.2` that don't even match SDK 55's `~55.x`). Widen these to `*` or a permissive caret so the apps' SDK-56 versions satisfy them. Peers must NOT be exact-pinned. `[VERIFIED: package.json]`
- **`pnpm.overrides`:** `axios ^1.16.0` and the security overrides are unrelated to expo and should remain. Confirm none of them collide with new SDK-56 transitive pins after `pnpm install`.
- **`@tesserix/native` patch:** `patches/@tesserix__native.patch` must still apply after install. If the patch fails (changed base), pnpm install will error — re-generate the patch against the new resolution.
- **`node-linker=hoisted` must stay** — codegen depends on it; do not switch to symlinked during this upgrade.

---

## 4. Verification Without a Native Build

**What CI/JS-level checks WILL catch:**
```sh
# from each app dir:
npx expo install --check     # reports any dep whose version is OUTSIDE the SDK-56 expected range (read-only audit)
npx expo-doctor              # 15+ checks: version mismatches, peer issues, invalid app.json, unsupported deps
pnpm -r test                 # jest-expo (apps) + vitest (mobile-shared) — preset must resolve under SDK 56
pnpm typecheck               # TS against new @types/react / RN types
npx expo start --clear       # Metro bundles JS — catches import-breaks (react-navigation codemod gaps, vector-icons), babel/worklets plugin errors, NativeWind transform
```

`expo install --check` = read-only ("are versions right?"); `expo install --fix` = write ("make versions right"). Run `--check` after `--fix` to confirm green. `[CITED: docs.expo.dev/workflow/upgrading-expo-sdk-walkthrough]`

**Known-acceptable warnings:**
- expo-doctor warnings about **unmaintained** or **untested-on-new-arch** third-party packages (e.g., react-native-maps) — informational unless a build fails.
- Peer-dependency range warnings from pnpm for `@tesserix/native` / firebase if their declared peers lag — acceptable if runtime works.

**What ONLY a native prebuild/build can catch (owner's final gate):**
- Hermes V1 runtime behavior, reanimated/worklets native binding load, firebase native init.
- **Xcode 26.4 + iOS 16.4 deployment target** compile (with `buildReactNativeFromSource:true`).
- react-native-webview (Razorpay) + react-native-maps + google-signin + firebase native module compilation under RN 0.85 new arch.
- `npx expo prebuild --clean` then `expo run:ios` / EAS build — the true pass/fail. CNG means `ios/`/`android/` are regenerated (not git-tracked — confirmed), so a clean prebuild is mandatory before the build gate.

---

## 5. Upgrade-Specific Pitfalls

1. **The brief's premise is wrong (biggest risk):** expecting RN to stay 0.83. It moves to 0.85. The already-bumped `expo-updates ~56.0.19` is consistent with *56*, so `--fix` will pull the rest of the world up to RN 0.85 — a far larger diff than "consistency fix." Plan/communicate accordingly.
2. **Mixed-version lockfile:** if `--fix` is run per-app but the lockfile isn't reconciled with a single final `pnpm install`, you can end up with two `react-native` versions resolved. After upgrade, verify: `pnpm why react-native` shows exactly one version across all 3 apps.
3. **mobile-shared peer ranges are already stale/wrong** (point to pre-55 versions). They'll silently "work" via hoisting today but should be corrected during this upgrade or they'll mask real mismatches.
4. **Reanimated babel:** worklets plugin ships inside `babel-preset-expo`. Do NOT add `react-native-reanimated/plugin` manually — double-registration breaks the build. Current babel.config.js is correct; leave it.
5. **Metro cache:** the apps use a custom `FileStore` at `.metro-cache/`. Clear it (`expo start --clear` and delete `.metro-cache/`) after the upgrade or stale transforms (NativeWind, worklets) cause phantom errors.
6. **`@tesserix/native` patch drift:** patched dep + design-system upgrade window; ensure the patch still applies post-install.
7. **CNG re-prebuild required:** `ios/`/`android/` are gitignored build output (confirmed not tracked). The existing on-disk `ios/` dirs are stale SDK-55 artifacts — delete and `expo prebuild --clean` before any native build.
8. **expo/fetch as default global fetch:** axios is unaffected, but any code relying on RN's old fetch quirks could regress at runtime only. Keep `EXPO_PUBLIC_USE_RN_FETCH=1` as a known escape hatch.

---

## Assumptions Log

| # | Claim | Risk if wrong |
|---|---|---|
| A1 | reanimated `~4.3.1` / worklets `~0.8.3` are the exact SDK-56 pins | Low — `expo install --fix` overrides; table is sanity-check only |
| A2 | Per-app `--fix` + one final root `pnpm install` yields a clean single-version lockfile | Medium — if pnpm leaves mixed versions, run `pnpm dedupe` / inspect `pnpm why react-native` |
| A3 | expo-notifications/secure-store/device have no SDK-56 breaking changes | Low/Medium — changelog didn't enumerate; expo-doctor + on-device test will surface issues |
| A4 | `react-native-webview` (Razorpay) compiles under RN 0.85 new arch | Medium — native; only the owner's build confirms |

## Sources

**Primary (HIGH):**
- [expo.dev/changelog/sdk-56](https://expo.dev/changelog/sdk-56) — RN 0.85, React 19.2, Xcode 26.4, iOS 16.4, expo/fetch, file-system async, vector-icons, dom-webview, router/react-navigation split
- [docs.expo.dev/router/migrate/sdk-55-to-56](https://docs.expo.dev/router/migrate/sdk-55-to-56/) — router codemod
- [docs.expo.dev/workflow/upgrading-expo-sdk-walkthrough](https://docs.expo.dev/workflow/upgrading-expo-sdk-walkthrough/) — install --fix / expo-doctor flow
- [reactnative.dev RN 0.83 blog](https://reactnative.dev/blog/2025/12/10/react-native-0.83) — RN baseline
- [github.com/reactwg/react-native-new-architecture #290](https://github.com/reactwg/react-native-new-architecture/discussions/290) — legacy arch frozen/ignored

**Secondary (MEDIUM):** WebSearch results for reanimated 4.3 / worklets 0.8 SDK-56 pins; RN 0.85 legacy-arch removal.

**Codebase (VERIFIED by grep):** package.json (all 4), babel.config.js, metro.config.js, app.json, `(tabs)/_layout.tsx`, checkout WebView usage, git ls-files (CNG confirmation), .npmrc.

**Note:** Direct version-API endpoints (`exp.host/--/api/v2/versions`) were unreachable from the sandbox. `npx expo install --fix` is therefore the authoritative version resolver — prefer it over any pinned number in this doc.
