# Quick Task #390: Net chef Razorpay/Stripe payout == settlement statement (single source of truth)

**One-liner:** The chef's actual gateway transfer now pays NET (gross − commission − TDS) via one `chefNetPayout(order)` helper feeding all three payout sites, equal to the statement's `ComputeOrderEarnings(order).NetPayout`; gross is `itemRevenue + Tax + ChefTip` (food GST in, delivery out); and the commission rate is frozen on the order at checkout so a mid-flight admin retune can't make the statement disagree with a transfer already sent.

Branch: `feat/390-net-chef-payout` · Issue: GH-390 · Risk: highest (live money-math)

## Files changed and why

| File | Why |
|------|-----|
| `apps/api/services/earnings.go` | Added `EarningsInput.Tax`; changed `gross = itemRevenue + Tax + ChefTip` (was `+ DeliveryFee`). `DeliveryFee` retained on `OrderEarnings` for display only. GST-on-commission + TDS logic unchanged. |
| `apps/api/services/earnings_test.go` | Rewrote the 4 Tax-dependent cases (Tax replaces DeliveryFee in gross); added `TestComputeOrderEarnings_GrossUsesTaxNotDelivery`. |
| `apps/api/models/order.go` | Added frozen `Order.CommissionRate float64` mirroring the `TaxRate`/`TaxName` freeze. |
| `apps/api/migrations/20260705000001_add_order_commission_rate.{up,down}.sql` | Reversible DDL for `orders.commission_rate`. |
| `apps/api/services/statement.go` | Row struct + query carry `tax` + `commission_rate`; caller passes `Tax` + `rowRate(r.CommissionRate, flatRate)`; migration-note comment. |
| `apps/api/services/statement_pdf.go` | Caller passes `Tax` + `rowRate(r.CommissionRate, commissionRate)`. |
| `apps/api/services/tds_certificate.go` | Inline query carries `tax` + `commission_rate`; caller passes `Tax` + `r.CommissionRate`. |
| `apps/api/services/commission.go` | Added unexported `rowRate(frozen, fallback)`. |
| `apps/api/handlers/chef_earnings.go` | Row struct + query carry `tax` + `commission_rate`; caller passes `Tax` + `rowRate(...)`; added local `rowRate`. |
| `apps/api/handlers/payment.go` | Replaced `chefGrossPayout` with `chefNetPayout(order)` reading frozen rate; all 3 payout sites route through it; rate frozen once at checkout (both gateways); determinism comment at verify. |
| `apps/api/handlers/payment_test.go` | Net-era tests: ChefNetPayout, OrderSettlements_ChefNetTransfer, non-tautological ChefTransferEqualsStatementNetPayout, FrozenRateSurvivesRetune, ConservationExcludesGSTOnCommission; harness DDL + platform_settings. |
| `apps/api/handlers/chef_delivered_gate_test.go`, `chef_earnings_test.go`, `order_lifecycle_money_test.go` | Caller-side test fixes for the formula change (Deviations). |
| `MIGRATION-NOTE.md` | Historical-rows + TDS-cert regulatory caveats + out-of-scope follow-ups. |

## The ₹999.30 conservation trace

Order: ₹1000 food + ₹50 tax + ₹70 delivery + ₹20 chef tip, intra-state, 6%, 1% TDS, no service fee, no refund. Customer pays ₹1140.

- commission = 6% × 1000 = 60
- gross = 1000 + 50 (tax) + 20 (tip) = 1070  (delivery ₹70 EXCLUDED)
- TDS = 1% × 1070 = 10.70
- netPayout = 1070 − 60 − 10.70 = 999.30  ← chef Route/Stripe transfer
- GST-on-commission = 18% × 60 = 10.80  ← platform downstream remittance, NOT deducted, NOT in order.Total

Captured conservation (GST-on-commission EXCLUDED):
net 999.30 + (commission 60 + TDS 10.70 + serviceFee 0) + (delivery 70 + driverTip 0) + refunds 0 = 1140 = order.Total ✓
Driver transfer (DeliveryFee + DriverTip = 70) unchanged. chefNetPayout(order) == ComputeOrderEarnings(...).NetPayout == 999.30.

## Transfer == statement (via the frozen rate)

- `chefNetPayout(order)` returns `ComputeOrderEarnings(...).NetPayout` — the statement figure.
- All 3 payout sites call it (grep confirms no residual `chefGrossPayout` caller).
- All 4 statement/TDS/earnings callers read the per-order frozen `commission_rate` (fallback via `rowRate` for legacy 0 rows).
- Rate frozen once at checkout for both gateways; verify reloads the persisted rate → deterministic create==verify split.
- `TestChefTransferEqualsStatementNetPayout` is non-tautological: builds payout-side input as payment.go maps it and statement-side input as statement.go maps a scanned row, asserts equal.

## Verification output

- gofmt -l on all touched files: empty (clean). Pre-existing gofmt violations in untouched files left as-is (out of scope).
- go vet ./...: clean.
- go build ./...: clean.
- go test -count=1 ./services/ ./handlers/: ok / ok (all pass), incl. ₹999.30 example, GrossUsesTaxNotDelivery, ChefFundedDiscount (906.30), InjectedRate (939.30), IntraState (Tax-driven), ChefNetPayout, OrderSettlements net, ChefTransferEqualsStatementNetPayout, FrozenRateSurvivesRetune, ConservationExcludesGSTOnCommission.
- Existing payout tests (#387/#388/#456/#457/#459): still green.

## Deviations from Plan

1. [Rule 1 — Bug] Caller-side earnings assertions asserted the old (delivery-in) formula.
   - Found: Task 7. `TestComputeOrderBreakdown_IntraState` and `TestOrderLifecycle_RefundRecomputeAndEarnings` asserted gross with DeliveryFee in it.
   - Fix: IntraState swapped DeliveryFee:50 → Tax:50 (gross 1070 / TDS 10.70 / net 999.30). Lifecycle test now passes surviving Tax (30); gross 660 → 650 (600 + 30 + 20; delivery 40 excluded).
   - Commit: e42da668

2. [Rule 3 — Blocking] Hand-DDL test harness missing `commission_rate`.
   - Found: Task 7. `chef_delivered_gate_test.go` orders DDL lacked commission_rate → GORM update 500 in 3 UpdateOrderStatus tests. Plan anticipated this.
   - Fix: added `commission_rate real DEFAULT 0` to the harness DDL (+ platform_settings and extra columns to payment_test.go harness for new tests).
   - Commit: e42da668

## Follow-ups (in MIGRATION-NOTE.md)

- Meal-plan-day payout (net): verify per-day meal-plan escrow transfers don't still pay gross (separate seam, not touched).
- TCS §52: CA-pending, not modelled.
- Vendor UI copy: earnings screens auto-reflect the DTO; explanatory copy is separate.
- Historical WeeklyStatement rows / already-issued TDS Form-16A: OLD basis — documented, NOT rewritten; regenerating a past TDS cert reports different TDS (regulatory follow-up).

## Commits

- 8d689e70 test(390): encode net-transfer formula, frozen rate, non-tautological transfer==statement equality (RED)
- f5d75984 feat(390): gross = itemRevenue + Tax + chefTip (food GST in, delivery out)
- 8250af59 feat(390): freeze Order.CommissionRate at checkout (column + reversible migration)
- e6c35d8a feat(390): statement, PDF, TDS cert, and earnings breakdown pass Tax + frozen per-order rate
- bbd2846e feat(390): chef Route/Stripe transfer pays NET via single chefNetPayout; freeze rate at checkout
- a824ae43 test(390): GST-excluded conservation identity; document historical-rows + TDS-cert migration caveats
- e42da668 test(390): update caller-side earnings assertions to new formula; add commission_rate to hand-DDL harness
