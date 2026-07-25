# Web ↔ Mobile feature parity

**Audit date:** 2026-07-25 · **Against:** `main` @ `ef29f8df`

Source of truth for the web-revival programme (owner decision 2026-07-25: bring
`fe3dr.com` back as a full ordering surface and `vendors.fe3dr.com` back as a
full chef portal, reversing the 2026-06 app-first sunsets).

---

## 1. Executive summary

Mobile is not level with web — it is **substantially ahead of it**. The web
surfaces were frozen in June while the mobile apps kept shipping.

| Surface | Commits since 2026-06-18 | Screens / routes | Builds? | Tests |
|---|---|---|---|---|
| `apps/mobile-customer` | **187** | 54 | ✅ | several |
| `apps/web` (customer) | 18 (paused) | 37 | ⚠️ **was broken**, fixed in this pass | **none** |
| `apps/mobile-vendor` | **100** | 58 | ✅ | — |
| `apps/vendor-portal` | 19 (sunset) | 25 | ✅ | — |

Headline numbers, measured by API surface (see method below):

- **27 endpoints** the customer mobile app calls that the customer web app never calls.
- **28 endpoints** the vendor mobile app calls that the vendor portal never calls.
- **1 active defect** where web calls a *different, weaker* endpoint than mobile
  for the same user action (order cancellation — see §5).

---

## 2. Live topology (verified 2026-07-25)

| URL | Serves | Status |
|---|---|---|
| `fe3dr.com` | `apps/web-landing` (Next.js marketing) | 200 |
| `vendors.fe3dr.com` | `apps/vendor-portal` (Vite SPA, orphaned image) | 200 |
| `api.fe3dr.com` | `apps/api` (Go) | 200 |

Both web clients already point at the **same** API (`api.fe3dr.com`) the mobile
apps use — `apps/web/src/shared/services/api-client.ts` resolves
`VITE_API_URL`. There is no second backend to reconcile; parity work is purely
client-side.

⚠️ `vendors.fe3dr.com` also hosts the **mobile vendor app's API + auth-bff
routes**. Any routing change to that host must preserve `/bff/*` and `/api/*`
or it takes the vendor mobile app offline.

---

## 3. Method

Screen counts alone overstate parity (one mobile screen can cover three web
routes and vice versa). The objective measure used here is **which API
endpoints each client actually calls** — if mobile calls an endpoint no web
file references, that capability does not exist on web.

Extraction: every `api.<verb>(…)` / `apiClient.<verb>(…)` call site, path
literals normalised (`${x}` → `{id}`, `/v1` prefix stripped), sorted, diffed.

Caveat: this finds *missing capability*, not *inferior implementation*. A
shared endpoint can still be handled worse on one client — §5 is an example
found by reading, not by the diff.

---

## 4. Customer parity — `apps/web` vs `apps/mobile-customer`

### 4.1 Confirmed AT parity

Verified as called by both clients:

- **Wallet** — `/customer/wallet`, `/customer/wallet/transactions`. Both are
  read-only balance + ledger views. **No gap.**
- **Loyalty** — `/customer/loyalty`, `/customer/loyalty/redeem`,
  `/customer/loyalty/transactions`. **No gap.**
- **Checkout core** — `/orders`, `/payments/order/{id}/verify`,
  `/promo/validate`. Order placement and Razorpay verification are present on
  both. **Core is fine**; the gaps are adjacent (§4.2 "Checkout").
- Referral, favourites, group orders, tipping, reorder, social feed,
  subscriptions, addresses.

### 4.2 Missing from web (27 endpoints)

**Meal plans / tiffin — entirely absent from web (biggest single gap)**
```
/meal-plans                              /meal-plans/{id}
/meal-plans/{id}/cancel                  /meal-plans/{id}/verify-payment
/meal-subscriptions/{id}/fulfillments    /tiffin/confirm-today
```
Mobile has 5 screens here (`(tabs)/plans`, `meal-plans/index`,
`meal-plans/[id]`, `meal-plans/refund-choices`, `book-meal-plan`). Web has
none. This is a whole revenue line missing.

**Order lifecycle after placement**
```
/orders/{id}/track            live tracking       → mobile order/[id]/track
/customer/orders/{id}/messages  chef chat         → mobile order/[id]/messages
/orders/{id}/report-issue     issue reporting     → mobile order/[id]/report-issue
/orders/{id}/confirm-received delivery confirm    → (no web equivalent)
/orders/{id}/invoice-link     receipt/invoice     → partially added 2026-07-18
```

**Checkout adjacencies**
```
/chefs/{id}/delivery-quote      delivery fee quote before pay
/chefs/{id}/fulfillment-times   delivery slot selection
```
Web places orders without quoting delivery or letting the customer pick a slot.

**Account & data rights — compliance-relevant (see §7)**
```
/customer/me/export             /customer/me/delete
/customer/me/deletion-eligibility
/customer/me/deactivate         /customer/me/reactivate
```

**Catering money flow**
```
/catering/requests/{id}         /catering/requests/{id}/cancel
/catering/requests/{id}/deposit /catering/requests/{id}/deposit/verify
```
Web can raise a catering request and view quotes but cannot pay the deposit,
cancel, or complete the booking.

**Discovery & auth**
```
/search/dishes                  dish-level search (mobile search-dishes)
/reviews                        review submission endpoint
/account/email/otp/request      email OTP sign-in
/account/email/otp/verify
```

### 4.3 Web-only routes that are NOT gaps

`apps/web` still hosts chef, admin and delivery role routes (`/chef/*`,
`/admin/*`, `/delivery/*`) that mobile splits into separate apps. These are not
parity gaps — they are a different packaging decision. Note `admin-portal` was
retired 2026-07-17, so the web admin routes may now be the only admin surface;
confirm before touching them.

---

## 5. Defect: web cancels orders through the wrong flow

**Severity: money-affecting.**

Both endpoints exist in `apps/api/routes/routes.go`:

| Client | Endpoint | Behaviour |
|---|---|---|
| `apps/web` | `POST /orders/{id}/cancel` (line 614) | Legacy. Sets `status=cancelled` and returns. No policy check, no refund calculation. |
| `apps/mobile-customer` | `POST /orders/{id}/cancel-request` (line 616) | Current. Cancellation policy, refund calculator, chef confirmation, dispute path, admin arbitration. |

A customer cancelling an accepted order **on web** therefore skips the entire
cancellation/refund engine that shipped across
`feat/cancellation-*` and `feat/admin-cancellation-arbitration`. Mobile does not
call the legacy endpoint at all.

**Fix:** migrate `apps/web` order cancellation onto `/cancel-request` +
`/cancel-request/dispute`, mirroring `hooks/useCancellation.ts`. Do this before
web ordering is re-exposed to customers.

---

## 5b. Defect: nobody can log in to the web portals

**Severity: total outage of the surface. Found by logging in, 2026-07-25.**

Signing in at `vendors.fe3dr.com` with a valid account fails. Observed order:

| Request | Result |
|---|---|
| `POST identitytoolkit.googleapis.com/…/accounts:signInWithPassword` | **200** — GIP accepts the credentials |
| `GET  /bff/auth/session` | 401 (expected, no session yet) |
| `POST /bff/auth/exchange` | **400** `{"error":"unknown_host"}` |

Cause: commit `0fd4e5cf` (#22, 2026-06-17) removed the `web`, `vendor-portal`
and `delivery-portal` entries from `apps/auth-bff/homechef-products.yaml`.
Only the OIDC browser handlers resolve apps by host, so with no entry for
`vendors.fe3dr.com` the exchange cannot resolve an app and rejects every
sign-in. The SPA itself still serves — the host returns 200 and looks healthy
from outside, which is why this went unnoticed.

`apps/vendor-portal/SUNSET.md` claims removing the registry entry was
"deferred". It was not — it shipped in #22. That doc is wrong.

**Fixed** in `46257856`: both entries restored verbatim, with host-resolution
tests. The `HOMECHEF_CUSTOMER_CLIENT_SECRET` / `HOMECHEF_BUSINESS_CLIENT_SECRET`
env vars they reference are still wired in `charts/apps/homechef-auth-bff` and
`external-secrets/prod/homechef`, so no infra change is required.

⚠️ **Requires a deploy to take effect.** `homechef-products.yaml` is baked into
the image (`apps/auth-bff/Dockerfile:19` `COPY … /etc/auth-bff/`), so the fix
needs an auth-bff rebuild + ArgoCD sync. Consider mounting the registry from a
ConfigMap instead, so host changes stop requiring an image rebuild.

---

## 6. Vendor parity — `apps/vendor-portal` vs `apps/mobile-vendor`

28 endpoints missing from the portal:

**Account lifecycle / data rights**
```
/chef/me/export  /chef/me/delete  /chef/me/deletion-eligibility
/chef/me/deactivate  /chef/me/reactivate
```

**Availability** — `/chef/availability/pause`, `/chef/availability/resume`
(the "Taking a break" state added 2026-07-24 does not exist on the portal).

**Order operations**
```
/chef/orders/{id}/cancel              /chef/orders/{id}/items/{id}/cancel
/chef/orders/{id}/delivery-failed
```
Per-line cancellation and delivery-failed handling are mobile-only.

**Catering** — `/chef/catering/bookings`, `/chef/catering/requests/{id}/quote`,
`/chef/catering/requests/{id}/complete`.

**Support tickets** — `/support/tickets`, `/support/tickets/{id}/close`
(3 mobile screens, no portal equivalent).

**Media & documents** — `/chef/profile-image`, `/chef/banner-image`,
`/chef/kitchen-photos`, `/chef/menu/items/{id}/images/{id}`,
`/chef/documents/ocr`.

**Other** — `/chef/notification-preferences`, `/chef/onboarding/status`,
`/chef/admin-requests/{id}/remind`, paginated/filtered `/chef/orders` queries.

Also mobile-only by screen: meal-plan management (`daily-menu`, `weekly-menu`
exists on portal, but `meal-plans/index|[id]|prep|refund-decisions` do not),
`account-lifecycle`, `chef-agreement`, `documents/renew`, `language`,
`upgrade-required`.

---

## 7. Compliance exposure

`fe3dr.com/account-deletion/` (the URL filed in the Play Console Data Safety
form) currently instructs users to delete their account **inside the app**. That
is consistent while web is a marketing page. The moment `apps/web` becomes a
signed-in ordering surface again, the DPDP Act 2023 access/erasure rights and
Play's account-deletion policy need a web path too — i.e. `/customer/me/export`,
`/customer/me/delete`, `/customer/me/deletion-eligibility` must be wired into
web before launch, not after. Same for chefs on the portal.

---

## 8. Build & CI state

| Item | State |
|---|---|
| `apps/web` build | **Was failing** (4 TS errors). Fixed 2026-07-25 — see §9. |
| `apps/web` tests | **Zero test files.** `vitest` exits 1 with "No test files found". |
| `apps/vendor-portal` build | Passes. |
| `homechef-web-build.yml` | Present but disabled (manual-dispatch only; deploy job gated to push-on-main). |
| vendor-portal CI | **Deleted.** Must be recreated. |
| `homechef-web` ksvc | Currently serves the **web-landing** image. Reviving `apps/web` needs a routing/cutover decision in `tesserix-k8s` first. |

---

## 9. Changes already made in this pass (2026-07-25)

1. **`apps/web` compiles again** — was dead on `tsc -b`:
   - `shared/types/index.ts` — added the `OrderChef` type and `Order.chef`.
     The API *does* return it (`OrderChefResponse`); the web type was stale.
   - `shared/hooks/useSwipe.ts` — `NodeJS.Timeout` → `ReturnType<typeof setTimeout>`.
   - `features/chef/pages/MenuPage.tsx` — split `z.input` / `z.output` so the
     zod `.default()` fields stop breaking the resolver's assignability.
2. **Download surface no longer ships dead links** — `lib/site.ts` placeholders
   (`idTODO`, `com.homechef.customerTODO`) replaced with a typed listing model
   carrying the real package ids (`com.tesserix.homechef.customer` / `.vendor`).
   Unpublished listings render an inert "Coming soon to" badge instead of a
   link, and the JSON-LD `installUrl` key is omitted entirely rather than
   advertising a crawlable 404. New `/download/` page covers both apps.

---

## 10. Suggested sequencing

**P0 — before web is exposed to customers again**
1. Fix the cancellation flow (§5). Money-affecting.
2. Wire the account/data-rights endpoints into web + portal (§7). Compliance.
3. Decide the `homechef-web` ksvc cutover (landing vs app) in `tesserix-k8s`.
4. Recreate `homechef-vendor-portal-build.yml`.

**P1 — close the revenue-visible gaps**
5. Meal plans / tiffin on web (5 screens, 6 endpoints).
6. Checkout adjacencies: delivery quote + fulfilment slot.
7. Order tracking, chef chat, report-issue.

**P2 — chef portal**
8. Availability pause/resume, per-item cancel, delivery-failed.
9. Catering quote/complete, support tickets, media uploads.

**P3 — hardening**
10. Introduce tests for `apps/web` (currently zero).
