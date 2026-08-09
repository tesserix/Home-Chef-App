# App Store & Play Store submission

Source of truth for submitting **Fe3dr** (customer) and **Fe3dr Vendor** to the
App Store and Google Play. Keep it current — both stores compare what you
declare here against what the binary actually does, and a mismatch is a
rejection, not a warning.

| | Customer | Vendor |
|---|---|---|
| Display name | Fe3dr | Fe3dr Vendor |
| Bundle / package | `com.tesserix.homechef.customer` | `com.tesserix.homechef.vendor` |
| ASC app id | `6780689976` | `6780689641` |
| Apple team | `2CRHRRYBPL` | `2CRHRRYBPL` |
| EAS project | `259c48bf-…a15f881` | `e97eea3f-…51ee40eab` |
| Source | `apps/mobile-customer` | `apps/mobile-vendor` |

**Not for submission:** `apps/mobile-delivery` is decommissioned (see its
`SUNSET.md` — the own-fleet backend routes were retired server-side, so it 404s
against prod). `apps/mobile-admin` is internal-only and still carries a
placeholder Google client id.

---

## 1. Before you build

### Secrets that must exist as EAS secrets

| Name | Used by | Notes |
|---|---|---|
| `NODE_AUTH_TOKEN` | both | Reads `@tesserix/*` from GitHub Packages during `eas-build-pre-install`. |

### Google Maps: no key exists, and the map is disabled because of it

`GOOGLE_MAPS_API_KEY` is **not** an EAS secret in any environment, and must not
be invented into one from what is currently in GCP.

- **No Maps SDK key exists** in project `tesseracthub-480811`. Neither
  `maps-android-backend.googleapis.com` nor `maps-ios-backend.googleapis.com`
  is an enabled service, so no valid key can even be minted right now.
- **`prod-homechef-google-maps-api-key` is NOT that key.** It holds
  `homechef-routes-api`, restricted to `routes.googleapis.com` with no
  application restriction — a server-side Routes key. **Do not** put it in EAS,
  `app.json`, or `app.config.ts`. A mobile key ships inside the binary in
  plaintext, so anyone who unzips the APK could pull it out and bill Routes API
  calls to the project. An earlier version of this document told you to do
  exactly that; do not reinstate it.
- Because the var is unset, `withMapsKey` (`apps/mobile-customer/lib/maps-config.js:29`)
  no-ops and the built Android manifest carries no `com.google.android.geo.API_KEY`.
  Any `MapView` inflation then throws `RuntimeException: API key not found`.
- Consequently the chefs map is disabled behind `CHEFS_MAP_ENABLED` in
  `apps/mobile-customer/lib/features.ts` — the header button is not rendered and
  `/chefs-map` redirects to the tabs, so a deep link cannot crash the app. Order
  tracking's `DeliveryMap` is a separate surface behind its own `showMap` gate
  and is out of scope here.

To actually enable maps later:

1. Enable `maps-android-backend.googleapis.com` (and the iOS backend only if the
   iOS provider ever moves off Apple MapKit — `PROVIDER_DEFAULT` on iOS is
   MapKit today and needs no key).
2. Mint a **new** key restricted to **Maps SDK for Android**, with an Android
   application restriction for `com.tesserix.homechef.customer` carrying the
   SHA-1 of **both** the EAS release keystore (`eas credentials -p android`)
   **and** the Play App Signing certificate (Play Console → Setup → App
   signing). Miss the second and the map works in internal testing and breaks in
   production.
3. Store it as a **new, separate** GCP secret — not the Routes one — then
   `eas secret:create --name GOOGLE_MAPS_API_KEY`.
4. Flip `CHEFS_MAP_ENABLED` to `true`. That also restores guest browsability:
   `apps/mobile-customer/lib/guest-routes.ts` derives the map's guest route from
   the same flag.

The key ships inside the binary — that is normal and unavoidable for mobile
Maps keys. Restriction is what secures it, not secrecy, which is exactly why
step 2 is non-negotiable and why an unrestricted server key can never be used.

### Backend env for Sign in with Apple revocation

App Review **5.1.1(v)** requires revoking the user's Apple token on account
deletion. Deleting the Identity Platform user is not enough — Apple keeps its
own record, and the app stays listed under *Settings → Apple ID → Sign in with
Apple*. A reviewer who deletes the test account and looks there will see it.

**Two apps, two client ids.** Both Customer and Vendor have native Sign in
with Apple enabled (`usesAppleSignIn: true` in each `app.json`), and each is a
separate App ID with its own bundle id. A native authorization code is bound
to the App ID that issued it, so Apple's `/auth/token` and `/auth/revoke`
reject a Customer-app code presented with the Vendor app's `client_id` and
vice versa — one shared client id cannot serve both apps, even though they
share a Team ID and a single Sign in with Apple private key.

| Env var | Value |
|---|---|
| `APPLE_TEAM_ID` | Apple Developer team id (`2CRHRRYBPL`) — shared by both apps |
| `APPLE_KEY_ID` | Key id of the Sign in with Apple `.p8` — shared by both apps |
| `APPLE_SERVICES_CLIENT_ID` | **Customer** app's bundle id (`com.tesserix.homechef.customer`) |
| `APPLE_SERVICES_CLIENT_ID_VENDOR` | **Vendor** app's bundle id (`com.tesserix.homechef.vendor`) |
| `APPLE_SIGNIN_PRIVATE_KEY_B64` | The `.p8` contents, base64-encoded — shared by both apps |

The API resolves which client id to use per request from the user's role
(customer vs. chef), not from a client-supplied header — see
`apps/api/services/apple_signin.go`. Each app's revocation is independently
gated: if only one app's client id is set, revocation works for that app and
degrades to a safe no-op for the other (nothing is sent to Apple with the
wrong client id). The shared vars (team id, key id, private key) plus at least
one client id must be set for revocation to do anything at all; a startup log
line names exactly which vars are missing and for which app(s), unconditional
in every environment. See `apps/api/config/config.go`
(`warnIfAppleSignInIncomplete`).

> **Status:** not yet provisioned in prod. This is the one remaining
> submission blocker that needs an action outside this repo — for BOTH client
> id vars, not just one, or only one app's revocation will actually work.

---

## 2. App Store — Privacy Nutrition Labels

Mirror `ios.privacyManifests.NSPrivacyCollectedDataTypes` in each `app.json`.
Nothing is used for tracking; `NSPrivacyTracking` is `false` and
`NSPrivacyTrackingDomains` is empty in both apps, so **do not** enable App
Tracking Transparency — there is no tracking SDK to justify it.

### Customer

| Data | Linked | Tracking | Purpose | Why |
|---|---|---|---|---|
| Email address | Yes | No | App Functionality | Account identity |
| Name | Yes | No | App Functionality | Order and delivery |
| Phone number | Yes | No | App Functionality | Delivery contact |
| Physical address | Yes | No | App Functionality | Delivery address |
| Payment info | Yes | No | App Functionality | Cashfree checkout |
| Purchase history | Yes | No | App Functionality | Order history, reorder |
| Customer support | Yes | No | App Functionality | Support tickets, order messaging |
| Photos or videos | Yes | No | App Functionality | Photo attached to an order-issue report |
| Device ID | Yes | No | App Functionality | Push notification token |

Precise location is **not** collected. The app has no `expo-location`
dependency and `DeliveryMap` sets `showsUserLocation={false}`; delivery
addresses are entered, not sensed.

### Vendor

Everything above minus purchase history and payment info, plus:

| Data | Linked | Tracking | Purpose | Why |
|---|---|---|---|---|
| Precise location | Yes | No | App Functionality | One-time kitchen-address autofill during onboarding |
| Other financial info | Yes | No | App Functionality | Payout and earnings |
| Sensitive info | Yes | No | App Functionality | FSSAI licence, PAN, identity documents |
| Photos or videos | Yes | No | App Functionality | Dish photos, document uploads |

Neither app collects crash or performance data — no analytics or crash SDK is
installed (only `@react-native-firebase/app` and `/auth`).

---

## 3. Google Play — Data Safety

Same inventory as above, in Play's vocabulary.

- **Is data encrypted in transit?** Yes — TLS everywhere.
- **Can users request data deletion?** Yes — in-app *and* at
  <https://fe3dr.com/account-deletion>. Play requires both.
- **Data shared with third parties:** Cashfree (payment info, to process
  payment) and the 3PL delivery provider (name, phone, delivery address, to
  perform delivery). Everything else is collected, not shared.
- **Ads:** none. `com.google.android.gms.permission.AD_ID` is explicitly listed
  under `blockedPermissions` in both apps, so declare **no** advertising ID.

### Android permissions to justify

| Permission | App | Justification |
|---|---|---|
| `POST_NOTIFICATIONS` | both | Order status updates |
| `USE_BIOMETRIC` / `USE_FINGERPRINT` | both | Optional biometric unlock |
| `ACCESS_COARSE_LOCATION` / `ACCESS_FINE_LOCATION` | vendor | One-time kitchen-address autofill in onboarding, foreground only |
| `READ_EXTERNAL_STORAGE` / `RECORD_AUDIO` | both | Added transitively by `expo-image-picker`; not used directly |

Neither app requests background location, and neither declares a foreground
service.

---

## 4. Account deletion (Apple 5.1.1(v), Play policy)

Both stores require deletion to be startable **inside** the app.

- Customer: Profile → *Pause or delete account* → confirm by typing your email.
- Vendor: More → Settings → *Pause or delete account* → same.
- Web, for people who uninstalled: <https://fe3dr.com/account-deletion>

Behaviour, which the privacy policy must keep matching
(`apps/api/services/account_lifecycle.go`):

- Sign-in is destroyed immediately and the profile leaves the marketplace.
- Data is retained **180 days** (`RestoreWindow`) so the account can be
  restored, then permanently erased by the purge sweeper.
- The Identity Platform credential is torn down, and the **Sign in with Apple
  grant is revoked**.
- Deletion is blocked while money or work is in flight (live order, unspent
  wallet credit, unreleased payout). The app names what is outstanding.
  Deactivating is always available and never blocked.

---

## 5. User-generated content (Apple 1.2)

The customer app carries UGC — chef social posts and comments, chef reviews,
and order messaging — so all four of Apple's requirements are implemented:

| Requirement | Where |
|---|---|
| Filter objectionable content | Auto-hide at 3 distinct reports (`services/moderation.go`, `AutoHideThreshold`) |
| Report mechanism | ⋯ on every post and review → report sheet, 9 reasons + free text |
| Block abusive users | Offered in the same sheet; managed at Profile → Blocked accounts |
| Published contact | `support@fe3dr.com` in the app's Legal screen and on fe3dr.com |

Reports land in an admin triage queue (`GET /v1/admin/reports`) and are resolved
as upheld or rejected. Blocking hides the blocked user's posts and reviews and
stops order messaging in both directions.

---

## 6. Review notes (paste into App Store Connect)

> Fe3dr is a marketplace for home-cooked food in India. Food is prepared by
> independent, FSSAI-licensed home chefs and delivered by third-party partners.
>
> **You do not need an account to browse.** Tap "Browse without an account" on
> the sign-in screen to see chefs and menus. An account is required only to
> place an order, save a chef, or view order history.
>
> **Payments** are handled by Cashfree. All purchases are physical goods — real
> meals delivered to a physical address — so In-App Purchase does not apply
> (guideline 3.1.1). There is no digital content in the app.
>
> **Reporting and blocking:** tap ⋯ on any chef post or review to report it or
> block its author. Blocked accounts are managed at Profile → Blocked accounts.
>
> **Account deletion:** Profile → Pause or delete account. Confirm by typing the
> account email. Also available at https://fe3dr.com/account-deletion.
>
> Test account credentials are in the "Sign-In Information" section below.

For **Fe3dr Vendor**, add:

> This is the companion app for chefs selling on Fe3dr. It requires an approved
> chef account — the test account provided is pre-approved with a live menu.
> The app manages an existing real-world business relationship; it is not a
> standalone product and has no consumer-facing purchase flow.

### Demo accounts

Both stores require working credentials that reach the full app. These exist on
production and were verified end to end on 2026-08-09 (GIP sign-in → BFF
`/auth/auto-login` → authenticated API reads).

| Account | Email | Password | Pool |
|---|---|---|---|
| Vendor / chef | `vendor@fe3dr.com` | `Fe3drDemo!2026` | `HomeChef-Business-8s8ql` |
| Customer 1 — Priya Sharma | `customer01@fe3dr.com` | `Fe3drDemo!2026` | `HomeChef-Customer-rqg8a` |
| Customer 2 — Rahul Verma | `customer02@fe3dr.com` | `Fe3drDemo!2026` | `HomeChef-Customer-rqg8a` |
| Customer 3 — Ananya Iyer | `customer03@fe3dr.com` | `Fe3drDemo!2026` | `HomeChef-Customer-rqg8a` |

- [x] **Vendor** — resolves to *Saffron Home Kitchen* (Indiranagar, Bengaluru),
  `verified: true`, published menu with categories, profile + banner + 4 kitchen
  photos, 6 orders, 8 reviews, rating 4.63, operating hours set,
  `acceptingOrders: true`. Meets "approved chef, published menu, ≥1 order".
  Native Sign in with Apple confirmed working on the Vendor app.
- [x] **Customer** — `customer01` has a default Bengaluru address (Domlur,
  560071, `isDefault: true`), which is what makes the chef list resolve for a
  reviewer outside India. See the test-mode warning below before submitting.

Do not submit the E2E users (`e2e-test@fe3dr.com`, `e2e-admin@fe3dr.com`) —
they are temporary and slated for removal.

### Test mode blocks the Customer submission

Every chef on production is `testMode: true`, and there are only two. Outside
the test-mode allowlist a test kitchen collapses to `ToClosedResponse()` — name
and photo, no menu, no prices, not orderable
(`applyTestModePresentation`, `apps/api/handlers/chefs.go:516`). The reviewer
sees a populated marketplace only because `customer01@fe3dr.com` was added to
the allowlist in admin; it is **not** in `DefaultTestModePolicy`
(`apps/api/services/test_mode_policy.go:39`).

Three consequences, all Customer-only — the Vendor app is unaffected, since a
chef's own app does not go through listing visibility:

1. `apps/mobile-customer/app/chef/[id].tsx:620` renders *"TEST kitchen —
   payments use the gateway's test mode, no real money is charged."* A reviewer
   reads that as a demo or non-final build — **guideline 2.1**. This is the most
   likely Customer rejection and it is self-inflicted.
2. The second kitchen, *My Kitchen* (Mangaluru), is junk seed data — its
   description is keyboard mash repeated ~20 times, no images, 0 orders, closed.
   Guideline 4.3 / 2.1, and it halves an already-small marketplace.
3. The whole listing depends on one allowlist row. Anyone editing the test-mode
   policy in admin during the review window empties the reviewer's marketplace.

Before submitting Customer: take at least one kitchen live
(`PATCH /admin/chefs/:id/mode` with `{"mode":"live","reason":"…"}`) and remove
*My Kitchen*. Note there is **no hard-delete endpoint for chefs** — the
available action is `PUT /admin/chefs/:id/suspend`, which sets `is_active:
false` and is reversible. All `/admin/*` routes require an internal-pool admin
(`RequirePool(PoolInternal)` + `RequireAdmin`), so the seeded business/customer
accounts cannot perform them.

---

## 7. Age rating & content

- Both apps: **4+** on iOS / **Everyone** on Play. No objectionable content, no
  gambling, no alcohol.
- Terms require users to be **18+** to transact. That is a contractual limit,
  not a content rating — do not raise the rating to match it.
- The customer app carries UGC, so the Play content-rating questionnaire must
  answer **yes** to "users can interact" and "users can share content", and
  point at the moderation described in §5.

---

## 8. Known gaps

Things a reviewer will not catch but you should know about:

- **`ios.deploymentTarget` is `18.0`** in both apps
  (`app.json` → `expo-build-properties`). React Native 0.86 supports iOS 16, so
  this excludes every device that cannot run iOS 18 for no stated reason. Not a
  rejection — just lost users. Lowering it needs a real device build to verify.
- **Operating entity mismatch.** The in-app Terms name *Tesserix Pty Ltd (ACN
  694 070 865), New South Wales, Australia*, while the service runs in India
  under DPDP, RBI and FSSAI rules with Cashfree settlement. The store listing's
  seller and the privacy policy's controller must name the entity that actually
  contracts with customers. Worth a look from whoever owns the legal side.
- **Vendor app icon** was regenerated in-repo to stop both apps shipping
  byte-identical artwork (an Apple 4.3 flag). It is functional and clearly
  distinct, but it is engineering output — have a designer sign it off.
- `docs/e2e-test-plan-2026-07-25.md` and `homechef-e2e-tests/` reference
  temporary Keycloak test users that must be removed before public launch.

---

## 9. Build & submit

```bash
# Customer
cd apps/mobile-customer
eas build --platform all --profile production
eas submit --platform ios --profile production
eas submit --platform android --profile production

# Vendor
cd apps/mobile-vendor
eas build --platform all --profile production
eas submit --platform ios --profile production
eas submit --platform android --profile production
```

`eas.json` sets `appVersionSource: remote` with `autoIncrement` on the
production profile, so build numbers advance on their own — do not bump them by
hand. Android submits to the `internal` track first; promote in the Play
Console once the internal build is verified.

### Screenshot builds

Store screenshots come from the `prod-sim` profile — a release-mode simulator
build pointed at production, so there is no Expo dev overlay and the data is
real:

```bash
cd apps/mobile-vendor
eas build --platform ios --profile prod-sim
```

`prod-sim` must carry `"environment": "production"`. EAS only maps a profile to
the production environment when the profile is *named* `production`; anything
else defaults to `preview`, and `NODE_AUTH_TOKEN` exists **only** in the
production environment. Without the explicit key the build dies in
`eas-build-pre-install` after ~40s, unable to read `@tesserix/*` from GitHub
Packages. The vendor profile was fixed on 2026-08-09; the customer profile has
no `prod-sim` and will need the same key when one is added.

Capture on the 6.9" device (iPhone 17 Pro Max, 1320 × 2868) — with
`supportsTablet: false` in both apps, that is the only size App Store Connect
requires:

```bash
U=$(xcrun simctl list devices available | grep "iPhone 17 Pro Max" | head -1 | sed -E 's/.*\(([0-9A-F-]{36})\).*/\1/')
xcrun simctl boot "$U"
xcrun simctl install "$U" /path/to/FeedrVendor.app
xcrun simctl launch "$U" com.tesserix.homechef.vendor
xcrun simctl io "$U" screenshot shot.png
```
