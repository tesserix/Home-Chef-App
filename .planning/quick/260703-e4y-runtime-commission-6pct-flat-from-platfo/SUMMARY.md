# Quick Task Summary: Runtime flat 6% platform commission

**Requirement:** GH #390 / ADR-0001 economics slice
**Branch:** `feat/payout-6pct-commission`
**One-liner:** Chef platform commission is now a flat **6%** default (`DefaultCommissionRate = 0.06`), runtime-tunable via `PlatformSettings.payout.commission_rate`, with the per-chef premium reduced-rate tier deleted from the payout math. GST (18%) and TDS (1% §194-O) untouched.

## What changed (files + why)

### New
- **`apps/api/services/commission.go`** — runtime rate resolution.
  - `parseCommissionRate(raw string) (float64, bool)` — pure parse+validate; accepts only `0 < rate < 1`, else `(0, false)`. Backs tamper mitigation T-quick-01: a bad admin value never charges a nonsense rate.
  - `GetCommissionRate(db *gorm.DB) float64` — reads `PlatformSettings` key `payout.commission_rate`; returns parsed rate when valid, else `DefaultCommissionRate`. Never panics. Follows the `order_issue.go` read pattern.
- **`apps/api/services/commission_test.go`** — table-driven `TestParseCommissionRate` covering `0.06`/`0.08` (valid), `""`/`0`/`1`/`1.5`/`abc` (fallback).

### Modified
- **`apps/api/services/earnings.go`** — added `DefaultCommissionRate = 0.06`; repointed `RateCommission = DefaultCommissionRate` (single source value keeps display refs compiling and showing 6%); `ComputeOrderEarnings` fallback branch now uses `DefaultCommissionRate`; rewrote the `EarningsInput.CommissionRate` doc. GST/TDS/Round2/NormaliseState untouched. Math stays pure — rate injected.
- **`apps/api/services/premium.go`** — deleted `PremiumCommissionRateForChef` and the `chefCountryCode` helper; updated file header to drop "lower commission". `PremiumChefIDs` / `IsChefPremium` kept.
- **`apps/api/handlers/chef_earnings.go`** — resolves `services.GetCommissionRate(database.DB)` once; `effectiveCommission` is the resolved runtime rate (removed the premium branch).
- **`apps/api/services/statement.go`** — resolves `flatRate := GetCommissionRate(database.DB)` once before the chef-bucket loop; bucket rate is the flat rate.
- **`apps/api/services/statement_pdf.go`** — resolves `commissionRate := GetCommissionRate(database.DB)`; label uses `commissionRate*100` (rate threaded into `addStatementSummary` — deviation 1).
- **`apps/api/handlers/subscription.go`** — `standardCommissionRate` + `standardCommissionPct` now from `services.GetCommissionRate(database.DB)`.
- **`apps/api/services/billing.go`** — kept `PremiumCommissionRate: 0.12` in `PlanConfig` but commented as legacy subscription-tier metadata only (no payout effect). Admin pricing response shape unchanged.
- **`apps/api/services/earnings_test.go`** — flipped all 15% expectations to 6%; added `TestComputeOrderEarnings_FlatSixPercent_MoneyConserved` (asserts `commission + tds + netPayout == gross` and captured-conservation `retained + net + refunds == gross`); added `TestComputeOrderEarnings_DefaultRateIsSixPercent`; renamed premium-override test to `TestComputeOrderEarnings_InjectedRate`.
- **`apps/api/handlers/chef_earnings_test.go`**, **`apps/api/handlers/order_lifecycle_money_test.go`** — retuned from 15% to 6% (both call `computeOrderBreakdown(..., 0)`, now defaulting to 6%).

## Money conservation (6%, itemRevenue 1000 / delivery 50 / tip 20, intra-state)
- commission = 6% × 1000 = 60.00 · gross = 1070.00 · CGST = SGST = 5.40 · TDS = 10.70 · netPayout = 999.30
- Conserved: 60 + 10.70 + 999.30 = 1070.00 == gross

## Test results
```
go build ./...  -> OK
go test ./services/... ./handlers/... -count=1
  ok  github.com/homechef/api/services  0.620s
  ok  github.com/homechef/api/handlers  0.595s
PASS: TestParseCommissionRate (+7 subtests)
PASS: TestComputeOrderEarnings_FlatSixPercent_MoneyConserved
PASS: TestComputeOrderEarnings_DefaultRateIsSixPercent
PASS: TestComputeOrderEarnings_InjectedRate / _IntraState / _InterState
grep -rn "PremiumCommissionRateForChef" -> no results (zero callers)
```
Coverage: `parseCommissionRate` 100.0%, `ComputeOrderEarnings` 94.4%. `GetCommissionRate` is DB-backed (its pure core covered; plan noted no DB harness needed).

## Deviations
1. `statement_pdf.go` — the label lives in `addStatementSummary`, not the function resolving `commissionRate`. Minimal fix: added a `commissionRate float64` parameter to `addStatementSummary` and pass the resolved rate at the call site. No behavior change beyond the intended label repoint.
2. `subscription.go` — introduced a local `standardCommission` var reused by both the response field and perks string so `GetCommissionRate` is called once.

Scope held: `handlers/payment.go` orderSettlements / Razorpay Route split untouched; admin pricing response shape unchanged.

## Commits
- 91165280 test: flat 6% commission conservation, default flip, and rate parsing (RED)
- 451824aa feat: flat 6% DefaultCommissionRate + runtime GetCommissionRate from PlatformSettings
- 9fcf2ef4 refactor: delete premium commission tier, resolve flat runtime rate at all earnings surfaces
- 987439ad test: retune handler earnings tests to flat 6% commission

## Follow-ups (out of scope)
- Remove the now-legacy `PremiumCommissionRate` field from `PlanConfig` (`billing.go`) and the admin pricing response (`subscription.go` `premiumCommissionRate` + premium-perks "lower commission" line). Left in place to avoid rippling the admin pricing response shape; flagged in `billing.go`.
- `GetCommissionRate` has no DB-integration test (only the pure `parseCommissionRate` layer is unit-tested). Add one when a DB harness exists for the services package.
