# Vendor App UI Modernization — Design

**Date:** 2026-07-05
**Status:** Approved (brainstormed + section-by-section sign-off)
**Scope:** `apps/mobile-vendor` only. Light mode. No backend changes.
**Sibling pass:** `2026-07-05-customer-ui-modernization-design.md` (customer app).

## Goal

Bring the vendor (chef) app up to the same modern visual language the customer
app just received — specifically the **floating Instagram-style dock** that
replaces the flush "2019 default" tab bar. Do it in the vendor's **Uber
monochrome** system, changing **no color tokens**.

## Reality check (scope honesty)

"Full parity with customer" is **mostly the dock**. The vendor app already
completed its own consistency sweep under UI-V2-SPEC:

- Semantic status chips (§2) — already present in Orders.
- Two-segment control (§5) — already present in Orders.
- Grouped white "section cards" (§1/§9) — already present in More (nav rows)
  and Orders (history rows).
- Motion already on the app-standard `bezier(0.22,1,0.36,1)` easing; every
  animation comment enforces "no bounce / no overshoot". Nothing to clean up.
- Vendor screens use **contextual** headers (dashboard hero, Orders segmented
  control, Menu search, More identity card) — NOT a uniform large-title. So
  porting the customer's `ScreenTitle` would fight the existing structure and
  is deliberately **not** done.

The single visible gap that makes vendor look dated next to the modernized
customer app is the **flush custom tab bar vs. the floating dock**. This design
targets that, plus the correctness work that a floating dock requires, plus one
small genuine divergence (Menu's bespoke empty state).

## Decisions (locked)

| Question | Decision |
|---|---|
| Scope | Full parity pass — but honestly scoped: dock + clearance + tiny EmptyState touch-up. Dashboard (already V2) left as-is. |
| Active-tab treatment | **Solid `ink` pill, white (`paper`) icon+label.** Inactive = icon-only `ink.muted`. (Customer used a coral-tint pill; vendor has no on-brand colored tint, and solid ink is the strongest monochrome expression.) |
| Cart / 5th slot | **Dropped.** Vendor has no cart; the customer dock's dynamic cart FAB does not port. |
| Color tokens | **Unchanged.** Only existing `ink` / `paper` / `mist` used. `herb` (persimmon) stays retired. |
| ScreenTitle | **Not ported** — vendor headers are contextual, not uniform titles. |
| Motion | **No change** — already bounce-free and on-standard-easing. |

## Section 1 — The floating monochrome dock

Port the customer `Dock.tsx` to the vendor app, swapping the coral accent for
**ink**.

**Geometry (identical to customer dock).** Detached rounded bar floating above
the home indicator: `left/right: 16`, `DOCK_BOTTOM_GAP` (4px) above the safe
area, `DOCK_HEIGHT` ~64px tall, `28px` corner radius, solid **white (`paper`)**
background, `mist` hairline border (`StyleSheet.hairlineWidth`), shadow
`{ offset: 0/4, opacity: 0.10, radius: 16 }`, `elevation: 10`. No blur, no
glass. Root `View` is `position: absolute` with `pointerEvents="box-none"` so
scenes get full screen height and content scrolls visibly through the gap
beneath.

**Tabs.** 4 slots — Dashboard (`index`), Orders, Menu, More — same routes and
Lucide icons as today (`LayoutDashboard`, `ClipboardList`, `UtensilsCrossed`,
`MoreHorizontal`).
- Inactive: icon only, `ink.muted`, 22px.
- Active: icon + label inside a **solid `ink` pill**; icon + text in
  **`paper` (white)**, 13px Inter-SemiBold. The pill is the only accent in the
  dock. Pill padding `14/9`, radius `20`.
- Active-pill entrance: `FadeIn` 250ms `bezier(0.22,1,0.36,1)`; under
  `useReducedMotion()` → instant (no `entering`).
- Every slot `flex: 1`, full bar height → ≥44px touch target.
- `accessibilityRole="tablist"` on the bar, `accessibilityRole="tab"` +
  `accessibilityState={{ selected }}` + `accessibilityLabel={label}` per slot.
- Icon quirk: use `Pressable` as the customer dock does (customer verified it
  works there); if iOS drops flex on the slot, fall back to the vendor's
  existing `TouchableOpacity` approach (see
  `feedback_ios_pressable_array_style.md`). Verify on-sim.

**No cart FAB.** The customer dock's dynamic 5th cart slot and its `CartFab`
are omitted entirely — vendor has no cart concept.

**Wiring.** The current `CustomTabBar` in `app/(tabs)/_layout.tsx` is removed
and replaced with `<Tabs tabBar={(props) => <Dock {...props} />}>`. Screen
registration (names, titles, icons) is unchanged.

## Section 2 — Dock clearance re-anchoring (required correctness)

The old flush `CustomTabBar` reserved layout height at the bottom; the floating
dock does not. Each tab screen must pad its scrollable bottom or the last row /
quiet block hides beneath the dock.

- Add `components/navigation/dock-metrics.ts` (ported from customer): exports
  `DOCK_HEIGHT`, `DOCK_BOTTOM_GAP`, and `useDockClearance()` =
  `insets.bottom + DOCK_BOTTOM_GAP + DOCK_HEIGHT + 12`. `Dock.tsx` re-exports
  these so screens import from either.
- Apply `useDockClearance()` as bottom padding on all 4 tab screens:
  - **Dashboard** (`index.tsx`): `ScrollView` `scrollContent.paddingBottom`
    AND ensure the `marginTop:auto` quiet block clears the dock.
  - **Orders**: `FlatList` `contentContainerStyle` bottom padding.
  - **Menu**: list bottom padding.
  - **More**: `ScrollView` content bottom padding.

**Trade-off accepted (same as customer):** a floating dock covers more content
edge than a flush bar; screens pad by `useDockClearance()` so the last row
scrolls clear. Bottom sheets / iOS action sheets overlay everything including
the dock (unchanged behavior).

## Section 3 — Minimal consistency touch-up

The only genuine divergence from vendor's own system: **Menu (`menu.tsx`) rolls
a bespoke inline empty state** (`emptyMenu` / `emptyHeadline` / `emptyBody` /
`emptyCta` styles) instead of the shared `EmptyState`
(`packages/mobile-shared/src/ui/EmptyState.tsx`).

Replace it with the shared component: `title` + `body` + `ctaLabel="Add first
item"` + `onCtaPress` routing to menu-item creation, keeping the existing copy.
The filter/search "no results" case (`No items match "…"`) also routes through
`EmptyState` with `title` only, no CTA.

Everything else — status chips, section cards, segmented control, motion — is
already conformant and **left untouched**. No token, radius, shadow, or spacing
changes.

## Structure

- `apps/mobile-vendor/components/navigation/Dock.tsx` — new; custom `tabBar`
  for expo-router `<Tabs>`. Owns pill render + safe-area math + a11y. Tokens
  from `@homechef/mobile-shared/theme` (`colors.ink` / `colors.paper` /
  `colors.mist`), nothing hardcoded except the shadow color `#000`.
- `apps/mobile-vendor/components/navigation/dock-metrics.ts` — new; geometry +
  `useDockClearance()`.
- `apps/mobile-vendor/app/(tabs)/_layout.tsx` — swap `CustomTabBar` → `Dock`.
- `apps/mobile-vendor/app/(tabs)/{index,orders,menu,more}.tsx` — bottom padding
  via `useDockClearance()`.
- `apps/mobile-vendor/app/(tabs)/menu.tsx` — adopt shared `EmptyState`.

## Rollout

**One focused PR** (feature branch → PR to `main`, per repo workflow; not a
direct push). Dock + clearance + Menu `EmptyState` swap. This is intentionally
smaller than the customer's 3-PR pass because vendor already swept.

## Verification

- `npx tsc --noEmit` clean in `apps/mobile-vendor`.
- Debug build on iOS simulator (local, per `feedback_local_sim_builds_only.md`).
- Before/after screenshots of all 4 tabs: Dashboard (quiet state + with pending
  orders), Orders (New + History segments), Menu (populated + empty), More.
- Confirm action sheets / bottom sheets still overlay the dock.
- A11y spot-check: dock tab roles/labels, 44px targets, reduced-motion renders
  the active pill instantly (no fade).
