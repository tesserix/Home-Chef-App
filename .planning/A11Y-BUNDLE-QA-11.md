# #11 — Accessibility + bundle-size QA (customer app)

Static audit of `apps/mobile-customer` (`app/` + `components/`). Baseline is strong: 171 `accessibilityLabel` + 132 `accessibilityRole`, every icon-only button already labelled, reduced-motion honored in 4/5 animation files. Gaps below, with the quick wins fixed in `feat/a11y-quick-wins` and the device-dependent items deferred to a real production build.

## Fixed in this PR (quick wins)
- **Cart steppers** (`components/cart/CartSheet.tsx`): decrease/increase/remove now have `accessibilityRole="button"` + `hitSlop={10}` (were 28px targets, no role) and item-specific labels; food thumbnail marked `accessible={false}` (decorative — dish name is adjacent).
- **Modifier sheet** (`components/cart/ModifierSheet.tsx`): qty steppers + close button given `accessibilityRole="button"` + `hitSlop`.
- **Reduced motion** (`components/orders/ActiveOrderStack.tsx`): the expand/collapse `LayoutAnimation` — the one motion path that ignored the setting — is now gated by `useReducedMotion()`.
- **Contrast** (`app/checkout.tsx`): promo-code placeholder `#9CA3AF` (~2.8:1 on canvas, below AA) → `#717171` (the charcoal-soft token used everywhere else), and the input got an `accessibilityLabel`.
- **Unlabeled inputs**: added `accessibilityLabel` to report-issue description, review title + body, chat message, and custom-tip amount.

## Remaining — follow-ups (not in this PR)
1. **Hand-rolled header back buttons missing `accessibilityRole`** (~10 screens: subscriptions, referral, book-meal-plan, meal-plans, meal-subscription, order tip/report-issue, group-order). Labelled but no "button" trait. Best fix: adopt the shared `components/ScreenHeader.tsx` (already correct: role + label + hitSlop) rather than per-screen patches. Several of these screens are behind v1 feature-flags (tiffin/group/meal-plans), so lower priority for launch.
2. **Text-label buttons missing role** in group-order / group screens (`group/[code].tsx`, `group-order/[id].tsx`). Group orders are flag-hidden for v1.
3. **Form error live-regions (systemic):** 0 `accessibilityLiveRegion` across the app — validation errors in onboarding (`user-info`, `address`) and `checkout` render but aren't announced. Add `accessibilityLiveRegion="polite"` (Android) / `AccessibilityInfo.announceForAccessibility` on submit. Highest-value follow-up for VoiceOver/TalkBack users.
4. **Order "ready photo" + issue-photo thumbnails** — add a label or mark decorative (low severity).

## Needs a real production build (device QA — carry into #20 TestFlight pass)
- **VoiceOver** sweep of primary flows (browse → order → pay → track).
- **Hindi** locale: confirm chef-facing copy renders without truncation.
- **Bundle size** measurement + asset-compression pass (Hermes already locked).
- **Contrast watch:** `charcoal.soft` `#717171` on canvas ≈ 4.5:1 — passes AA for normal text by a hair; audit at `text-xs` on a device.

## Notes
- Accent token is Airbnb coral `#FF385C` (token still named `herb`/`coral`), not persimmon — colors audited against that.
