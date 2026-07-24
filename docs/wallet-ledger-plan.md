# HomeChef / Fe3dr Wallet → Ledger: Implementation Plan

**Status:** Draft for review · **Principle (non-negotiable):** *HomeChef Wallet is a ledger, not a balance.* PostgreSQL owns financial truth; Temporal owns orchestration; NATS distributes facts.

This plan adapts the target wallet-ledger architecture to the **actual HomeChef codebase** (`apps/api`, Go 1.26 / Gin / GORM / PostgreSQL / Temporal / NATS JetStream). It is grounded in what exists today, calls out the migration risk of a **live money system**, and sequences the work so nothing money-critical is rewritten in a big bang.

**Shipped so far (all behind `LEDGER_SHADOW_ENABLED`, default off — zero behaviour change until the ledger schema is deployed and the flag is on):**
- **Phase 1 · Financial core** — `Money` paise type, double-entry ledger (`ledger_transactions`/`ledger_entries`), posting API + balance projection, dual-write from every `CreditWallet`/`DebitWallet`, opening-balance backfill, and a shadow reconcile cron asserting `ledger == legacy` to the paise.
- **Phase 2 · Buckets** — user value split into provenance buckets (refund/referral/promo/goodwill/cashback) while the customer keeps ONE balance; credits route by source, debits drain in spending priority (promo/cashback first, refund last).
- **Phase 3 · Holds** — generic wallet hold (available/held) with reserve/capture/release, expressed entirely in the ledger, per-user row lock, available-balance guard, idempotent, captured-XOR-released, source-preserving release.

**Customer wallet is also live end-to-end (independent of the shadow ledger):** the balance view is on for all customers (`WALLET_ENABLED`), a home-header balance chip taps through to it, and applying wallet credit at checkout is enabled (`WALLET_CHECKOUT_ENABLED`, API + mobile). Refund credit from cancelled/undelivered meal-plan days lands in the wallet and is spend-on-HomeChef-only (non-withdrawable) — within the pre-regulatory scope of §1.1.

---

## 0. Current state (grounded in code)

| Concern | Today | Target |
|---|---|---|
| Money type | `float64` everywhere (`Wallet.Balance`, `WalletTxn.Amount`, `CreditWallet(amount float64)`, `Round2()`) | **paise `int64`** (`₹250.75 → 25075`), never float |
| Model | Single `Balance` + append-only `wallet_txns` journal (`type/source/amount/balance_after/idempotency_key UNIQUE`) | **Double-entry ledger** (accounts + entries, Σdebit = Σcredit) |
| Buckets | One indistinguishable balance | Separate **REFUND / REFERRAL / PROMO / GOODWILL / CASHBACK** value with per-bucket rules |
| Holds | None generic — order checkout uses `order.WalletApplied`; meal-plan uses per-day payout holds | Generic wallet **HOLD** (available / held), capture / release |
| Allocation | Wallet applied then platform tops up chef/driver (`settleOrderWallet`) | Policy-driven **spending priority** + FEFO within buckets |
| Refund allocation | `CreditWallet` straight to balance (keyed e.g. `mealplan-refund:<dayID>`, `issue:<id>`) | Source-preserving restoration (promo→promo, referral→referral, external→PSP) |
| Reconciliation | Payout reconcile exists; no ledger↔balance↔PSP wallet reconcile | Daily Temporal reconciliation, mismatch → case (never auto-correct) |
| Admin | Adjustments create `wallet_txns` (has `created_by`); no maker/checker | Maker/checker on sensitive adjustments; immutable ledger |
| State machines | Implicit | Explicit refund + credit state machines |

**Already built (reuse, do not rebuild):** transactional outbox (`outbox_events` → `OutboxRelay`, PubAck + `Nats-Msg-Id` dedup + DLQ) → durable JetStream consumers with a `processed_events` inbox; Temporal (`homechef` ns, `OrderSaga`/`ConfirmReceipt`/cron workflows, exactly-once Schedules); idempotency keys (`wallet_txns.idempotency_key UNIQUE`); NATS streams incl. `PAYMENTS`.

**Callers that touch the wallet today** (all must keep working through the migration): meal-plan refunds (`RefundDay`/`refundDayAmount`), order refunds + issue refunds, referral reward (`MaybeGrantReward`), wallet-at-checkout (`order.WalletApplied` + `DebitWallet` + platform top-ups), admin adjustments, the (now-fixed) capture-guarded meal-plan flow.

---

## 1. Decisions to lock BEFORE building (owner input required)

These change the design; do not start Phase 1 code until they're settled.

1. **Regulatory model — the hard gate.** RBI separates *closed-system* instruments (usable only for goods/services from the issuing entity, **no** third-party settlement or cash withdrawal) from regulated PPIs. HomeChef is a **marketplace with independent chefs**, so "HomeChef Wallet" is *not* automatically closed-system. **Get Indian payments counsel** to confirm the merchant-of-record / funds-flow model **before** refund-derived balances become broadly spendable or withdrawable. This gates the withdrawability + refund-UX parts of Phases 4/6/7. Until settled, refund credits stay *spend-on-HomeChef-only, non-withdrawable*.
2. **Money-type migration approach.** Recommend: introduce a `Money` (paise `int64`) type in the **new ledger only**; the legacy float wallet is kept in sync during transition; the ledger is authoritative. No app-wide float→int rewrite up front.
3. **Service shape.** The target diagram shows a `homechef-wallet` microservice. **Recommendation: build the ledger as a bounded package inside the existing `apps/api` monorepo first** (`internal/ledger`, `internal/wallet`) — this matches your own §33 advice ("don't prematurely split ledger/credit/hold services; it makes consistency *harder*"). Extract to a service only if scale demands. Schema DDL goes in **`tesserix-k8s/.../db-schema-bootstrap`** (per repo rule: app repo holds GORM models only, never raw SQL).
4. **Bucket set + spending priority.** Confirm buckets (REFUND/REFERRAL/PROMO/GOODWILL/CASHBACK) and default priority (recommend PROMO → REFERRAL → GOODWILL → REFUND → external, so expiring/non-cash value is spent before refund value). Make it **policy-driven** (a `platform_settings` / `policy` key), not hard-coded.
5. **Hold model.** Confirm generic wallet holds replace the ad-hoc `order.WalletApplied` at checkout (available/held split), reusing the existing per-order idempotency.

---

## 2. Migration strategy (the part the greenfield design omits — and the riskiest)

There are **live wallets with float balances, live history, and live callers.** Never swap the model in place. Use a **strangler-fig** approach:

1. **Build the ledger alongside** the existing wallet (Phase 1). Every `CreditWallet`/`DebitWallet` **dual-writes**: the legacy `wallet_txns`/`Balance` (unchanged) **and** a double-entry ledger posting in paise, in the **same DB tx**.
2. **Backfill** each wallet's current float balance into the ledger as an `OPENING_BALANCE` entry (paise), so the ledger's projected balance equals the legacy balance at cutover-start.
3. **Shadow-reconcile** for a bake period: a check asserts `ledger_projection(user) == legacy_balance(user)` (in paise) on every mutation + a daily sweep. Any drift → alert, do **not** auto-correct.
4. **Flip reads** to the ledger projection behind a flag once reconciliation is clean for N days.
5. **Retire** the float `Balance` column + legacy journal writes (Phase 10), after a full bake.

This keeps the money system correct at every step and lets us abort at any phase without customer impact.

---

## 3. Phased plan (adapted to this repo)

Each phase is independently shippable, gated where noted, and adds automated **financial-invariant tests** (Section 4).

| Phase | Deliverable | Notes / repo shape |
|---|---|---|
| **0 · Gate** | Regulatory review + lock §1 decisions | Blocks spendability/withdrawability in later phases |
| **1 · Financial core** | `Money` (paise int64) type; double-entry ledger (`ledger_accounts`, `ledger_entries`, posting API, balance projection); idempotency; **dual-write + backfill + shadow reconcile** | `apps/api/internal/ledger`; schema in tesserix-k8s; **no behavior change** — runs in shadow |
| **2 · Buckets** | Per-user bucket accounts (REFUND/REFERRAL/PROMO/GOODWILL/CASHBACK); one UI balance + internal breakdown | Ledger accounts, not new tables per bucket |
| **3 · Holds** | Generic wallet hold (available/held), capture/release; migrate order-checkout wallet-apply onto holds | Replaces ad-hoc `order.WalletApplied` path |
| **4 · Refund allocation** | `RefundAllocationEngine` (source-preserving restoration) + refund/credit **state machines** | **Gated (§1.1)** on withdrawability/UX |
| **5 · Temporal** | `WalletPaymentWorkflow` (mixed wallet + UPI/card, hold→external→capture/release compensation), `WalletRefundWorkflow` | Reuse existing `homechef` Temporal setup + task queues |
| **6 · Referral** | Reward policy config, `ReferralRewardWorkflow` (credit only after delivery + cancellation window), fraud risk score + states | Reuse existing referral + `MaybeGrantReward` |
| **7 · Promotions + expiry** | Promo credits, `CreditExpiryWorkflow`, FEFO spend within buckets | Append expiry entries; never delete grants |
| **8 · Reconciliation** | `WalletReconciliationWorkflow` (ledger ↔ balance projection ↔ PSP ↔ order) as a daily Temporal Schedule; mismatch → `OPEN_RECONCILIATION_CASE` | Fits the existing cron-Schedule registry |
| **9 · Admin** | Search, ledger view, maker/checker adjustments (create entries, never edit balance), freeze/unfreeze | tesserix-home admin proxy (existing pattern) |
| **10 · Cutover + hardening** | Retire float `Balance`; invariant/chaos/load tests, negative-balance + concurrency guards | Only after a clean bake |

**Events / outbox (all phases):** reuse the outbox → JetStream path with the versioned subjects (`homechef.wallet.v1.*`) and the standard envelope (`eventId/eventType/eventVersion/aggregateId/occurredAt/correlationId/causationId/data`). Producer at-least-once (outbox) + consumer idempotent (inbox) = effectively-once. **No wallet event is fire-and-forget.**

**Temporal vs NATS (kept strict):** Temporal for *processes with state/retries/timers/compensation* (payment, refund, referral qualification, expiry, reconciliation). NATS for *facts other services react to* (credited, refunded, hold captured/released). Never run a multi-step financial saga purely through loosely-coupled NATS consumers.

---

## 4. Acceptance criteria (automated invariant tests — not docs)

- Retry an operation 100× → credited/debited **exactly once** (idempotency key).
- Two concurrent ₹500 orders vs a ₹500 wallet → **exactly one** reserves (row lock / optimistic version; `available - requested >= 0` enforced in the ledger tx).
- External payment fails → any wallet **hold is eventually released** (Temporal compensation).
- Worker crash mid-flow → payment/refund resumes with **no duplicate money movement**.
- NATS down → ledger tx still commits; event publishes later via outbox.
- Ledger entries are **immutable** (no edit/delete); a reversal posts a **new reversing entry**.
- Σdebit == Σcredit at all times (mismatch = SEV-1 `wallet_ledger_mismatch_total`).
- `₹10.99` is stored as `1099` (paise int), never `10.99` float.
- Shadow-reconcile: `ledger_projection == legacy_balance` throughout the migration.

---

## 5. Observability & guardrails

Metrics: `wallet_credit_total`, `wallet_debit_total`, `wallet_payment_failure_total`, `wallet_stale_hold_total`, `wallet_ledger_mismatch_total` (**SEV-1**), `wallet_negative_balance_total`, `wallet_reconciliation_difference_minor`. Full traceability on every entry (`wallet_id/tenant_id/user_id/order_id/payment_id/refund_id/referral_id/workflow_id/run_id/trace_id/correlation_id/actor/reason_code`). Security: immutable ledger, maker/checker on sensitive adjustments, RLS as defence-in-depth, secrets via External Secrets (existing pattern), no PAN storage.

---

## 6. Checkout & payment UX — hassle-free by default

The ledger + holds architecture isn't only for safety — it's what *lets* the UX be frictionless: because a failed leg can always be released and a drop can always resume, we can default aggressively without risking the customer's money.

1. **Smart default split, zero decisions.** Checkout pre-computes the best split from policy — apply wallet (buckets in priority order, FEFO) up to the order total, remainder on the customer's **last-used** method — and shows **one line + one button**: *"₹300 wallet + ₹550 Google Pay → Pay ₹850"*. Expanding to change it is optional; the default is already right.
2. **Wallet-only = instant, no gateway.** If the wallet covers the full order, there is **no Razorpay round-trip** — capture the hold, mark paid, confirm: one tap, instant. (Today a full-wallet order already bypasses the gateway via `settleFullWalletOrder`; make this a visible, celebrated fast-path.)
3. **Minimize the external-leg friction.** For the remainder, prefer **UPI intent** (deep-link into Google Pay / PhonePe — approve in the UPI app, no card or OTP typing) and **tokenised saved cards** (1-tap, RBI-compliant token, no re-entering PAN/CVV/OTP). Full card entry becomes the fallback, not the default — removing the mobile → card → OTP → save-prompt slog the current Razorpay sheet forces.
4. **Transparent, not noisy.** Show *"₹1,750 available"* and the split up front; the bucket breakdown (refund/referral/promo) and "₹100 expiring in 3 days" nudges live behind a tap. Plain language only — *"Refund — Order #HC8928 +₹550"* — never CR/DR/ledger jargon.
5. **Never strand, never double-charge.** Holds + Temporal make drops safe: a dropped/failed external leg **auto-releases the wallet hold** (the customer never loses wallet money to a failed order), and re-tapping pay **resumes the same payment** (the resume fix shipped this cycle). A worker crash resumes with no duplicate movement. The user just sees a working **Retry** — the money is always consistent underneath, so no "Payment not completed" dead-ends.
6. **Fast, honest states.** Wallet-only confirms instantly; a mixed payment shows a brief, durable *"confirming…"* that survives app-kill (Temporal-backed) and lands on a clear confirmed/failed screen.
7. **Refund UX matches the model.** On cancel: *"Where would you like your ₹750?"* → HomeChef Wallet (instant, for future orders) or original method (bank timing) — wording gated on the regulatory model (§1.1). Wallet refunds land instantly and show in a plain history.

Net: the common path — wallet + one UPI approval, or wallet-only — becomes a **1–2 tap, no-OTP** experience, while the ledger/holds guarantee it stays financially correct.

## 7. Reality check

This is a multi-week, money-critical program, not a single PR. The two dominant risks are (a) migrating a **live float wallet** without drift (Section 2) and (b) the **regulatory gate** (Section 1.1) — both must be handled before any customer-facing withdrawability. The money-flow hardening already shipped this cycle (capture-guarded refunds, durable confirm, hold-outside-tx, gateway-aware expiry, day-terminalize) is compatible groundwork: it already treats the wallet as append-only + idempotent, which is the ledger mindset.

**Recommended immediate first step (Phase 1, shadow-only):** the `Money` paise type + the double-entry ledger + dual-write + backfill + shadow reconcile — zero behavior change, fully de-risked, and it's the foundation every later phase builds on.
