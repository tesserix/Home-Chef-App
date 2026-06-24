# Chef-delivery fulfillment lifecycle — design

**Date:** 2026-06-23
**Status:** Approved (brainstorm). Ready for implementation plan.

## Problem

The order fulfillment model churned repeatedly and chef-delivery is "not streamlined":
- A delivery order to a self-delivering chef auto-routed to chef-delivery with **no way to choose 3PL**.
- `picked_up` is bucketed to History, so a chef-delivery order **vanishes from the active list the moment the chef taps "Out for delivery"** — the chef can't mark it Delivered ("workflow not update").
- Customer/vendor status copy says "awaiting pickup"/"awaiting driver" on chef-delivery orders, which is wrong.
- No clear, single decision point for "who carries this order."

## Model (locked)

**Chef profile = capabilities** (already exist): `Allow pickup`, `I deliver myself` (+ self-delivery range/pricing from #342).

**Customer checkout:** customer picks **Delivery** or **Pickup** only (Pickup shown only when the chef allows it). The customer **never** picks the carrier. (Matches shipped #344 — no customer-facing "Chef delivery" chip.)

**The carrier decision is the chef's, made once at "Mark Ready."**

### Lifecycle by path

| Path | Flow |
|------|------|
| **Pickup** | pending → accept → preparing → **Ready for pickup** → customer collects → chef taps **Handed over** → done |
| **Delivery, self-delivery OFF** | pending → accept → preparing → **Ready** → *Awaiting rider* → 3PL picks up → delivered |
| **Delivery, self-delivery ON** | pending → accept → preparing → **Mark Ready → choose carrier** (below) |

At **Mark Ready** for a self-delivering chef, two actions:
- **"I'll deliver"** → fulfillment becomes `chef_delivery` → full customer address + phone reveal → **Out for delivery** (stays **active**, chef en route) → chef taps **Mark Delivered** → done.
- **"Hand to a rider"** → fulfillment becomes `delivery` (3PL) → *Awaiting rider* → rider picks up → delivered. Address stays **area-only** to the chef.

### Cross-cutting rules

- **Distance (#342):** at Mark Ready, if the drop is beyond the chef's `SelfDeliveryMaxDistanceKm`, show the soft warning and **recommend "Hand to a rider."** Chef can still deliver, hand off, or cancel→refund.
- **Privacy:** the chef sees **area-only + an approximate distance** until they choose "I'll deliver"; only then does the full address + phone reveal (the documented chef_delivery privacy exception — see `project_privacy_area_not_address`).
- **Carrier reversibility:** the chef may switch deliver↔rider **until someone is en route**. Once "Out for delivery" (chef) or rider-picked-up (3PL), the carrier is **locked** (finish or cancel→refund).
- **History bucketing (the bug fix):** `picked_up` must be **fulfillment-aware**:
  - `chef_delivery` + `picked_up` (= chef out delivering) → **ACTIVE** until `delivered`.
  - 3PL `delivery` + `picked_up` (= rider took it, out of chef's hands) → History.
- **No-3PL-yet rule:** a Delivery order for a chef who doesn't self-deliver is **allowed** and parks in *Awaiting rider* until Shadowfax is live. (Owner decision — no enforcement that a chef must offer a working method.)

## Scope

**Build + testable now** (everything except actual 3PL dispatch):
1. Backend: at the ready transition, accept a **carrier choice** (`chef_delivery` vs `delivery`) instead of auto-routing at order creation. Don't reveal the full address until `chef_delivery` is chosen.
2. Backend: make the `picked_up`→History/active bucketing fulfillment-aware (vendor `HISTORY_STATUSES` + any server-side grouping).
3. Vendor **Mark Ready** UI: present the two carrier actions (with the distance warning steering far drops to "Hand to a rider"); the **"Hand to a rider" option is present now** even though dispatch parks until 3PL.
4. Carrier-aware status **wording** (vendor + customer): kill "awaiting pickup"/"awaiting driver" on chef-delivery; chef-delivery reads "Out for delivery" / customer "on the way".
5. Carrier **switch-before-en-route** affordance.

**Deferred to the next workstream (Shadowfax / 3PL):** the actual rider dispatch, quote, tracking, cancellation. The owner will not test the 3PL path until then; the "Hand to a rider" choice merely parks the order in *Awaiting rider*.

## Testing

- Go unit tests for the ready-transition carrier resolution + the fulfillment-aware bucketing.
- Manual sim (chef-delivery path only): place Delivery order → chef accepts → prepares → Mark Ready → "I'll deliver" → address reveals → Out for delivery (stays active) → Mark Delivered. Verify the customer sees carrier-neutral "on the way" wording.
- 3PL path not tested until Shadowfax lands.

## References

- Settled-model memory: `project_fulfillment_model_settled`
- Distance warning (shipped): PR #342
- Customer-choice removal (shipped): PR #344
- 3PL deferral: `project_delivery_3pl_own_fleet_retired`, `project_shadowfax_creds_available`
