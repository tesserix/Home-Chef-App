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
| `GOOGLE_MAPS_API_KEY` | customer | Injected by `app.config.ts` → `lib/maps-config.js`. **Without it `react-native-maps` crashes with "API key not found"** the moment a map renders — that is order tracking and the chefs map, both gone. |
| `NODE_AUTH_TOKEN` | both | Reads `@tesserix/*` from GitHub Packages during `eas-build-pre-install`. |

Provision the Maps key in project `tesseracthub-480811`:

1. Restrict it to **Maps SDK for Android** and **Maps SDK for iOS** only.
2. Add an Android application restriction for `com.tesserix.homechef.customer`
   with the SHA-1 of **both** the EAS release keystore (`eas credentials -p
   android`) **and** the Play App Signing certificate (Play Console → Setup →
   App signing). Miss the second and the map works in internal testing and
   breaks in production.
3. Store it as GCP secret `prod-homechef-google-maps-api-key`, then
   `eas secret:create --name GOOGLE_MAPS_API_KEY`.

The key ships inside the binary — that is normal and unavoidable for mobile
Maps keys. Restriction is what secures it, not secrecy.

### Backend env for Sign in with Apple revocation

App Review **5.1.1(v)** requires revoking the user's Apple token on account
deletion. Deleting the Identity Platform user is not enough — Apple keeps its
own record, and the app stays listed under *Settings → Apple ID → Sign in with
Apple*. A reviewer who deletes the test account and looks there will see it.

| Env var | Value |
|---|---|
| `APPLE_TEAM_ID` | Apple Developer team id (`2CRHRRYBPL`) |
| `APPLE_KEY_ID` | Key id of the Sign in with Apple `.p8` |
| `APPLE_SERVICES_CLIENT_ID` | The app's bundle id |
| `APPLE_SIGNIN_PRIVATE_KEY_B64` | The `.p8` contents, base64-encoded |

All four must be set or revocation silently disables itself (logged at
startup). See `apps/api/services/apple_signin.go`.

> **Status:** not yet provisioned in prod. This is the one remaining
> submission blocker that needs an action outside this repo.

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
| Payment info | Yes | No | App Functionality | Razorpay checkout |
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
- **Data shared with third parties:** Razorpay (payment info, to process
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
> **Payments** are handled by Razorpay. All purchases are physical goods — real
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

Both stores require working credentials that reach the full app.

- [ ] Customer demo account — email + password, onboarding already completed
- [ ] Vendor demo account — approved chef, published menu, at least one order

> **Status:** not yet created. The E2E users (`e2e-test@fe3dr.com`,
> `e2e-admin@fe3dr.com`) are marked temporary and slated for removal, so do not
> submit those — create dedicated review accounts.

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
  under DPDP, RBI and FSSAI rules with Razorpay settlement. The store listing's
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
