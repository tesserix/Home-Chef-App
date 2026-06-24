# Shadowfax 3PL delivery integration — design

**Date:** 2026-06-25
**Status:** Approved (brainstorm). Ready for implementation plan.

## Problem

"Hand to a rider" already routes a delivery order through a provider-agnostic
pipeline (Temporal `DeliveryWorkflow` → `DispatchOrderDelivery` → `ProviderService`),
but no real 3PL is wired, so every such order falls back to a deterministic
**mock** booking and never moves. The Shadowfax adapter (`shadowfax_client.go`)
was written swap-ready against a *guessed* hyperlocal contract. The owner has now
provided sandbox credentials + the real **Shadowfax Unified (Forward) API** docs,
and the guessed contract is materially wrong. This workstream wires the real
Shadowfax sandbox end-to-end so the owner can test, gated dark until activated.

**Milestone:** sandbox-verified and ready to flip to prod (prod creds + go-live
are a later flip). No real money/riders in this milestone.

## The real Shadowfax contract (Unified/Forward API)

Source: https://sfxunifiedapi.docs.apiary.io/ (Forward/Unified; reverse-pickup not needed).

- **Base URLs:** staging `https://dale.staging.shadowfax.in/api/`, prod `https://dale.shadowfax.in/api/`.
- **Auth:** header `Authorization: Token <token_id>` (token in GCP Secret Manager `shadowfax-api-token`, project `tesseracthub-480811`, sandbox).
- **Serviceability (gate, no price):** `GET /v1/clients/serviceability/?service={service}&pincodes={csv}` — returns the serviceable pincodes for a service (`customer_delivery`, `seller_pickup`, …). It is a yes/no **pincode** check; it returns **no delivery price**.
- **Create order:** `POST /v3/clients/orders/` with `order_type:"marketplace"`. `order_details` (`client_order_id`, optional `awb_number` — **auto-assigned if omitted**, `actual_weight` g, `product_value`, `cod_amount`, `payment_mode` `Prepaid|COD`), `customer_details` (drop = customer: name/contact/address/city/state/pincode/lat/lng), `pickup_details` (chef), `rts_details` (return-to-source = chef), `product_details[]` (≥1 sku). Response returns the assigned **`awb_number`** + `customer_track_url`.
- **Cancel / update:** `POST /v3/clients/order_update/` (awb + cancel action).
- **Track:** `GET /v4/clients/orders/{awb_number}/track/` → `status`, `status_display`, addresses, **`customer_track_url`** (Shadowfax-hosted live page). **No raw rider GPS.**
- **Push Callback (webhook):** fields `order_id` (= our `client_order_id`), `status`, `event`, `current_location` (a **hub-name string**, e.g. `"CHN_Ambattur_EXP"` — not coordinates), `event_timestamp`, `comments`, `otp_verified`. Configured via the Shadowfax Client Portal "Webhook" tab; optional auth-key header.
- **Statuses:** `new, ofp (out for pickup), picked, assigned_for_delivery, ofd (out for delivery), delivered, cancelled_by_seller, cancelled_by_customer, rts*, lost, on_hold, …` (hub/parcel vocabulary; we map the on-demand subset).

### Three capability gaps (drove the design)

1. **No live rider GPS.** Track + webhook expose status + a hub-name string + Shadowfax's hosted `customer_track_url` — never rider coordinates. → We do **not** plot a 3PL rider on our own map.
2. **No delivery price quote.** Serviceability is a pincode yes/no; there is no rate endpoint (Shadowfax bills Fe3dr per commercial contract). → Customer delivery fee stays the **flat configured fee**; serviceability is used only as a serviceable/not gate.
3. **Parcel/marketplace flow.** AWB-based, hub statuses. We use the on-demand subset and map statuses to our `Delivery` enum; verify the real sandbox transitions.

## Design decisions

- **Customer tracking** = in-app **status timeline** (driven by webhook events) **+ a "Track live" affordance** that opens Shadowfax's `customer_track_url` in an in-app WebView (same pattern as the Razorpay checkout WebView). Our own live map remains for `chef_delivery` only (chef app supplies GPS).
- **Delivery fee** = existing flat/configured fee. Serviceability gates whether an order *can* go 3PL (unserviceable pincode → cannot hand to a rider; chef keeps it or it stays parked).
- **Chef rider-visibility** = vendor order screen shows 3PL status + "picked up"/"delivered" from webhook events (no rider PII beyond what Shadowfax returns).
- **Activation** = a `shadowfax` `DeliveryProvider` row, base URL = staging, `APIKey` from Secret Manager, `WebhookSecret` set, **`active` flag off until verified**. Flip to prod later by swapping base URL + the prod token version and re-enabling.
- **Privacy** unchanged: 3PL rider leg keeps precise drop coords (the customer's own address); customer↔chef still see approximate area only. See `project_privacy_area_not_address`.

## Architecture (what changes, by unit)

Provider-agnostic wiring (dispatch, Temporal workflow, webhook route, provider
registry, quote/cancel entry points) **stays put** — only Shadowfax-specific
contract details and two UI surfaces change.

1. **`services/shadowfax_client.go` (rewrite to the real contract).** Replace the
   guessed endpoints/payloads with: serviceability (GET pincode gate), create
   order (`/v3/clients/orders/`, marketplace, auto-AWB), cancel (`/v3/clients/order_update/`),
   track (`/v4/clients/orders/{awb}/track/`). `GetQuote` becomes a **serviceability
   check** returning `Serviceable` only (Fee=0, caller keeps flat fee). `CreateTask`
   sends `pickup_details`=chef, `customer_details`=drop, `rts_details`=chef, a
   single synthetic `product_details` sku, `payment_mode:"Prepaid"`, `cod_amount:0`;
   reads back `awb_number` + `customer_track_url`. `CancelTask` posts `order_update`.
   `TrackTask` returns status + `customer_track_url` (no rider lat/lng).

2. **Status mapping (new small unit).** A pure `mapShadowfaxStatus(string) models.DeliveryStatus`
   used by both `TrackTask` and the webhook, unit-tested against the documented
   vocabulary (`ofp/picked → picked-up-ish`, `ofd → out for delivery`, `delivered`,
   `cancelled_* → cancelled`, `rts*/lost/on_hold → failed/exception`).

3. **Webhook (`handlers/delivery_provider.go` `HandleWebhook`).** Parse the Push
   Callback body (`order_id`=client_order_id, `status`, `event`, `event_timestamp`,
   `current_location`, `otp_verified`); HMAC/auth-key verify; resolve the `Delivery`
   by `client_order_id`/awb; update `Delivery.Status` + timestamps + propagate to
   order status; store `customer_track_url` if present. Idempotent on repeat events.

4. **Provider config + secrets.** Seed/enable a `shadowfax` `DeliveryProvider`
   (code, staging base URL, serviceable cities, `APIKey` ← Secret Manager, `WebhookSecret`).
   Token injected via env/ESO; **never committed** (public repo). `active=false`
   until sandbox-verified.

5. **Customer app.** For a 3PL (`delivery`) order: status timeline from `Delivery`
   status + a "Track live" button opening `customer_track_url` (WebView). No 3PL
   map plotting.

6. **Vendor app.** Chef order screen surfaces 3PL status ("Rider assigned" →
   "Picked up" → "Delivered") from the same `Delivery` status.

## Data flow

`Hand to a rider (chef marks ready, carrier=delivery)` → `EnqueueDeliveryDispatch`
→ Temporal `DeliveryWorkflow` (idempotent) → `DispatchOrderDelivery` →
`FindAvailableProvider` (serviceable city/pincode) → `ProviderService.CreateProviderDelivery`
→ `shadowfaxClient.CreateTask` (`POST /v3/clients/orders/`) → persist `Delivery`
(`ExternalDeliveryID`=awb, `customer_track_url`). Shadowfax → `POST /api/webhooks/delivery/shadowfax`
(status events) → `HandleWebhook` → `Delivery`/order status → customer timeline +
chef status. Cancel/refund → `CancelOrderDelivery` → `order_update`.

## Error handling

- Unserviceable pincode / no provider → order left for manual handling (logged,
  **not** an error; chef's status update still succeeds) — existing behaviour.
- Shadowfax non-2xx → surfaced through the Temporal activity retry (booking is the
  flaky external call); after retries exhausted, order parks for manual handling.
- Webhook: unknown `order_id`/awb → 200 + log (don't make Shadowfax retry forever);
  bad signature → 401; duplicate event → idempotent no-op.
- Never expose raw provider errors to end users (`feedback_no_raw_error_codes_to_users`).

## Testing

- **Go unit tests:** `mapShadowfaxStatus` table test; `shadowfaxClient` request
  shaping + response parsing against captured sample payloads (httptest server);
  webhook parse/idempotency test (extend `delivery_webhook_hmac_test.go`).
- **Sandbox (owner-run):** with the staging provider `active`, place a delivery
  order → "Hand to a rider" → verify create returns an AWB + `customer_track_url`,
  serviceability gate works, webhook events flip the customer timeline, cancel
  works. No unit test drives the live sandbox; the owner exercises it.
- `tsc` clean (vendor: except pre-existing `app/reviews.tsx`); `go build`/`go test` green.

## Scope

**In:** adapter rewrite + status mapping; webhook parse→status; provider config +
secret wiring (dark); customer status timeline + Shadowfax track-link WebView;
vendor 3PL status; serviceability gate.

**Deferred:** fee reconciliation / admin charge-vs-collected view (Wave 7E); prod
credentials + city rollout + go-live flip; any reverse-pickup/returns flow.

**Dropped (not feasible with this API):** in-app live-rider map for 3PL; live
checkout delivery-fee quote.

## References

- `project_shadowfax_creds_available`, `project_delivery_3pl_own_fleet_retired`,
  `project_fulfillment_model_settled`, `project_privacy_area_not_address`,
  `reference_kargo_deploy_gar_mirror`.
- Prior art: `services/{shadowfax_client,provider,provider_dispatch,delivery_client}.go`,
  `temporal/workflows/delivery.go`, `handlers/delivery_provider.go`, `models/delivery*.go`.
