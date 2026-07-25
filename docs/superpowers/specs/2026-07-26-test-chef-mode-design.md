# Test Chef Mode — Design

**Date:** 2026-07-26
**Status:** Approved for planning

## Problem

We need to exercise the full Fe3dr flow — chef onboarding, menu publish, customer
browse, order, pay, chef accept/cook, delivery, confirm, refund, payout — against
**production** infrastructure (prod API, prod GKE, prod Keycloak/GIP, real mobile
builds on real devices) **without moving real money** and **without any real
customer ever seeing the fake kitchen**.

Today there is exactly one Razorpay credential set, shared by every payment path,
and every admin-approved kitchen is visible to every customer. There is no way to
have a sandbox kitchen coexist with live kitchens.

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

## Non-goals

- A separate test environment, database, or namespace. This runs on prod.
- Test-mode wallet top-ups or chef premium-subscription purchases. Those are not
  chef-attributable payments and stay **live-only**; see Limitations.
- Automated purge of test data. Test rows are excluded from reporting and can be
  deleted manually if needed.

## Decisions taken during brainstorming

| Question | Decision |
|---|---|
| Mode mutability | **Freely flippable both ways** at any time by an admin. This forces per-record mode snapshots (see §2). |
| Allowlist storage | **Admin-editable `platform_settings` row**, mirroring the existing `security_policy` pattern. No deploy to change. |
| Test blast radius | Bypass 3PL dispatch, payout release, analytics/reporting. Plus wallet + loyalty (found during exploration — see §4). |
| Notifications | **Kept ON**, prefixed `[TEST]`. Recipients are only ever the allowlist plus the test chef/driver, and this is the only way to verify the FCM/Temporal push path on prod. |
| Route linked accounts | **Full Route in test mode** — test chefs get a real linked account against the Razorpay test key so transfers, holds and releases are exercised. |
| Rollout | **One PR, live on merge.** Safe by construction: every chef defaults to `live`, so nothing changes until an admin explicitly marks a chef as test. |

---

## 1. The mode dimension

### `ChefProfile.Mode`

```go
// Mode selects which Razorpay credential set and which visibility rules apply
// to this kitchen. "live" (default) is a real kitchen taking real money from
// real customers. "test" is a sandbox kitchen: visible only to the test-mode
// viewer allowlist, paid for with Razorpay test credentials, and excluded from
// every real-money and reporting path.
Mode string `gorm:"type:varchar(4);not null;default:'live';index" json:"mode"`
```

Constants live in `models`: `ChefModeLive = "live"`, `ChefModeTest = "test"`.

An empty/unknown value **reads as live**. This is the fail-safe direction: a
column-default glitch can never accidentally hide a real kitchen, and can never
route a real payment through test keys (a test key on a real order fails loudly
at Razorpay rather than silently taking no money).

### Why a per-record snapshot is mandatory

Because mode is freely flippable, the chef row is not a reliable source of truth
for money operations on an *existing* record. Concretely: chef flipped test→live,
then a customer requests a refund on an order paid in test mode. Reading
`chef.Mode` at refund time would call the **live** Razorpay API with a **test**
payment ID — which 400s, and the refund silently strands.

So every row that stores a `razorpay_order_id` gains:

```sql
payment_mode varchar(4) NOT NULL DEFAULT 'live'
```

Rows identified during exploration (`grep RazorpayOrderID models/`):

| Model | File |
|---|---|
| `Order` | `models/order.go` |
| `GroupOrder` | `models/group_order.go` |
| `MealPlan` | `models/meal_plan.go` |
| `MealTrial` / `MealSubscription` | `models/meal_subscription.go` |
| `CateringRequest` | `models/catering.go` |
| `Tip` | `models/tip.go` |
| `ChefPromotion` | `models/promotion.go` |

Implementation must re-run that grep and cover **every** hit, not just this list.

### The rule

> **Chef-scoped configuration reads `chef.Mode`. Money operations on an existing
> record read that record's `payment_mode`.**

A mode flip therefore never touches in-flight money. Orders created before the
flip settle, refund and pay out under the credentials they were created with.

`services.PaymentModeForChef(chefID) (string, error)` is the single resolver used
at record-creation time. It is the only place `chef.Mode` is read for money.

---

## 2. Razorpay: two credential slots

### Secret Manager layout

| Slot | Secret names |
|---|---|
| **Live** | `prod-homechef-razorpay-key-id`, `-key-secret`, `-webhook-secret` |
| **Test** | `prod-homechef-razorpay-test-key-id`, `-test-key-secret`, `-test-webhook-secret` |

The live slot names are the **existing** ones, unchanged, so nothing currently
configured breaks. Both slots are seeded to the literal `placeholder` by the
Helm bootstrap; `isPlaceholderValue` already treats that as "not configured", so
an unconfigured slot degrades to a clear admin-facing error rather than a crash.

### Client factory

`services/razorpay.go` currently exposes a single cached singleton via
`GetRazorpay()`. It becomes a two-entry cache:

```go
type PaymentMode string // "live" | "test"

// GetRazorpayFor returns the cached client for the given mode, fetching that
// mode's credentials from Secret Manager on a cache miss. Returns nil when that
// slot is not configured — callers must handle nil, as they already do.
func GetRazorpayFor(mode string) *RazorpayClient

// GetRazorpay is retained as GetRazorpayFor(live) so the ~20 existing
// non-chef-scoped call sites keep compiling unchanged.
func GetRazorpay() *RazorpayClient { return GetRazorpayFor(ChefModeLive) }
```

`InvalidateRazorpay()` gains a mode-scoped sibling so saving test keys does not
evict a healthy live client (and vice versa).

Call sites are converted deliberately, not mechanically. The chef-scoped ones —
`handlers/payment.go`, `handlers/chef_order_cancel.go`, `handlers/tips.go`,
`handlers/group_order.go`, `handlers/meal_plan.go`, `handlers/catering.go`,
`handlers/promotion.go`, `handlers/chefs.go` (payout details),
`services/meal_plan_escrow.go`, `services/orderrefund_gateway.go`,
`services/payout_release.go` — take the record's `payment_mode`. The genuinely
global ones (`handlers/admin.go` gateway status, `services/reconciliation.go`)
stay explicit about which slot they mean.

`razorpayKeyId` is already returned to clients in the checkout payloads
(`handlers/payment.go`, `tips.go`, `group_order.go`, `meal_plan.go`,
`catering.go`). Those payloads now carry the **mode-correct** key, so the mobile
Razorpay SDK opens the test checkout for a test chef with no client-side change
to key handling.

### Prefix guardrail

Razorpay keys are self-describing: `rzp_test_*` vs `rzp_live_*`.

- **On write** (`UpdatePaymentGatewayKeys`): saving `rzp_test_*` into the live
  slot, or `rzp_live_*` into the test slot, is rejected with an explicit error.
  This is the single highest-value safety check in the feature — it makes the
  catastrophic mix-up (real customers charged against a sandbox, or test orders
  taking real money) unrepresentable.
- **On read** (`GetPaymentGatewayStatus`): an already-stored key whose prefix
  does not match its slot surfaces a warning in the admin card, so a
  pre-existing mismatch is self-correcting rather than silent.

### Webhooks

There is one webhook endpoint (`https://api.fe3dr.com/webhooks/razorpay`) and
both Razorpay dashboards (test and live) must be pointed at it. Each signs with
its own secret.

`VerifyWebhookSignature(payload, signature)` becomes
`VerifyWebhookSignatureMode(payload, signature) (ok bool, mode string)`: it tries
the live secret first, then the test secret, and reports which one matched. The
handler then:

1. Rejects the event if neither matched (unchanged behaviour).
2. Resolves the local record from the event's order/payment ID.
3. **Rejects the event if the matched mode differs from the record's
   `payment_mode`** — a live-signed webhook must never mutate a test order, and
   vice versa. This closes the cross-mode confusion window.

Signature comparison uses `hmac.Equal` for both attempts (constant time,
matching the existing implementation), and the second attempt is only made when
the first fails, so the common case costs one extra comparison at most.

`VerifyPaymentSignature` gains a mode parameter and is called with the order's
`payment_mode` at verify time.

### Admin surface

The Payment Gateway page becomes **two cards**, Live and Test, each with its own
key/secret/webhook-secret fields, save action, health check and status readout.
The existing endpoints gain a mode parameter:

- `GET  /admin/payment-gateway/status?mode=live|test`
- `PUT  /admin/payment-gateway/keys` with `{"mode":"live|test", ...}`

Mode defaults to `live` when absent, so the current admin UI keeps working during
the deploy window.

---

## 3. Visibility

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

`TestModePolicy.MayViewTestChefs(email string) bool` does a case-insensitive,
whitespace-trimmed comparison — mirroring `SecurityPolicy.IsTwoFactorExempt`. An
empty email (anonymous caller) is always false.

Defaults ship in code with those three emails, so the feature works immediately
on deploy even before anyone touches the admin UI.

### Where it is enforced

The `/chefs` route group already runs `bffAuthOptional`, which populates
`userEmail` in the Gin context when a session is present, so the check is
available on public browse endpoints without making them authenticated.

**List/search surfaces** — a GORM scope, sibling to the existing
`services.ExcludeFSSAILocked`:

```go
// ExcludeTestChefs removes test-mode kitchens from a chef_profiles query.
// Applied to every customer discovery surface unless the caller is on the
// test-mode viewer allowlist.
func ExcludeTestChefs(db *gorm.DB) *gorm.DB
```

Applied in `handlers/chefs.go` `ListChefs` and `SearchDishes`, and to the
chef-subquery in the dietary filter.

**Single-chef surfaces** — every route under the `/chefs/:id` group returns
**404** (not 403) for a non-allowlisted caller: `GetChef`, `GetChefMenu`,
`GetChefReviews`, `GetPublicWeeklyMenu`, `GetPublicDailyMenu`, `GetChefOffer`,
`GetChefDeliverySlots`, `GetChefFulfillmentTimes`, `QuoteDeliveryFee`. 404 rather
than 403 so a shared link discloses nothing about the kitchen's existence.

A small helper keeps this from being copy-pasted nine times:

```go
// testChefVisible reports whether the caller may see this chef. Handlers call
// it right after loading the chef and 404 when it returns false.
func testChefVisible(c *gin.Context, chef *models.ChefProfile) bool
```

**Order creation** — `CreateOrder` in `handlers/orders.go` hard-rejects with 403
when the chef is test-mode and the customer is not allowlisted, regardless of how
the chef ID was obtained. Same guard on the group-order, meal-plan, meal-
subscription and catering create paths. This is defence in depth: even if a
discovery surface is missed, no real customer can transact with a test kitchen.

**Reorder / favourites** need no guard: a non-allowlisted customer can never have
ordered from or favourited a test chef in the first place, and the create-path
guard catches any residual case.

**web-landing** (the SSG chef index on `fe3dr.com`) calls the API anonymously and
therefore never receives test chefs — no change needed there.

---

## 4. What a test order bypasses

A helper `services.IsTestOrder(order)` (and equivalents per record type) drives
all of these. Each is a small, explicit branch — not a global kill switch.

### 3PL delivery dispatch — blocked

`services/provider_dispatch.go` must never hand a test order to Shadowfax or any
external courier: a real rider would be sent to a real address for a fake order.
Test orders support **pickup** and **chef self-delivery**, and remain assignable
to our own platform drivers through the delivery app — so the driver flow stays
fully testable while no external courier is ever engaged.

### Payout release — blocked

Test orders never enter `services/payout_release.go`, `payout_automation.go`,
`payout_mealplan_release_cron.go`, `statement.go` or the chef statements PDF.
Route transfers for a test order are created and released **inside the Razorpay
test account** via the test client, so the split-payment code path is genuinely
exercised, but nothing reaches the real settlement engine or a real bank.

### Analytics and reporting — excluded

An `ExcludeTestOrders` scope is applied to admin revenue analytics, chef earnings
and analytics, GST/TDS reporting, reconciliation crons and the ledger. Test rows
must never move a real number.

### Wallet and loyalty — blocked

Found during exploration, and the most dangerous leak in the feature:

- A test order refunding to the customer **wallet** would create real,
  spendable balance out of sandbox money.
- **Loyalty points** earned on a test order would be redeemable against a live
  chef.

Therefore: test orders cannot use wallet as a payment source or a refund
destination (refunds go back to the Razorpay test payment), and they neither earn
nor redeem loyalty points. Referral rewards and promo-code usage counters are
likewise not credited by test orders.

### Notifications — kept on, tagged

Emails and push for test orders fire normally, with `[TEST]` prefixed to the
email subject and the push title. Recipients are structurally limited to the
allowlist plus the test chef and assigned driver, so there is no blast radius,
and this is the only way to verify the FCM/Temporal dispatch path on prod.

---

## 5. Admin and client surfaces

### API

| Endpoint | Change |
|---|---|
| `POST /admin/approvals/:id/approve` | accepts optional `{"mode":"live"\|"test"}`, defaults `live` |
| `POST /admin/approvals/bulk-approve` | always approves as `live` (bulk is never the right place to mint test kitchens) |
| `PATCH /admin/chefs/:id/mode` | flips a chef between live and test; audited via `LogAudit` |
| `GET/PUT /admin/test-mode-policy` | reads/writes the viewer allowlist |
| `GET /admin/chefs` | returns `mode`, and accepts a `mode=` filter |
| `GET/PUT /admin/payment-gateway/*` | mode-scoped as described in §2 |

`services.ActivateChefOnboarding` gains the mode so the approval → activation
path (which runs both inline and as a Temporal activity) sets `chef.Mode`
idempotently alongside `is_verified`.

### tesserix-home admin (`apps/web/app/admin/apps/homechef/`)

All admin UI lives in the **tesserix-home** repo and reaches the HomeChef Go API
through the existing `/api/admin/apps/homechef/gw` HMAC proxy, so no proxy or
backend plumbing is needed for new endpoints.

- `approvals/[id]` — the approve action becomes a small dialog with a Live/Test
  choice, Live preselected, and a plain-language note about what Test means.
- `chefs` — a `TEST` badge on test rows, a mode filter, and a switch-mode action
  with a confirmation step.
- `platform-settings` — an editor for the test-mode viewer allowlist.
- `payment-gateway` — the two-card Live/Test layout.

### mobile-customer

Allowlisted users need to tell a test kitchen from a real one at a glance,
otherwise the whole exercise is confusing:

- `TEST` badge on the chef card in listings and on the chef detail header.
- A test-mode notice on the checkout screen above the pay button.

Non-allowlisted users never receive a test chef from the API, so these render for
nobody else.

### mobile-vendor

A persistent `TEST MODE` banner when the signed-in chef is test-mode, so a chef
can never mistake sandbox money for real earnings.

---

## 6. Schema

Per the repo convention, DDL lives in
`tesserix-k8s/charts/apps/db-schema-bootstrap/schemas/homechef/homechef/homechef_db.sql`
as idempotent `ALTER TABLE ... ADD COLUMN IF NOT EXISTS` statements, with the
matching GORM tags in `apps/api/models/`. The API's `AutoMigrate` covers local
dev; the bootstrap CronJob is the production source of truth.

Columns added:

```sql
ALTER TABLE chef_profiles ADD COLUMN IF NOT EXISTS mode varchar(4) NOT NULL DEFAULT 'live';
CREATE INDEX IF NOT EXISTS idx_chef_profiles_mode ON chef_profiles(mode);

ALTER TABLE orders                ADD COLUMN IF NOT EXISTS payment_mode varchar(4) NOT NULL DEFAULT 'live';
ALTER TABLE group_orders          ADD COLUMN IF NOT EXISTS payment_mode varchar(4) NOT NULL DEFAULT 'live';
ALTER TABLE meal_plans            ADD COLUMN IF NOT EXISTS payment_mode varchar(4) NOT NULL DEFAULT 'live';
ALTER TABLE meal_subscriptions    ADD COLUMN IF NOT EXISTS payment_mode varchar(4) NOT NULL DEFAULT 'live';
ALTER TABLE meal_trials           ADD COLUMN IF NOT EXISTS payment_mode varchar(4) NOT NULL DEFAULT 'live';
ALTER TABLE catering_requests     ADD COLUMN IF NOT EXISTS payment_mode varchar(4) NOT NULL DEFAULT 'live';
ALTER TABLE tips                  ADD COLUMN IF NOT EXISTS payment_mode varchar(4) NOT NULL DEFAULT 'live';
ALTER TABLE chef_promotions       ADD COLUMN IF NOT EXISTS payment_mode varchar(4) NOT NULL DEFAULT 'live';
```

Table names must be confirmed against the live schema during implementation.

---

## 7. Testing

Written test-first, following the repo's existing money-test conventions
(`sqlite` in-memory + `httptest` Razorpay stubs, as in
`services/gateway_idempotency_test.go` and `handlers/payment_test.go`).

**Mode resolution**
- Chef defaults to `live`; blank/unknown mode reads as live.
- Order snapshots the chef's mode at creation.
- Flipping the chef test→live does **not** change an existing order's
  `payment_mode`, and a refund on that order still routes to the test client.

**Credential routing**
- `GetRazorpayFor` returns independent clients per slot and caches them
  independently; invalidating one leaves the other intact.
- Prefix guardrail rejects `rzp_test_*` into live and `rzp_live_*` into test.
- Webhook signature verification identifies the signing mode correctly, and an
  event whose mode disagrees with the record's `payment_mode` is rejected.

**Visibility**
- Anonymous `ListChefs` excludes test chefs.
- Non-allowlisted authenticated `ListChefs`/`SearchDishes` exclude test chefs.
- Allowlisted `ListChefs` includes them.
- `GET /chefs/:id` returns 404 for a non-allowlisted caller and 200 for an
  allowlisted one.
- Allowlist matching is case-insensitive and trims whitespace.

**Guards**
- `CreateOrder` 403s for a non-allowlisted customer against a test chef.
- Allowlisted customer ordering from a **live** chef is unaffected and charges
  real money (regression guard — the allowlist must not make these users
  free-riders).
- A test order cannot pay from wallet, cannot refund to wallet, and earns zero
  loyalty points.
- A test order is never dispatched to a 3PL provider.
- A test order never appears in payout release or chef statements.
- A test order is excluded from admin revenue analytics and chef earnings.

**Regression**
- With no test chefs configured, every existing money and discovery test passes
  unchanged. This is the primary safety evidence for shipping live.

---

## 8. Limitations (accepted)

- **Wallet top-ups and chef premium-subscription purchases are always live.**
  They are not chef-attributable, so there is no mode to derive. Testing those
  flows still costs real money. Documented rather than solved.
- **Both Razorpay dashboards must point their webhook at the same URL.** This is
  a manual, one-time console step per dashboard, called out in the runbook.
- **No automated test-data purge.** Test rows are excluded from reporting; if
  they need to go, they are deleted by hand.
- **Bulk approve always mints live chefs.** Test kitchens are created one at a
  time, on purpose.

## 9. Rollout

One PR, merged and deployed live. Safety comes from the defaults, not from a
flag: every existing chef is `live`, every existing record is `payment_mode
= 'live'`, and the test credential slot starts as `placeholder`. Nothing changes
in production until an admin (a) enters test keys and (b) explicitly marks a
chef as test.

Post-merge runbook:

1. Deploy the API (Kargo/ArgoCD) and confirm the schema bootstrap has run.
2. Admin → Payment Gateway → **Test** card → enter the Razorpay test keys; the
   card's health check must go green.
3. Point the Razorpay **test** dashboard's webhook at
   `https://api.fe3dr.com/webhooks/razorpay` and save the test webhook secret in
   the same card.
4. Admin → Platform Settings → confirm the three viewer emails.
5. Onboard a chef, approve as **Test**, and verify it is invisible on a
   non-allowlisted account and visible on an allowlisted one.
6. Place a test order end to end and confirm: test checkout opens, no real money
   moves, no 3PL dispatch, `[TEST]` notifications arrive, and the order is
   absent from admin revenue analytics.
