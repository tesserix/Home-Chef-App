# PLAN — #462 leftover: provider/delivery webhook replay dedup

Epic #403. One of the #462 "medium cluster" leftovers (items 1 & 5 already shipped in PR #499).
Security/correctness, **not escrow-flag-gated** — worth doing regardless of the flag flip.

## Problem

Both inbound webhook paths verify only HMAC-over-body — **no timestamp window, no event-level dedup**.
The *only* replay protection today is each handler's downstream conditional-UPDATE guard
(`WHERE status <> completed`, `RowsAffected==0 → skip`). That is per-*effect*, not per-*event*, so:

- **Razorpay** (`handlers/payment.go:1024 RazorpayWebhook`): a replayed (or provider-retried)
  `payment.captured` re-runs the whole handler. The status flip is guarded, but the *side effects*
  keyed off `RowsAffected>0` (referral grant, saga start, **chef new-order push**) only fire on the
  single winning transition — a genuine Razorpay retry after a slow first delivery can still double-fire
  the chef push under a verify/webhook race (#395). `refund.processed`, `subscription.charged/halted`
  each rely on their own ad-hoc guard.
- **Delivery** (`handlers/delivery_provider.go:623 HandleWebhook` → `services.HandleProviderWebhook`):
  a replayed `delivered` callback re-stamps `delivered_at = now()` (timestamp drift), **re-enqueues the
  `provider.delivery.updated` outbox event** every time, and re-runs the escrow-park
  (`SetOrderHoldAwaitingConfirmation`) which is only *status-guarded*. A "delivered" event releases held
  escrow downstream — this is the most sensitive replay surface.

Issue #462 ask: *"add a ts window or a (provider, event-id) dedup table (currently saved only by
downstream conditional-UPDATE guards)."*

## Design — event-level dedup, reuse `processed_events`

Reuse the existing consumer-idempotency ledger `models.ProcessedEvent` (composite PK
`(consumer, msg_id)`, already AutoMigrated, already used by the NATS backbone in `services/consumers.go`
with `OnConflict{DoNothing}`). **No new table / migration.**

New file `apps/api/services/webhook_dedup.go`:

```go
// ClaimWebhookEvent atomically claims (consumer, eventID). Returns firstTime=true
// when this is the first time we've seen the event (caller should process it),
// false when it's a replay (caller should skip). Atomic via INSERT ... ON CONFLICT
// DO NOTHING + RowsAffected — no check-then-act TOCTOU under concurrent deliveries.
func ClaimWebhookEvent(db *gorm.DB, consumer, eventID, subject string) (firstTime bool, err error)

// ReleaseWebhookEvent removes a claim so a provider RETRY re-processes the event
// after a failed dispatch (delivery path returns 500 → provider redelivers).
// Best-effort; logs on error.
func ReleaseWebhookEvent(db *gorm.DB, consumer, eventID string)

// WebhookEventID returns headerID when non-empty AND ≤ 64 chars, else a stable
// hash of the body ("h:"+base64rawurl(sha256(body)), 45 chars). ALWAYS ≤ 64 so
// it never overflows processed_events.msg_id varchar(64) — Postgres ERRORS on
// overflow (→ 500 poison-loop), and sqlite silently accepts it (test blind spot),
// so the cap is enforced in code, not the column. Body-hash makes an EXACT replay
// a duplicate while genuinely-distinct events (different status/ts) stay unique.
func WebhookEventID(headerID string, body []byte) string   // len ≤ 64 guaranteed
```

**Plan-check fix (HIGH):** cap the returned id at 64 — a header id longer than the column
would 500 on Postgres but pass on sqlite, a latent prod-only failure. If `headerID` is empty
OR `len(headerID) > 64`, fall back to the `h:` body-hash.

`ClaimWebhookEvent` body:
```go
rec := &models.ProcessedEvent{Consumer: consumer, MsgID: eventID, Subject: subject}
res := db.Clauses(clause.OnConflict{DoNothing: true}).Create(rec)
if res.Error != nil { return false, res.Error }
return res.RowsAffected == 1, nil
```

Consumer labels (≤ varchar(64)): `webhook:razorpay`, `webhook:stripe`, `webhook:delivery:<code>`.
Subject (audit, varchar(255)): the event type (`event.Event` / `event.Type` / `delivery:<code>`).

### Wiring — ordering rule: **verify signature BEFORE claim** (never let a forged/unauthed request write a dedup row)

**Razorpay** (`RazorpayWebhook`): after `VerifyWebhookSignature` + minimal unmarshal:
```
eventID := WebhookEventID(c.GetHeader("X-Razorpay-Event-Id"), body)
firstTime, err := ClaimWebhookEvent(db, "webhook:razorpay", eventID, event.Event)
if err != nil { 500 }                         // DB down → let provider retry
if !firstTime { 200 {"status":"duplicate"}; return }  // replay → skip dispatch
<existing switch>
200 {"status":"ok"}
```
**Plan-check fix (MEDIUM, finding 4):** claim-first-**keep** would strand a *swallowed* handler
failure — e.g. `handlePaymentCaptured`'s conditional UPDATE returns `res.Error` (logged + swallowed
today); with the claim kept, a later duplicate delivery / manual replay is deduped and the payment can
never be re-driven to `completed`. Fix: make the Razorpay sub-handlers **return error**, and on a
non-nil dispatch error **`ReleaseWebhookEvent` + still return 200** (preserve today's always-200 —
Razorpay won't auto-retry, but the released claim restores the current accidental "a duplicate delivery
re-runs the conditional UPDATE" recovery net). Parse-fail inside a handler = poison → return nil (keep
claim, nothing to retry). Best-effort side-effects (referral/notify/saga) stay logged-not-returned.

**Stripe: OUT OF SCOPE** for this PR — the domestic gateway is Razorpay (Stripe is dormant in India).
Adding dedup only to Razorpay leaves Stripe exactly as-is (no dedup, no regression). Fast-follow with the
same helper (`event.ID` is already parsed) noted below.

**Delivery** (`HandleWebhook`) — failure-aware, because this endpoint returns 500 → provider redelivers.
**Plan-check fix (HIGH, finding 2):** releasing on *every* error poison-loops the deterministic ones
(unmapped status, parse-fail, no-id, no-status all re-error identically → release → 500 → retry storm,
dedup never engages). Classify with a new sentinel `services.ErrWebhookPermanent`:
```
eventID := WebhookEventID(c.GetHeader("X-Webhook-Event-Id"), body)   // after sig verify
firstTime, err := ClaimWebhookEvent(db, "webhook:delivery:"+provider.Code, eventID, "delivery:"+provider.Code)
if err != nil { 500; return }
if !firstTime { 200 {"status":"duplicate"}; return }
if err := providerService.HandleProviderWebhook(provider.Code, body); err != nil {
    if errors.Is(err, services.ErrWebhookPermanent) {   // un-processable → ACK, keep claim, no retry storm
        200 {"status":"ignored"}; return
    }
    ReleaseWebhookEvent(db, "webhook:delivery:"+provider.Code, eventID)  // transient → let the retry re-process
    500; return
}
200
```
`HandleProviderWebhook` wraps its **deterministic** returns with `%w ErrWebhookPermanent` (parse-fail,
no-delivery-id, no-status, unmapped-status). **Transient** returns stay bare (default → release + 500,
preserving today's retry): delivery-not-found (booking-vs-callback race), the delivery `Updates` DB
failure, provider re-lookup DB error.

**Accepted narrow window (finding 3, documented):** claim + `HandleProviderWebhook` are not one tx, so a
concurrent redelivery to a second instance during a transient-failing first attempt could see the claim,
200-"duplicate", and drop — but the first attempt returned 500, so the provider redelivers again and the
(now-released) claim is re-taken and processed. No double-mutation (every `HandleProviderWebhook` error
return precedes its single `Updates`). Full atomicity would need threading a tx through
`HandleProviderWebhook` (it uses the global `database.DB` throughout) — deferred as scope.

### Why body-hash fallback is safe
Providers that send a stable event/delivery id in a header → we dedup by it. Absent that, an EXACT
byte-identical POST is the definition of a replay; any real state change (new status, new timestamp,
moved rider coords) yields a different body → not deduped. Downstream conditional guards remain the
mutation-level backstop; this is defense-in-depth, not a replacement.

**Reality (plan-check finding 7):** the delivery handler today only reads `X-Webhook-Signature`; there's
no evidence Borzo/Shadowfax send an event-id header, so delivery dedup rests **in practice on the
body-hash** (`X-Webhook-Event-Id` is kept only as forward-compat). Worst case of an over-dedup is a
dropped rider-location refresh (money is guarded downstream) — acceptable.

## Tests (RED-first, table + concurrency)

`apps/api/services/webhook_dedup_test.go` (sqlite in-mem like existing service tests):
- `TestClaimWebhookEvent_FirstThenReplay` — first claim `firstTime=true`; identical second `false`;
  different consumer OR different id → `true` (independent).
- `TestClaimWebhookEvent_Concurrent` — N goroutines claim the same (consumer,id) with the sqlite pool
  pinned to **1 conn** (`SetMaxOpenConns(1)` — `:memory:` opens a fresh empty DB per pooled conn, so an
  unpinned pool flakes on "no such table"): **exactly one** gets `firstTime=true`. Validates the claim
  idiom returns a single winner when serialized (true concurrent arbitration is a Postgres property, not
  asserted here).
- `TestReleaseWebhookEvent_AllowsReprocess` — claim → release → claim again `firstTime=true`.
- `TestWebhookEventID` — header ≤64 wins; empty header OR header >64 → deterministic `h:` hash, result
  **always ≤ 64 chars**; different bodies → different ids.

`apps/api/handlers/payment_test.go` (extend existing harness):
- `TestRazorpayWebhook_ReplaySkipsDispatch` — POST captured twice (same `X-Razorpay-Event-Id`); order
  flips once; **second response `{"status":"duplicate"}`**. (The no-second-side-effect part is already
  green via the existing `RowsAffected==0` guard on sequential replay — the RED assertion here is the
  duplicate response + the claim row existing.)
- signature-fail path still 401 and writes **no** dedup row (regression guard).

`apps/api/handlers/delivery_provider_test.go` (extend `delivery_webhook_hmac_test.go` harness):
- `TestDeliveryWebhook_ReplayNoReenqueue` — deliver twice (same body); `delivered_at` stamped once (no
  drift), **only one `provider.delivery.updated` outbox row**, second response `duplicate`. **Genuinely
  RED** (today re-stamps + re-enqueues every call).
- `TestDeliveryWebhook_TransientErrorReleasesClaim` — first POST for an external id with **no delivery
  row yet** → transient not-found → 500 + claim released; then create the row + re-POST the **same body**
  → claim gone → re-processes → 200 + status updated. Proves transient release.
- `TestDeliveryWebhook_PermanentErrorAcks` — unmapped status → **200 `ignored`**, claim **kept**, no
  retry storm (guards against the finding-2 poison loop).

## Verification
`go build ./...`, `go vet ./...`, `go test ./apps/api/services/ ./apps/api/handlers/ -count=1`, then `go test ./...`.
Money-path gates: **plan-check before code, independent verify after**. RED-first proven for each new test.

## Out of scope → follow-ups
- **Stripe webhook dedup** — same helper, `event.ID` already parsed; dormant gateway (India=Razorpay),
  so deferred to keep this diff focused on the escrow-relevant paths. Fast-follow.
- **`processed_events` pruning/retention** for webhook rows (table grows unbounded at webhook volume —
  low). Fast-follow ticket.
- **Full tx-atomic delivery claim** (thread a tx through `HandleProviderWebhook`) — finding 3; the narrow
  window above is provider-retry-recoverable, so deferred.
- The other #462 leftovers (partial-amount reverse, queue Amount gross→net, wallet-only reconciliation)
  touch `payout_release.go`/`order_payout.go` — deferred until PRs #530/#532 merge to avoid conflict.
