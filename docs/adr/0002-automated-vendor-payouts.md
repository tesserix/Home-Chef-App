# ADR 0002 — Automated Vendor Payouts

- **Status:** Superseded in part (2026-07-23) — see Superseded below
- **Date:** 2026-07-22
- **Deciders:** Mahesh Sangawar (product/eng)

## Superseded (2026-07-23)

Confirming Razorpay Route as the rail reversed three decisions in this ADR. The
originals are left in place below — an ADR records what was decided and why it
changed, it is not edited into looking correct. The replacement is
[docs/superpowers/specs/2026-07-22-payout-release-governor-design.md](../superpowers/specs/2026-07-22-payout-release-governor-design.md).

| Decision here | Superseded by | Why |
|---|---|---|
| Rail: RazorpayX push payouts | Razorpay Route | Route is already implemented and live (transfers held at checkout). It never custodies chef funds and the platform fee simply stays unremitted. RazorpayX would have added a current account and custody. |
| Cadence: daily batch, 24h maturation | 15-minute sweep, 2h maturation | Our hold and Razorpay's fixed 2-working-day linked-account settlement are sequential; a 24h hold pushed the chef to ~3 days. 2h keeps the total inside two days while preserving a late-refund window. |
| Recovery: netted across a daily batch | deducted from the next order's transfer at creation | Route transfers are per-payment, so a debt cannot be netted after the fact — it is taken off the next transfer before it is built. |

The batch builder (PR #762) is retired accordingly. The payee-agnostic core
(money, ledger, rules, payee registry — #754) is kept; only the batch-assembly
step is replaced by a per-order release decision.
- **Extends:** [ADR 0001](0001-vendor-payout-settlement-model.md) — vendor payout & settlement model
- **Epic:** [#736](https://github.com/tesserix/Home-Chef-App/issues/736) — automated vendor payouts
- **Resolves:** [#737](https://github.com/tesserix/Home-Chef-App/issues/737)
- **Supersedes:** [#30](https://github.com/tesserix/Home-Chef-App/issues/30) (scope moves to [#742](https://github.com/tesserix/Home-Chef-App/issues/742))

## Context

ADR 0001 settled **how much** a vendor is owed: escrow-first capture, dual confirmation, admin-approved release, 6% commission + 18% GST on commission − 1% TDS, with weekly statements as the authoritative payable ledger. That control plane is code-complete and flag-gated (#403).

It did **not** settle how the money physically leaves the platform account. Today it does not: disbursement is a human typing a bank reference into `POST /admin/statements/:id/mark-paid`, and `handlers/chef_statements.go:51` says so in as many words. `#30` held the placeholder for automating this and never grew past three lines.

Three facts shape the decision.

**The vendor has no verified destination.** Payout setup is optional and unenforced. `handlers/chefs.go:2063 SavePayoutDetails` is a Settings action; onboarding explicitly defers it (`StepPolicies.tsx:140`, `StepDocuments.tsx:106` — *"optional, can add later"*); mobile-vendor's seven onboarding screens have no payout step; and no server-side check gates approval, go-live or order acceptance on it. Drivers compute `payoutSet` (`handlers/driver_onboarding.go:47`) but only report it (`:105`). A chef can therefore take customer money and accrue `released` holds with nowhere to send their share.

**Route is the wrong rail for this.** Razorpay Route settles to a linked sub-merchant account and cannot pay an arbitrary bank account or UPI handle. ADR 0001 already moved us off payment-linked Route transfers at checkout. Paying out to a vendor's *nominated* method — bank preferred, UPI permitted — needs RazorpayX payouts against fund accounts, which in turn needs a RazorpayX current account and a resolved Indian entity (#31). That prerequisite is external and unscheduled.

**The durability substrate already exists.** Temporal is live (`go.temporal.io/sdk v1.46.0`, a worker binary at `apps/api/cmd/worker/main.go`, 19 crons already running as Schedules via `services/cron_temporal.go:25`), `TaskQueuePayouts` is declared and unused (`temporal/queues.go:7`), the transactional outbox is done (`services/outbox.go`), and `payments.hold_released` is already published with no consumer (`services/nats.go:52-54`). Building payouts durably is mostly wiring, not construction.

## Decision

Build a **payee-agnostic payout engine** in a new domain-free package `apps/api/payouts/`, fed by the existing hold-release events, orchestrated by Temporal, and executed through a provider interface.

### 1. Rail — provider-agnostic, two adapters

A `PayoutProvider` interface (`CreatePayout`, `GetPayout`, `GetPayoutByReference`, `ValidateFundAccount`, `GetBalance`) with two implementations:

- **`manual`** — emits a bank-upload CSV; an admin returns UTRs. **Ships enabled.**
- **`razorpayx`** — contacts, fund accounts (`bank_account` and `vpa`), payouts.

Adapter choice is configuration. This is the load-bearing decision of this ADR: it decouples the entire engine from #31. Every guardrail, the ledger, penalties, reconciliation and analytics are built, tested and *operated* against the manual adapter, and switching to RazorpayX is a config change on a system already proven in production.

**Rejected:** RazorpayX-only (blocks all work on an unscheduled external dependency); Route linked accounts (cannot pay arbitrary bank/UPI, no penny-drop control, contradicts ADR 0001).

### 2. Cadence — daily batch, 24h maturation

A released hold becomes sweepable at `released_at + payout_maturation_hours` (default **24**). A daily sweep batches all matured credits per payee into one payout.

The maturation window is not a formality. It is the interval during which a late refund, a disputed order or a fraud signal can still cancel a ledger credit **before** the money is irreversible. Paying instantly on release would convert every one of those into a clawback against a vendor who has already been paid.

Weekly statements survive as the tax and accounting artefact — TDS certificates, GST invoices, vendor records. They stop being the payment trigger.

**Rejected:** automating weekly disbursement only (simplest, but leaves vendors waiting up to seven days for money the platform is already holding); vendor-selectable cadence (a good retention lever, deferred until the daily path is proven).

### 3. Approval — a runtime mode ladder

One `PlatformSettings` key, three modes, plus a global kill switch:

| Mode | Behaviour |
| --- | --- |
| `manual` | Every batch waits for an admin decision. **Launch value.** |
| `risk_tiered` | Auto-pay when all guardrails are green **and** amount ≤ cap **and** payee risk is clean; everything else queues. |
| `auto` | Everything guardrail-green pays; admin intervenes only to freeze. |

Promotion is a runtime change, no deploy, audit-logged — consistent with the `payout_auto_approve_after_hours` pattern ADR 0001 established. We launch on `manual` because the first weeks are when the guardrails are least trusted, and dial up as reconciliation proves clean.

### 4. Payout destination — mandatory, verified, cooled-down

- **Collection is mandatory at onboarding** (a wizard step, not a Settings afterthought). Conversion while the vendor is motivated is far higher than chasing them later.
- **Verification is asynchronous** — penny-drop fund-account validation plus a name match against the PAN/KYC name. Approval is **never** blocked on verification, which can take hours and can fail benignly.
- **`accepting_orders` is gated on a verified method.** A vendor may not take a customer's money until there is a verified destination for their share.
- **A new or edited method pays out nothing for `payout_method_cooldown_hours` (default 48)**, and every change fires an all-channel notification with a report-fraud link. Account takeover is the dominant fraud vector in vendor payouts and it is a fast attack; the cooldown is what makes it survivable.
- Existing approved-but-unpayable chefs get a banner, a nag and a grace window. They are **not** silently switched off.

Sensitive fields stay where they already are — GCP Secret Manager, via `services.StoreVendorSecret` — with only metadata (type, status, masked last-4, provider fund-account id, verification result, cooldown timestamps) in Postgres.

### 5. Guardrails

A payout may only include a hold that was **placed, paid, accepted, fulfilled, delivered by an allowed path, dual-confirmed, dispute-free, refund-free, released with an audit row, and matured** — re-evaluated at batch-build **and again at execution**, plus a drift guard that recomputes `ComputeOrderEarnings().NetPayout` and refuses on mismatch. Batch-level rules cover payee standing, FSSAI validity, method verification and cooldown, caps, and provider balance.

Failing a **batch** rule queues for manual review. Failing a **hold** rule leaves that hold in the ledger for the next sweep with a recorded reason. **Money is never dropped, only deferred.**

#### The settlement-timing constraint

Razorpay PG captures land in a gateway balance that settles to the bank on **T+2**. RazorpayX pays out from a current account. Payout capacity is therefore bounded by settlement timing, **not** by what has been captured. The engine performs a pre-flight balance check and **pauses the whole batch rather than partially paying it**. Recorded here because it is the most common way payout systems fail in production, and because it means the platform must carry working capital to pay vendors daily.

### 6. Penalties, clawbacks and recovery

There is no penalty subsystem today — no `penalty`, `sla` or `strike` model exists anywhere in the codebase. This ADR establishes one as ledger **debits**: vendor-fault cancellation, no-show/late SLA, misrepresentation (food not as described, short quantity, misleading photos), food-safety, post-payout refund clawback, and manual adjustment.

Three constraints, all deliberate:

- **Every penalty is appealable** — evidence, rule id, and a 7-day window before it finalises, with contested penalties routed to the existing arbitration surface. A system that silently deducts from a vendor's earnings without recourse will cost us chefs faster than the penalties recover.
- **Penalties are capped** — per-order at that order's net payout, per-cycle at `penalty_cycle_cap_percent` (default 30%). No vendor is zeroed out without a human deciding to.
- **Negative balances carry forward** as `recovery_balance`, blocking payouts and flagging manual review. The engine never emits a negative payout. Recovery beyond carry-forward is a commercial matter, not an engineering one.

A rolling reserve (`payout_reserve_percent`) exists as a knob but is **0 at launch**. Withholding a slice of every vendor's earnings against hypothetical future clawbacks is a real cost to them and is not justified until the clawback rate is measured.

### 7. Durability

`PayoutSweepWorkflow` as a Temporal Schedule; `VendorPayoutWorkflow` per `(payee, business_date)` with workflow ID `homechef:payouts:<payeeType>:<payeeID>:<date>`, so Temporal's workflow-ID dedupe *is* the idempotency guarantee. Admin approval arrives as a signal. Webhooks persist raw then signal, never process inline. Events publish through the existing outbox in the same transaction as the ledger write.

**On a `CreatePayout` timeout the engine never blind-retries** — it calls `GetPayoutByReference` on the idempotency key first, then decides. Double-paying a vendor is the highest-consequence failure this system can produce.

### 8. Reusability

The core is domain-free and keyed by `(payee_type, payee_id)`, with domain logic entering through a single `PayeeAdapter` interface. An **import-boundary test** fails the build if the package imports a domain package. Delivery partners — who already carry an identical bank/UPI/`RazorpayAccountID` shape at `models/delivery.go:107-136` — become the second adapter and thereby the proof that the abstraction is real.

Extraction to `go-shared/payouts` is **gated on a second product needing it** (#749). Extracting a financial abstraction before a second consumer exists reliably produces the wrong abstraction, and would mean versioning a shared library against invariants that are still settling.

### 9. Money representation

The payout core uses **integer minor units** (`amount_minor int64` + `currency`) and never touches `float64`, converting at the boundary until #396 migrates the rest of the codebase. Every ledger, method and batch row carries `tenant_id` from the first migration. Both are cheap now and expensive to retrofit onto financial tables.

## Consequences

**Positive.** Vendors are paid within roughly a day of confirmed delivery instead of up to a week. The platform gains a single auditable ledger for every rupee owed and paid. No work is blocked on the Indian entity. The engine is proven on a real rail before real automation. A second payee type costs one adapter.

**Negative.** Daily payouts require working capital, because PG settlement is T+2 and the platform must front the gap. Mandatory payout collection adds an onboarding step and will cost some signup conversion — measured, not assumed (#739). The manual adapter means genuine ops effort until RazorpayX lands. A penalty system with appeals is materially more work than one without, and is worth it.

**Invariant.** Platform-wide, at all times:

```
captured == refunded + paid_out + platform_retained + held + reserve
```

enforced by property tests in the style of `meal_plan_escrow_fees_test.go`.

## Implementation

Sub-issues #737–#749 under epic #736. #738 (core), #739 (mandatory setup), #741 (batch builder), #744 (guardrails) and #745 (penalties) are all shippable before RazorpayX exists.

Schema DDL goes in `tesserix-k8s/charts/apps/db-schema-bootstrap/schemas/homechef/homechef/` — never as `.sql` in this repo. In-repo `apps/api/migrations/*.sql` is dead (`database/database.go:344-349`); live DDL is GORM `AutoMigrate` plus the idempotent `postMigrate` block.

## Open items

- [ ] **Self-delivery fee ownership.** `services/earnings.go:126` excludes the delivery fee from chef gross because it is "the driver's money". For `chef_delivery` orders the chef *is* the driver, and #699/#703 now price that fee. Unless resolved, the engine will systematically underpay self-delivering chefs. **Blocks #741.**
- [ ] **TCS under CGST §52** — carried forward unresolved from ADR 0001; still not modelled. Confirm applicability and rate with the CA before any live payout.
- [ ] **RazorpayX + Indian entity** — owner and date (#31). Does not block engineering; does bound when `razorpayx` can replace `manual`.
- [ ] **Working-capital float** — confirm the platform can fund daily payouts against T+2 settlement, and set the minimum balance that trips the pre-flight pause.
- [ ] **Penalty amounts** — the debit catalogue defines *types*; per-type amounts and the 30% cycle cap need commercial sign-off.
- [ ] **Onboarding conversion impact** of mandatory payout collection — baseline before #739 ships, compare after.
