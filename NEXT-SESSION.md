# Next session — Home Chef mobile redesign continuation

## Where we left off (2026-06-01)

**Brand direction locked:** Uber-like utility UI for vendor + driver; Airbnb-style cream/coral for customer when we get there. See `~/.claude/projects/-Users-Mahesh-Sangawar-personal-tesserix-new-Home-Chef-App/memory/feedback_no_editorial_style.md` for the full rationale.

**Tokens (Uber-faithful):**
- `paper: #FFFFFF` · `bone: #F5F5F4` · `mist: #E5E5E5 / #D6D6D6`
- `ink: #0E0E0C / #525252 / #888888`
- `herb (persimmon): #C2410C / #9A3412 / #FFEDD5` — **accent only**, NEVER a primary CTA
- Primary `<Button>` variant fills with **ink** (not persimmon)
- Links render as **ink underlined SemiBold** (not persimmon)
- Geist-Bold display + Inter 400/500/600 body

Source of truth: `packages/mobile-shared/src/theme/tokens.ts`. Mirrored in `apps/mobile-vendor/tailwind.config.js`. 8 tests in `packages/mobile-shared/src/__tests__/theme-tokens.test.ts` lock the values.

**Shared primitives live in `packages/mobile-shared/src/ui/`:**
- `<Screen>` · `<Button>` · `<Input>` · `<EmptyState>` · `<Skeleton>` / `<SkeletonGroup>` · `<ToastProvider>` + `useToast()` · `<Sheet>` (gorhom) · `<UndoSnackbarProvider>` + `useUndoSnackbar()` · `<OnboardingScaffold>`
- All exported via `@homechef/mobile-shared/ui`
- `resolveAuthErrorMessage()` in `@homechef/mobile-shared/auth` maps cryptic auth errors to plain English

**UX policy:** `docs/UX-PATTERNS.md` covers feedback channels, save policy, loading ladder, destructive checklist.

## Done in vendor app

| Screen | State |
|---|---|
| `(auth)/login` | ✅ Uber-faithful: black CTA, ink underlined links, persimmon dropped |
| `(auth)/register` | ✅ Same, with Google + Apple |
| `(auth)/forgot-password` | ✅ Same |
| `(onboarding)/personal-info` | ✅ Uses `<OnboardingScaffold>` |
| `(tabs)/_layout` | ✅ Ink active / muted inactive, hairline top border |
| `(tabs)/index` (dashboard) | ⚠️ Tokens migrated but user said **"looks basic"** — needs structural rethink, see below |
| Splash overlay | ✅ White bg, ink spinner, proper auth-key-keyed routing |
| `ErrorBoundary` | ✅ Wraps RootLayout |
| `multipartConfig()` helper | ✅ Used by all 5 upload sites |

## Not yet redesigned

**Onboarding wizard (6 screens — apply OnboardingScaffold pattern):**
- `(onboarding)/kitchen-details.tsx` — 9 fields incl. address; consider chunking via Scaffold subsections
- `(onboarding)/operations.tsx`
- `(onboarding)/documents.tsx`
- `(onboarding)/policies.tsx`
- `(onboarding)/review.tsx`
- `(onboarding)/pending.tsx`

**Main app tabs:**
- `(tabs)/orders.tsx` — order list, accept/reject actions, status flows
- `(tabs)/menu.tsx` — list of items + category filter (button card + create flow are already mostly working)
- `(tabs)/more.tsx` — likely a settings/profile menu

**Secondary screens:**
- `analytics.tsx` · `earnings.tsx` · `reviews.tsx` · `review/[reviewId].tsx`
- `menu/new.tsx` · `menu/[itemId]/edit.tsx`
- `profile.tsx` · `settings.tsx`

## What the user said about dashboard

> "looks basic"

It's currently a stack of: greeting → optional next-action → 2×2 stat grid → toggle → recent orders. The structure is fine for Uber's compact ops surface, but lacks visual rhythm. Things to try in restructure:

1. **Hero stat** — make today's orders or earnings the single hero number at the top (40-50pt Geist Bold), then small chips for the other 3 stats below. Right now all four numbers compete equally.
2. **Time-of-day context** — morning shows prep window for lunch; evening shows tomorrow's prep window. The "what's next" should be temporal, not just queue-based.
3. **Order velocity sparkline** — a tiny inline chart showing the last 7 days, ink hairline strokes. Gives the screen something visual without going decorative.
4. **Combined toggle + status** — replace the standalone "Accepting orders" row with a top-right status pill ("Open" / "Closed") that's tappable. Saves a row, reads more pro.
5. **Smarter recent orders** — show pending orders as actionable rows (Accept / Reject inline) rather than just status badges. Pending orders are the only thing the chef actually clicks on the dashboard.

User signaled openness to "complete restructure" — not just visual restyle. Don't be timid.

## Open backend items

- Customer + delivery mobile apps still build but use old tokens + Button — port the same patterns when redesigning those.
- Need to wire customer app with Airbnb cream/coral theme via per-app token override (not yet implemented; mobile-shared's `theme` is Uber for now).
- Apple Sign-In: client + GIP configured for vendor's bundle ID. Customer + delivery will need same setup.
- `auto_login_502` resolved — root cause was Docker postgres being stopped. If it returns, check `docker ps` first.

## How to continue

1. Read this file
2. Read `~/.claude/projects/-Users-Mahesh-Sangawar-personal-tesserix-new-Home-Chef-App/memory/feedback_no_editorial_style.md` for the brand direction
3. Read `packages/mobile-shared/src/theme/tokens.ts` for the design tokens
4. Pick a starting surface — recommendation: **redesign the dashboard structurally first** (user pain point), then onboarding wizard (next surface a vendor sees), then orders tab (highest-frequency usage)
5. Per redesigned screen: import primitives from `@homechef/mobile-shared/ui`, use `theme` from `@homechef/mobile-shared/theme`, no NativeWind className for new code (use StyleSheet for consistency with the new screens)
6. After each screen: build via `cd apps/mobile-vendor && find . -maxdepth 1 -name 'build-*.tar.gz' -delete && EXPO_NO_TELEMETRY=1 npx eas-cli build --local --platform ios --profile local-sim --non-interactive`
7. Install on iPhone 17 Pro sim: `xcrun simctl install AD109A46-2F99-43C3-8AAA-FEE68DC8499E /tmp/hcv-app/HomeChefVendor.app`

## Suggested next-session prompt to paste

```
Continue the Home Chef mobile redesign. Read NEXT-SESSION.md and
~/.claude/projects/-Users-Mahesh-Sangawar-personal-tesserix-new-Home-Chef-App/memory/feedback_no_editorial_style.md
before starting. Direction is locked: Uber-like (white + ink + persimmon
accent only) for vendor and driver, Airbnb-style for customer later.

Start with a structural redesign of the vendor dashboard
(apps/mobile-vendor/app/(tabs)/index.tsx) — the user said it "looks basic"
and is open to a complete restructure, not just visual polish. Use the
suggestions in NEXT-SESSION.md as a starting menu, but feel free to
propose your own composition. The screen should reward a chef glancing
at it during prep, surface the one thing they need to act on, and feel
like a pro tool — not a generic SaaS stat grid.

After dashboard, move to the onboarding wizard (6 remaining screens —
apply <OnboardingScaffold>), then the main tabs (orders, menu, more),
then the long tail (profile, settings, analytics, earnings, reviews).

Build + install on iPhone 17 Pro (sim id AD109A46-2F99-43C3-8AAA-FEE68DC8499E)
after every screen so we catch regressions early.
```
