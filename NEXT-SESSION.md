# Next session — Home Chef vendor app continuation

## Where we are (2026-06-02, end of build #10)

The **vendor app** has been redesigned end-to-end against the Uber-like brand
direction (white + ink + persimmon accent only). The customer + delivery apps
still use old tokens and remain untouched. This doc picks up the vendor app
for the next sprint.

### Brand direction (locked)
- Source of truth: `~/.claude/projects/-Users-Mahesh-Sangawar-personal-tesserix-new-Home-Chef-App/memory/feedback_no_editorial_style.md`
- Tokens: `packages/mobile-shared/src/theme/tokens.ts` (mirrored in `apps/mobile-vendor/tailwind.config.js`)
- Geist-Bold display, Inter body. Ink (#0E0E0C) primary CTAs. Persimmon
  (#C2410C / `herb.DEFAULT`) is STRICTLY accent — never a primary fill.
- Light-first; dark mode not yet built.

### Redesigned in current pass
All in **StyleSheet (no NativeWind)**, with the dashboard v3.1 / orders v3 /
menu v3 as the visual reference set:

| Screen | File | Notes |
|---|---|---|
| Dashboard | `app/(tabs)/index.tsx` | 4-zone: command bar / action queue / in-progress / today strip. PendingOrderCard extracted to shared. |
| Orders | `app/(tabs)/orders.tsx` | Queue / History as bare-text underline tabs. History hairline rows grouped by date. Surge banner reused. |
| Menu | `app/(tabs)/menu.tsx` | Header `+ Add` (FAB killed). Category underline tabs. MenuItemRow with 44pt thumb + DietIcon + inline ink switch. |
| More | `app/(tabs)/more.tsx` | "Account" Geist title. 44pt ink avatar + email. Hairline nav rows with captions + chevrons. Ink underlined Log out. |
| Profile | `app/profile.tsx` | 72pt ink avatar. 3-col command bar: ChevronLeft / title / Edit-or-Cancel. Hairline LabeledRow / EditableField. Sticky Save when dirty. |
| Settings | `app/settings.tsx` | Caption-spaced section labels (NOTIFICATION PREFERENCES / AVAILABILITY / ACCOUNT). Switches ink track + paper thumb. |
| Earnings | `app/earnings.tsx` | Geist 44pt ₹ hero. Week/Month/All underline tabs. Hairline transaction rows grouped by date. Payout account row → `/profile`. |
| Analytics | `app/analytics.tsx` | Chart DROPPED (no honest baseline data). Today-strip summary + hairline popular-items rows. |
| Reviews | `app/reviews.tsx` | No hero. Compact `4.7 ★ · 38 reviews` inline. Filter underline tabs (All / 5★ / 4★ / 3★ / ≤2★). |
| Review reply | `app/review/[reviewId].tsx` | Review hero first. Compact ★ N /5. Growing TextInput cap 180pt. Inline errors. |
| Menu form | `app/menu/MenuItemForm.tsx` (+ `new.tsx`, `[itemId]/edit.tsx`) | Shared 1006-line form. `onBack` prop wired. |
| Onboarding wizard | `app/(onboarding)/{kitchen-details,operations,documents,policies,review,pending}.tsx` | All use `<OnboardingScaffold>`. Cuisine + prep-time as ink-bordered pills. Ink checkboxes/radios. |
| Pending / app status | `app/(onboarding)/pending.tsx` | 3-step timeline (Submitted → Reviewing → Approved). "WHAT YOU'LL GET" preview. Visible Check status button. |
| Auth shared | `packages/mobile-shared/src/screens/{LoginScreen,RegisterScreen,ForgotPasswordScreen}.tsx` | Side-by-side SocialIconButtons (Google G + Apple silhouette, both SVG via `_socialIcons.tsx`). |
| Tab bar | `app/(tabs)/_layout.tsx` | `letterSpacing 0` + `fontSize 10` (was truncating "Menu" → "Me…"). |

### Shared components added
- `components/vendor/PendingOrderCard.tsx` — used by dashboard + orders queue. `showInstructions` prop surfaces special instructions on orders tab only.
- `components/vendor/MenuItemRow.tsx` — replaces old MenuItemCard.
- `components/vendor/DietIcon.tsx` — FSSAI-style square outline + centered dot. Green for veg, paprika for non-veg.
- `components/vendor/UndoSnackbar.tsx` — patched: `Animated.spring` → `Animated.timing` with `motion.entrance`. Now supports `errorMessage` + `onRetry`.
- `packages/mobile-shared/src/screens/_socialIcons.tsx` — `SocialIconButton`, `GoogleGlyph`, `AppleGlyph` (SVG via `react-native-svg`). Apple path sourced from Tesserix design system.

### Tokens added
- `theme.colors.diet.veg = '#2A9D3E'` and `theme.colors.diet.nonVeg = '#B22B0E'`. Used ONLY by DietIcon.

### Auth / routing hard-lock (in `app/_layout.tsx`)
- Onboarding-status query: `retry: false`, `refetchOnMount: 'always'`, `staleTime: 0`. No more stale `verified` data letting non-verified chefs roam.
- Routing useEffect now strict-redirects on ANY pathname drift from `expectedPath`. Back gesture / deep link from `/pending` gets yanked back to pending.
- Approval toast: when status transitions to `verified` mid-session, fires `"Your kitchen is approved. Welcome aboard."` via ToastProvider.
- Splash overlay shows `"Setting up your kitchen…"` during the post-Google-sign-in chain so it doesn't read as stuck.

### Veg-bug hack (frontend-only, needs proper backend fix)
Backend `MenuItem` model has NO `IsVeg` boolean — it stores `DietaryTags []string`.
The frontend pretends `isVeg: boolean` exists via:
- `useVendorMenu` derives `isVeg` from `dietaryTags.includes('vegetarian')` (case-insensitive).
- `useCreateMenuItem` / `useUpdateMenuItem` translate `isVeg: true/false` →
  `dietaryTags: ['vegetarian' | 'non-vegetarian']` before POST/PUT.
This works but is fragile. Add a real `is_veg` column when convenient.

### iOS Pressable function-style guard
**Recurring bug**: iOS strips `backgroundColor` / `borderWidth` / `flexDirection`
from a Pressable with a function-based `style={({ pressed }) => [...]}` prop
under certain conditions. Every row-shaped Pressable in this codebase uses the
**inner-View pattern**:

```tsx
<Pressable onPress={...}>
  {({ pressed }) => (
    <View style={[styles.row, pressed && { backgroundColor: bone }]}>
      ...children
    </View>
  )}
</Pressable>
```

When adding new Pressables, always use this pattern unless it's a tiny
button with no flex children.

### Approving / verifying a chef (dev)
The backend chef_profiles table uses `state: text` + `is_verified: bool` +
`verified_at: timestamp`. To approve via direct DB:

```bash
docker exec homechef-db psql -U homechef -d homechef -c \
  "UPDATE chef_profiles SET state='verified', is_verified=true, verified_at=NOW() WHERE user_id=(SELECT id FROM users WHERE email='you@example.com');"
```

Admin endpoint exists at `PUT /admin/approvals/:id/approve` (requires admin auth).

### Backend prerequisites still pending
- **Real `is_veg` boolean** on MenuItem (remove the dietaryTags hack).
- **`packaging_fee`, `min_order_value`, `service_radius_km`** columns on `chef_profiles` (the kitchen rules). Default `serviceRadius` already lives in `operations`, just needs to flow into `chef_profiles` on submit.
- **GST split** on order totals (compute backend, surface on Earnings + customer checkout). User signaled GST-inclusive MRP pricing.
- **`?q=` search** on `/chef/orders`, `/chef/menu/items`, `/chef/reviews` for Sprint 3.

---

## Plan: next 4 sprints

### Sprint 1 — Foundations (≈2.5 days)
1. **Order detail screen** `app/orders/[orderId].tsx` (new file)
   - Hero: customer + total + status pill + age
   - Hairline sections: items list, delivery address, special instructions, customer contact
   - Footer actions vary by status (Accept/Reject pending → Mark preparing → Mark ready)
   - Linked from queue + history rows in `(tabs)/orders.tsx`

2. **Onboarding draft persistence** `store/onboarding-store.ts`
   - Add Zustand `persist` middleware with `createJSONStorage(() => AsyncStorage)`
   - Wizard resumes wherever chef left off
   - `store.reset()` already clears on submit — no change needed
   - ~5 lines

3. **Push notifications end-to-end audit**
   - Verify FCM token registration post-login
   - Verify Android channel registration
   - Test foreground notification UI
   - Verify tap → routes to `/(tabs)/orders` with orderId param
   - Lock-screen Accept/Reject actions already coded in `_layout.tsx:160-200`
   - Add a "Test notification" button in dev-only Settings section

### Sprint 2 — Quality of life (≈3–4 days)
4. **Address autocomplete** in `(onboarding)/kitchen-details.tsx`
   - `react-native-google-places-autocomplete` package
   - Needs `EXPO_PUBLIC_GOOGLE_PLACES_API_KEY`
   - Replaces 5 manual text fields with 1 search + parse

5. **Preview as customer**
   - "Preview" right-side link on Menu command bar
   - Opens bottom sheet rendering kitchen page as customers see it
   - Use existing customer menu endpoint OR add `/chef/preview` server-side
   - Photos + name + cuisines + sorted menu with prices/descriptions/DietIcon

### Sprint 3 — Scale (≈2 days)
6. **Search** on orders history, menu, reviews
   - Backend: add `q` param to 3 endpoints (`WHERE LOWER(name) ILIKE '%' || $q || '%'`)
   - Frontend: search field in each list's command bar, 300ms debounce, re-fetch on change

7. **Reply templates** on review reply screen
   - Bottom sheet with 5 templates ("Thank you for the kind words!", "We're sorry to hear that…", etc.)
   - Tap populates the TextInput

### Sprint 4 — Dark mode (≈3–5 days, own cycle)
8. Extend `theme.colors` with `light` + `dark` variants
9. Root-level `useColorScheme()` hook drives token resolution
10. Audit every screen for hardcoded `paper`/`ink` references
11. Validate WCAG AA contrast in dark mode
12. Toggle in Settings + system follow

---

## Bigger deferrals
- **Customer + delivery apps** still on old tokens. Major parallel effort eventually needed for brand cohesion.
- **i18n / Hindi** — `expo-localization` + `i18next`. Doubles addressable market in India.
- **Offline / poor network handling** — `OfflineBanner` exists in `mobile-shared` but isn't wired anywhere visible. Queue+retry pattern for orders accept/reject when offline.
- **Bank account verification** — needs Razorpay/Cashfree integration for payouts.
- **FSSAI number validation** — backend regex.

## Build + install workflow
```bash
cd apps/mobile-vendor
find . -maxdepth 1 -name 'build-*.tar.gz' -delete
EXPO_NO_TELEMETRY=1 npx eas-cli build --local --platform ios --profile local-sim --non-interactive
# When done:
BUILD=$(ls -t build-*.tar.gz | head -1)
rm -rf /tmp/hcv-app && mkdir -p /tmp/hcv-app
tar -xzf "$BUILD" -C /tmp/hcv-app
xcrun simctl terminate AD109A46-2F99-43C3-8AAA-FEE68DC8499E com.homechef.vendor 2>/dev/null
xcrun simctl install AD109A46-2F99-43C3-8AAA-FEE68DC8499E /tmp/hcv-app/HomeChefVendor.app
xcrun simctl launch AD109A46-2F99-43C3-8AAA-FEE68DC8499E com.homechef.vendor
```

Each build takes ~5–10 min. Bundle multiple changes into one build whenever possible.

## Suggested next-session prompt

```
Continue the Home Chef vendor app. Read NEXT-SESSION.md and
~/.claude/projects/-Users-Mahesh-Sangawar-personal-tesserix-new-Home-Chef-App/memory/feedback_no_editorial_style.md
before starting. The full design pass landed in build #10 — verify it
installs cleanly, then start Sprint 1: Order detail screen + Onboarding
draft persistence + Push notifications audit. All three are independent
and ship together in one build.

Use the inner-View Pressable pattern, hairline rows for non-action data,
bone cards only for things demanding a decision. Ink primary CTAs;
persimmon is accent only.
```
