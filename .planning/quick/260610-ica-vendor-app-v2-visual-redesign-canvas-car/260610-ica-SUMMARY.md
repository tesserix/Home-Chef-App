---
status: complete
date: 2026-06-10
commits:
  - 115e4ff feat(260610-ica): wave A — bone canvas + white elevated cards, status chips, segmented controls, pill chips, persimmon tab bar, greeting header, merged action-required, inline today stat card
  - f2a5814 feat(260610-ica): wave B + wow pass — dark ink dashboard hero with pulsing open pill and today stats, reanimated motion, fixed accept bar, canvas+cards across all secondary screens
---

# Quick Task 260610-ica: Vendor app v2 visual redesign ("canvas + cards")

Owner rejected the flat hairline-minimal look as "basic and outdated." Full visual redesign of
all 15 vendor screens + 4 shared components to a modern-consumer language while keeping brand
(paper/ink/persimmon, Geist/Inter, single accent). Spec: `260610-ica-UI-V2-SPEC.md`.
`.impeccable.md` updated with the "Vendor v2 surface model" addendum per its own pivot rule.

## What shipped

- **Surface flip:** bone (#F5F5F4) page canvas everywhere; content in white cards (radius 16,
  shadow tokens); iOS-style group-card lists with inset hairlines replace edge-to-edge rows.
- **Dashboard hero (wow pass):** dark ink card — greeting + name in light type, persimmon Open
  pill with live pulsing dot (reanimated, reduced-motion aware), today's earnings Geist-Bold 30
  + orders/rating on near-black. Replaced separate TODAY card.
- **Motion layer:** staggered FadeInDown section/card entrances (250ms ease-out-quart bezier
  0.22,1,0.36,1 — no bounce), press-scale 0.97 on Accept, pulse on Open dot. All gated by
  `useReducedMotion`.
- **Status chips** (tint bg + same-hue text) replace bare dots on dashboard, orders history,
  order detail, admin requests.
- **Controls:** iOS segmented controls (Queue/History, periods, bank/UPI), pill chips (menu
  categories, review filters with counts, form selectors), ink primary + ghost secondary
  buttons (no more naked underlined links; links are herb, de-underlined).
- **Tab bar:** white, top-elevated, active item persimmon, indicator line removed.
- **Fix:** squashed Accept bar — flex moved to a plain wrapper View (iOS Pressable
  function-style bug), row alignItems stretch, 52pt height.

## Verification

- `npx tsc --noEmit`: 108 errors — identical to pre-redesign baseline (duplicate @types/react
  classes: TS2305 lucide, TS2786 JSX, documents.tsx TS2322, reviews.tsx TS2345). Zero new.
- Verified live on iPhone 17 Pro simulator via Metro (dev client → vendors.fe3dr.com prod API).

## Notes

- Customer + driver apps still on v1 flat language — migrate as touched (v2 addendum in
  .impeccable.md is the direction).
- Dark mode remains queued as a phase (STATE.md pending todos).
