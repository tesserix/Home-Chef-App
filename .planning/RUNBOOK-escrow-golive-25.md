# Runbook — Razorpay live switch + escrow flag flip (#25, gated by #218)

**Goal:** take the (already backend-complete) escrow control plane live — switch Razorpay
test→live keys, register the live webhook, and flip the escrow flags so held chef/rider
transfers actually release/reverse. Both escrow flags are OFF today; nothing here moves live
money until Phase 3.

**Owner actions are flagged 🔑 (Razorpay dashboard / secrets / infra). Everything else is
verification.**

---

## 0. Facts this runbook depends on

**Flags (env vars, `apps/api/config/config.go`, all default `false`):**
| Env var | Config field | Gates |
|---|---|---|
| `ORDER_PAYOUT_AUTO_RELEASE_ENABLED` | `OrderPayoutAutoReleaseEnabled` | regular-order Route transfer release/reverse + partial claw-backs (`order_payout.go`, `order_payout_partial.go`) |
| `MEAL_PLAN_ESCROW_ENABLED` | `MealPlanEscrowEnabled` | meal-plan per-day hold/release/reverse (`meal_plan_escrow.go`) |
| `GROUP_ORDERS_ENABLED` | `GroupOrdersEnabled` | group/office order money flow |

**Secrets (env vars, injected via `homechef-api-secrets` → k8s ExternalSecret):**
`RAZORPAY_KEY_ID`, `RAZORPAY_KEY_SECRET`, `RAZORPAY_WEBHOOK_SECRET`.

**Webhook:** `POST /webhooks/razorpay` (public, no auth — HMAC-verified via
`VerifyWebhookSignature`), handler `paymentHandler.RazorpayWebhook`. Public URL
**`https://api.fe3dr.com/webhooks/razorpay`**. Events consumed: `payment.captured`,
`refund.processed`, `transfer.processed`. Webhook wallet-settlement is idempotent and runs
from BOTH the client verify path and `payment.captured` (#395·3), so a dropped client is safe.

**Model:** escrow-first — capture 100% to the platform, chef/rider Route transfers are created
**on-hold** at checkout, held through dual-confirmation, then the flag-gated seam
releases (or reverses) them. Admin surfaces: **Release Queue** (`/admin/apps/homechef/payout-queue`)
and **Delivery Failures** (`/admin/apps/homechef/delivery-failures`, #613).

---

## 1. Pre-flight — do NOT proceed until all true

- [ ] #218 sandbox sign-off complete (Section 2 below).
- [ ] Razorpay **Route** is activated on the live account and the platform balance can fund
      held transfers (Route needs a funded platform balance to settle on release).
- [ ] All chefs/riders who can receive payouts have a **live** linked account
      (`chef_profiles.razorpay_account_id` / delivery partner account) — a release to a missing
      account fails at the gateway. Query for `payout_hold_status='release_eligible'` rows whose
      chef/rider has no linked account BEFORE flipping.
- [ ] `#395·1` unique payment-id indexes applied via `tesserix-k8s/db-schema-bootstrap`
      (dedup-safe apply — a naive unique index on existing dup payment ids crashes the fatal
      `Migrate()` on prod boot). This is the one external migration; confirm it's live first.
- [ ] CA sign-off on the tax model (TCS §52 not yet modeled — #19) if required for live payouts.
- [ ] A named on-call owner for the flip window + a comms channel.

---

## 2. Phase 0 — sandbox sign-off (#218) — in a NON-prod env with TEST keys + flags ON

Do this in dev/staging with `rzp_test_*` keys and the escrow flags **ON** there (never flip
prod yet). Verify against Razorpay Route **test** accounts. For each, confirm the money AND the
DB hold state.

- [ ] **Capture → hold:** place a paid order → Razorpay captures 100% to platform; chef+rider
      Route transfers exist **on-hold**; `payout_hold_status` = `awaiting_customer_confirmation`
      on delivery.
- [ ] **Release:** customer confirms (or the 24h auto-confirm sweep fires) → `release_eligible`
      → admin releases from the Release Queue → held transfers flip to settled; chef/rider paid.
- [ ] **Reverse (refund):** refund a delivered+released order → hold → `reversed`; the Route
      transfer is clawed back to the platform; customer refunded.
- [ ] **Partial refund:** goodwill/issue partial → only the refunded portion clawed from the
      chef transfer; the remainder stays releasable (#549/#586); `refunded_at` NOT stamped.
- [ ] **Meal-plan day:** per-day hold → release-on-delivery → per-day refund/reverse
      (`MEAL_PLAN_ESCROW_ENABLED`).
- [ ] **Group order:** multi-payer charge → one consolidated held chef transfer → release, and
      cancel → per-participant wallet refund + chef transfer reversed (`GROUP_ORDERS_ENABLED`).
- [ ] **Delivery failure:** force a failed delivery → freeze (hold→`disputed`, issue/day/group
      surfaces in **Delivery Failures**) → admin resolves customer-fault (release) and
      platform/chef-fault (refund + withhold) → money matches the ruling.
- [ ] **Conservation:** for each, `captured == released + reversed + retained + refunded`
      (paise-level; the reconcile cron currently tolerates 1 paise).
- [ ] Sign-off recorded on #218; then close #218.

---

## 3. Phase 1 — LIVE cutover (#25)

> Sequence matters: keys+webhook FIRST (payments keep working on live rails with flags still
> OFF = state-only), then flip the flags once a live smoke transaction is clean.

### 3a. Swap keys (flags still OFF)
1. 🔑 In `homechef-api-secrets`, set `RAZORPAY_KEY_ID`/`RAZORPAY_KEY_SECRET` to `rzp_live_*`.
2. 🔑 Roll the API deployment to pick up the new secret (ExternalSecret refresh → pod restart).
3. [ ] Confirm the pods came up healthy (`/health`, `/ready`) and read the live key
       (`RAZORPAY_KEY_ID` starts `rzp_live_`).

### 3b. Register the live webhook
4. 🔑 In the Razorpay **live** dashboard, add webhook **`https://api.fe3dr.com/webhooks/razorpay`**
   for events **`payment.captured`, `refund.processed`, `transfer.processed`** (add
   `payment.failed` only if a handler exists — today the three above are consumed).
5. 🔑 Set the webhook secret to a fresh value and put the SAME value in
   `RAZORPAY_WEBHOOK_SECRET` (`homechef-api-secrets`) → roll pods. (Signature verification
   rejects any payload signed with the wrong secret → 4xx, no state change — safe.)
6. [ ] Fire Razorpay's "test webhook" from the dashboard → expect `2xx` and a signature-verified
       log line; a `signature mismatch` means the secret in the pod ≠ the dashboard secret.

### 3c. Live smoke (flags STILL OFF)
7. [ ] Place a real ₹-small order (e.g. ₹1–₹10) end-to-end on the live app: pay → `payment.captured`
       webhook lands → order `completed` → deliver → hold parks at `awaiting_customer_confirmation`
       → customer confirms → `release_eligible`. With flags OFF the payout does NOT release
       (state-only) — the Release Queue shows it as eligible. Confirm no error.
8. [ ] Refund that ₹-small order from the admin/customer path → customer refunded on the live
       gateway; hold cross-guarded to withheld/reversed (state-only).

### 3d. Flip the escrow flags (money now moves)
> Flip **one at a time**, watch, then the next. Recommended order: orders → meal-plan → group.
9. 🔑 Set `ORDER_PAYOUT_AUTO_RELEASE_ENABLED=true` → roll pods.
10. [ ] Release ONE eligible order payout from the Release Queue → confirm the chef/rider
        actually receive the settled transfer on the live Route account; `payout_settled_at`
        stamped; reconcile shows no drift.
11. 🔑 Set `MEAL_PLAN_ESCROW_ENABLED=true` → roll → verify one day release + one day refund live.
12. 🔑 Set `GROUP_ORDERS_ENABLED=true` (if launching group orders) → roll → verify one group
        release + one cancel-refund live.

### 3e. Close out
13. 🔑 Rotate/disable the old `rzp_test_*` keys and any exposed test webhook secret.
14. [ ] Update `docs/MONEY-FLOW.md` §5 + close #25; note the flip timestamp on #403.

---

## 4. Monitoring (first 48h)

- **Reconcile cron** (`services/payout_reconcile_cron.go`) — watch for `payout_settle_attempts`
  climbing / stranded `released`/`reversed`-but-unsettled rows / cancelled-group ALERTs
  (capped at ~5 attempts then alerts).
- **Delivery-failure reconcile** — stranded failed deliveries getting auto-frozen.
- **Webhook 4xx/5xx rate** on `/webhooks/razorpay` (signature mismatches, parse failures,
  Razorpay retries).
- **Gateway errors** on release/reverse: missing linked account, insufficient platform balance,
  already-reversed (tolerated).
- **Escrow conservation** — spot-check `captured == released + reversed + retained + refunded`
  on a sample of settled orders.
- **Admin queues** — Release Queue age (24h SLA) + Delivery Failures backlog.

---

## 5. Rollback

The seams are flag-gated, so rollback is a flag flip (fast) — but money already moved does NOT
un-move.

- **Bad release behaviour / gateway errors after a flip:** 🔑 set the offending flag back to
  `false` → roll pods. New releases/reverses immediately stop (state-only again). Already-settled
  transfers stay settled; use the admin Reverse action per-order for any that must be clawed back.
- **Webhook broken (signature/parse):** payments still capture; the client verify path is the
  idempotent backstop (#395·3), so completion still happens. Fix the secret and re-fire.
- **Live keys bad:** 🔑 revert the secret to the previous key, roll pods; new payments fail
  fast (503 "razorpay unavailable") — no wrong charges.
- **Full stop:** all three flags `false` returns the system to state-only escrow (today's safe
  posture) with live keys still capturing.

Rollback does NOT require a code deploy — only a secret/ConfigMap change + pod roll.

---

## 6. What's explicitly NOT blocking this flip (deferred)

- `#396` Phase-2 paise migration (float math is fine at rupee level; 1-paise reconcile tolerance).
- `#611` (InitiateRefund partial-aware reserve) — narrow concurrency window, pre-existing.
- `#398`/`#602` escrow drift-ledger reconcile — more meaningful to tighten post-flip.
- `#387` customer-facing confirm-fulfilment mobile UI — the 24h auto-confirm sweep covers release
  meanwhile.
- `#389`/`#30` RazorpayX disbursement — manual weekly payouts are fine interim.

---

## 7. Pre-flip control-plane completeness (already DONE — reference)

- Dual-confirm hold state machine + confirm endpoints + 24h auto-confirm sweep (#387 backend).
- Admin Release Queue: release / withhold / reverse / bulk (#388) + super-admin `SPManagePayouts` gate (#515).
- Refund cross-guards symmetric across every refund path incl. partials (#457/#458/#498/#549/#586/#609).
- Delivery-failure freeze→resolve→reconcile for every order shape/channel (#393/#589/#594) +
  admin Delivery Failures surface (#613).
- 6% commission (frozen per-order/day/group) + 18% GST + 1% TDS economics (#390/#547).
- Gateway idempotency keys on money-moving Razorpay POSTs (#574).
