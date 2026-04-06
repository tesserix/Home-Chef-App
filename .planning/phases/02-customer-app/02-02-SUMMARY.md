---
phase: 02-customer-app
plan: "02"
subsystem: mobile-customer
tags: [react-native, expo, browse, cart, bottom-sheet, react-query, nativewind]
dependency_graph:
  requires:
    - 02-01  # tab shell, cart store, customer types, API client
  provides:
    - chef-browse-grid
    - chef-detail-screen
    - cart-bar
    - cart-sheet
    - useChefs-hooks
  affects:
    - 02-03  # checkout needs cart populated from this plan
tech_stack:
  added:
    - react-native-maps@1.27.2
    - expo-image@~55.0.8
    - "@gorhom/bottom-sheet@^5.2.8"
    - expo-haptics@~55.0.11
  patterns:
    - FlatList numColumns=2 chef browse grid
    - React Query staleTime caching (2min chefs, 5min menu)
    - Zustand cart store addItem/cross_chef_conflict flow
    - forwardRef BottomSheet for cart sheet
    - NativeWind className via nativewind-env.d.ts type reference
key_files:
  created:
    - apps/mobile-customer/hooks/useChefs.ts
    - apps/mobile-customer/components/chef/ChefCard.tsx
    - apps/mobile-customer/components/chef/ChefGrid.tsx
    - apps/mobile-customer/components/chef/MenuItemCard.tsx
    - apps/mobile-customer/components/cart/CartBar.tsx
    - apps/mobile-customer/components/cart/CartSheet.tsx
    - apps/mobile-customer/app/chef/[id].tsx
    - apps/mobile-customer/nativewind-env.d.ts
  modified:
    - apps/mobile-customer/app/(tabs)/index.tsx
    - apps/mobile-customer/package.json
    - apps/mobile-customer/tsconfig.json
    - pnpm-lock.yaml
decisions:
  - "Use @ts-expect-error for @gorhom/bottom-sheet ref and BottomSheetView children — library types not yet updated for React 19 JSX; suppression is the standard pattern until upstream fixes"
  - "nativewind-env.d.ts added and referenced in tsconfig.json to enable className prop on all RN components via NativeWind type augmentation"
  - "FlatList with numColumns=2 in home screen (not ChefGrid component) to support ListHeaderComponent with search/filter controls above the grid"
  - "limit=20 default in useChefs filters per threat model T-02-02-04 (FlatList virtualization constraint)"
metrics:
  duration_minutes: 35
  completed_date: "2026-04-06"
  tasks_completed: 2
  files_created: 8
  files_modified: 4
---

# Phase 02 Plan 02: Chef Browse, Detail, and Cart UI Summary

**One-liner:** 2-column chef browse grid with React Query hooks, chef detail screen with sticky menu category tabs, and `@gorhom/bottom-sheet` cart UI wired to Zustand store.

## What Was Built

### Task 1 — Install packages + React Query hooks (commit `eed4e94`)

Installed four new SDK 55-compatible packages via `npx expo install`:
- `react-native-maps@1.27.2` — map display for future tracking screen
- `expo-image@~55.0.8` — performant chef/menu photo caching
- `@gorhom/bottom-sheet@^5.2.8` — cart bottom sheet
- `expo-haptics@~55.0.11` — haptic feedback on add-to-cart

Created `apps/mobile-customer/hooks/useChefs.ts` with three React Query hooks:
- `useChefs(filters)` — paginated chef list, `staleTime: 2min`, keyed `['chefs', filters]`
- `useChef(id)` — single chef detail, enabled guard on `!!id`
- `useChefMenu(chefId)` — chef menu items, `staleTime: 5min`

### Task 2 — Browse home, chef detail, cart UI (commit `086ca57`)

**`components/chef/ChefCard.tsx`** — Card with expo-image hero, name, cuisine, star rating + review count, open/closed badge, delivery time, minimum order. Taps navigate to `/chef/[id]`.

**`components/chef/ChefGrid.tsx`** — Thin FlatList wrapper with 4-card skeleton loading state.

**`app/(tabs)/index.tsx`** — Full home screen replacing Phase 1 stub:
- Debounced (400ms) search TextInput
- Horizontal cuisine filter chips (All / North Indian / South Indian / Chinese / Continental / Italian / Healthy)
- Open Now toggle + sort row (Recommended / Top Rated / Newest / Price)
- Social Feed and Catering quick-access pill buttons below sort row
- `FlatList numColumns={2}` chef grid with pull-to-refresh and empty state
- All filter state in `useState<ChefFilters>`, passed to `useChefs(filters)` with `limit: 20`

**`components/chef/MenuItemCard.tsx`** — Menu item card with expo-image, name, description (2-line truncation), price, dietary tags, Add button. On add: calls `useCartStore.getState().addItem()`; on `cross_chef_conflict` shows `Alert.alert` to replace cart; on `ok` triggers `Haptics.impactAsync(Light)`.

**`components/cart/CartBar.tsx`** — Absolute-positioned floating bar, renders only when `items.length > 0`. Shows item count badge, cart icon, "View Cart" label, subtotal. Press calls `onPress` prop (parent passes `cartSheetRef.current?.expand()`).

**`components/cart/CartSheet.tsx`** — `@gorhom/bottom-sheet` BottomSheet with `snapPoints=['60%','90%']`, `enablePanDownToClose`. Contains FlatList of cart items with +/- steppers (`updateQty`) and trash remove button (`removeItem`). Footer has subtotal + "Proceed to Checkout" → `router.push('/checkout')`.

**`app/chef/[id].tsx`** — Chef detail stack screen:
1. Full-width 200px hero image
2. Chef name, cuisine, rating, open/closed badge, delivery info card
3. Horizontal category tab strip (derived from unique `item.category` values)
4. FlatList of `MenuItemCard` for active category
5. `CartBar` overlay + `CartSheet` with forwarded ref

**`nativewind-env.d.ts`** — Added `/// <reference types="nativewind/types" />` and wired it into `tsconfig.json` includes to enable NativeWind `className` prop type augmentation on all React Native components.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing critical] Added nativewind-env.d.ts for NativeWind className types**
- **Found during:** Task 2 TypeScript verification
- **Issue:** `className` prop rejected by TypeScript on all RN components — `nativewind-env.d.ts` was missing; without it NativeWind's type augmentation of `react-native` module is not loaded
- **Fix:** Created `apps/mobile-customer/nativewind-env.d.ts` with `/// <reference types="nativewind/types" />` and added it to `tsconfig.json` includes
- **Files modified:** `apps/mobile-customer/nativewind-env.d.ts`, `apps/mobile-customer/tsconfig.json`
- **Commit:** `086ca57`

**2. [Rule 1 - Bug] Fixed implicit `any` in setIsOpenOnly callback**
- **Found during:** Task 2 TypeScript verification
- **Issue:** `setIsOpenOnly((prev) => !prev)` — `prev` inferred as `any` under strict mode
- **Fix:** Typed as `(prev: boolean) => !prev`
- **Files modified:** `apps/mobile-customer/app/(tabs)/index.tsx`
- **Commit:** `086ca57`

**3. [Rule 1 - Bug] Fixed @gorhom/bottom-sheet React 19 type incompatibility**
- **Found during:** Task 2 TypeScript verification
- **Issue:** `BottomSheet` ref type and `BottomSheetView` children prop incompatible with React 19 JSX types — library not yet updated for React 19
- **Fix:** Added `// @ts-expect-error` suppressions with explanatory comments; imported `BottomSheetMethods` type from library's internal types path for correct forwardRef typing
- **Files modified:** `apps/mobile-customer/components/cart/CartSheet.tsx`
- **Commit:** `086ca57`

## Known Stubs

None — all data flows are wired to live React Query hooks consuming the Go API. The Social Feed (`/social`) and Catering (`/catering`) routes are entry points only; the screens themselves are delivered in Plan 05 (Wave 2).

## Threat Flags

None — all surfaces introduced (browse query params, cart store) are covered by the plan's threat model (T-02-02-01 through T-02-02-04). Cart prices are stored from API response at add-time and will be re-validated by Go API during order creation.

## Self-Check: PASSED

All created files verified present on disk. Both commits (`eed4e94`, `086ca57`) confirmed in git log.
