# ADR 0001 — Vendor Payout & Settlement Model

- **Status:** Accepted (pending finance/CA sign-off on TCS — see Open Items)
- **Date:** 2026-07-03
- **Deciders:** Mahesh Sangawar (product/eng)
- **Resolves:** [#390](https://github.com/tesserix/Home-Chef-App/issues/390) (settlement-model contradiction)
- **Unblocks epic:** [#403](https://github.com/tesserix/Home-Chef-App/issues/403) — Escrow settlement control plane
- **Related:** #387, #388, #389, #391, #400, #30

## Context

The codebase carried **two mutually inconsistent settlement models for regular orders**, both partially wired:

- **Model A — Razorpay Route split at checkout (gross, zero commission):** transfers attached to the Razorpay order at payment creation (`handlers/payment.go:198-213`), chef paid `Subtotal + Tax + ChefTip − ChefFundedDiscount` gross, released on delivery (`services/order_payout.go:47`).
- **Model B — Weekly statements (net of commission + GST + TDS):** `ComputeOrderEarnings` (`services/earnings.go:96-140`) computes `commission = 15% × (subtotal − chefFundedDiscount)`, 18% GST on commission, 1% TDS (§194-O); frozen weekly by `GenerateWeeklyStatements` (`services/statement.go:61-117`).

If both go live, the platform **double-pays** (Route settles gross at delivery while the statement still reports a net payable) and **never collects commission or withholds statutory TDS**. This must be resolved before any money-movement flag (`MEAL_PLAN_ESCROW_ENABLED`, `ORDER_PAYOUT_AUTO_RELEASE_ENABLED`) is enabled (#218 sandbox → #25 live).

## Decision

Adopt **escrow-first settlement** with a dual-confirmation, admin-approved release control plane. Money flow:

> Customer pays → **100% captured to the platform (owner) Razorpay account, held** → order/day fulfilled → **both vendor and customer confirm fulfilment** → **admin approves release** (manual at launch; runtime auto-approve timer later) → funds accrue to the chef's payable ledger → **disbursed weekly**. Disputes/failures freeze or refund deterministically.

Payment-linked Route transfers are **no longer attached at checkout**; vendors are paid from approved releases computed net of commission and TDS. Weekly statements become the **authoritative payable ledger**.

### 1. Economics (per delivered order, on food subtotal)

| Line | Value | Notes |
| --- | --- | --- |
| Commission | **6%** | Runtime-configurable via `PlatformSettings.commission_rate_percent` (default 6). **Flat for all chefs — no premium tier.** |
| GST on commission | **18%**, separate line | Effective platform take ≈ 6% + GST ≈ **7.08%**. GST-registered chefs may claim input credit. |
| TDS (§194-O) | **1% of gross**, withheld | Statutory; platform withholds and deposits. |
| TCS (CGST §52) | **TBD** | Not coded — flagged for finance/CA sign-off (see Open Items). |

`net_payout = gross − (6% commission) − (1% TDS)`; GST invoiced on the commission.

**Why 6% (not 5% or lower):** the platform absorbs the Razorpay gateway fee (~2% + 18% GST ≈ 2.36%) on the *full* captured amount, plus goodwill-refund leakage (order-issue auto-refunds up to the `auto_approve_cap`, default ₹300) and reconciliation/support cost. At 5% the per-order net after gateway is ~2.4%; 6% roughly doubles the refund/ops cushion while remaining radically below Swiggy/Zomato (18–25%). The rate is runtime-tunable, so launch promos (e.g. 0% first month) or per-cohort changes need no deploy.

### 2. Release control plane

- **Customer-confirm window:** 24h (`PlatformSettings.payout_customer_confirm_window_hours`, default 24). A delivered, dispute-free order/day auto-confirms after the window if the customer does not tap "confirm received". Filing an order-issue freezes the hold as `disputed` (#400).
- **Proof of delivery (self-delivery):** **Customer OTP** shown at handoff, entered by the chef, before `MarkMealPlanDayDelivered` / order-delivered counts the vendor half of confirmation. Chef `Ready → Delivered` is **restricted to `FulfillmentChefDelivery`**; courier/3PL orders may only reach `delivered` via the delivery pipeline or provider webhooks. Fixes the #391 HIGH self-attest exploit.
- **Admin approval:** **Manual-first at launch** — admin approves every release from the payout queue (#388), with bulk-release for the daily sweep. Runtime knob `PlatformSettings.payout_auto_approve_after_hours` exists but is **disabled at launch**; dial to ~24h once reconciliation is proven, no deploy needed.
- **Hold state machine:** `awaiting_customer_confirmation → release_eligible → released`, with side-states `disputed`, `withheld`, `reversed`. Delivery hooks stop calling `ReleaseDayPayout`/`ReleaseOrderPayouts` directly (#387); they only mark holds.

### 3. Disbursement

- **Weekly cadence** — existing weekly statement (`services/statement.go`). `POST /admin/statements/:id/mark-paid` records `Status=paid`, `PaidAt`, `PayoutRef`, actor; audit-logged; idempotent/race-safe (#389). Manual bank/UTR now → RazorpayX auto-payout later (#30 auto-fills `PayoutRef`).

### 4. Refunds & disputes

- A disputed hold **never releases** from any path (delivery hook, admin bulk, auto-approve timer) until the issue resolves (#400).
- **Vendor-fault refund default:** admin picks per case at resolve time; **default is full clawback** from the vendor for genuine vendor-fault, with platform-goodwill as an explicit override. Both sides written to the audit trail.
- **Conservation invariant** everywhere: `refund + vendor release + platform retained = captured amount`, enforced by a money-conservation unit test in the style of `meal_plan_escrow_fees_test.go`.

## Consequences

**Positive:** single source of truth for money; commission and TDS actually collected; statutory compliance path; human-in-the-loop before real money moves at launch; every setting runtime-tunable; consistent across single/group orders and meal plans.

**Negative / cost:** admin operational load at launch (mitigated by bulk-release + auto-approve knob); requires building the confirmation + approval + disbursement surfaces (#387/#388/#389); vendors wait for weekly disbursement (mitigate later with #30 / faster cadence).

## Implementation alignment (code to change)

- Stop attaching Route transfers in `orderSettlements` (`handlers/payment.go:198-213`); capture 100% to platform.
- Align `services/earnings.go` to 6% commission (from `PlatformSettings`), keep GST + 1% TDS; drop the premium-rate branch (`earnings.go:49-52`).
- Statements (`services/statement.go`) become authoritative; wire `mark-paid` ledger (#389).
- Decouple delivery hooks from release (#387); add hold state machine + admin queue (#388).
- Restrict chef `→ Delivered` + add OTP PoD (#391).
- Link order-issues to holds (#400).
- Migration note for any orders already settled under Model A.
- Update vendor-facing earnings screens (`/chef/earnings/breakdown`, `PayoutsPage.tsx`) to match.

## Sequencing

1. This ADR + #390 (model) and #391/#394/#395 (security) — must precede any flag flip.
2. #387 → #388 → #389 build the dual-approval control plane in order.
3. #392/#393/#399/#400 complete refunds/disputes against the hold state machine.
4. #397/#398/#396 harden; #402 anytime.
5. #218 sandbox sign-off → #25 live switch. #30 (RazorpayX) automates disbursement later.

## Open items (require finance/CA sign-off)

- [ ] **TCS under CGST §52** (1% e-commerce operator collection) — separate from §194-O income-tax TDS, not currently modeled. Confirm applicability + rate with CA before live switch. Ties into #19 legal sign-off.
- [ ] Confirm GST treatment (CGST/SGST vs IGST by chef state) matches `earnings.go` logic at 6%.
- [ ] Confirm 6% commission is final for launch pricing.
