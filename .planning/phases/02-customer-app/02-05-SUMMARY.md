---
phase: 02-customer-app
plan: "05"
subsystem: mobile-customer
tags: [react-native, expo, orders, favorites, profile, social, catering, react-query, react-hook-form, zod]
dependency_graph:
  requires:
    - 02-01  # tab shell, customer types, Zustand auth store, API client
    - 02-02  # ChefCard, expo-image, useChefs (for favorite toggle invalidation)
  provides:
    - orders-tab
    - favorites-tab
    - profile-tab
    - order-detail-screen
    - social-feed-screen
    - catering-screen
    - useOrderHistory-hooks
    - useFavorites-hooks
    - useProfile-hooks
    - useSocial-hooks
    - useCatering-hooks
  affects:
    - 02-03  # checkout calls useOrders to refresh after order placed
tech_stack:
  added: []
  patterns:
    - React Query paginated load-more with local state merge
    - Optimistic UI for favorite toggle and social like (revert on error)
    - react-hook-form + Zod schema validation on profile and catering forms
    - Tab switcher (Request / My Requests) in single screen component
    - Initials avatar placeholder for chef and user
key_files:
  created:
    - apps/mobile-customer/hooks/useOrderHistory.ts
    - apps/mobile-customer/hooks/useFavorites.ts
    - apps/mobile-customer/hooks/useProfile.ts
    - apps/mobile-customer/hooks/useSocial.ts
    - apps/mobile-customer/hooks/useCatering.ts
    - apps/mobile-customer/components/orders/OrderCard.tsx
    - apps/mobile-customer/app/order/[id].tsx
    - apps/mobile-customer/app/social.tsx
    - apps/mobile-customer/app/catering.tsx
  modified:
    - apps/mobile-customer/app/(tabs)/orders.tsx
    - apps/mobile-customer/app/(tabs)/favorites.tsx
    - apps/mobile-customer/app/(tabs)/profile.tsx
decisions:
  - "useOrderHistory.ts exports useOrders+useOrder (not useOrders.ts) — filename keeps checkout hooks separate in useOrderCheckout.ts per Plan 03"
  - "Favorites list response wraps entries in FavoriteChefEntry shape (id, chefId, chef) matching API FavoriteChefResponse; toggle mutationFn sends chefId in body for POST, path param for DELETE"
  - "Profile endpoint is PUT /v1/customer/profile (not PATCH) — confirmed from routes.go; addresses at GET /v1/addresses"
  - "Social feed endpoint is GET /v1/social/feed (not /v1/social/posts) — confirmed from social.go; like toggle is POST /v1/social/posts/:id/like"
  - "Catering form uses tab switcher within single screen; on submit success auto-switches to My Requests tab"
  - "Profile tab More section wires /social and /catering routes — Plan 02 home screen can add its own entry points without conflict"
metrics:
  duration_minutes: 40
  completed_date: "2026-04-06"
  tasks_completed: 2
  files_created: 9
  files_modified: 3
---

# Phase 02 Plan 05: Secondary Customer Screens Summary

**One-liner:** Paginated order history with status filter, order detail with Track Order button, favorites tab with optimistic heart toggle, profile form with Zod validation, social feed with like toggle, and catering request form with submitted quotes list — all wired to confirmed Go API endpoints.

## What Was Built

### Task 1 — Data hooks (commit `3d1ac83`)

Five React Query hooks covering all secondary screen data needs:

**`hooks/useOrderHistory.ts`** — `useOrders(params)` (paginated, 30s stale, keyed `['orders', params]`) and `useOrder(id)` (single detail, enabled guard). Both import from this file — no `useOrders.ts` created.

**`hooks/useFavorites.ts`** — `useFavorites()` (list of `FavoriteChefEntry` from `GET /v1/favorites/chefs`) and `useToggleFavorite()` mutation (DELETE for `isFavorited=true`, POST body `{chefId}` for false). Invalidates `['favorites']` and `['chefs']` on success.

**`hooks/useProfile.ts`** — `useProfile()` (`GET /v1/customer/profile`), `useUpdateProfile()` (`PUT /v1/customer/profile`), `useAddresses()` (`GET /v1/addresses`). Endpoint path corrected from plan's `/v1/profile` to `/v1/customer/profile` based on routes.go.

**`hooks/useSocial.ts`** — `useSocialFeed(params)` (`GET /v1/social/feed`, 1min stale, pagination support) and `useLikePost()` mutation (`POST /v1/social/posts/:id/like`). Endpoint path corrected from plan's guess to confirmed route.

**`hooks/useCatering.ts`** — `useCateringRequests(params)` (`GET /v1/catering/requests`) and `useCreateCateringRequest()` mutation (`POST /v1/catering/requests`). Request body matches `CateringRequestInput` confirmed from catering.go.

### Task 2 — Screens and components (commit `4653589`)

**`components/orders/OrderCard.tsx`** — Tappable card with order number, chef name, color-coded status badge (7 status values), item count, amount, date. Navigates to `/order/[id]` on press.

**`app/(tabs)/orders.tsx`** — Replaces Plan 01 stub. FlatList of OrderCards with horizontal status filter chips (All/Active/Delivered/Cancelled), pull-to-refresh via RefreshControl, and load-more via `onEndReached` incrementing page. Local state merges pages immutably.

**`app/order/[id].tsx`** — Stack screen using `useLocalSearchParams`. Shows chef name + order number header, large status badge, items list with per-item subtotal, delivery address, price breakdown (subtotal + delivery fee + total), order date. **Track Order** button visible only when status is one of `confirmed|preparing|ready|picked_up` — calls `router.push('/order/[id]/track')`.

**`app/(tabs)/favorites.tsx`** — Replaces Plan 01 stub. FlatList of `FavoriteChefCard` (inline component with expo-image, name, cuisine, rating/reviews, open badge, red filled heart). Heart press triggers optimistic removal + `useToggleFavorite.mutate({chefId, isFavorited: true})`; reverts on error with Alert.

**`app/(tabs)/profile.tsx`** — Replaces Plan 01 stub. Sections: (1) Avatar initials + email display + react-hook-form personal info form (firstName, lastName, phone with Zod validation per T-02-05-01, save button shown only when `isDirty`); (2) Cuisine preference chip grid with Save Preferences button; (3) More section with Social Feed (`router.push('/social')`) and Catering (`router.push('/catering')`) rows; (4) Logout button calling `useAuthStore.getState().logout()` + `router.replace('/(auth)/login')`.

**`app/social.tsx`** — Stack screen. FlatList of PostCard (author initials avatar, name, date, content text, optional image via expo-image, hashtags, heart button). Optimistic like toggle: updates local `isLiked` + `likesCount` immediately, reverts on API error. Pagination and pull-to-refresh.

**`app/catering.tsx`** — Stack screen with tab switcher (Request Catering / My Requests). Request tab: react-hook-form + Zod form (T-02-05-02) with event type chip selector, date (YYYY-MM-DD), guest count (positive integer), budget (positive number), city, state, description. On success shows alert and switches to My Requests tab. My Requests tab: FlatList of RequestCard with event type, status badge, guest count, date, city, and "Quotes available" hint for `status=quoted`.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Corrected profile endpoint path**
- **Found during:** Task 1 useProfile hook
- **Issue:** Plan suggested `GET /v1/profile` and `PATCH /v1/profile` but routes.go registers `GET /v1/customer/profile` and `PUT /v1/customer/profile` (PUT, not PATCH)
- **Fix:** Used `/v1/customer/profile` with `api.put()` instead of `api.patch()`
- **Files modified:** `apps/mobile-customer/hooks/useProfile.ts`
- **Commit:** `3d1ac83`

**2. [Rule 1 - Bug] Corrected favorites endpoint — POST body not path param**
- **Found during:** Task 1 useFavorites hook
- **Issue:** Plan suggested `POST /v1/favorites/chefs/:id` but favorites.go takes `POST /v1/favorites/chefs` with body `{chefId}` (no path param for add)
- **Fix:** POST sends `{ chefId }` in request body; DELETE uses path param `/v1/favorites/chefs/:chefId`
- **Files modified:** `apps/mobile-customer/hooks/useFavorites.ts`
- **Commit:** `3d1ac83`

**3. [Rule 1 - Bug] Corrected social feed endpoint path**
- **Found during:** Task 1 useSocial hook
- **Issue:** Plan guessed `POST /v1/social/feed` (POST) but social.go registers `GET /v1/social/feed`
- **Fix:** Used `api.get('/v1/social/feed')` with pagination params
- **Files modified:** `apps/mobile-customer/hooks/useSocial.ts`
- **Commit:** `3d1ac83`

## Known Stubs

None — all data flows are wired to live React Query hooks consuming confirmed Go API endpoints. The `/order/[id]/track` route is pushed by the Track Order button but will be implemented in Plan 04 (live GPS tracking). This is an intentional forward reference, not a stub in this plan's deliverables.

## Threat Flags

None — no new network endpoints or trust boundaries beyond what the plan's threat model covers. Profile PATCH (T-02-05-01) and Catering form (T-02-05-02) mitigations are implemented via Zod schemas. Social feed pagination (T-02-05-05) is implemented with `limit=20`.

## Self-Check: PASSED

All files created and commits confirmed:
- `apps/mobile-customer/hooks/useOrderHistory.ts` — FOUND
- `apps/mobile-customer/hooks/useFavorites.ts` — FOUND
- `apps/mobile-customer/hooks/useProfile.ts` — FOUND
- `apps/mobile-customer/hooks/useSocial.ts` — FOUND
- `apps/mobile-customer/hooks/useCatering.ts` — FOUND
- `apps/mobile-customer/components/orders/OrderCard.tsx` — FOUND
- `apps/mobile-customer/app/order/[id].tsx` — FOUND
- `apps/mobile-customer/app/social.tsx` — FOUND
- `apps/mobile-customer/app/catering.tsx` — FOUND
- `apps/mobile-customer/app/(tabs)/orders.tsx` — MODIFIED (confirmed)
- `apps/mobile-customer/app/(tabs)/favorites.tsx` — MODIFIED (confirmed)
- `apps/mobile-customer/app/(tabs)/profile.tsx` — MODIFIED (confirmed)
- Commit `3d1ac83` — FOUND in git log
- Commit `4653589` — FOUND in git log
