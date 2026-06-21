# Flexible Fulfillment: Pickup · Chef Self-Delivery · 3PL

**Date:** 2026-06-21
**Status:** Approved (design)
**Branch:** `feat/flexible-fulfillment`

## Problem

Today every order assumes **3PL delivery** (Shadowfax-class). We just shipped customer↔chef address **masking** (PR #320, deployed `main-3f63e9c`) to stop the two sides arranging delivery off-platform and bypassing the app.

The owner wants two more fulfillment options:
- **Customer pickup** — the customer collects from the chef. No delivery fee.
- **Chef self-delivery** — the chef delivers themselves. Free or a minimal, distance-based fee.

These modes **legitimately require the address** (a customer picking up needs the chef's address; a chef delivering needs the customer's). So blanket masking is wrong — visibility must depend on the fulfillment mode. Masking is **kept only for 3PL**, where neither side has a reason to see the other.

## Goals

- Add a per-order **fulfillment mode**: `delivery` (3PL, default) · `chef_delivery` · `pickup`.
- Let each chef opt into offering pickup and/or self-delivery, and price self-delivery.
- Make address visibility **mode-aware** (turn the shipped masking conditional).
- Capture real **chef coordinates** (prerequisite for pickup + distance pricing).

## Non-goals

- Masked calling between parties — tracked separately ([issue #321](https://github.com/tesserix/Home-Chef-App/issues/321)).
- Changing the 3PL/Shadowfax integration itself.
- A manual map-pin picker for chef location (we geocode the existing address instead).

## Design

### 1. Fulfillment mode on the order

`Order.FulfillmentType` (`varchar`, default `delivery`): one of `delivery`, `chef_delivery`, `pickup`. Added via migration + model field + `OrderResponse`. Set at order creation from the customer's checkout choice; validated against what the chef offers.

### 2. Per-chef offering + self-delivery pricing

On `ChefProfile` (migration + model):
- `OffersPickup bool` (default false)
- `OffersSelfDelivery bool` (default false)
- Self-delivery pricing fields: `SelfDeliveryBaseFee`, `SelfDeliveryFreeRadiusKm`, `SelfDeliveryPerKm`, `SelfDeliveryMaxFee` (all float, default 0).

3PL delivery remains the always-available baseline (no toggle).

Fee formula (pure, unit-tested):
```
selfDeliveryFee = baseFee + max(0, distanceKm − freeRadiusKm) × perKm
                  then clamp to maxFee when maxFee > 0
```
- all zeros → free
- baseFee only → flat minimal fee
- freeRadius + perKm → distance-based
- Pickup fee is always 0 (not computed).

`distanceKm` = `haversineDistance(chef.lat, chef.lng, dropLat, dropLng)` (helper already exists in `services/`).

### 3. Address visibility — mode-aware (conditional masking)

| Mode | Customer sees of chef | Chef sees of customer |
|---|---|---|
| `pickup` | **exact** address + map pin (to collect) | area only — no delivery, no phone |
| `chef_delivery` | chef **area** (fuzzed circle) | **full** address **+ phone** (chef delivers; needs door contact) |
| `delivery` (3PL) | chef **area** | area only — **masked (as shipped)**, rider gets precise data |

Implementation: the chef-coordinate fuzzing (`FuzzCoordinate`) and `Order.ToChefResponse()` redaction from PR #320 become **conditional on `FulfillmentType`**:
- `TrackOrder` / chef detail endpoints return exact vs fuzzed chef coords based on mode.
- `ToChefResponse()` returns full address + phone for `chef_delivery`, area-only for `delivery`, area-only (no contact) for `pickup`.
- Customer-facing chef browse/list stays fuzzed (mode is per-order, not per-browse).

### 4. Status & dispatch per mode

- `delivery` → 3PL auto-dispatch on `ready` (unchanged: `EnqueueDeliveryDispatch`).
- `chef_delivery` → **no** 3PL dispatch; chef advances through the existing statuses `ready → picked_up` (rendered as "Out for delivery" by the chef) `→ delivered` from the vendor app. No new status enum values.
- `pickup` → **no** delivery; `ready` means "ready to collect"; completion is `ready → delivered` (customer collected). No driver, no rider pin.

The dispatch trigger (`UpdateOrderStatus` on `ready`) branches on `FulfillmentType`: only `delivery` enqueues a 3PL dispatch.

### 5. Chef location capture (prerequisite)

Pickup and distance pricing need real chef coords; today they are `0,0`. Approach: **geocode the chef's existing profile address → lat/lng** on profile save, plus a one-time backfill of existing chefs. Reuse the same geocoder already used for customer-address capture at checkout. No manual map picker.

### 6. Checkout & vendor UX

- **Customer checkout**: a fulfillment selector showing only the modes the chef offers (always at least 3PL delivery), with the computed fee per mode (₹0 for pickup; formula for self-delivery; 3PL quote for delivery). Pickup shows the chef's address + a "collect from" note.
- **Vendor profile**: toggles for pickup / self-delivery + the self-delivery pricing fields.
- **Vendor order detail**: for `chef_delivery`, show the customer's full address + phone + a "you're delivering this" affordance and the self-delivery status transitions; for `pickup`, show "customer will collect"; for `delivery`, the area-only view shipped in #320.

## Phasing

- **Phase 1 — Pickup + conditional masking + chef-coords capture.** Fulfillment-type plumbing (migration, model, checkout selector with pickup, order creation), chef `OffersPickup` toggle, chef-coordinate geocoding/backfill, make #320 masking conditional (pickup reveals chef address; 3PL stays masked), pickup status flow (no dispatch). Smallest slice that stands up the foundation.
- **Phase 2 — Chef self-delivery.** `OffersSelfDelivery` + pricing fields, the fee formula at checkout, `chef_delivery` visibility (customer address + phone to chef), vendor self-delivery status flow.

Each phase is its own PR off `main`, deployed via the Kargo/GAR-mirror flow; mobile apps hot-reload (chef-coords geocoding and order fields need an API deploy).

## Testing

- **Unit (Go):** self-delivery fee formula (free / flat / distance / cap / zero-coords), fulfillment-type validation against chef offerings, `ToChefResponse` visibility per mode, geocoding fallback when address is ungeocodable.
- **Handler (Go):** order creation per mode sets the right fee + dispatch path; `TrackOrder` returns exact vs fuzzed chef coords per mode.
- **Frontend (tsc + manual sim):** checkout selector shows only offered modes with correct fees; vendor order detail renders the right visibility per mode.

## Risks / edge cases

- **Chef has no geocodable address** → pickup/self-delivery can't be offered for that chef; fall back to 3PL only, and surface a "add your kitchen location" prompt in the vendor profile.
- **Self-delivery contact** reintroduces chef→customer phone visibility (intentional, scoped to `chef_delivery`); masked-calling (#321) can replace it later.
- **Mode chosen at checkout but chef later disables it** — validate at order-creation time; existing orders keep their mode.
- **Distance pricing with stale coords** — fee is frozen on the order at creation (like tax), so later coord/profile edits don't change a placed order.
