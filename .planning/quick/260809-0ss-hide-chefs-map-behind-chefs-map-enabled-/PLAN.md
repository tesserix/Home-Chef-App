---
type: quick
scope: apps/mobile-customer + docs/store-release
files_modified:
  - apps/mobile-customer/lib/features.ts
  - apps/mobile-customer/lib/guest-routes.ts
  - apps/mobile-customer/lib/guest-routes.test.ts
  - apps/mobile-customer/lib/chefs-map-gated.test.ts
  - apps/mobile-customer/app/(tabs)/index.tsx
  - apps/mobile-customer/app/chefs-map.tsx
  - docs/store-release/README.md
autonomous: true
---

<objective>
Hide the chefs-map entry point in the customer app behind a new
`CHEFS_MAP_ENABLED = false` flag, and correct the store-release doc that tells
the reader to ship the Routes-only API key inside the binary.

Purpose: no Maps SDK key exists (GCP `tesseracthub-480811` has only
`homechef-routes-api`, restricted to `routes.googleapis.com`;
`maps-android-backend.googleapis.com` is not an enabled service). `withMapsKey`
no-ops without `GOOGLE_MAPS_API_KEY`, so the Android manifest lacks
`com.google.android.geo.API_KEY` and any `MapView` inflation crashes with
`RuntimeException: API key not found`. Maps cannot be fixed for this release, so
the surface must be unreachable — including by deep link.

Output: chefs map unreachable on all paths while the flag is false; one flag
flip (plus a real key) restores it, including guest browsability.
</objective>

<verified_context>
Facts established before planning — do not re-investigate:

- `styles.addressRow` is `flexDirection: row; alignItems: center; gap: 8` with
  `addressRowPill: { flex: 1 }`. Removing the 40x40 map button leaves the
  address pill flexing to fill and the bell button at the row end. No layout fix
  needed. `styles.mapButton` is ALSO used by the notifications bell
  (`index.tsx:344`) — **do not delete the style**.
- `Map` (lucide) is imported at `index.tsx:40` and used only at line 329 (inside
  the map button). `MapPin` is a separate import used at line 751 — keep it.
  `noUnusedLocals` is on, so `Map` must stay referenced (it will, inside the
  gated JSX) or be removed.
- `isGuestBrowsable` has exactly one consumer: `app/_layout.tsx:318`.
- Baseline is green: `npx tsc --noEmit -p tsconfig.json` exits 0 today.
- Jest here is a hand-rolled config with `testEnvironment: 'node'` and zero
  `.tsx` tests. RTL is a devDependency but unproven in this suite — **do not
  write a component-render test**. Use the source-scan pattern from
  `lib/no-razorpay.test.ts`.
- `docs/store-release/README.md` lines 28-45 hold the wrong instruction.
</verified_context>

<tasks>

<task type="auto" tdd="true">
  <name>Task 1: Add CHEFS_MAP_ENABLED and derive guest browsability from it</name>
  <files>
    apps/mobile-customer/lib/features.ts,
    apps/mobile-customer/lib/guest-routes.ts,
    apps/mobile-customer/lib/guest-routes.test.ts
  </files>
  <behavior>
    - `isGuestBrowsable(['chefs-map'])` is `false` while `CHEFS_MAP_ENABLED` is false.
    - `isGuestBrowsable(['search-dishes'])`, `['chef','[id]']`, `['chefbook']` and
      every legal page stay `true` — the gate must not widen.
    - `isGuestBrowsable([])` stays `false`.
  </behavior>
  <action>
Update `guest-routes.test.ts` FIRST (RED). Replace the `chefs-map` assertion in
the "lets a guest browse the map and dish search" case (currently line ~22)
with a rename to "lets a guest browse dish search" keeping the `search-dishes`
assertion, and add a separate case asserting `isGuestBrowsable(['chefs-map'])`
is `false`, with a comment naming the reason (no Maps SDK key; the screen
crashes on Android) and pointing at `CHEFS_MAP_ENABLED`. Run it and confirm it
fails.

Then in `lib/features.ts`, append `export const CHEFS_MAP_ENABLED: boolean = false;`
with a block comment in the same voice as the existing `GROUP_ORDERS_ENABLED` /
`MESSAGING_ENABLED` comments. The comment must state the unblock condition
concretely so the next reader is not left guessing: no Maps SDK key exists in
`tesseracthub-480811`; `prod-homechef-google-maps-api-key` holds the
Routes-API-only key and is NOT usable; `maps-android-backend.googleapis.com` is
not enabled; without a key `withMapsKey` no-ops and Android throws
"API key not found" on MapView inflation. Note this gates only the chefs map —
`DeliveryMap` in order tracking is a separate, `showMap`-gated surface left
untouched by this change.

In `lib/guest-routes.ts`, import `CHEFS_MAP_ENABLED` from `./features` and build
the set so `'chefs-map'` is present only when the flag is true, e.g. spread a
conditional array into the `new Set([...])` literal — do not mutate the set
after construction (immutability rule). Update the file's header comment: the
map is currently out of the guest browse surface and comes back with the flag,
so the App Review 5.1.1(iv) claim stays accurate.

Re-run the test (GREEN).
  </action>
  <verify>
    <automated>cd apps/mobile-customer && npx jest lib/guest-routes.test.ts</automated>
  </verify>
  <done>`CHEFS_MAP_ENABLED` exists and is false; guest-routes derives `chefs-map` from it; guest-routes test passes and asserts the false case.</done>
</task>

<task type="auto" tdd="true">
  <name>Task 2: Gate the home header button and the route itself</name>
  <files>
    apps/mobile-customer/app/(tabs)/index.tsx,
    apps/mobile-customer/app/chefs-map.tsx,
    apps/mobile-customer/lib/chefs-map-gated.test.ts
  </files>
  <behavior>
    - No source file navigates to `/chefs-map` outside a `CHEFS_MAP_ENABLED` guard.
    - `app/chefs-map.tsx` references `CHEFS_MAP_ENABLED` — the screen refuses to
      render its MapView when the flag is off, so a deep link cannot crash Android.
  </behavior>
  <action>
Write `lib/chefs-map-gated.test.ts` FIRST (RED), modeled on
`lib/no-razorpay.test.ts` (same `sourceFiles` recursion over
`['app','components','hooks','lib','store','types']`, same `ROOT` resolution,
self-exclusion of the test file). Two cases:
  1. Every file containing `/chefs-map` as a navigation target (match
     `router.push('/chefs-map')`, `router.replace('/chefs-map')` and
     `href="/chefs-map"` / `href={'/chefs-map'}`) must also contain
     `CHEFS_MAP_ENABLED`. Assert the offender list is `[]` so the failure names
     the file.
  2. `app/chefs-map.tsx` must contain `CHEFS_MAP_ENABLED` — assert the file is
     found (guard against a silent path typo) and that the flag appears in it.
Include a header comment recording why: the screen inflates a `react-native-maps`
`MapView` with `PROVIDER_DEFAULT`, which on Android is Google Maps and throws
without a manifest API key that no environment supplies.

Then in `app/(tabs)/index.tsx`: add `CHEFS_MAP_ENABLED` to the existing
`lib/features` import on line 61 (keep the members alphabetized as they are).
Wrap the map-button `Pressable` (~line 317, `accessibilityLabel="View chefs on a
map"`) in `{CHEFS_MAP_ENABLED ? ( ... ) : null}`, matching the
`{WALLET_ENABLED && !isGuest ? (...) : null}` form used directly above it. Add a
one-line comment saying the map is flag-hidden pending a Maps SDK key. Do NOT
touch the notifications bell `Pressable`, `styles.mapButton` (the bell uses it),
or the `MapPin` import.

Then in `app/chefs-map.tsx`: import `CHEFS_MAP_ENABLED` from `../lib/features`
and `Redirect` from `expo-router`, and early-return
`<Redirect href="/(tabs)" />` from `ChefsMapScreen` before any MapView is
constructed. Keep the rest of the screen intact — this is a gate, not a
deletion; the screen must work unchanged when the flag flips. Add a comment
explaining the redirect exists because the button being hidden does not stop a
deep link, and a deep-linked MapView still crashes Android.

Re-run the guard test (GREEN).
  </action>
  <verify>
    <automated>cd apps/mobile-customer && npx jest lib/chefs-map-gated.test.ts && npx tsc --noEmit -p tsconfig.json</automated>
  </verify>
  <done>Home header renders no map button; `/chefs-map` redirects to the tabs; guard test passes; typecheck exits 0 with no unused-import error for `Map`.</done>
</task>

<task type="auto">
  <name>Task 3: Correct the store-release Maps key instructions</name>
  <files>docs/store-release/README.md</files>
  <action>
Rewrite the `GOOGLE_MAPS_API_KEY` row in the "Secrets that must exist as EAS
secrets" table (line ~30) and the three-step provisioning block below it (lines
~33-45).

The current text is wrong and actively harmful: step 3 says to store the key as
GCP secret `prod-homechef-google-maps-api-key` and run `eas secret:create`. That
secret already exists and holds `homechef-routes-api` — a key restricted to
`routes.googleapis.com` with no application restriction. Following the
instruction ships a server-side Routes key inside a public binary, where anyone
can extract it and bill Routes API calls to the project.

Replace with the real state:
  - No Maps SDK key exists in `tesseracthub-480811`. Neither
    `maps-android-backend.googleapis.com` nor `maps-ios-backend.googleapis.com`
    is an enabled service.
  - `prod-homechef-google-maps-api-key` is the Routes API key. **Do not** put it
    in EAS or in app config — call out the exfiltration/billing risk explicitly
    so nobody "helpfully" reinstates the old step.
  - `GOOGLE_MAPS_API_KEY` is therefore set in no EAS environment, so
    `withMapsKey` (`lib/maps-config.js:29`) no-ops and the Android manifest has
    no `com.google.android.geo.API_KEY`.
  - Consequently the chefs map is disabled via `CHEFS_MAP_ENABLED` in
    `apps/mobile-customer/lib/features.ts`. Order tracking's `DeliveryMap` is a
    separate surface, gated behind `showMap` and out of scope here.

Then give the genuine path to enabling maps later, as an ordered list:
  1. Enable `maps-android-backend.googleapis.com` (and the iOS backend if the
     iOS provider ever moves off Apple MapKit — `PROVIDER_DEFAULT` on iOS is
     MapKit today and needs no key).
  2. Mint a NEW key restricted to Maps SDK for Android, with an Android
     application restriction for `com.tesserix.homechef.customer` carrying the
     SHA-1 of BOTH the EAS release keystore (`eas credentials -p android`) AND
     the Play App Signing certificate (Play Console → Setup → App signing).
     Missing the second works in internal testing and breaks in production —
     preserve that warning, it is correct.
  3. Store it as a NEW, separate GCP secret (not the Routes one), then
     `eas secret:create --name GOOGLE_MAPS_API_KEY`.
  4. Flip `CHEFS_MAP_ENABLED` to `true` — which also restores guest
     browsability, since `lib/guest-routes.ts` now derives it from the flag.

Keep the existing true note that a mobile Maps key ships inside the binary and
is secured by restriction rather than secrecy — it is what makes step 2
non-negotiable.
  </action>
  <verify>
    <automated>grep -n "prod-homechef-google-maps-api-key" docs/store-release/README.md | grep -q "eas secret:create" && echo FAIL || echo OK; grep -q "CHEFS_MAP_ENABLED" docs/store-release/README.md && echo OK</automated>
  </verify>
  <done>Doc no longer instructs shipping the Routes key; states no Maps SDK key exists, that the map is flag-disabled, and the real enablement path.</done>
</task>

</tasks>

<verification>
Full-suite regression, from `apps/mobile-customer`:

1. `npx jest` — whole customer suite green (15 existing lib tests + the new
   guard test). `lib/maps-config.test.ts` must still pass untouched; this change
   does not alter `withMapsKey`.
2. `npx tsc --noEmit -p tsconfig.json` — exits 0. Baseline was 0, so any error
   is ours. Watch specifically for `noUnusedLocals` on the `Map` lucide import in
   `app/(tabs)/index.tsx`.
3. `git diff --stat` — confirm nothing under `apps/mobile-vendor`,
   `components/tracking/DeliveryMap.tsx`, `app.config.ts`, `lib/maps-config.js`,
   `eas.json` or `app.json` was touched.
</verification>

<success_criteria>
- `CHEFS_MAP_ENABLED: boolean = false` exists in `lib/features.ts` with a comment
  naming the missing-Maps-SDK-key blocker and the unblock condition.
- The home-screen map button is not rendered; the bell button and address pill
  are unchanged and the row still lays out correctly (`addressRowPill` flexes).
- A deep link to `/chefs-map` redirects to the tabs instead of inflating MapView.
- `isGuestBrowsable(['chefs-map'])` is false, derived from the flag rather than
  hard-deleted, so one flip restores it.
- `docs/store-release/README.md` no longer tells anyone to ship the Routes key.
- `npx jest` and `npx tsc --noEmit` both clean.
- Commit: single-line conventional, e.g.
  `fix(mobile-customer): hide the chefs map until a Maps SDK key exists`.
  No signature, no attribution.
</success_criteria>

<out_of_scope>
- `components/tracking/DeliveryMap.tsx` and order-tracking behavior (already
  `showMap`-gated).
- Adding any API key to EAS, `app.json`, or `app.config.ts`.
- `apps/mobile-vendor` (no `react-native-maps` dependency).
- Deleting `app/chefs-map.tsx` — it is gated, not removed.
</out_of_scope>
