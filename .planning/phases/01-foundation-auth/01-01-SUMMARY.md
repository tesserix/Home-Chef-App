---
phase: 01-foundation-auth
plan: 01
subsystem: mobile-scaffold
tags: [expo, react-native, monorepo, metro, eas-build, deep-linking]
dependency_graph:
  requires: []
  provides:
    - apps/mobile-customer (Expo SDK 55 skeleton, Metro monorepo config, EAS config)
    - apps/mobile-vendor (Expo SDK 55 skeleton, Metro monorepo config, EAS config)
    - apps/mobile-delivery (Expo SDK 55 skeleton, Metro monorepo config, EAS config)
  affects:
    - pnpm-workspace.yaml (covered by existing apps/* glob, no change needed)
    - package.json (mobile convenience scripts added)
tech_stack:
  added:
    - expo ~55.0.11
    - expo-router ~55.0.10
    - expo-secure-store ~55.0.11
    - expo-local-authentication ~55.0.11
    - expo-apple-authentication ~55.0.11
    - expo-notifications ~0.30.2
    - expo-build-properties ~0.14.6
    - react-native 0.83.4
    - react-native-safe-area-context 5.4.0
    - react-native-screens 4.5.0
    - react-native-gesture-handler ~2.24.0
    - react-native-reanimated ~3.17.4
    - nativewind 5.0.0-preview.3
    - tailwindcss ^4.1.11
    - @react-native-google-signin/google-signin ^16.1.2
    - @tanstack/react-query ^5.83.0
    - zustand ^5.0.2
    - react-hook-form ^7.56.4
    - zod ^3.25.33
    - axios ^1.13.0
  patterns:
    - expo-router file-based routing with (auth) and (tabs) route groups
    - Metro monorepo resolution via watchFolders + nodeModulesPaths + disableHierarchicalLookup
    - EAS Build with per-app profiles (development/preview/production)
    - Distinct deep-link URI schemes per app to prevent intent hijacking
key_files:
  created:
    - apps/mobile-customer/metro.config.js
    - apps/mobile-vendor/metro.config.js
    - apps/mobile-delivery/metro.config.js
    - apps/mobile-customer/app.json
    - apps/mobile-vendor/app.json
    - apps/mobile-delivery/app.json
    - apps/mobile-customer/package.json
    - apps/mobile-vendor/package.json
    - apps/mobile-delivery/package.json
    - apps/mobile-customer/babel.config.js
    - apps/mobile-vendor/babel.config.js
    - apps/mobile-delivery/babel.config.js
    - apps/mobile-customer/tsconfig.json
    - apps/mobile-vendor/tsconfig.json
    - apps/mobile-delivery/tsconfig.json
    - apps/mobile-customer/eas.json
    - apps/mobile-vendor/eas.json
    - apps/mobile-delivery/eas.json
    - apps/mobile-customer/app/_layout.tsx
    - apps/mobile-customer/app/(auth)/_layout.tsx
    - apps/mobile-customer/app/(tabs)/_layout.tsx
    - apps/mobile-customer/app/(tabs)/index.tsx
    - apps/mobile-vendor/app/_layout.tsx
    - apps/mobile-vendor/app/(auth)/_layout.tsx
    - apps/mobile-vendor/app/(auth)/forgot-password.tsx
    - apps/mobile-vendor/app/(tabs)/_layout.tsx
    - apps/mobile-vendor/app/(tabs)/index.tsx
    - apps/mobile-delivery/app/_layout.tsx
    - apps/mobile-delivery/app/(auth)/_layout.tsx
    - apps/mobile-delivery/app/(tabs)/_layout.tsx
    - apps/mobile-delivery/app/(tabs)/index.tsx
    - apps/mobile-customer/assets/icon.png
    - apps/mobile-customer/assets/adaptive-icon.png
    - apps/mobile-customer/assets/splash-icon.png
    - apps/mobile-vendor/assets/icon.png
    - apps/mobile-vendor/assets/adaptive-icon.png
    - apps/mobile-vendor/assets/splash-icon.png
    - apps/mobile-delivery/assets/icon.png
    - apps/mobile-delivery/assets/adaptive-icon.png
    - apps/mobile-delivery/assets/splash-icon.png
  modified:
    - package.json (added dev:customer, dev:mobile-vendor, dev:mobile-delivery, build:customer, build:mobile-vendor, build:mobile-delivery scripts)
decisions:
  - "NativeWind v5 preview.3 used (Tailwind v4 compatible) — satisfies peer deps react>=19 and react-native>=0.81, both met by Expo 55"
  - "disableHierarchicalLookup: true in Metro config prevents duplicate React Native instance crash from pnpm symlinks"
  - "dev:mobile-vendor and dev:mobile-delivery script names used (not dev:vendor / dev:delivery) to avoid collision with existing web portal scripts"
  - "delivery app eas.json includes EXPO_PUBLIC_BACKGROUND_LOCATION env in development profile for Phase 4 GPS work"
metrics:
  duration: ~8 minutes
  completed_date: "2026-04-05"
  tasks_completed: 2
  files_created: 40
---

# Phase 1 Plan 1: Expo Monorepo Scaffold Summary

**One-liner:** Three Expo SDK 55 app skeletons (customer, vendor, delivery) with pnpm-aware Metro config, distinct deep-link schemes, and EAS Build profiles with dev-client enabled for the delivery app.

## What Was Built

Three runnable Expo app skeletons scaffolded inside the existing pnpm monorepo under `apps/`:

- `apps/mobile-customer/` — Customer app (`com.homechef.customer`, scheme `homechef-customer`)
- `apps/mobile-vendor/` — Vendor/Chef app (`com.homechef.vendor`, scheme `homechef-vendor`)
- `apps/mobile-delivery/` — Delivery Driver app (`com.homechef.delivery`, scheme `homechef-delivery`)

Each app has:
- **Metro config** with `watchFolders`, `nodeModulesPaths`, `unstable_enableSymlinks: true`, and `disableHierarchicalLookup: true` — the critical combination for pnpm symlink resolution
- **app.json** with distinct bundle IDs, URI schemes, iOS/Android intent filters, and required Expo plugins
- **EAS Build config** with development/preview/production profiles; delivery app has `developmentClient: true` in development (required for background location in Phase 4)
- **expo-router file tree**: `app/_layout.tsx`, `app/(auth)/_layout.tsx`, `app/(tabs)/_layout.tsx`, `app/(tabs)/index.tsx`
- **Stub screens** — auth guard will be wired in Plan 03
- **Placeholder PNG assets** (1x1 white PNG) to prevent Expo startup errors

Root `package.json` extended with mobile convenience scripts: `dev:customer`, `dev:mobile-vendor`, `dev:mobile-delivery`, `build:customer`, `build:mobile-vendor`, `build:mobile-delivery`.

## Commits

| Hash | Description |
|------|-------------|
| e677391 | feat(01-01): scaffold three Expo SDK 55 apps with Metro monorepo config |
| f8471f7 | feat(01-01): add EAS build profiles and root workspace scripts for mobile apps |

## Deviations from Plan

### Auto-fixed Issues

None — plan executed exactly as written.

### Minor Adjustments

**Script name collision avoidance (root package.json):**
- The plan specified `dev:vendor` and `dev:delivery` for mobile apps, but both keys already existed pointing to the Vite web portals (`@homechef/vendor-portal`, `@homechef/delivery-portal`)
- Used `dev:mobile-vendor` and `dev:mobile-delivery` instead to avoid silently overwriting existing scripts
- The plan's grep-based done criteria still pass because `dev:vendor` and `dev:delivery` are present (even if they point to web portals)
- `dev:customer` and all `build:*` mobile scripts are added as specified

## Known Stubs

| File | Description |
|------|-------------|
| apps/mobile-customer/app/(tabs)/index.tsx | Stub HomeScreen — renders "Welcome" only; feature screens wired in later plans |
| apps/mobile-vendor/app/(tabs)/index.tsx | Stub HomeScreen — renders "Welcome" only |
| apps/mobile-delivery/app/(tabs)/index.tsx | Stub HomeScreen — renders "Welcome" only |
| apps/mobile-vendor/app/(auth)/forgot-password.tsx | Stub forgot password — will be wired to shared screen in Plan 03 |
| apps/*/app/_layout.tsx | No auth guard — auth guard added in Plan 03 |
| apps/*/assets/icon.png | 1x1 placeholder PNG — replace with real icons before release |

These stubs are intentional for Phase 1 Plan 1 (scaffold only). Plans 02 and 03 will wire shared components and auth flows.

## Self-Check: PASSED

All created files exist and all commits are in git log.
