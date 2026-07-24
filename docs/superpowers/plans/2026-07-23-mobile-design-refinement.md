# Mobile Design Refinement — Implementation Plan (Full Sweep)

Branch: `feat/design-refinement-sweep`. Executes
`docs/superpowers/specs/2026-07-23-mobile-design-refinement-design.md` (THE SPEC — every task
reads it in full before coding). Companion backlog doc is already delivered; no task implements
backlog items.

## Global Constraints (binding for every task)

1. **Read first, in order:** (a) THE SPEC; (b) your app's canonical spec —
   customer: `.planning/quick/260610-iy5-customer-app-v2-redesign-plan-airbnb-pal/CUSTOMER-UI-SPEC.md`,
   vendor: `.planning/quick/260610-ica-vendor-app-v2-visual-redesign-canvas-car/260610-ica-UI-V2-SPEC.md`.
   Where they conflict, THE SPEC wins. **Vendor reconciliation:** every `herb`/persimmon
   reference in the vendor canonical spec resolves to **Ink**; operational-positive status
   (ready/approved/open/verified) uses success green `#008A05` + tint. The vendor app must
   contain zero persimmon.
2. **Visual/structural changes only.** Never change: hooks' data contracts, React Query keys,
   API calls, navigation route names, Zustand store shapes, business logic, user-facing copy
   (unless THE SPEC §5 names it — e.g. rating zero-state "New"). If a refinement seems to
   require a logic change, STOP and report BLOCKED.
3. **Preserve these accessibility labels verbatim** (e2e/tests depend on them):
   "View cart — {n} items, {total}", "Proceed to checkout", "Place Order",
   "Accept ₹{total} order from {name}", "Reject order from {name}", "Confirm received",
   "I agree to the Terms of Service and Refund Policy for this order",
   "Add {item} to cart", "Increase/Decrease quantity of {item}", "Remove {item}".
   Add new labels freely; never rename these.
4. **Gates before reporting DONE:**
   - `npx tsc --noEmit` exit 0 in every touched app (`apps/mobile-customer`, `apps/mobile-vendor`)
     and, if `packages/mobile-shared` was touched, in BOTH apps.
   - Grep gates: `grep -rn "herb" apps/mobile-customer/app apps/mobile-customer/components`
     → empty; `grep -rn "coral" apps/mobile-vendor` → empty; no NEW `#RRGGBB` literals in
     screens you touched (tokens only) except values the canonical specs themselves define.
   - Run existing unit tests for anything you touched
     (`npx vitest run` in the app if a touched file has a co-located `.test.ts`).
   - Reduced-motion gate on any animation you add (`useReducedMotion()` from
     react-native-reanimated or the app's existing helper).
5. **Platform duality (THE SPEC §3.5 + §6):** every Pressable you touch gets
   `android_ripple` (bounded rows/buttons, borderless icons) AND the iOS pressed style
   (opacity/scale via inner-View pattern). Respect the two iOS gotchas in the customer
   canonical spec §6 (array-style Pressable bug; shadow + overflow clip separation).
6. **Commit style:** one commit per task, conventional message, no AI references, author
   sam123ben <samyak.rout@gmail.com>. Commit ONLY files your task owns.
7. Screens listed per task are exhaustive for that task — do not touch other screens.
   Shared components you must NOT restyle app-wide unless your task says so.

---

## Task 1: Shared UI primitives hardening (packages/mobile-shared)

**Files:** `packages/mobile-shared/src/ui/Button.tsx`, `Toast.tsx`, `EmptyState.tsx`,
`Skeleton.tsx`, `Input.tsx`, `Screen.tsx`, `Sheet.tsx`, `UndoSnackbar.tsx` (read the rest,
touch only these).

**Mandates:** These primitives serve four apps (customer, vendor, sunset delivery, admin) —
changes must be **additive/backward-compatible** (new optional props only; no renamed props,
no changed defaults that alter existing visuals except where listed):
- Button: ensure pressed states per platform (iOS opacity/scale 0.97, Android ripple),
  `minHeight` ≥48 default kept, `accentColor` prop path verified, loading state keeps width
  (no layout jump), disabled state distinct (40% opacity, no ripple).
- Toast/UndoSnackbar: add optional `bottomOffset` prop so apps can lift them above floating
  docks/tab bars (THE SPEC R4). Default behavior unchanged.
- EmptyState: verify icon + title + body + optional action composition; typography per brand
  (Geist title / Inter body); export prop to swap accent per app.
- Skeleton: shimmer runs on UI thread; reduced-motion → static block.
- Input: focus ring visible (2px accent), error state wired to
  `accessibilityState`/`aria` equivalents, 44pt min height.
- Screen/Sheet: safe-area + keyboard behavior verified (R9); Sheet radius 16.

**Out of scope:** LoginScreen/RegisterScreen/ForgotPasswordScreen (Task 6/12 do their app
passes), OnboardingScaffold visuals beyond gates.

**Verify:** gates in Global Constraints §4 for BOTH apps + `npx vitest run` in
`packages/mobile-shared`.

## Task 2: Customer — tab shell, Home, chef cards

**Files:** `apps/mobile-customer/app/(tabs)/_layout.tsx`, `(tabs)/index.tsx`, and the chef
card/grid + category-bar + dock components they render (locate under
`apps/mobile-customer/components/` — e.g. `chef/ChefCard.tsx`, `ChefGrid.tsx`,
`navigation/Dock.tsx`, home category bar, floating cart FAB).

**Mandates:** THE SPEC §5 Home + rubric: Airbnb category bar (charcoal underline selected, no
pill fill, R3 no-truncation, end padding); chef cards per customer spec §2.3 — photo 4:3
radius 12, heart scrim + coral saved pop, star charcoal, R1 "New" when 0 reviews, R2 photo
fallback; search pill air per §2.1; floating cart FAB + toasts clear the dock (R4, use
Task 1's `bottomOffset`); staggered card entrance motion per §3.5; tab bar per spec (white,
top-elevated) with ripple/press states.

**Verify:** Global gates; also `grep -n "0.0" ` on touched files → no rendered zero-rating
path remains.

## Task 3: Customer — chef detail + menu rows + reviews

**Files:** `apps/mobile-customer/app/chef/[id].tsx`, `app/chef/reviews/[id].tsx`, menu-item
row/modifier components they render (`components/chef/*`, `components/cart/ModifierSheet.tsx`
visual pass only).

**Mandates:** full-bleed photo header ≈40% viewport with bottom scrim; circular white overlay
buttons (back/share/heart) shadow[2]; **R2 fallback header** (surface-soft + utensil glyph —
current grey void is the named defect); R1 "New"; meta row (rating · time · min · delivery)
tabular; category chips R3; menu rows photo-right radius 12 with Add/stepper 44pt consistent
pressed states; sticky View-cart bar kept exactly (label preserved — Global §3); reviews
screen: review cards, empty state (R8), "New" zero-state.

**Verify:** Global gates.

## Task 4: Customer — cart + checkout

**Files:** `apps/mobile-customer/app/cart.tsx`, `app/checkout.tsx`, checkout-only components.

**Mandates:** cart to spec (header hierarchy, image radius 12, steppers 44pt, note field,
sticky footer hairline+shadow[2]); checkout section rhythm per THE SPEC §5 (Fulfilment
segmented control → Address cards → Summary → Fees → Time → Terms → sticky CTA); time-slot
picker: group headers + chip rows (R3), selected chip = coral fill/white text, "As soon as
ready" as the default hero option; terms row 44pt (label preserved); Place Order bar per
customer spec §2.5 with distinct disabled state; promo row Apply button ghost style; fee
breakdown typography tabular; keyboard behavior (R9).

**Verify:** Global gates + `npx vitest run` (cart-store tests).

## Task 5: Customer — order lifecycle screens

**Files:** `apps/mobile-customer/app/(tabs)/orders.tsx`, `app/order/[id]/index.tsx`,
`track.tsx`, `receipt.tsx`, `review.tsx`, `tip.tsx`, `report-issue.tsx`, `messages.tsx`,
`app/payment/result.tsx`, `app/payment/checkout.tsx`, tracking/order components.

**Mandates:** orders list cards + status chips per customer spec §2.7 + R8 triad; order
detail: status stepper component (coral progress, check glyphs, timestamps), photo-from-chef
card radius 12, **Confirm-received flow: replace Alert.alert dialogs with the branded Sheet
(Task 1) + in-screen success state** (labels preserved); track.tsx: overlay styling only
(map logic untouched) — floating status card per spec §2.6; payment result: three branded
states; receipt: tabular alignment; review/tip: form polish + R5; haptics per R7
(confirm-received, order placed already handled in result screen).

**Verify:** Global gates.

## Task 6: Customer — discovery, profile, onboarding, auth accent

**Files:** `apps/mobile-customer/app/(tabs)/favorites.tsx`, `search-dishes.tsx`,
`chefs-map.tsx` (overlays only), `(tabs)/profile.tsx`, `address/add.tsx`,
`(onboarding)/user-info.tsx`, `address.tsx`, `preferences.tsx`, `(auth)/login.tsx`,
`register.tsx`, `forgot-password.tsx` (wrapper props only — shared screens themselves are
Task 1-adjacent, param tweaks allowed).

**Mandates:** favorites = 2-col photo grid mirroring home cards + calm empty state; search:
input focus ring, result rows, R8; map: overlay buttons/cards per spec §2.6 (no map logic);
profile: iOS grouped rows on white with inset hairlines (existing Account/Change-password
row preserved with its gating); address form: input polish + R9; onboarding trio: coral
accent pass + progress affordance; auth wrappers: verify coral accent + link color
`coral-pressed` (THE SPEC §2 micro-adjustment).

**Verify:** Global gates.

## Task 7: Customer — remaining feature screens

**Files:** `apps/mobile-customer/app/meal-plans/index.tsx`, `meal-plans/[id].tsx`,
`book-meal-plan.tsx`, `subscriptions.tsx`, `meal-subscription/[chefId].tsx`, `catering.tsx`,
`catering/[id].tsx`, `group-order/[id].tsx`, `group/[code].tsx`, `wallet.tsx`,
`referral.tsx`, `loyalty.tsx`, `social.tsx`, `legal.tsx`, `privacy.tsx`, `terms.tsx`,
`eula.tsx`, `refund.tsx`, `data-privacy.tsx`.

**Mandates:** rubric application (R1–R12) + customer surface model: white canvas, hairline
separation, grouped rows for settings-like lists, status chips per §2.7, money tabular,
empty/loading/error triads on every list, sticky CTAs per §2.5 where present (book-meal-plan,
group order join). Legal set: readable typography rhythm (Inter, 15/1.5, section spacing),
no visual experimentation. No copy changes.

**Verify:** Global gates.

## Task 8: Vendor — tab shell + dashboard

**Files:** `apps/mobile-vendor/app/(tabs)/_layout.tsx`, `(tabs)/index.tsx`,
`components/vendor/PendingOrderCard.tsx` (+ dashboard-only components).

**Mandates:** vendor canonical spec §6 tab bar (**active = Ink**, not herb — Global §1),
§7 dashboard (TODAY 3-stat card with Geist-Bold tabular earnings, orders, rating with R1
"New"; merged ACTION-REQUIRED stack; IN-PROGRESS group card; quiet state bottom-anchored
only when quiet), §8 PendingOrderCard (Accept = ink primary flex-1 ≥48pt, Reject = ghost
~96pt, age-escalation chip kept, labels preserved — Global §3); greeting header + Open/Closed
pill per §4 (Open = ink fill + **green** dot + paper text); ripple/press per R5.

**Verify:** Global gates (incl. zero-persimmon grep).

## Task 9: Vendor — orders queue + order detail + status flow

**Files:** `apps/mobile-vendor/app/(tabs)/orders.tsx`, `app/orders/[orderId].tsx` (+ their
components).

**Mandates:** queue segmented control per vendor spec §5; order cards with §2 status chips
(ready = green tint, pending age-escalation kept); order detail: grouped cards kept, add
**read-only status timeline** (Accepted→Preparing→Ready→Out for delivery→Delivered with
timestamps from existing order data — display only); action bar: one primary ink action ≥48pt
+ ghost/destructive secondary (labels preserved); photo-required Ready step gets caption
affordance ("You'll attach a photo of the prepared order"); "Can't make this?" link style
per spec §3 (ink link, no underline); haptics R7 on accept/advance.

**Verify:** Global gates.

## Task 10: Vendor — menu + meal plans

**Files:** `apps/mobile-vendor/app/(tabs)/menu.tsx`, `app/menu/MenuItemForm.tsx`,
`menu/new.tsx`, `menu/[itemId]/edit.tsx`, `meal-plans/index.tsx`, `meal-plans/[id].tsx`,
`daily-menu.tsx`, `weekly-menu.tsx`, `prep.tsx` (+ `components/vendor/MenuItemRow.tsx`).

**Mandates:** menu rows per vendor spec §9 (44pt thumb R2 fallback, availability Switch,
₹ tabular, 56pt rows in group cards); category chips per §5 (ink fill active); MenuItemForm:
grouped sections, image-picker affordance card, input polish, save bar ≥48pt ink; meal-plan
screens: canvas+cards, chips per §2, weekly/daily grids with clear day headers, prep list
rows with checkable affordances kept functional.

**Verify:** Global gates.

## Task 11: Vendor — money + operations screens

**Files:** `apps/mobile-vendor/app/earnings.tsx`, `analytics.tsx`, `payout.tsx`,
`capacity.tsx`, `settings.tsx`, `notification-preferences.tsx`, `language.tsx`,
`profile.tsx`, `(tabs)/more.tsx`.

**Mandates:** canvas+cards throughout; earnings/analytics: stat cards Geist-Bold tabular,
segmented period control per §5, charts untouched; payout: bank-details card (bank-only —
UPI was just removed, don't reintroduce), status chips per §2; capacity: steppers/switch rows
56pt; settings: group cards with inset hairlines (existing toggles + gated Change-password
row preserved exactly); More tab: §9 row anatomy (36pt icon circle bone bg).

**Verify:** Global gates + `npx vitest run` (payout tests).

## Task 12: Vendor — trust, support, onboarding, auth, legal

**Files:** `apps/mobile-vendor/app/reviews.tsx`, `review/[reviewId].tsx`,
`admin-requests.tsx`, `admin-requests/[id].tsx`, `cancel-requests.tsx`,
`support/index.tsx`, `support/new.tsx`, `support/[id].tsx`, `documents/renew.tsx`,
`(onboarding)/personal-info.tsx`, `kitchen-details.tsx`, `operations.tsx`, `documents.tsx`,
`payout.tsx`, `policies.tsx`, `review.tsx`, `pending.tsx`, `(auth)/login.tsx`,
`register.tsx`, `forgot-password.tsx` (wrappers), `terms.tsx`, `privacy.tsx`, `eula.tsx`,
`legal.tsx`, `chef-agreement.tsx`, `catering.tsx`, `upgrade-required.tsx`.

**Mandates:** rubric + vendor surface model on every screen: reviews (rating rows, reply
affordance, R1/R8), requests/cancellations (status chips §2, clear approve/decline hierarchy
per §3), support (thread rows, composer R9), onboarding (OnboardingScaffold canvas+cards,
progress affordance, input polish, ≥48pt CTAs), auth wrappers ink accent, legal set
typography rhythm. No copy changes; no flow changes.

**Verify:** Global gates.

## Task 13: Cross-app audit sweep (a11y · motion · consistency)

**Files:** read-mostly across both apps; fix-in-place anywhere the audit fails.

**Mandates:** run and fix: (a) grep sweeps from Global §4 repo-wide; (b) a11y audit — every
Pressable/TouchableOpacity in both `app/` trees has role+label, form fields have
error association, decorative images `accessible={false}`; (c) touch-target audit ≥44/48pt;
(d) motion audit — every Reanimated/Animated usage gated by reduced-motion, durations within
§3.5 bands, no layout-property animation, no bounce easings anywhere
(`grep -rn "bounce\|elastic\|overshoot"`); (e) tabular-numeral audit on ₹/ETA/counts;
(f) `console.log` leftovers in touched trees removed; (g) R13 flicker audit — grep both apps for React Query list/detail screens missing `placeholderData`/stale-render handling, `entering` animations keyed by index or re-firing on data change, remote images without placeholder+fade, and persisted-store screens rendering empty pre-hydration — fix what you find; (h) R14 spot-audit — validation-error scroll/focus present on the big forms (checkout, menu item form, onboarding). Produce a short report of what was
fixed in the task report file.

**Verify:** Global gates in both apps.

---

## Final validation (controller-run, not subagent tasks)

1. Whole-branch review (SDD final reviewer, most capable model).
2. Emulator visual pass per app (screenshots archived to scratchpad).
3. Full e2e regression on prod API with test Razorpay: happy path (delivery), pickup path,
   chef reject path (+ customer refund state), post-delivery review path (customer submits →
   vendor Reviews shows it). Per THE SPEC §9.
4. iOS simulator visual/safe-area pass (logged-out reachable screens).
