# Test Chef Mode — Design

**Date:** 2026-07-26
**Status:** Approved for planning

## Problem

We need to exercise the full Fe3dr flow — chef onboarding, menu publish, customer
browse, order, pay, chef accept/cook, delivery, confirm, refund, payout — against
**production** infrastructure (prod API, prod GKE, prod Keycloak/GIP, real mobile
builds on real devices) **without moving real money** and **without any real
customer ever seeing the fake kitchen**.

Separately, when a production issue hits a real kitchen, we want to flip that
kitchen into a sandbox, reproduce the issue against a copy of its real setup,
fix it, and flip back — with the kitchen's real data untouched throughout, on
the same database server, with no restore step.

Today there is exactly one Razorpay credential set shared by every payment path,
every admin-approved kitchen is visible to every customer, and there is no
concept of sandbox data at all.

## Goals

1. At approval time an admin chooses **Live** (default) or **Test** for a chef.
2. A **test chef** is invisible to every customer except an admin-managed
   allowlist of emails (seeded with `samyak.rout@gmail.com`,
   `unidevidp@gmail.com`, `mahesh.sangawar@gmail.com`).
3. Payments involving a test chef run against **Razorpay test credentials**;
   everything else keeps running against **live credentials**, automatically.
4. Both credential sets are owner-managed from the admin UI (Secret Manager
   backed), so switching or rotating either one needs no deploy.
5. Test activity never contaminates real money, real reporting, or real couriers.
6. Mode is a **data partition**: while a chef is in test mode, every chef-facing
   and admin-facing surface shows only that chef's test rows. Flipping to live
   shows only live rows. Nothing is ever deleted.
7. Flipping an established live chef to test **clones** their current setup and
   recent orders into a fresh test session, so the sandbox behaves like the real
   kitchen. Flipping back resumes live exactly where it left off.

## Non-goals

- A separate test environment, database, or namespace. This runs on prod, in the
  same database, on purpose — fast diagnosis is the point.
- Test-mode wallet top-ups or chef premium-subscription purchases. Those are not
  chef-attributable, so there is no mode to derive; they stay **live-only**.
- Automatic archival or expiry of test data. Sessions are retained until an
  admin explicitly purges one.

## Decisions taken during brainstorming

| Question | Decision |
|---|---|
| Mode mutability | **Freely flippable both ways**, subject to the in-flight guard below. |
| Allowlist storage | **Admin-editable `platform_settings` row**, mirroring the existing `security_policy` pattern. No deploy to change. |
| Test blast radius | Bypass 3PL dispatch, payout release, analytics/reporting. Plus wallet + loyalty (found during exploration — see §5). |
| Notifications | **Kept ON**, prefixed `[TEST]`. Recipients are only ever the allowlist plus the test chef/driver, and this is the only way to verify the FCM/Temporal push path on prod. |
| Route linked accounts | **Full Route in test mode** — test chefs get a real linked account against the Razorpay test key so transfers, holds and releases are exercised. |
| Chef stats | **Per-mode stats table.** `chef_profiles` keeps the live numbers so customer-facing rating stays truthful. |
| Admin console | **Global Test/Live toggle**, Stripe-style, across the whole HomeChef admin. |
| Live→test flip guard | **Blocked while orders are in flight**; allowed once settled. |
| Customer view of a flipped kitchen | **Listed but Closed, name only.** Born-test chefs stay fully hidden. |
| Clone depth | **Config + last 30 days of orders.** |
| Repeat flips | **New numbered session per flip**, previous sessions retained. |
| Rollout | **One PR, live on merge.** Safe by construction: every chef defaults to `live`. |

---

## 1. The mode dimension

### `ChefProfile.Mode`

```go
// Mode selects which Razorpay credential set, which visibility rules and which
// data partition apply to this kitchen. "live" (default) is a real kitchen
// taking real money from real customers. "test" is a sandbox: paid for with
// Razorpay test credentials, excluded from every real-money and reporting path,
// and serving only test-partition data.
Mode string `gorm:"type:varchar(4);not null;default:'live';index" json:"mode"`

// FirstLiveAt is stamped the first time this chef becomes live and never
// cleared. It distinguishes a born-test kitchen (nil — fully hidden from
// customers) from an established kitchen temporarily flipped to test (set —
// listed as Closed so its regulars don't think it shut down). See §4.
FirstLiveAt *time.Time `gorm:"" json:"firstLiveAt,omitempty"`

// ActiveTestSessionID points at the open ChefTestSession when Mode is "test".
// Nil when live. See §3.
ActiveTestSessionID *uuid.UUID `gorm:"type:uuid" json:"activeTestSessionId,omitempty"`
```

Constants in `models`: `ChefModeLive = "live"`, `ChefModeTest = "test"`.

An empty or unknown value **reads as live**. This is the fail-safe direction: a
column-default glitch can never accidentally hide a real kitchen, and can never
route a real payment through test keys (a test key against a real payment fails
loudly at Razorpay rather than silently taking no money).

### Mode on data rows

Mode is the partition key. Every chef-scoped transactional row carries it:

```sql
mode            varchar(4) NOT NULL DEFAULT 'live'
test_session_id uuid NULL          -- which test session this row belongs to
cloned_from_id  uuid NULL          -- set on rows produced by the clone (§3)
```

Rows in scope (`grep RazorpayOrderID models/` plus the chef-scoped transactional
tables): `orders` (+ `order_items`), `group_orders`, `meal_plans` (+
`meal_plan_days`), `meal_subscriptions`, `meal_trials`, `catering_requests`,
`tips`, `chef_promotions`, `reviews`, `refund_transactions`, `payout_*`,
`statements`, `order_issues`, `cancellation_requests`, `delivery_*`.

Implementation must enumerate these from the schema rather than trusting this
list, and the plan includes an explicit inventory step.

### Two distinct reads of mode

> **Chef-scoped configuration and data visibility read `chef.Mode`.
> Money operations on an existing record read that record's own `mode`.**

The second rule exists because mode is freely flippable. Chef flipped test→live,
then a refund is requested on an order paid in test mode: reading `chef.Mode` at
refund time would call the **live** Razorpay API with a **test** payment ID,
which 400s and strands the refund. Reading the order's own `mode` routes it
correctly forever.

`services.PaymentModeForChef(chefID)` is the single resolver used at record
creation. It is the only place `chef.Mode` is read for money.

### Why live data survives a round trip untouched

While a chef is in test mode every write goes to a test-partition row. Live rows
are never read-modify-written. So "flip to test, break things, flip back, live
continues exactly as it was" requires **no restore step and no snapshot of live**
— it is a property of the partitioning, not a feature. The clone (§3) is
one-directional: live → test, at flip time only.

---

## 2. Razorpay: two credential slots

### Secret Manager layout

| Slot | Secret names |
|---|---|
| **Live** | `prod-homechef-razorpay-key-id`, `-key-secret`, `-webhook-secret` |
| **Test** | `prod-homechef-razorpay-test-key-id`, `-test-key-secret`, `-test-webhook-secret` |

The live slot names are the **existing** ones, unchanged, so nothing currently
configured breaks. The new test slot is created and **seeded by copying the
current live-slot values into it**, because the live slot today holds a Razorpay
*test* key. The live slot keeps that same test key until a real live key is
generated and entered by the owner.

That interim state — a `rzp_test_*` key sitting in the live slot — is expected
and must be **savable**. Therefore:

- **The key-prefix check is a loud warning, not a hard block.** Saving
  `rzp_test_*` into the live slot succeeds, and the admin card renders a
  persistent `⚠ Live slot is holding a test key — no real payments will be
  captured` banner until a real `rzp_live_*` key replaces it.
- Saving `rzp_live_*` into the **test** slot is likewise warned, not blocked,
  but with stronger wording since that direction risks charging real cards for
  sandbox orders.

The Secret Manager copy is a one-time operational step, run against
`tesseracthub-480811` at deploy time and confirmed with the owner before
execution. It is recorded in the runbook (§10), not automated in code.

### Client factory

`services/razorpay.go` currently exposes a single cached singleton via
`GetRazorpay()`. It becomes a two-entry cache:

```go
// GetRazorpayFor returns the cached client for the given mode, fetching that
// mode's credentials from Secret Manager on a cache miss. Returns nil when that
// slot is not configured — callers must handle nil, as they already do.
func GetRazorpayFor(mode string) *RazorpayClient

// GetRazorpay is retained as GetRazorpayFor(live) so the ~20 existing
// non-chef-scoped call sites keep compiling unchanged.
func GetRazorpay() *RazorpayClient { return GetRazorpayFor(ChefModeLive) }
```

`InvalidateRazorpay()` gains a mode-scoped sibling so saving test keys does not
evict a healthy live client, and vice versa.

Call sites are converted deliberately, not mechanically. The chef-scoped ones —
`handlers/payment.go`, `handlers/chef_order_cancel.go`, `handlers/tips.go`,
`handlers/group_order.go`, `handlers/meal_plan.go`, `handlers/catering.go`,
`handlers/promotion.go`, `handlers/chefs.go`, `services/meal_plan_escrow.go`,
`services/orderrefund_gateway.go`, `services/payout_release.go` — take the
record's `mode`. The genuinely global ones (`handlers/admin.go` gateway status,
`services/reconciliation.go`) stay explicit about which slot they mean.

`razorpayKeyId` is already returned to clients in the checkout payloads
(`handlers/payment.go`, `tips.go`, `group_order.go`, `meal_plan.go`,
`catering.go`). Those payloads now carry the **mode-correct** key, so the mobile
Razorpay SDK opens the test checkout for a test chef with no client-side change
to key handling.

### Webhooks

One endpoint (`https://api.fe3dr.com/webhooks/razorpay`); both Razorpay
dashboards must point at it, each signing with its own secret.

`VerifyWebhookSignature` becomes `VerifyWebhookSignatureMode(payload, signature)
(ok bool, mode string)`: it tries the live secret first, then the test secret,
and reports which matched. The handler then:

1. Rejects the event if neither matched (unchanged behaviour).
2. Resolves the local record from the event's order/payment ID.
3. **Rejects the event if the matched mode differs from the record's `mode`** —
   a live-signed webhook must never mutate a test order, or vice versa.

Both attempts use `hmac.Equal` (constant time, matching the existing
implementation); the second is only made when the first fails.

`VerifyPaymentSignature` gains a mode parameter, called with the order's `mode`.

**Interim caveat:** while both slots hold the same test key, both secrets are
identical and step 1 always matches on the live attempt. Step 3 is what keeps
this correct — the record's own mode decides. This is called out because it
means the mode-disagreement rejection is load-bearing during the interim period,
not merely defensive.

### Admin surface

The Payment Gateway page becomes **two cards**, Live and Test, each with its own
key/secret/webhook-secret fields, save action, health check, status readout and
prefix warning. Endpoints gain a mode parameter, defaulting to `live` when
absent so the current UI keeps working during the deploy window:

- `GET /admin/payment-gateway/status?mode=live|test`
- `PUT /admin/payment-gateway/keys` with `{"mode":"live|test", ...}`

---

## 3. Test sessions and the live→test clone

### `ChefTestSession`

```go
type ChefTestSession struct {
    ID          uuid.UUID
    ChefID      uuid.UUID
    SessionNo   int        // 1, 2, 3… per chef, for human reference
    Status      string     // "open" | "closed" | "purged"
    Reason      string     // admin-entered: why we flipped this kitchen to test
    ClonedAt    *time.Time
    CloneSummary datatypes.JSON // {"menuItems":42,"orders":118,…}
    OpenedBy    uuid.UUID
    OpenedAt    time.Time
    ClosedAt    *time.Time
}
```

Every live→test flip opens a **new** session with a fresh clone. Flipping back to
live closes it. Previous sessions are retained and browsable, so last month's
debugging evidence is still there. An admin can explicitly purge a closed
session, which deletes its rows and marks it `purged`.

A chef approved directly as Test (born-test) also gets session 1, but with no
clone — there is nothing to clone from.

### What the clone copies

Depth is **configuration in full, plus a bounded window of order history**
(default 30 days, admin-adjustable at flip time). That is enough to reproduce
essentially any prod issue, and it completes in seconds rather than recursively
copying a year of ledger entries.

**Configuration (full copy):** menu items and their modifiers/options, menu
categories, weekly menu, daily menus, chef schedule, capacity settings, delivery
slots, self-delivery configuration, subscription/meal-plan offer config, and
dietary tags. The chef's profile row itself is **not** cloned — there is one
chef row; `Mode` on it is what switches worlds.

**History (windowed copy):** orders and their items within the window, with
their status, totals and fulfilment fields. Cloned orders are **historical
replicas, not re-payable**: gateway identifiers are cleared, so a cloned order
can be inspected and its status machine driven, but it can never be re-charged
or refunded against a real payment.

**Not cloned:** ledger entries, payout records, statements, invoices, wallet
balances, loyalty lots. Those are real-money artefacts; a copy would be
meaningless in test and dangerous if it leaked into reporting.

### Clone execution rules

- Runs inside one transaction; a failure rolls back and the flip is refused.
- **Emits no side effects.** No NATS events, no Temporal signals, no
  notifications, no webhooks. Cloned rows are inserted directly, bypassing the
  hooks that would fire on a real create. This is the single biggest correctness
  risk in the feature and gets dedicated tests.
- Every cloned row records `cloned_from_id`, so a cloned order is always
  distinguishable from one actually placed in the sandbox.
- PII columns (including the encrypted companions from #710) are copied verbatim
  rather than re-encrypted, so no key material is touched.
- Cloned orders reference **real customers**. They are therefore never surfaced
  to any customer (§4) — only to the chef, the admin console and allowlisted
  testers.

### The in-flight guard

A live→test flip is **refused** when the chef has any of: an active order (not in
a terminal state), a pending or in-progress payout, an open refund or
cancellation request, or an active meal plan / subscription. Flipping mid-dinner
would strand real customers holding an order they can no longer see.

The API returns a structured 409 listing exactly what is blocking, so the admin
UI can show "3 active orders, 1 pending payout" rather than a generic failure.

Test→live is always allowed; it simply closes the session.

---

## 4. Visibility

### The policy row

New `platform_settings` key `test_mode_policy`, following the `security_policy`
pattern in `services/platform_config.go` exactly (typed struct, 5-minute cache,
`Get`/`Save`/`Invalidate`, DB-backed, admin-editable):

```json
{
  "viewerEmails": [
    "samyak.rout@gmail.com",
    "unidevidp@gmail.com",
    "mahesh.sangawar@gmail.com"
  ]
}
```

`TestModePolicy.MayViewTestChefs(email string) bool` — case-insensitive,
whitespace-trimmed, mirroring `SecurityPolicy.IsTwoFactorExempt`. An empty email
(anonymous caller) is always false. Defaults ship in code with those three
emails so the feature works on deploy before anyone opens the admin UI.

### Two kinds of test chef

This is the distinction `FirstLiveAt` exists to make:

| | `FirstLiveAt == nil` (**born-test**) | `FirstLiveAt != nil` (**flipped**) |
|---|---|---|
| In listings/search | **Absent entirely** | **Present**, shown as Closed |
| Chef detail | 404 | Name, photo, Closed state; **no menu, no prices** |
| Ordering | Blocked | Blocked ("not accepting orders") |
| Allowlisted viewer | Full access, TEST badge | Full access, TEST badge |

A kitchen nobody has heard of should not exist. A kitchen with regulars should
not vanish — it should look closed, which it effectively is.

### Where it is enforced

The `/chefs` route group already runs `bffAuthOptional`, populating `userEmail`
in the Gin context when a session is present, so this works on public browse
endpoints without making them authenticated.

**List/search** — a GORM scope, sibling to the existing
`services.ExcludeFSSAILocked`:

```go
// TestChefVisibility filters chef_profiles for the calling viewer: allowlisted
// viewers see everything; everyone else has born-test kitchens removed and
// flipped kitchens forced to a closed presentation.
func TestChefVisibility(viewerEmail string) func(*gorm.DB) *gorm.DB
```

Applied in `handlers/chefs.go` `ListChefs`, `SearchDishes` and the dietary
sub-query. For flipped kitchens that survive the filter, the serializer forces
`acceptingOrders=false` and strips menu/price fields — done in the response
mapper, not by mutating the model.

**Single-chef routes** — every route under `/chefs/:id` consults a shared helper
and returns **404** for a born-test chef, or the reduced closed payload for a
flipped one: `GetChef`, `GetChefMenu`, `GetChefReviews`, `GetPublicWeeklyMenu`,
`GetPublicDailyMenu`, `GetChefOffer`, `GetChefDeliverySlots`,
`GetChefFulfillmentTimes`, `QuoteDeliveryFee`. 404 rather than 403 so a shared
link discloses nothing.

**Order creation** — `CreateOrder` hard-rejects with 403 when the chef is in
test mode and the customer is not allowlisted, regardless of how the chef ID was
obtained. Same guard on the group-order, meal-plan, meal-subscription and
catering create paths. Defence in depth: even if a discovery surface is missed,
no real customer can transact with a test kitchen.

**Customer order history** — always filtered to `mode = 'live'`, plus test orders
the allowlisted viewer actually placed themselves. Rows with `cloned_from_id`
set are **never** shown to any customer: a cloned order belongs to a real
customer who did not place it in the sandbox and must never see it. This is a
hard rule in the query layer, not a UI concern.

**web-landing** — the SSG chef index calls the API anonymously and therefore
never receives born-test chefs. Flipped kitchens appear as closed, which is
correct.

---

## 5. What a test order bypasses

Driven by `services.IsTestOrder(order)` and per-record equivalents. Each is a
small, explicit branch — not a global kill switch.

**3PL delivery dispatch — blocked.** `services/provider_dispatch.go` must never
hand a test order to Shadowfax or any external courier; a real rider would be
sent to a real address for a fake order. Test orders support pickup and chef
self-delivery, and remain assignable to our own platform drivers, so the driver
flow stays fully testable while no external courier is ever engaged.

**Payout release — blocked.** Test orders never enter `payout_release.go`,
`payout_automation.go`, `payout_mealplan_release_cron.go`, `statement.go` or the
statements PDF. Route transfers for a test order are created and released
**inside the Razorpay test account** via the test client, so the split-payment
path is genuinely exercised without reaching the real settlement engine.

**Analytics and reporting — excluded.** An `ExcludeTestOrders` scope is applied
to admin revenue analytics, chef earnings and analytics, GST/TDS reporting,
reconciliation crons and the ledger.

**Wallet and loyalty — blocked.** Found during exploration, and the most
dangerous leak in the feature: a test-order refund to the customer wallet would
mint real spendable balance from sandbox money, and loyalty points earned on a
test order would be redeemable against a live chef. So test orders cannot use
wallet as a payment source or refund destination (refunds go back to the
Razorpay test payment), earn or redeem no loyalty points, and credit no referral
rewards or promo-usage counters.

**Notifications — kept on, tagged.** Emails and push fire normally with `[TEST]`
prefixed to the subject and push title. Recipients are structurally limited to
the allowlist plus the test chef and assigned driver. Cloned rows emit nothing
at all (§3).

---

## 6. Chef stats

`chef_profiles` stores `Rating`, `TotalReviews`, `TotalOrders` and `IssueCount`.
Fake orders and reviews must never move them.

New table, one row per (chef, mode):

```go
type ChefModeStats struct {
    ChefID       uuid.UUID
    Mode         string
    TotalOrders  int
    Rating       float64
    TotalReviews int
    IssueCount   int
}
```

- `chef_profiles` continues to hold the **live** numbers verbatim, so every
  customer-facing surface keeps reading them with no query change and a real
  rating can never be polluted.
- Aggregate updates are routed through a single writer that targets the row for
  the record's mode; the live path additionally mirrors into `chef_profiles` so
  the two never drift.
- Vendor and admin dashboards read the stats for the **currently active** mode.

---

## 7. Admin console (tesserix-home)

All admin UI lives in the **tesserix-home** repo under
`apps/web/app/admin/apps/homechef/`, reaching the HomeChef Go API through the
existing `/api/admin/apps/homechef/gw` HMAC proxy — so new endpoints need no
proxy or backend plumbing.

**Global Test/Live toggle.** A switch in the HomeChef admin header, persisted per
admin user. In Test, every page — orders, payouts, analytics, refunds,
cancellations, wallets, reviews — shows only test-partition data. Defaults to
Live. When Test is active the console renders a persistent, high-contrast bar so
there is never ambiguity about which world an admin is acting in. The toggle is
sent to the API as a `X-HomeChef-Mode` header; the API applies it as a scope on
admin list/aggregate queries.

**Per-page changes:**

- `approvals/[id]` — the approve action becomes a dialog with a Live/Test choice,
  Live preselected, plus plain-language explanation of what Test means.
- `chefs` — TEST badge, mode filter, and a flip action that shows the in-flight
  blockers (§3) inline when refused and asks for a reason when allowed.
- `chefs/[id]/test-sessions` — session list with clone summary, open/close
  timestamps, reason, and an explicit purge action per closed session.
- `platform-settings` — editor for the test-mode viewer allowlist.
- `payment-gateway` — the two-card Live/Test layout with prefix warnings.

## 8. Client surfaces

**mobile-customer** — allowlisted users need to tell a sandbox kitchen from a
real one at a glance: a `TEST` badge on the chef card and detail header, and a
test-mode notice on checkout above the pay button. Non-allowlisted users never
receive a test chef from the API, so these render for nobody else.

**mobile-vendor** — a persistent `TEST MODE` banner when the signed-in chef is in
test mode, naming the session, so a chef can never mistake sandbox money for
real earnings.

---

## 9. Schema

Per repo convention, DDL lives in
`tesserix-k8s/charts/apps/db-schema-bootstrap/schemas/homechef/homechef/homechef_db.sql`
as idempotent `ALTER TABLE ... ADD COLUMN IF NOT EXISTS`, with matching GORM tags
in `apps/api/models/`. The API's `AutoMigrate` covers local dev; the bootstrap
CronJob is the production source of truth.

```sql
ALTER TABLE chef_profiles ADD COLUMN IF NOT EXISTS mode varchar(4) NOT NULL DEFAULT 'live';
ALTER TABLE chef_profiles ADD COLUMN IF NOT EXISTS first_live_at timestamptz NULL;
ALTER TABLE chef_profiles ADD COLUMN IF NOT EXISTS active_test_session_id uuid NULL;
CREATE INDEX IF NOT EXISTS idx_chef_profiles_mode ON chef_profiles(mode);

CREATE TABLE IF NOT EXISTS chef_test_sessions (...);
CREATE TABLE IF NOT EXISTS chef_mode_stats (...);

-- repeated for every chef-scoped transactional table (see §1):
ALTER TABLE <t> ADD COLUMN IF NOT EXISTS mode varchar(4) NOT NULL DEFAULT 'live';
ALTER TABLE <t> ADD COLUMN IF NOT EXISTS test_session_id uuid NULL;
ALTER TABLE <t> ADD COLUMN IF NOT EXISTS cloned_from_id uuid NULL;
CREATE INDEX IF NOT EXISTS idx_<t>_mode ON <t>(mode) WHERE mode <> 'live';
```

The partial index keeps the live hot path unaffected — the overwhelming majority
of rows are `live` and are not indexed at all.

Exact table names must be confirmed against the live schema during
implementation; the plan includes an inventory step for this.

---

## 10. Testing

Written test-first, following the repo's money-test conventions (in-memory
sqlite + `httptest` Razorpay stubs, as in `services/gateway_idempotency_test.go`
and `handlers/payment_test.go`).

**Mode resolution**
- Chef defaults to live; blank/unknown mode reads as live.
- A record snapshots the chef's mode at creation.
- Flipping the chef test→live does not change an existing record's `mode`, and a
  refund on that record still routes to the test client.

**Credential routing**
- `GetRazorpayFor` returns and caches independent clients per slot; invalidating
  one leaves the other intact.
- Prefix mismatch warns and still saves (the interim state must be reachable).
- Webhook verification identifies the signing mode; an event whose mode
  disagrees with the record's mode is rejected — including when both slots hold
  the same key.

**Visibility**
- Anonymous and non-allowlisted `ListChefs`/`SearchDishes` exclude born-test
  chefs and show flipped chefs as closed with no menu.
- Allowlisted viewers see both fully.
- `GET /chefs/:id` 404s for born-test, returns the reduced payload for flipped,
  full payload for allowlisted.
- Allowlist matching is case-insensitive and trims whitespace.
- A cloned order never appears in any customer's order history, including the
  customer it was cloned from.

**Guards**
- `CreateOrder` 403s for a non-allowlisted customer against a test chef.
- An allowlisted customer ordering from a **live** chef is unaffected and is
  charged real money — the allowlist must not create free-riders.
- A test order cannot pay from or refund to wallet, and earns zero loyalty.
- A test order is never dispatched to a 3PL and never enters payout release.
- A test order is excluded from admin revenue analytics and chef earnings.
- Live→test flip is refused with a structured blocker list when orders are in
  flight, and succeeds once settled.

**Clone**
- Produces the expected counts per configuration table and respects the order
  window.
- Emits **no** NATS events, Temporal signals, notifications or webhooks.
- Cloned orders carry no usable gateway identifiers and cannot be charged or
  refunded.
- Rolls back wholly on failure, leaving the chef live.
- A second flip opens session 2 and leaves session 1's rows intact.

**Round trip (the headline behaviour)**
- Flip a live chef with data to test, mutate heavily in the sandbox, flip back:
  every live row is byte-identical to before the flip, and the live dashboard
  shows exactly the pre-flip state.

**Regression**
- With no test chefs configured, every existing money and discovery test passes
  unchanged. This is the primary safety evidence for shipping live.

---

## 11. Limitations (accepted)

- **Wallet top-ups and chef premium-subscription purchases are always live.**
  Not chef-attributable, so no mode to derive. Testing those still costs real
  money.
- **Both Razorpay dashboards must point their webhook at the same URL.** A
  one-time manual console step per dashboard.
- **The live slot holds a test key until a real one is issued.** No real payment
  can be captured until it is replaced. Surfaced loudly in the admin card.
- **Clone is bounded at 30 days of orders and excludes ledger/payout artefacts.**
  A bug that only reproduces against a year-old order or a specific ledger state
  will not reproduce from a clone.
- **Bulk approve always mints live chefs.** Test kitchens are created one at a
  time, on purpose.
- **One PR.** Landing all of this at once means a large review surface and a
  longer verification pass than a phased ship would have needed. Accepted
  deliberately.

## 12. Rollout

One PR, merged and deployed live. Safety comes from the defaults, not a flag:
every existing chef is `live`, every existing row is `mode = 'live'`, and the
admin console defaults to Live. Nothing changes in production until an admin
explicitly marks a chef as test.

Post-merge runbook:

1. Deploy the API and confirm the schema bootstrap has run.
2. Copy the current live-slot Razorpay secrets into the new `-test-*` secrets in
   `tesseracthub-480811` (confirm with the owner before running).
3. Admin → Payment Gateway → confirm both cards read green, with the expected
   "live slot holds a test key" warning.
4. Point the Razorpay **test** dashboard's webhook at
   `https://api.fe3dr.com/webhooks/razorpay` and save its secret in the Test card.
5. Admin → Platform Settings → confirm the three viewer emails.
6. Onboard a chef, approve as **Test**, verify invisible on a non-allowlisted
   account and visible on an allowlisted one.
7. Place a test order end to end: test checkout opens, no real money moves, no
   3PL dispatch, `[TEST]` notifications arrive, order absent from revenue
   analytics.
8. On a settled live chef, flip to test, confirm the clone summary and that the
   kitchen shows as Closed to a real customer; flip back and confirm live data
   is exactly as it was.
