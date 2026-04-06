---
phase: 03-vendor-app-driver-core
plan: 03
subsystem: mobile-vendor
tags: [menu-management, photo-upload, earnings, analytics, reviews, profile, settings, react-native]
dependency_graph:
  requires:
    - 03-02 (useVendorOrders pattern, api.ts client, auth store)
  provides:
    - Full vendor menu CRUD with availability toggle
    - Camera/gallery photo upload flow (two-step: create then upload)
    - All More-tab destination screens
    - Complete vendor app feature set (VEND-01 through VEND-10 satisfied)
  affects:
    - apps/mobile-vendor/app/(tabs)/menu.tsx
    - apps/mobile-vendor/app/(tabs)/more.tsx
    - apps/mobile-vendor/app/menu/new.tsx
    - apps/mobile-vendor/app/menu/[itemId]/edit.tsx
    - apps/mobile-vendor/app/earnings.tsx
    - apps/mobile-vendor/app/analytics.tsx
    - apps/mobile-vendor/app/reviews.tsx
    - apps/mobile-vendor/app/review/[reviewId].tsx
    - apps/mobile-vendor/app/profile.tsx
    - apps/mobile-vendor/app/settings.tsx
tech_stack:
  added:
    - expo-image-picker ~16.0.6 (added to package.json; was missing despite being used in onboarding)
  patterns:
    - Two-step photo upload: create item first to get itemId, then POST multipart to :itemId/images
    - Optimistic UI for availability toggle and delete (same pattern as useVendorOrders)
    - Inline availability Switch on MenuItemCard — no navigation required
    - Plain View-based bar charts for earnings/analytics (no charting library)
    - Zod client-side validation on review reply (min 10 chars) per threat model T-03-03-03
    - Bank account display masked to last 4 digits per threat model T-03-03-02
key_files:
  created:
    - apps/mobile-vendor/hooks/useVendorMenu.ts
    - apps/mobile-vendor/components/vendor/MenuItemCard.tsx
    - apps/mobile-vendor/app/menu/new.tsx
    - apps/mobile-vendor/app/menu/[itemId]/edit.tsx
    - apps/mobile-vendor/app/earnings.tsx
    - apps/mobile-vendor/app/analytics.tsx
    - apps/mobile-vendor/app/reviews.tsx
    - apps/mobile-vendor/app/review/[reviewId].tsx
    - apps/mobile-vendor/app/profile.tsx
    - apps/mobile-vendor/app/settings.tsx
  modified:
    - apps/mobile-vendor/app/(tabs)/menu.tsx (replaced placeholder)
    - apps/mobile-vendor/app/(tabs)/more.tsx (replaced emoji-based nav with lucide icons + user header)
    - apps/mobile-vendor/package.json (added expo-image-picker)
decisions:
  - Used plain View-based bar charts instead of a charting library — keeps bundle small and matches plan guidance
  - Profile form uses useState with initialization guard instead of RHF — profile edits are free-form, not validation-heavy
  - Settings toggles call PUT immediately on each change — no Save button per plan spec
  - expo-image-picker added to package.json (Rule 3 deviation — already used in onboarding but missing from manifest)
metrics:
  duration: ~45 minutes
  completed_date: 2026-04-06
  tasks_completed: 2
  files_created: 10
  files_modified: 3
---

# Phase 3 Plan 3: Vendor App Remaining Screens Summary

Completed vendor menu management and all More-tab destination screens with two-step photo upload, inline availability toggles, earnings/analytics visualizations, customer review reply flow, and settings with per-toggle PUT calls.

## Screens Created and Their Routes

| Screen | Route | Requirement |
|--------|-------|-------------|
| Menu list with FAB | `/(tabs)/menu` | VEND-02 |
| Create menu item | `/menu/new` | VEND-03 |
| Edit menu item | `/menu/[itemId]/edit` | VEND-02 |
| More tab (updated) | `/(tabs)/more` | VEND-05 |
| Earnings | `/earnings` | VEND-06 |
| Analytics | `/analytics` | VEND-07 |
| Reviews list | `/reviews` | VEND-08 |
| Review reply | `/review/[reviewId]` | VEND-08 |
| Profile | `/profile` | VEND-09 |
| Settings | `/settings` | VEND-10 |

## Photo Upload Approach

All photo uploads use a two-step pattern:

1. **New item**: POST `/chef/menu/items` first → receive `item.id` → POST multipart FormData with `file` field to `/chef/menu/items/${itemId}/images`
2. **Edit item**: `itemId` already known → upload immediately on photo selection
3. **Profile photo**: POST multipart to `/chef/profile-image`
4. **Kitchen photos**: POST multipart to `/chef/kitchen-photos`

FormData pattern follows established Phase 2 convention (same as onboarding documents.tsx):
```typescript
formData.append('file', { uri, name: 'photo.jpg', type: 'image/jpeg' } as unknown as Blob);
```

## Threat Model Mitigations Applied

- **T-03-03-02** (bank account disclosure): Bank account number masked to last 4 digits on earnings screen via `maskAccount()` helper
- **T-03-03-03** (review reply injection): Zod `z.string().min(10)` validation on review reply form; React Native Text renders as native text (no HTML injection risk)
- **T-03-03-04** (photo upload auth): All uploads use authenticated Bearer token via `api` instance; no unsigned URLs

## Web Reference Differences

- `ReviewsPage.tsx` (web): inline reply textarea per card → Mobile: navigate to dedicated `/review/[reviewId]` screen (cleaner UX on small screen)
- `EarningsPage.tsx` (web): uses Recharts bar chart → Mobile: plain View/width-percentage bars (no charting library added)
- `AnalyticsPage.tsx` (web): similar — same plain-bar approach on mobile

## Vendor App Completion Status

All VEND-01 through VEND-10 requirements are now implemented:

| Req | Description | Status |
|-----|-------------|--------|
| VEND-01 | Auth (login, register, forgot password) | Done — Plan 03-01 |
| VEND-02 | Menu management (list, create/edit, availability toggle) | Done — this plan |
| VEND-03 | Menu photo upload (camera + gallery) | Done — this plan |
| VEND-04 | Chef onboarding wizard | Done — Plan 03-01 |
| VEND-05 | Dashboard + live orders | Done — Plan 03-02 |
| VEND-06 | Earnings and payouts | Done — this plan |
| VEND-07 | Analytics with period filter | Done — this plan |
| VEND-08 | Customer reviews with reply | Done — this plan |
| VEND-09 | Profile and kitchen setup | Done — this plan |
| VEND-10 | Settings with notification preferences | Done — this plan |

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Missing Dependency] Added expo-image-picker to package.json**
- **Found during:** Task 1 implementation
- **Issue:** `expo-image-picker` was imported and used in `app/(onboarding)/documents.tsx` (pre-existing) but was missing from `package.json` dependencies
- **Fix:** Added `"expo-image-picker": "~16.0.6"` to dependencies
- **Files modified:** `apps/mobile-vendor/package.json`
- **Commit:** 3b8a2e6

## Commits

| Hash | Task | Description |
|------|------|-------------|
| 3b8a2e6 | Task 1 | Menu CRUD hooks, menu list with availability toggle, and menu item create/edit screens |
| 8165b72 | Task 2 | More-tab destination screens — earnings, analytics, reviews, profile, settings |

## Self-Check: PASSED
