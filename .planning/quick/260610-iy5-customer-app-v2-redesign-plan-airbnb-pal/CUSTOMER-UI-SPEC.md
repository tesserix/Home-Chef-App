# Customer App v2 Visual Language — "Airbnb"

Owner decision 2026-06-10: the customer app (`apps/mobile-customer`) gets a premium
consumer-marketplace look in the Airbnb mold — **white-first, airy, photo-forward, rausch coral
as the single accent**. This is the binding contract for the redesign waves, modeled on the
vendor v2 spec. The vendor app stays ink + persimmon; the two never share an accent.

The customer app renders via **NativeWind `className`** (e.g. `bg-coral`, `text-charcoal`), NOT
inline `theme` styles. All colours come from the customer palette in
`apps/mobile-customer/tailwind.config.js` (mirrored as `customerColors` /  `customerTheme` in
`@homechef/mobile-shared/theme` for Reanimated + programmatic colour). **No hex literals in
screens** — every colour traces to a token.

## 0. Palette (the vocabulary)

| Class root | Hex | Use |
|---|---|---|
| `coral` (`bg-coral`, `text-coral`) | `#FF385C` | Primary CTA, links, selected, heart-saved, focus |
| `coral-pressed` | `#E00B41` | Pressed/active on coral surfaces |
| `coral-tint` | `#FFE8EC` | Chip/badge tint backgrounds |
| `charcoal` | `#222222` | Primary text + dark buttons (Airbnb near-black) |
| `charcoal-soft` | `#717171` | Secondary text, captions, placeholders |
| `canvas` | `#FFFFFF` | Page background — **white-first, NOT grey-canvas** |
| `hairline` | `#EBEBEB` | Dividers, card borders |
| `surface` / `surface-soft` | `#FFFFFF` / `#F7F7F7` | Elevated cards / input fills, image placeholders |
| `success` | `#008A05` | Delivered / confirmation only |

- Legacy `paper`/`bone`/`ink`/`mist` are repointed to this system (white/charcoal/hairline) as a
  migration safety-net. **New code uses the explicit names above.** Never introduce `herb` in a
  customer screen (grep-banned); never let `coral` leak into vendor/driver.
- Typography stays **Geist** (`font-display`) + **Inter** (`font-sans`). Tabular figures for every
  price, ETA, rating, count.

## 1. Surface model (the big difference from vendor)

- **Page canvas:** white (`bg-canvas`). NOT grey. Airbnb is white-first; the photos carry the colour.
- **Separation by hairline, not card-soup.** Content sits directly on white separated by `hairline`
  rules and whitespace. Cards are reserved for genuinely elevated/floating elements.
- **Floating elements** (search pill, sticky CTA bars, map controls, FABs) get white bg +
  `shadow[2]` (`shadowOffset {0,2}`, opacity 0.06, radius 6). Everything else is flat on white.
- **Radius:** `12` for images and content cards (`rounded-xl`); `full` for pills, chips, search,
  circular overlay buttons; `8` for filled CTA buttons.
- Grouped lists (profile, settings rows): white rows separated by inset hairlines aligned to text —
  iOS-grouped feel, on white, NOT a grey canvas.

## 2. Signature patterns (implement these)

1. **Floating search pill** (home): radius-full white pill, `shadow[2]`, `charcoal-soft` magnifier +
   placeholder "What are you craving?". The brand's first impression — give it air above the cards.
2. **Category chip row** (under search): icon + label chips, horizontal scroll. Selected = `charcoal`
   text + 2px `charcoal` underline (Airbnb category-bar style), unselected = `charcoal-soft`, no
   border, no pill fill. (This replaces the old bordered persimmon pills.)
3. **Photo-led chef cards** (2-col grid): full-width food photo, `rounded-xl`, ~4:3. Heart toggle
   top-right (white glyph + subtle charcoal scrim; **saved = coral fill + 150ms scale-pop**). Below
   the photo, on white: chef name Inter-SemiBold `charcoal` + inline `★ 4.9 (212)` (star is
   `charcoal`, NOT gold), cuisine line `charcoal-soft`, then delivery-time · min-order line.
   Open/Closed as a small text state, not a loud badge.
4. **Chef detail:** full-bleed photo header (~40% viewport). Circular white overlay buttons
   (`shadow[2]`) for back / share / heart over the photo. Content sheet scrolls up over it. Menu
   items = rows with photo on the right. **Sticky bottom CTA bar**: white, top hairline + `shadow[2]`,
   coral filled "Add to cart · ₹540".
5. **Sticky CTA bars** (cart/checkout): coral filled, radius 8, 52pt min-height, Inter-SemiBold
   `canvas` text. "Continue", "Place order · ₹780". Pressed = `coral-pressed` (or 0.9 opacity).
6. **Order tracking:** map full-bleed; floating white status card at the bottom (`rounded-xl`,
   `shadow[2]`) — driver, ETA, progress dots in coral; circular floating back button top-left.
7. **Status chips:** tint bg + dark text of same family, radius-full, Inter-SemiBold caption:
   in-progress → `coral-tint` bg + `coral-pressed` text; delivered → `success-tint` bg +
   `success` text; cancelled → a neutral/`surface-soft` bg + `charcoal-soft` text. Coral reserved
   for active/selected — don't tint everything coral.
8. **Profile / favorites:** profile = iOS grouped rows on white with hairlines. Favorites = 2-col
   photo grid mirroring the home chef card (same heart behavior), with a calm empty state.

## 3. Buttons

- **Primary:** `bg-coral` fill, `canvas` (white) text, radius 8, `minHeight: 52`, Inter-SemiBold body.
  Pressed = `coral-pressed` or 0.9 opacity.
- **Dark/secondary action** (Airbnb "Reserve"-style where coral would be too loud, e.g. a neutral
  confirm): `charcoal` fill, white text, same radius/height.
- **Ghost/outline:** white bg, `borderWidth 1` `hairline`, `charcoal` text, same radius/height.
- **Text links:** `coral` Inter-SemiBold, no underline.
- Keep the iOS Pressable inner-View pattern everywhere (visual styles on an inner View; if a
  Pressable must fill a row, `flex:1` on a plain wrapper). See §6 gotchas.

## 4. Headers

- Screen titles: Geist-Bold ~26–28 `charcoal`, letterSpacing -0.3, on white canvas.
- Sub-screens: back chevron (`charcoal`) + title; flat white, hairline under header only if the
  content scrolls beneath it.
- Home has no big title bar — the floating search pill IS the header anchor.

## 5. Motion (Wave C — same recipe as vendor, coral-tuned)

- Staggered card entrances: `FadeInDown.duration(250).easing(Easing.bezier(0.22,1,0.36,1))`,
  delay-stepped per card/section (`.delay(60/140/...)`).
- Heart toggle: 150ms scale-pop (1 → 1.2 → 1) on save, coral fill.
- Sticky bar: slide-in from bottom on mount.
- Pressed scale 0.97 on CTAs.
- **All gated by `useReducedMotion()`** — no animation when reduced motion is on.
- No bounce, no elastic, no overshoot. Animate `opacity`/`transform` only.

## 6. Codebase gotchas (respect these)

- **iOS Pressable array-style bug:** function-style `style` props returning arrays drop
  flex/bg/padding on iOS. Put visual styles on an inner View; push `flex:1` to a plain wrapper.
- **Group cards clipping pressed bg:** shadow on the outer View, `overflow:'hidden'` + radius on an
  inner clip View (iOS kills shadows if both live on one View).
- **Photo dependency:** card quality depends on chef photo from the API. Missing image →
  `surface-soft` placeholder + a utensil glyph (`charcoal-soft`). Required, not optional.
- **Map screen** (`track.tsx`) uses react-native-maps — overlay styling only; do not touch map logic.

## 7. What does NOT change

- Business logic, hooks, React Query, navigation, data fetching, copy (except where spec says).
- Accessibility: labels, roles, 44pt touch targets, visible coral focus, reduced-motion.
- Fonts (Geist + Inter), spacing scale, motion easings/durations.
- Dark mode stays OUT of scope (queued phase, both apps together).

## File assignments

**Wave A** (4 parallel, disjoint): `(tabs)/_layout.tsx` + `(tabs)/index.tsx` + `components/chef/ChefCard.tsx`/`ChefGrid.tsx` ·
`chef/[id].tsx` + `components/chef/MenuItemCard.tsx` · `checkout.tsx` + `components/cart/*` ·
`(tabs)/favorites.tsx`.
**Wave B** (3 parallel): `order/[id]/index.tsx` + `order/[id]/track.tsx` + `components/tracking/*` +
`components/orders/*` · `(tabs)/orders.tsx` + `payment/result.tsx` · `(tabs)/profile.tsx` +
`social.tsx` + `catering.tsx`.
**Wave C** (main session): `(auth)/*` + `(onboarding)/*` accent pass (shared LoginScreen
parameterized, NOT forked) · motion layer · live QA on sim.

## TypeScript gate

`cd apps/mobile-customer && npx tsc --noEmit` — **77 pre-existing errors** (duplicate @types/react:
TS2786 JSX-component in packages/mobile-shared + a couple unused `@ts-expect-error` directives) are
NOT regressions. Only NEW errors fail the gate. Grep gate after each wave: no `herb` in
`apps/mobile-customer/app` or `components`; no `coral` outside `apps/mobile-customer`.
