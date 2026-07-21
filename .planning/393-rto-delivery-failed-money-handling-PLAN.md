# Plan — #393 RTO / delivery-failed money handling

Part of the vendor-payout epic (#403). Goal: no reachable `DeliveryStatus` (or chef
self-delivery outcome) leaves the linked order/day in a non-terminal state with
unresolved money. **Escrow conservation must hold in every branch: captured = refunded
+ released + retained.** Do NOT re-attempt the #395·6 oversell trap (capacity release on
failure — see `project_capacity_release_on_failed_unsafe`).

## Owner decisions (locked 2026-07-06)
1. **Customer-fault terminal RTO** (confirmed address, customer not home after attempts):
   **NO customer refund; vendor hold released in FULL; delivery fee retained.** (Escrow:
   refund=0, hold→released, fee retained.)
2. **Attempt cap = 2**: two `failed` attempts, then terminal `returned`.
3. **Platform/chef/driver-fault** terminal failure: full customer refund + reverse the
   vendor hold; chef may still be compensated for prepared food via the admin
   dispute/payout queue (admin decision, not auto).
4. **Fault attribution = HYBRID (driver reason + admin confirm):** the driver/chef picks a
   structured failure reason code at mark-failed time that SUGGESTS a fault class, but the
   money outcome only executes after an **admin confirms** in the payout/dispute queue
   (pre-filled from the reported reason). Human-in-the-loop on every refund. Matches the
   epic's manual-first launch stance.
5. **Chef self-delivery is in scope** too (owner add 2026-07-06): a chef on
   `FulfillmentChefDelivery` (customer-OTP PoD) who fails to deliver must go through the
   same terminalization + admin-confirm money resolution — NOT only the driver/3PL path.
6. Ambiguous fault → admin queue, never auto-resolve.

## Current state (grounding)
- `handlers/delivery.go:596-740` — the driver `UpdateDeliveryStatus` switch has NO case
  for `failed`/`returned`: the row is saved but the order is untouched, no refund, hold
  neither released nor reversed → **order stranded in limbo, customer charged**.
- Transition table (`delivery.go:565-573`) allows `at_dropoff → failed` and
  `failed → returned` only. No retry-back-to-dispatch path exists yet.
- Delivery model ALREADY has retry/fault scaffolding: `AttemptNumber int (default 1)`,
  `MaxAttempts int (default 3 — owner wants 2)`, `FailureReason string` (free text today:
  customer_unavailable/wrong_address/refused). No structured fault-class field.
- `Order` status enum (`models/order.go:32-45`): pending/accepted/preparing/ready/
  picked_up/delivering/delivered/cancelled/rejected/refunded — **no terminal
  delivery-failed/returned order status**; need one (or a pending-resolution flag).
- Shadowfax webhook (`services/shadowfax_webhook.go`) only handles delivered;
  `DeliveryCancelled` just stamps `cancelled_at` — no money handling, no RTO event.
- Meal-plan: `allDaysTerminal` (`services/meal_plan_fulfillment.go:248-263`) gates plan
  completion; a failed/returned day must reach a terminal day status + route the day's
  escrow per the same policy.
- Chef self-delivery marks delivered via the chef order-status path (OTP PoD, restricted
  to `FulfillmentChefDelivery`); a corresponding chef "could not deliver" path is missing.
- Repo runs `DB.AutoMigrate` (`database/database.go:145`) — nullable column adds are
  feasible in-repo, BUT prod schema changes for this epic have been treated as
  tesserix-k8s-gated (unique indexes / type changes). **New columns here are nullable adds
  → confirm whether they need a tesserix-k8s migration or AutoMigrate suffices before
  relying on them in prod.** (Open dependency.)

## Money-policy matrix (the invariant to encode)
| Fault class (admin-confirmed) | Customer refund | Vendor hold | Delivery fee | Chef comp |
|---|---|---|---|---|
| Customer-fault | none | **release full** | retained | n/a (already paid) |
| Platform/driver-fault | **full** | **reverse** | refunded as part of full | admin goodwill via queue |
| Chef-fault | **full** | **reverse (clawback)** | refunded | none |
| Ambiguous | frozen | frozen | frozen | admin decides |

Every branch must satisfy captured = refunded + released + retained.

## Slice breakdown (each = one PR, RED-first TDD + independent adversarial verify)

### Slice 1 — Terminalization (driver path), NO money movement — SHIPPED (PR #580)
- Structured `DeliveryFailureReason` enum + `SuggestedFaultClass` mapping (pure, tested).
- `failed`/`returned`: open a pending `delivery_failed` OrderIssue + **freeze** the payout
  hold to `disputed` via new guarded `SetOrderHoldDisputed` (reuses the #458 dispute
  machinery — no new columns, no new order status, no migration). Emit `delivery.failed`
  once. Regular gateway orders only (meal-plan/group → later slice, returns froze=false).
- **No refund / release / reverse** — that's Slice 3 (admin confirm).
- **Retry / 2-attempt cap DEFERRED → #579.** `Delivery.OrderID` is a hard uniqueIndex
  (one Delivery per order, no soft-delete) so the "back to ready + new row" retry would
  500 on re-accept and strand the order UNFROZEN. Adversarial verify caught this → slice 1
  ALWAYS terminalizes (never strands); #579 tracks the retry redesign (reuse the Delivery
  row + AttemptNumber). Owner's cap=2 lands with #579.

### Slice 2 — Chef self-delivery failure (parity with Slice 1)
- Add a chef "could not deliver" action on the chef order-status path (mirror OTP-delivered
  path), same structured reason + terminalization + freeze-hold + admin item + notifs.
- Chef-fault reason codes default the suggested class to chef-fault.

### Slice 3 — Admin-confirm money execution (backend)
- Extend the admin payout/dispute queue: a terminal-pending-resolution item shows the
  reported reason + suggested fault; admin confirms a fault class → execute the policy
  atomically: customer-fault → release hold (reuse release path) + no refund; platform/
  chef-fault → full refund (reuse the atomic refund claim path) + reverse hold (reuse
  `WithholdOrReverseOrderHoldForRefund`). Race-safe flip-before-gateway. Reuse #574
  idempotency keys on the gateway calls (refund is guarded by the claim).
- Escrow-conservation tests per branch.

### Slice 4 — Meal-plan day failure terminalization
- A failed/returned meal-plan day → terminal day status so `allDaysTerminal` completes the
  plan; route the day's escrow (release vs reverse) via the same admin-confirm policy.

### Slice 5 — Shadowfax RTO webhook
- Handle Shadowfax failure/RTO events → same terminalization + admin-resolution path as
  Slice 1 (do not auto-move money; land in the admin queue).

### Slice 6 — tesserix-home admin UI (separate repo)
- Surface the failed-delivery resolution items in the payout/dispute queue with the
  reported reason + fault-confirm control. Depends on Slice 3's backend contract.

## Dependencies / open items
- **Migration:** new order status value + any new columns (fault class, pending-resolution
  flag, resolution audit) — confirm AutoMigrate vs tesserix-k8s migration.
- **Admin UI** in tesserix-home (Slice 6) — separate PR/repo.
- Notifications templates (customer + vendor) for failed delivery with money outcome.

## Sequencing
Slice 1 (safe, no money, fixes limbo) → Slice 2 (chef parity) → Slice 3 (admin money
execution) → Slice 4 (meal-plan) → Slice 5 (Shadowfax) → Slice 6 (UI). Slices 1/2 can land
before 3 because they only freeze money (never move it), so they are strict improvements.
