# Customer App UI Modernization — Design

**Date:** 2026-07-05
**Status:** Approved (brainstormed + section-by-section sign-off)
**Scope:** `apps/mobile-customer` only. Light mode only. No backend changes.

## Goal

Modernize the customer app's visual language — the bottom tab bar reads
"2019 default" and Home is chrome-heavy with competing accents. Direction:
Instagram-style floating dock (no glassmorphism), decluttered photo-forward
Home, and one consistent visual system across all four tabs. Clean, simple
UX journey throughout.

## Decisions (locked)

| Question | Decision |
|---|---|
| Scope | Full app pass (dock + Home + all tabs + motion) |
| Dock labels | Active-label hybrid: inactive = icon-only, active = icon+label in coral-tint pill |
| Cart / active orders | Cart merges into dock as dynamic 5th slot; active-order card stays the single floating layer |
| Discovery links | Fold into search row: search pill routes to unified search, map becomes icon button; kill both pink text links |
| Dark mode | Out of scope; new components must consume theme tokens (no hardcoded colors) so dark stays possible |

## Section 1 — The Dock

**Geometry.** Detached rounded bar floating above the home indicator:
16px side margins, 12px bottom gap above safe area, ~64px tall, 28px corner
radius, solid `canvas` background, hairline border, shadow `0 4 16 / 0.10`.
No blur, no glass. Content remains visible in the gap beneath.

**Tabs.** 4 slots — Home, Orders, Saved, Profile.
- Inactive: icon only, `charcoal.soft`, 22px.
- Active: icon + label inside a `coral.tint` pill, `coral.DEFAULT` icon+text,
  13px Inter-SemiBold. The pill is the only accent in the dock.
- Pill transition: opacity/transform only, 250ms,
  `cubic-bezier(0.22,1,0.36,1)`, `prefers-reduced-motion` → instant.
- Every slot ≥44px touch target.

**Cart pill (dynamic 5th slot).** When cart is non-empty, a solid-coral pill
appears at the dock's right end after a thin divider: bag icon + total
(item count as mini badge on the bag). Tap → `/checkout`. Fade+scale entrance,
no bounce. `CartBar` retires from the tabs layout (checkout flow elsewhere
unaffected).

**Anchoring.** Active-order stack and list `paddingBottom` re-anchor to
dock top + gap. `Dock` exports `DOCK_CLEARANCE` (height + bottom gap) for
screens to pad by. Bottom sheets overlay everything including the dock.

**Trade-off accepted:** a floating dock covers more content edge than a flush
bar; list screens pad bottom by `DOCK_CLEARANCE` so the last row scrolls clear.

## Section 2 — Home Declutter

Header collapses 7 rows → 4:

1. **Address + map** — address pill left (unchanged), map entry becomes a
   circular icon button right-aligned on the same row (replaces "Map view →").
2. **Search** — pill becomes a *button* (no inline TextInput on Home) routing
   to a unified search screen covering dishes and chefs (extends the existing
   `/search-dishes` screen; "Search dishes →" link dies). Placeholder stays
   "What are you craving?".
3. **Contextual slot** — winback banner / meal-plan chip only when present.
   Plan chip demotes to neutral: canvas bg, hairline border, small coral
   calendar icon only.
4. **Categories + filters** — structure unchanged; "Open Now" chip's coral
   dot goes neutral (selected state is the accent, not the dot).

**Accent discipline:** coral on Home = active dock pill + selected chips +
cart pill when present. Everything else neutral.

**Chef cards:**
- Missing-photo placeholder: warm `bone` gradient + chef initial (replaces
  flat gray fork-and-knife).
- Meta tightens to one line; "Open" becomes small green dot + "Open" inline.
- Photos stay the hero; card structure otherwise untouched.

## Section 3 — Other Tabs + Motion

**Shared screen-header rhythm.** Orders / Saved / Profile: large title
(Geist 600, ~28px), 16px gutters, consistent top spacing via a shared
`ScreenTitle` component.

**Orders.** Structure stays. Status chips align to one semantic set:
in-flight = coral-tint, delivered = green-tint, cancelled = neutral gray.
Empty state adopts shared component.

**Saved.** Chefs/dishes segmented control restyles to dock language:
active segment = coral-tint pill, inactive = plain text.

**Profile.** Header rhythm; NavRows grouped into rounded hairline-border
section cards; avatar block tightened. No structural change.

**Shared `EmptyState`.** Icon in soft `bone` circle, title, one-line body,
optional coral text CTA. Used by Orders, Saved, search.

**Motion.** Entrances = fade+slide 250ms standard easing; pressed states =
opacity 0.85 (no scale-bounce); dock pill 250ms; all reduced-motion-gated.
Remove any bounce/elastic keyframes. No new animation types.

## Section 4 — Structure + Rollout

**Components** (tokens from `packages/mobile-shared/src/theme`, nothing
hardcoded):
- `components/navigation/Dock.tsx` — custom `tabBar` for expo-router
  `<Tabs tabBar={...}>`; owns pill animation + safe-area math; exports
  `DOCK_CLEARANCE`.
- `components/navigation/DockCartPill.tsx` — reads existing cart store
  (reuses CartBar logic).
- `components/shared/EmptyState.tsx`, `components/shared/ScreenTitle.tsx`.
- Unified search: extend existing `/search-dishes` screen with chef results.

**Rollout — 3 PRs, each shippable + sim-verified (tsc clean, before/after
screenshots):**
1. **PR 1 — Dock:** Dock + cart pill + re-anchor order stack + list padding.
2. **PR 2 — Home declutter:** header consolidation, unified search routing,
   accent demotions, chef-card polish.
3. **PR 3 — Consistency sweep:** ScreenTitle, Orders chips, Saved segmented
   control, Profile sections, EmptyState, motion cleanup.

## Verification

Per PR: `npx tsc --noEmit` clean; Debug build on iOS simulator; screenshot
review of affected screens (Home with/without cart + active order; each tab;
sheet overlays unaffected). Accessibility spot-check: tab roles/labels on
dock slots, 44px targets, reduced-motion.
