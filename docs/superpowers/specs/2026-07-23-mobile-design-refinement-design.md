# Mobile Design Refinement — Full Sweep (Customer + Vendor)

Date: 2026-07-23 · Branch: `feat/design-refinement-sweep` · Status: approved direction (owner: refine within sanctioned systems, full sweep, iOS + Android)

## 1. Goal

Bring every screen of `apps/mobile-customer` and `apps/mobile-vendor` up to best-in-class
execution of the two owner-sanctioned design systems — the quality bar of Airbnb, Uber Eats
(merchant side), Cash App — grounded in Apple HIG and Material 3 platform guidance and WCAG 2.1 AA.
Visual/structural refinement only: **zero business-logic, navigation, data-fetching, or copy
changes** except where this spec names them. Both platforms (iOS + Android) are first-class.

## 2. Direction decision + alternatives considered

The owner asked for exploration of alternative palettes/directions before committing. Considered:

| Direction | Verdict | Why |
|---|---|---|
| **A. Keep sanctioned dual system** — customer = Airbnb (white · charcoal · rausch coral `#FF385C`); vendor = Uber-merchant monochrome (ink CTAs · canvas+cards · functional green) | **CHOSEN** | Owner-sanctioned (2026-06-10 pivots), ~80% implemented, matches category-best references (Airbnb consumer / Uber Eats Manager). The gap is execution, not direction. |
| B. Unify both apps on Paper·Ink·Persimmon (brand doc's editorial orange) | Rejected | Reverses two explicit owner pivots; 49-file token rename; weeks of churn with no UX gain. |
| C. Saffron/turmeric + deep-green "Indian home kitchen" | Rejected | Reads artisanal-kitsch — explicitly an anti-reference in `.impeccable.md` ("terracotta-and-cream homemade kitsch"). |
| D. Deep teal + warm neutrals (Deliveroo-adjacent) | Rejected | Cold for food; appetite suppression; no brand equity. |
| E. Tomato-red saturation (Swiggy/Zomato/DoorDash class) | Rejected | Named anti-reference; urgency-chasing contradicts "Confident · Appetizing · Quietly modern". |

Micro-adjustment adopted with A (the only palette change in this sweep):
- **Customer text links on white use `coral-pressed` `#E00B41`** (≈4.9:1, AA-clean) instead of
  `#FF385C` (≈3.9:1, fails AA for body-size text). Coral `#FF385C` remains for **fills** (CTA
  bg + white label ≥16 SemiBold), selected states, heart-saved, focus — the look is unchanged.

## 3. Binding references (precedence order)

1. This spec (resolves conflicts below).
2. `.impeccable.md` — brand personality, typography (Geist display / Inter body, tabular
   numerals), motion (ease-out-quart entrances, 150/250/400ms, no bounce/elastic/overshoot,
   `opacity`/`transform` only, reduced-motion mandatory), a11y baseline, spacing rhythm.
3. Customer: `.planning/quick/260610-iy5-customer-app-v2-redesign-plan-airbnb-pal/CUSTOMER-UI-SPEC.md` (palette table, surface model, signature patterns §2, buttons §3, headers §4, motion §5, gotchas §6).
4. Vendor: `.planning/quick/260610-ica-vendor-app-v2-visual-redesign-canvas-car/260610-ica-UI-V2-SPEC.md` **as amended by the `.impeccable.md` vendor note**: the vendor app uses **ZERO persimmon** — every `herb` reference in that spec resolves to **Ink** (tab-active, links, callouts) except *operational-positive* status (ready / approved / open / verified) which uses functional **success green `#008A05`** with its tint. Ink fill = primary buttons; amber = attention; destructive = problems.

Platform guidance: Apple HIG (navigation/back gestures, safe areas, haptics, modality) and
Material 3 (touch ripple, back handling, edge-to-edge, elevation semantics) inform *behavioral*
platform correctness; the visual brand stays one cross-platform language per app (the
Airbnb/Uber approach — HIG and M3 both endorse brand-consistent cross-platform apps).

### 3.5 Reference-grounded motion + smoothness contract (HIG / Material 3)

The owner's bar is "smoother and very professional." Concrete, non-negotiable values,
reconciling the brand curve with the platform systems:

- **Signature easing stays the brand's** `cubic-bezier(0.22, 1, 0.36, 1)` for entrances —
  it sits inside M3's *emphasized-decelerate* family (`cubic-bezier(0.05, 0.7, 0.1, 1)`), so
  entrances feel platform-native on Android and HIG-calm on iOS. State changes use
  `cubic-bezier(0.4, 0, 0.2, 1)` (M3 standard). **Exits accelerate** (M3 rule): leaving
  elements use ~150ms with standard-accelerate feel — never slow, attention-grabbing exits.
- **Duration bands (M3-aligned):** micro (press, toggle, chip) 100–150ms · component
  (card entrance, sheet content, chip group) 250ms · surface (sheet/modal/page) 300–400ms.
  Nothing animated exceeds 400ms. Stagger steps 40–60ms, max 3 steps.
- **60fps discipline:** every animation runs on the UI thread (Reanimated / `useNativeDriver`)
  and animates ONLY `opacity`/`transform`. No `setState`-driven per-frame animation, no
  layout-property animation, no shadow animation.
- **List smoothness:** long lists use `FlatList`/`FlashList` hygiene — stable `keyExtractor`,
  memoized row components, `expo-image` with blurhash + 150ms fade (no pop-in), no inline
  anonymous renderItem closures capturing whole screens.
- **Perceived speed:** skeletons appear instantly (no spinner-then-content double paint);
  optimistic press states within 50ms; sticky CTAs slide in once per screen mount, not per
  re-render.
- **iOS (HIG):** ≥44pt targets; edge back-swipe never blocked; sheets for transient tasks;
  haptics are semantic (`impactLight` for add/confirm, `notificationSuccess` for order
  placed/accepted) and never decorative; respect Reduce Motion (cross-fade instead of slide).
- **Android (M3):** ≥48dp targets; `android_ripple` on every touchable (bounded on rows/buttons,
  borderless on bare icons); predictive-back friendly (no custom back traps); edge-to-edge with
  correct inset padding; elevation communicates hierarchy only (our 3-step shadow scale maps to
  M3 levels 1/2/3 — never decorative).

## 4. Cross-cutting refinement rubric (applies to every screen)

**R1 — Ratings zero-state.** Never render `★ 0.0 (0)`. When `reviewCount === 0`, show a `New`
text chip (customer: `surface-soft` bg + `charcoal-soft` text; vendor: `mist` bg + ink.soft
text). Applies to chef cards, chef detail, vendor dashboard rating stat, reviews screens.

**R2 — Image fallbacks.** Every remote image gets: `surface-soft`/bone placeholder + utensil
glyph (`charcoal-soft`/ink.soft) when absent, blurhash placeholder + 150ms fade transition when
loading (expo-image already in both apps). No grey voids, no single-letter placeholders on
photo surfaces (letter avatars remain fine for *people*).

**R3 — Text never truncates mid-word in chips/tabs.** Category tabs/chips size to content
(`paddingHorizontal` ≥ 12, no fixed widths); horizontal scroll containers get
`contentContainerStyle` padding so the last item is fully visible ("Chines"/"Continent"
truncation class of bug).

**R4 — Floating-layer discipline.** Toasts, snackbars, and FABs must never overlap the tab
bar/dock or each other: they stack above the dock with an 8pt gap (use the existing dock
clearance hooks; Toast gains an offset prop if missing). Dev-only LogBox is exempt.

**R5 — Touch targets + press states.** ≥44pt (customer) / ≥48pt (vendor primary actions).
Every Pressable has a visible pressed state: iOS = opacity/scale per spec; Android =
`android_ripple` (borderless for icons, bounded for rows/buttons) — never a dead press.

**R6 — Tabular numerals** on every price, ID, ETA, count, rating (`fontVariant:
['tabular-nums']` or the token style).

**R7 — Haptics on money-moments.** expo-haptics light impact on: add-to-cart, place-order
success, order accept (vendor), status advance (vendor), confirm-received (customer).
Never on mere navigation. Wrap in try/catch (simulators).

**R8 — Empty/loading/error triad.** Every list screen has: branded EmptyState (icon + one
sentence + optional action), Skeleton loading (no spinners on full screens), and a retry
error state. No raw "No data" strings.

**R9 — Safe areas + keyboards.** Every screen respects insets (`SafeAreaView`/`useSafeAreaInsets`
— including Android edge-to-edge); every form scrolls above the keyboard
(KeyboardAwareScrollView already shared).

**R10 — A11y.** Every interactive element: `accessibilityRole` + meaningful
`accessibilityLabel`; form fields keep label/error association; decorative images
`accessible={false}`. Preserve existing labels verbatim where e2e/tests rely on them
(e.g. "View cart — N items…", "Accept ₹… order from …", "Proceed to checkout").

**R11 — Motion.** Entrances `FadeInDown 250ms ease-out-quart` staggered ≤3 steps; pressed
scale 0.97 on CTAs; all gated by reduced-motion; no layout-property animation. Remove any
bounce/elastic if found.

**R12 — No new hex literals in screens.** Colors trace to tokens (customer: NativeWind classes
/ `customerColors`; vendor: `theme`). Exceptions already sanctioned by the June specs stay.

## 5. Screen-specific mandates (observed defects to fix)

Customer:
- **Home**: category bar → Airbnb style (charcoal underline selected, no pill fills), R3 fix;
  chef cards per spec §2.3 (R1, R2, star = charcoal not gold); floating cart FAB clears dock
  (R4); search pill air per §2.1.
- **Chef detail**: full-bleed photo header ~40% viewport with scrim + circular white overlay
  buttons (share/heart/back) — R2 fallback when no photo (current grey void + icon is the
  defect); R1 for `0.0 (0)`; menu rows photo-right per spec; sticky View-cart bar stays
  (recently added — keep its a11y label).
- **Cart** (`cart.tsx`, new): bring to spec — header hierarchy, image radius 12, qty steppers
  44pt, sticky footer with top hairline + shadow[2].
- **Checkout**: group sections with clear rhythm (Fulfilment toggle → Address → Summary →
  Fees → Time → Terms → sticky CTA); time-slot picker: group headers (`Today · Dinner`)
  with chip rows (R3), selected = coral fill + white text; terms row 44pt with checkbox
  visual; Place Order bar per spec §2.5 (enabled/disabled states distinct).
- **Order tracking/detail**: status stepper as shared-quality component (coral progress,
  check icons, timestamps); "photo from your chef" card polish; Confirm-received: replace
  `Alert.alert` confirm + thanks with branded Sheet/dialog + success state in-screen.
- **Payment result**: success/pending/failure states branded (current success screen is close).
- **Orders tab**: order cards with status chips per spec §2.7, R8 triad.
- Favorites, search-dishes, map overlays, profile (grouped rows), addresses, auth accent pass,
  onboarding, and remaining feature screens (meal plans, subscriptions, catering, group
  orders, wallet, referral, loyalty, social, legal set): apply rubric + spec patterns.

Vendor:
- **Dashboard**: TODAY stat card (earnings Geist-Bold tabular / orders / rating with R1);
  merged ACTION-REQUIRED stack; PendingOrderCard hero per spec §8 (Accept primary ink +
  Reject ghost); quiet state stays bottom-anchored only when quiet.
- **Orders queue**: segmented control per spec §5; cards with status chips §2 (green=ready,
  amber=pending age-escalation kept).
- **Order detail**: grouped cards kept; status action bar: single primary ink action ≥48pt +
  destructive/ghost secondary; add a compact status timeline (Accepted → Preparing → Ready →
  Out for delivery → Delivered with timestamps — read-only, data already present); photo-required
  "Ready" step gets an explanatory affordance (caption under the action: "You'll attach a
  photo of the prepared order").
- **Menu**: item rows per spec §9 (44pt thumb, availability switch, ₹ tabular); MenuItemForm
  grouped sections + image picker affordance.
- **More/settings/profile/earnings/analytics/payout/reviews/requests/support/onboarding/auth**:
  canvas+cards + rubric throughout; earnings/analytics numerals Geist-Bold tabular; charts remain
  as-is (no new chart work).

## 6. Platform correctness (iOS + Android)

- Android: `android_ripple` press feedback everywhere (R5); hardware back never traps
  (expo-router default — verify modals/sheets close on back); edge-to-edge insets (R9);
  status-bar style per screen background.
- iOS: safe-area headers; sheet modality for pickers; no Android-style caps/elevation borders;
  haptics via expo-haptics (R7).
- Both: identical brand visuals; only *feedback + system affordances* diverge.

## 7. Out of scope (explicit)

- Dark mode (queued as its own phase per June specs).
- `apps/mobile-delivery` (sunset) and `apps/mobile-admin`, all web portals.
- New features, chart libraries, copy rewrites, i18n, API changes, dependency upgrades.
- Photo *content* (chef-uploaded imagery quality is a data problem).

## 8. Quality gates (every task)

1. `npx tsc --noEmit` exit 0 in the touched app (both apps currently at 0 — no regressions).
2. Grep gates: no `herb` in customer app; no `coral` outside customer app; no new hex literals
   in screens (`#\w{6}` diff-scan against the exceptions list).
3. Existing unit tests pass (`payout.test.ts`, cart-store tests, etc. where present).
4. No changes to: hooks' data contracts, route names, store shapes, API calls, test-relied
   accessibility labels (R10 list).
5. Reduced-motion audit on any added animation.

## 9. Validation after the sweep (controller-run, on emulator + prod API)

1. Visual pass per cluster on Pixel_9 emulator (screenshots archived).
2. Full e2e regression: login (both apps) → browse → cart → checkout (delivery) → Razorpay
   test card → chef accept → preparing → ready(photo) → out-for-delivery → delivered →
   customer confirm-received.
3. Alternative paths: **pickup order** end-to-end; **chef reject** (+ customer sees rejection
   & refund state); **customer leaves a review** post-delivery (and vendor sees it in Reviews).
4. iOS: build the customer app once on the iOS simulator for a layout/safe-area visual pass
   (auth is keychain-blocked on sim — visual-only, logged-out screens + any state reachable
   without sign-in; full iOS behavioral pass happens on the next TestFlight build).

## 10. Companion deliverable — retention feature backlog

Owner request: alongside the design sweep, produce a curated backlog of the best-possible
*missing* functionality for this food-service platform — features that measurably drive
repeat usage — as a document only (no implementation in this sweep). Delivered at
`docs/superpowers/specs/2026-07-23-retention-feature-backlog.md`.
