---
status: complete
plan: 260610-grk-PLAN.md
date: 2026-06-10
commits:
  - 2687ff3 feat(260610-grk-01): order urgency — age escalation on PendingOrderCard + waiting-for-driver elapsed counter
  - 2de6011 feat(260610-grk-01): 44px touch targets, login error alerts, and menu search input
  - d4ef086 feat(260610-grk-01): feedback toasts, review filter counts, payout UX, radius tokens, and empty-state copy
  - fix: remove unreachable duplicate ready-status footer block in order detail (post-merge)
---

# Quick Task 260610-grk: Vendor app UX improvements from design critique

All 10 design-critique items implemented across 12 files in `apps/mobile-vendor/`. 3/3 tasks complete.

## Key changes

- **PendingOrderCard.tsx** — live 45-second ticker; age color escalates `ink.soft → amber → destructive` at 5 and 10 minute thresholds; interval clears on unmount
- **app/orders/[orderId].tsx** — waiting-for-driver footer shows elapsed minutes (` · Nm`) using `timing.preparedAt` as baseline; contact buttons `minHeight 36→44`; duplicate unreachable `ready` footer block removed post-merge (TS2367)
- **app/(tabs)/menu.tsx** — search `TextInput` (44px minHeight) combined with category filter via `useMemo`; `+ Add` button `minHeight 36→44`; `No items match "…"` empty state when query active
- **app/menu/MenuItemForm.tsx** — photo-remove button hitSlop 12 on all sides (effective target ≥44px); success toast after category creation
- **app/(auth)/login.tsx** — Google/Apple/email sign-in handlers wrapped in try/catch with `Alert.alert('Sign-in failed', …)`; FCM registration stays non-fatal
- **app/notification-preferences.tsx** — quiet-hours inputs `minHeight 44` + `theme.radius.DEFAULT`; success toast on each persisted change; inline HH:MM validation error
- **app/reviews.tsx** — filter tabs show review counts `(N)`; `≤2★` renamed `Low (1–2★)`
- **app/payout.tsx** — `(re-enter required)` helper under masked account/UPI fields; dirty-aware save label (`Save changes` vs `Save payout details`)
- **app/admin-requests.tsx** — badge and notes-block radii aligned to `theme.radius.sm`
- **app/analytics.tsx / app/(tabs)/orders.tsx** — prescriptive empty-state copy; removed "We poll every 10 seconds" implementation detail
- **app/profile.tsx** — kitchen photo upload success toast

## Verification

- `npx tsc --noEmit`: no new errors from this change set. 107 pre-existing errors remain — all from a duplicate `@types/react` environment issue (TS2305 lucide-react-native exports / TS2786 JSX component types in `packages/mobile-shared`), present in files untouched by this task.

## Deferred

- Dark mode — phase-sized (dark token set + theme-reactive styles across all static StyleSheets); queued separately on the roadmap.
- Bulk availability toggle for menu categories — needs UX decision on placement (category-level action vs multi-select mode).
