# Vendor App v2 Visual Language — "Canvas + Cards"

Owner feedback 2026-06-10: the flat hairline-minimal look reads "basic and outdated."
This spec shifts the vendor app to a richer modern-consumer feel (Cash App / Revolut / Airbnb
class) while keeping the brand: paper/ink/persimmon palette, Geist + Inter, single accent.

ALL values come from `theme` (`@homechef/mobile-shared/theme`). No hardcoded colors.

## 1. Surface model (the big flip)

- **Page canvas:** `theme.colors.bone` (#F5F5F4) — NOT white. Every screen root.
- **Content surfaces:** white `theme.colors.paper` cards, `borderRadius: theme.radius.lg` (16),
  `...theme.shadow[1]`. No border. Cards sit on the bone canvas with `marginHorizontal: spacing[4]`.
- **Interactive/action cards** (pending orders, primary content): `...theme.shadow[2]`.
- **Grouped lists** (settings rows, nav rows, history rows, menu items): iOS-modern "group card" —
  one white rounded card (radius.lg) containing rows separated by inset hairlines
  (`marginLeft` aligned to text, `borderBottomColor: mist.DEFAULT`, last row no border).
  NOT edge-to-edge hairlines on the page.
- Section labels stay: Inter-SemiBold caption, letterSpacing 1.4, ink.muted, UPPERCASE —
  placed above each card group with spacing[2] gap, spacing[6] between sections.

## 2. Status chips (replace bare dots + grey text)

Pill chip: `paddingHorizontal: spacing[2]`, `paddingVertical: 3`, `borderRadius: theme.radius.full`,
Inter-SemiBold caption text. Tint bg + dark text of same hue:
- pending / preparing → `amber.tint` bg, text `#7A5C16` (amber dark — use literal only if no token; prefer ink.DEFAULT on amber.tint)
- ready → `herb.tint` bg, `herb.soft` text
- accepted → `info.tint` bg, `info.DEFAULT` text
- delivered / picked_up → bg `#E7F2E9`-class: use `diet.veg` at low alpha is NOT available — use `mist.DEFAULT` bg with `diet.veg` text
- cancelled / rejected → `destructive.tint` bg, `destructive.DEFAULT` text
Age escalation (PendingOrderCard): age text/chip shifts ink.soft → amber.DEFAULT → destructive.DEFAULT at 5/10 min (keep existing ticker logic).

## 3. Buttons

- **Primary:** ink.DEFAULT fill, paper text, `borderRadius: theme.radius.md` (12), `minHeight: 52`,
  Inter-SemiBold body. Pressed = 0.85 opacity.
- **Secondary/ghost:** paper bg, `borderWidth: 1`, `borderColor: mist.strong`, ink text, same radius/height.
  (Replaces naked underlined text links for Reject etc. Reject = ghost button ~96px wide beside full-flex Accept.)
- **Text links:** herb.DEFAULT Inter-SemiBold, no underline (modern apps drop underlines).
- Keep iOS Pressable inner-View pattern everywhere (visual styles on inner View).

## 4. Screen headers

- Title: Geist-Bold 28, ink, letterSpacing -0.3 (unchanged) but now sits on bone canvas with
  paddingBottom spacing[4].
- Dashboard: two-line greeting — caption "Good morning/afternoon/evening" (Inter, ink.muted) over
  name (Geist-Bold 24). Open/Closed control unchanged in behavior; restyle as pill
  (radius.full, minHeight 40): Open = ink fill + herb dot + paper text; Closed = paper bg,
  mist.strong border, ink text. shadow[1].
- Sub-screens keep back-chevron + title pattern; just inherit canvas.

## 5. Segmented tabs (Queue/History, week/month/year, review filters)

iOS-style segmented control: track = mist.DEFAULT bg, radius.md (12), padding 3;
active segment = paper bg, radius 9, shadow[1], ink.DEFAULT Inter-SemiBold; inactive = ink.muted.
minHeight 40. (Replaces bare-text + underline tabs.) For scrolling category tabs (menu) keep
horizontal scroll but use chip style: active = ink fill + paper text pill, inactive = paper bg +
mist.strong border pill, radius.full, minHeight 36.

## 6. Tab bar (app/(tabs)/_layout.tsx)

White paper bar, NO top hairline — use shadow[3]-like top elevation:
shadowOffset {0,-2}, shadowOpacity 0.06, shadowRadius 12, elevation 8.
Active tab: herb.DEFAULT icon + label (selected-state accent IS allowed), strokeWidth 2.4.
Inactive: ink.muted. Remove the top-line indicator entirely. Label Inter-SemiBold 11.

## 7. Dashboard specifics (app/(tabs)/index.tsx)

- Merge the two "ACTION REQUIRED" sections into ONE label + one card stack (admin requests first,
  then doc expiry). Compact alert card: amber.tint bg, radius.md, 2 rows max —
  row 1: dot + status label + CTA link right-aligned (herb, no underline);
  row 2: single-line body (numberOfLines 1).
- PendingOrderCard becomes the hero: white, radius.lg, shadow[2] (see §8).
- IN PROGRESS: group card (white, radius.lg, shadow[1]) of rows with status chips + ₹ totals.
- Today strip → "TODAY" stat card: white card, 3 columns — earnings (Geist-Bold 24 tabular),
  orders count, rating ★ (Inter values, caption labels under each, ink.muted).
  When screen has active content render inline after IN PROGRESS (not bottom-anchored);
  keep bottom-anchored placement ONLY for the quiet state.
- Quiet state: keep copy; center-left block unchanged otherwise.

## 8. PendingOrderCard (components/vendor/PendingOrderCard.tsx)

White paper card, radius.lg, shadow[2], padding spacing[4]:
- Row 1: customer name (Inter-SemiBold body) left; total right (Geist-Bold 22, tabular, ink).
- Row 2: items summary (Inter bodySm ink.soft) + age chip right (escalating per §2, keep ticker).
- Special instructions: herb.tint callout radius.DEFAULT (when showInstructions).
- Row 3: Reject ghost button (~96 wide) + Accept primary (flex 1), both minHeight 48, radius.md.

## 9. Lists/rows (orders history, menu items, more nav)

Rows live inside group cards (§1). Row anatomy: 44pt thumb (menu) or 36pt icon circle
(bone bg, radius.full, ink.soft icon — More nav), title Inter-SemiBold bodySm,
caption Inter caption ink.muted, right side: ₹ (Geist-Bold tabular) or chevron or Switch.
minHeight 56. Pressed state = bone bg on the row.

## 10. What does NOT change

- Palette tokens, fonts, motion easings/durations, accessibility labels, all business logic,
  hooks, navigation, polling, haptics, copy (except where spec says).
- Persimmon discipline: accent for primary action/selected/focus/success + links only.
- No gradients, no glassmorphism, no bounce. `prefers-reduced-motion` untouched.
- DietIcon FSSAI colors untouched.

## File assignments (Wave A core / Wave B secondary)

Wave A: (tabs)/_layout.tsx · (tabs)/index.tsx · components/vendor/PendingOrderCard.tsx ·
(tabs)/orders.tsx · (tabs)/menu.tsx + components/vendor/MenuItemRow.tsx · (tabs)/more.tsx
Wave B: earnings.tsx · orders/[orderId].tsx · reviews.tsx · analytics.tsx · payout.tsx ·
settings.tsx · notification-preferences.tsx · profile.tsx · admin-requests.tsx ·
menu/MenuItemForm.tsx · components/vendor/UndoSnackbar.tsx

TypeScript gate: `cd apps/mobile-vendor && npx tsc --noEmit` — 107 pre-existing errors
(duplicate @types/react: TS2305 lucide / TS2786 JSX-component) are NOT regressions; only NEW
errors fail.
