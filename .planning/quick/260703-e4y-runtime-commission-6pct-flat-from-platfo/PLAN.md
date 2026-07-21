---
phase: quick
plan: 01
type: execute
wave: 1
depends_on: []
mode: quick
autonomous: true
requirements: [GH-390, ADR-0001-economics]
files_modified:
  - apps/api/services/earnings.go
  - apps/api/services/commission.go
  - apps/api/services/commission_test.go
  - apps/api/services/earnings_test.go
  - apps/api/services/premium.go
  - apps/api/services/statement.go
  - apps/api/services/statement_pdf.go
  - apps/api/handlers/chef_earnings.go
  - apps/api/handlers/subscription.go
  - apps/api/services/billing.go

must_haves:
  truths:
    - "Every chef is charged a flat 6% platform commission by default — no premium reduced-rate tier exists in the payout math."
    - "An admin can change the commission rate at runtime by writing PlatformSettings key `payout.commission_rate`, with no redeploy."
    - "18% GST on commission and 1% TDS (§194-O) are unchanged."
    - "Money is conserved: commission + tds + netPayout == gross at 6%."
  artifacts:
    - path: "apps/api/services/commission.go"
      provides: "GetCommissionRate(db) + parseCommissionRate + DefaultCommissionRate wiring"
      contains: "payout.commission_rate"
    - path: "apps/api/services/earnings.go"
      provides: "DefaultCommissionRate = 0.06, flat fallback in ComputeOrderEarnings"
      contains: "DefaultCommissionRate"
  key_links:
    - from: "apps/api/handlers/chef_earnings.go"
      to: "services.GetCommissionRate"
      via: "flat rate injected into EarningsInput.CommissionRate"
      pattern: "GetCommissionRate"
    - from: "apps/api/services/statement.go"
      to: "services.GetCommissionRate"
      via: "flat rate resolved once per statement run"
      pattern: "GetCommissionRate"
---

<objective>
Implement the **economics half** of ADR `docs/adr/0001-vendor-payout-settlement-model.md` and GH #390: make the chef platform commission a **flat 6%, runtime-configurable from PlatformSettings**, and **delete the premium-chef reduced-rate tier** from the payout math. GST (18% on commission) and TDS (1% §194-O) stay untouched. Add a money-conservation unit test at 6% and a rate-override test.

Purpose: The codebase currently hardcodes a 15% base rate and applies a per-chef premium override (#44). The ADR sets the launch model to a single flat 6% rate, tunable at runtime for launch promos / per-cohort changes without a deploy.

Output: A commission rate resolved from `PlatformSettings.payout.commission_rate` (default `0.06`), injected into the pure `ComputeOrderEarnings`, with the premium application path removed and tests locking the money-conservation invariant.

Scope guard: This is the **economics** slice only. Do NOT touch `handlers/payment.go` `orderSettlements` / the Razorpay Route split — that lands with the control-plane work (#387/#388). Do NOT build the hold state machine, OTP PoD, or admin approval queue.
</objective>

<execution_context>
@$HOME/.claude/get-shit-done/workflows/execute-plan.md
</execution_context>

<context>
@docs/adr/0001-vendor-payout-settlement-model.md

<interfaces>
<!-- Verified from the codebase — executor should use these directly, no exploration needed. -->

apps/api/services/earnings.go (current):
```go
const (
    RateCommission   = 0.15  // base rate — becomes 0.06
    RateGST          = 0.18  // UNCHANGED
    RateTDS          = 0.01  // UNCHANGED
    EarningsCurrency = "INR"
)

type EarningsInput struct {
    // ...
    // CommissionRate: currently the #44 premium override; becomes
    // "the resolved flat commission rate for this order".
    CommissionRate float64
    ChefFundedDiscount float64
}

// ComputeOrderEarnings — PURE. Keep it pure; the rate is injected via
// EarningsInput.CommissionRate. Fallback branch at ~line 98:
//   rate := in.CommissionRate
//   if rate <= 0 || rate >= 1 { rate = RateCommission }  // -> DefaultCommissionRate
func ComputeOrderEarnings(in EarningsInput, chefState string) OrderEarnings
```

Runtime settings read pattern — COPY THIS (apps/api/services/order_issue.go:66):
```go
func GetIssueConfig(db *gorm.DB) IssueConfig {
    cfg := IssueConfig{Enabled: true, AutoApproveCap: 300}
    var settings []models.PlatformSettings
    db.Where("key LIKE ?", "order_issue.%").Find(&settings)
    for _, s := range settings {
        switch s.Key {
        case "order_issue.auto_approve_cap":
            if v, err := strconv.ParseFloat(s.Value, 64); err == nil {
                cfg.AutoApproveCap = v
            }
        }
    }
    return cfg
}
```
Model is `models.PlatformSettings` (key/value/updated_by) — apps/api/models/admin.go:9.
DB handle: `database.DB` (package `github.com/homechef/api/database`).

Premium application path to DELETE — apps/api/services/premium.go:53:
```go
func PremiumCommissionRateForChef(chefID uuid.UUID) float64 { ... } // + chefCountryCode helper
```
Its only three callers (all resolve a per-chef rate, then pass via CommissionRate):
- apps/api/handlers/chef_earnings.go:122  -> commissionRate := services.PremiumCommissionRateForChef(chef.ID)
- apps/api/services/statement.go:87       -> chefBucket{ commissionRate: PremiumCommissionRateForChef(r.ChefID) }
- apps/api/services/statement_pdf.go:48   -> commissionRate := PremiumCommissionRateForChef(stmt.ChefID)

Display sites referencing the old base rate (repoint to the resolved runtime rate):
- apps/api/handlers/chef_earnings.go:150   -> effectiveCommission := services.RateCommission
- apps/api/services/statement_pdf.go:164   -> label "Platform commission (%.0f%%)" using RateCommission
- apps/api/handlers/subscription.go:95,107 -> standardCommissionRate display

KEEP UNTOUCHED: `PremiumChefIDs`, `IsChefPremium` (used by chef-list ranking/badges),
RateGST, RateTDS, GST intra/inter-state logic, Round2, NormaliseState,
handlers/payment.go orderSettlements (out of scope).
</interfaces>

Reference money-conservation test style: @apps/api/services/meal_plan_escrow_fees_test.go
</context>

<tasks>

<task type="auto" tdd="true">
  <name>Task 1: Write failing tests — 6% conservation, rate parsing, and default flip (RED)</name>
  <files>apps/api/services/earnings_test.go, apps/api/services/commission_test.go</files>
  <behavior>
    In earnings_test.go:
    - Add TestComputeOrderEarnings_FlatSixPercent_MoneyConserved: input itemRevenue=1000,
      deliveryFee=50, chefTip=20, DeliveryState="maharashtra", CommissionRate=0.06, chefState="Maharashtra".
      Expect commission=60.00, gross=1070.00, cgst=5.40, sgst=5.40 (9% each of 60), tds=10.70,
      netPayout = 1070 - 60 - 10.70 = 999.30.
      Assert gross - commission - tds == netPayout (money conservation).
      Assert captured-conservation: platformRetained(commission+tds) + netPayout + refunds(0) == gross.
    - Add TestComputeOrderEarnings_DefaultRateIsSixPercent: input WITHOUT CommissionRate set
      (0/unset) must fall back to 6% -> commission=60.00, NOT 150.00.
    - Repurpose the existing premium-override test: rename to
      TestComputeOrderEarnings_InjectedRate (injecting 0.12 still yields commission=120) —
      the injection point survives; only the "premium" framing/comment is dropped.
    - Update ANY existing assertion that expects the old 15% default (commission 150 / net 909.3)
      to the new 6% expectations.

    In commission_test.go:
    - TestParseCommissionRate: "0.06"->(0.06,true); "0.08"->(0.08,true); ""->(_,false);
      "0"->(_,false); "1"->(_,false); "1.5"->(_,false); "abc"->(_,false).
      (Covers "a PlatformSetting override changes the rate" at the pure layer, plus
      invalid/out-of-range falls back to default.)
  </behavior>
  <action>
    Write the tests above. They MUST fail to compile/pass initially (DefaultCommissionRate and
    parseCommissionRate do not exist yet, and the current default is 15%). Do not implement
    production code in this task. Keep tests table-driven where natural; no DB harness needed —
    ComputeOrderEarnings and parseCommissionRate are pure.
  </action>
  <verify>
    <automated>cd apps/api && go test ./services/ -run 'CommissionRate|FlatSixPercent|DefaultRateIsSixPercent|ParseCommissionRate|InjectedRate' 2>&1 | tail -20 || true</automated>
  </verify>
  <done>New tests exist and FAIL (compile error on missing symbols or assertion failure on the 15% default). No production code changed yet.</done>
</task>

<task type="auto" tdd="true">
  <name>Task 2: Add flat 6% default + runtime GetCommissionRate (GREEN core)</name>
  <files>apps/api/services/earnings.go, apps/api/services/commission.go</files>
  <action>
    earnings.go:
    - Add const `DefaultCommissionRate = 0.06` with a doc comment: "flat platform commission
      on food subtotal (ADR-0001 / #390); runtime-tunable via GetCommissionRate".
    - Repoint the base const to the new default: `RateCommission = DefaultCommissionRate`
      (single source value; keeps existing display references compiling and showing 6%).
    - In ComputeOrderEarnings, change the fallback branch to use `DefaultCommissionRate`
      instead of `RateCommission`; update the inline comment — the rate is now the injected
      flat per-order rate, not a "premium (#44)" override. Rewrite the `EarningsInput.CommissionRate`
      field doc to: "resolved flat commission rate for this order; 0/unset uses DefaultCommissionRate".
    - Do NOT touch RateGST, RateTDS, the GST intra/inter-state logic, Round2, or NormaliseState.

    New apps/api/services/commission.go (small, focused file — follow the order_issue.go read pattern):
    - `func parseCommissionRate(raw string) (float64, bool)`: `strconv.ParseFloat`, valid only when
      `0 < rate < 1`; returns `(0, false)` otherwise. Pure, no DB. Use `%w`-style errors only if
      you return errors (here a bool is sufficient).
    - `func GetCommissionRate(db *gorm.DB) float64`: read `models.PlatformSettings` where
      `key = "payout.commission_rate"`; if found and parseCommissionRate ok -> return it; else
      return `DefaultCommissionRate`. Never panics; a missing/invalid setting silently falls back.
      Keep the function <50 lines, early-return style.
  </action>
  <verify>
    <automated>cd apps/api && go build ./... && go test ./services/ -run 'ParseCommissionRate|FlatSixPercent|DefaultRateIsSixPercent|InjectedRate' 2>&1 | tail -20</automated>
  </verify>
  <done>Package builds; the conservation, default-6%, injected-rate, and parse tests PASS. `payout.commission_rate` is the only new settings key; math stays pure (rate injected).</done>
</task>

<task type="auto">
  <name>Task 3: Delete premium tier application + migrate callers to the flat runtime rate</name>
  <files>apps/api/services/premium.go, apps/api/services/statement.go, apps/api/services/statement_pdf.go, apps/api/handlers/chef_earnings.go, apps/api/handlers/subscription.go, apps/api/services/billing.go</files>
  <action>
    premium.go:
    - DELETE `PremiumCommissionRateForChef` and the now-unused `chefCountryCode` helper.
      KEEP `PremiumChefIDs` and `IsChefPremium` (ranking/badges still use them). Update the
      file-header comment to drop "lower commission" from the premium perks list.

    handlers/chef_earnings.go:
    - Line ~122: replace `commissionRate := services.PremiumCommissionRateForChef(chef.ID)`
      with `commissionRate := services.GetCommissionRate(database.DB)` (flat, chef-independent).
    - Line ~150: replace `effectiveCommission := services.RateCommission` + the `if commissionRate > 0`
      block with `effectiveCommission := commissionRate` (the resolved runtime rate is always set).
    - Ensure the `database` package is imported (it likely already is via other handlers; add if needed).

    services/statement.go:
    - Resolve the flat rate ONCE before the bucket loop: `flatRate := GetCommissionRate(database.DB)`.
    - Line ~87: `chefBucket{ ..., commissionRate: flatRate }` (drop `PremiumCommissionRateForChef(r.ChefID)`).
      Update the comment: rate is a flat platform rate, no longer per-chef premium.

    services/statement_pdf.go:
    - Line ~48: `commissionRate := GetCommissionRate(database.DB)` (drop PremiumCommissionRateForChef).
    - Line ~164: the "Platform commission (%.0f%%)" label should use `commissionRate*100`
      (the resolved rate), not `RateCommission`.

    handlers/subscription.go (lines ~95,107): repoint the displayed standard commission rate to
    `services.GetCommissionRate(database.DB)` so the pricing screen reflects the runtime rate
    instead of a compile-time const.

    services/billing.go (line ~83, `PremiumCommissionRate: 0.12`): leave the struct field in place
    (removing it ripples through admin pricing response) but add a comment:
    "// Premium tier no longer affects payout commission (ADR-0001/#390 flat rate); retained as
    legacy subscription-tier metadata only." Do NOT change admin_subscription_pricing.go response
    shape — full field removal is an explicit follow-up, out of scope here.
  </action>
  <verify>
    <automated>cd apps/api && grep -rn "PremiumCommissionRateForChef" . ; test -z "$(grep -rl 'PremiumCommissionRateForChef' --include='*.go' .)" && echo "NO_REMAINING_CALLERS" && go build ./...</automated>
  </verify>
  <done>`PremiumCommissionRateForChef` is deleted and has zero remaining references; all three earnings surfaces resolve the flat rate via `GetCommissionRate`; package builds. No changes to payment.go orderSettlements or admin pricing response shape.</done>
</task>

<task type="auto">
  <name>Task 4: Full suite + coverage verification</name>
  <files>apps/api/services/earnings_test.go, apps/api/services/commission_test.go</files>
  <action>
    Run the full services + handlers test suites. Fix any test that still encodes the old 15%
    base rate or the premium-commission behavior (e.g. an earnings/statement test asserting
    commission 150 or a premium override producing a per-chef rate). Confirm the money-conservation
    invariant holds at 6% and that overriding `payout.commission_rate` (via parseCommissionRate)
    changes the rate. Ensure coverage for the touched services files stays healthy (repo target 80%+).
  </action>
  <verify>
    <automated>cd apps/api && go test ./services/... ./handlers/... -count=1 2>&1 | tail -30</automated>
  </verify>
  <done>All service + handler tests pass; no references to a 15% base rate or premium commission remain in tests; conservation and rate-override tests green.</done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| admin → PlatformSettings | Admin-set `payout.commission_rate` value crosses into payout math |
| DB row → earnings math | Persisted setting value is parsed into a float used in money calculations |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-quick-01 | Tampering | `GetCommissionRate` parsing `payout.commission_rate` | mitigate | `parseCommissionRate` accepts only `0 < rate < 1`; any invalid/out-of-range value silently falls back to `DefaultCommissionRate` (0.06). Never charges a nonsensical rate. |
| T-quick-02 | Elevation of Privilege | Who can write `payout.commission_rate` | accept | Write path is the existing admin-only `setPlatformSetting` (admin_subscription_pricing.go); this plan adds no new write endpoint. Existing admin authz governs it. |
| T-quick-03 | Repudiation | Rate change attribution | accept | `PlatformSettings.updated_by` already records the actor on write; no change needed here. |
| T-quick-04 | Information Disclosure | Commission rate exposure on chef/pricing screens | accept | Rate is non-sensitive business config already surfaced to chefs. |
</threat_model>

<verification>
- `cd apps/api && go build ./...` — compiles.
- `go test ./services/... ./handlers/... -count=1` — all green.
- `grep -rn "PremiumCommissionRateForChef" apps/api` — no results.
- `grep -rn "DefaultCommissionRate\|payout.commission_rate" apps/api/services` — present.
- Manual sanity: default (no setting) → 6%; setting `payout.commission_rate=0.08` → 8%; `payout.commission_rate=1.5` or garbage → 6% fallback.
</verification>

<success_criteria>
- Default platform commission is a flat 6% for every chef; premium reduced-rate tier removed from payout math (`PremiumCommissionRateForChef` deleted, no callers).
- Rate is runtime-configurable via `PlatformSettings.payout.commission_rate` with in-range validation and 0.06 fallback.
- 18% GST (CGST/SGST/IGST) and 1% TDS logic unchanged.
- Money-conservation test at 6% passes: `commission + tds + netPayout == gross`.
- Rate-override behavior covered by tests; full services + handlers suites green.
- No changes to `handlers/payment.go` orderSettlements or the admin pricing response shape.
</success_criteria>

<output>
After completion, create `.planning/quick/260703-e4y-runtime-commission-6pct-flat-from-platfo/SUMMARY.md` noting: files touched, the new `payout.commission_rate` key + `DefaultCommissionRate=0.06`, deletion of `PremiumCommissionRateForChef`, and the explicit follow-up (remove the now-legacy `PremiumCommissionRate` field from `PlanConfig`/admin pricing response).
</output>
