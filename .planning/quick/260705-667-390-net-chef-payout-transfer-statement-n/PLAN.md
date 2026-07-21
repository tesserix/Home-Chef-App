---
mode: quick
issue: 390
branch: feat/390-net-chef-payout
title: Net chef Razorpay payout == statement math (single source of truth)
type: execute
autonomous: true
risk: highest — live money-math; changes the chef Route transfer amount
files_modified:
  - apps/api/services/earnings.go
  - apps/api/services/earnings_test.go
  - apps/api/models/order.go
  - apps/api/database/migrations (new commission_rate column migration)
  - apps/api/services/statement.go
  - apps/api/services/statement_pdf.go
  - apps/api/services/tds_certificate.go
  - apps/api/handlers/chef_earnings.go
  - apps/api/handlers/payment.go
  - apps/api/handlers/payment_test.go
requirements: [GH-390]
must_haves:
  truths:
    - "Chef Razorpay Route transfer amount == ComputeOrderEarnings(order).NetPayout (no drift)"
    - "ComputeOrderEarnings gross = itemRevenue + Tax + ChefTip (delivery OUT, tax IN)"
    - "All 4 ComputeOrderEarnings callers pass the order's Tax"
    - "The 3 chef-payout sites (FSSAI audit, Route split, Stripe transfer) share one net helper — cannot drift"
    - "Commission rate is FROZEN on the order at checkout; verify + statement + tds + earnings all read order.CommissionRate — so a mid-flight admin rate change cannot make the statement disagree with the transfer already sent"
    - "Driver transfer (DeliveryFee + DriverTip) unchanged"
    - "Money conservation at 6%: chef net + platform retained (commission + TDS + serviceFee) + driver + refunds == captured order.Total. GST-on-commission is EXCLUDED — it is a downstream carve-out of the platform's commission, not money the customer paid, so it is absent from order.Total"
  artifacts:
    - path: apps/api/services/earnings.go
      provides: "EarningsInput.Tax + gross = itemRevenue + Tax + ChefTip"
    - path: apps/api/models/order.go
      provides: "frozen Order.CommissionRate column (mirrors TaxRate freeze)"
    - path: apps/api/handlers/payment.go
      provides: "shared chefNetPayout helper; net Route + Stripe transfer; rate persisted at checkout, read from column at verify"
  key_links:
    - from: apps/api/handlers/payment.go (chefNetPayout)
      to: apps/api/services/earnings.go (ComputeOrderEarnings.NetPayout)
      via: "single helper called by all 3 payout sites, rate = order.CommissionRate"
    - from: apps/api/services/statement.go / tds_certificate.go / chef_earnings.go
      to: apps/api/models/order.go (order.CommissionRate)
      via: "SELECT o.commission_rate; per-row frozen rate (fallback GetCommissionRate/default for legacy 0 rows)"
---

<objective>
GH #390 — the chef's actual Razorpay payout and the settlement statement disagree.
Fix BOTH to ONE correct formula so the transfer equals the statement (single source
of truth), and FREEZE the commission rate on the order so the equality survives a
mid-flight admin rate change.

Two contradictory models today:
- Actual transfer (`handlers/payment.go:chefGrossPayout`) pays GROSS =
  Subtotal + Tax + ChefTip − ChefFundedDiscount. No commission, no TDS.
- Statement math (`services/earnings.go:ComputeOrderEarnings`) computes
  net = gross − commission − TDS, but with gross = itemRevenue + DeliveryFee + ChefTip
  — wrongly including delivery and excluding tax.

CONFIRMED formula (owner decision 2026-07-05 — chef receives the food GST; delivery excluded):
- itemRevenue = food_subtotal − ChefFundedDiscount   (floored 0)
- commission  = rate × itemRevenue   (rate = FROZEN order.CommissionRate, default 6%)
- gross       = itemRevenue + Tax + ChefTip          (Tax IN, DeliveryFee OUT)
- GST-on-commission = 18% × commission (CGST/SGST intra vs IGST inter) — platform's downstream remittance, UNCHANGED, and NOT part of order.Total
- tds         = RateTDS × gross   (1%)
- netPayout   = gross − commission − tds
- Chef Route transfer amount = netPayout   (was gross)
- Driver transfer = DeliveryFee + DriverTip   — UNCHANGED (the driver's)

Worked check (₹1000 food, ₹50 tax, ₹70 delivery, ₹20 tip, 6%, 1%):
gross=1070, commission=60, tds=10.70, net=999.30; delivery ₹70 → driver;
platform keeps ₹60 commission (+GST on it, remitted downstream) + retains TDS.

Purpose: eliminate the transfer-vs-statement mismatch — the reconciliation single source of truth, stable across rate retunes.
Output: net-of-commission-and-TDS chef payout that equals the statement, guarded by a conservation test and a frozen per-order rate.
</objective>

<grounding>
Verified against apps/api on 2026-07-05:

- services/earnings.go — EarningsInput (line 49) has NO Tax field; gross = Round2(itemRevenue + in.DeliveryFee + in.ChefTip) (line 118). OrderEarnings (line 68) already has DeliveryFee, Gross, NetPayout. GST-on-commission + TDS logic lines 120-132. ComputeOrderEarnings already falls back to DefaultCommissionRate when in.CommissionRate <= 0 || >= 1 (lines 107-110). DefaultCommissionRate=0.06, RateTDS=0.01, Round2.
- models/order.go — Order.Tax (line 81), Subtotal (78), DeliveryFee (79), ServiceFee (80), ChefTip (88), DriverTip (89), ChefFundedDiscount (95), Total (96), DeliveryAddressState (line 112 — FLAT column, NOT nested). TaxRate/TaxName (lines 82-86) are the exact frozen-at-placement pattern to copy for CommissionRate.
- The 4 ComputeOrderEarnings callers (non-test), ALL must now pass Tax AND read the per-row frozen rate:
  1. services/statement.go:85 — from statementOrderRow (struct line 31, query loadStatementOrderRows line 121). Row has NO tax and NO commission_rate. flatRate resolved once at line 69 (becomes the legacy fallback).
  2. services/statement_pdf.go:55 — reuses statementOrderRow via loadStatementOrderRows; extending that struct/query covers it.
  3. services/tds_certificate.go:92 — has its OWN inline statementOrderRow query (line 66).
  4. handlers/chef_earnings.go:170 — from earningsOrderRow (struct line 39, query line 103); commissionRate resolved once at line 122 (becomes the legacy fallback).
- handlers/payment.go — chefGrossPayout(order) (line 190) used at 3 sites: FSSAI-withhold audit (line 107), orderSettlements Route split (line 208), Stripe transfer chefAmount (line 336). orderSettlements(order) (line 198) is called at 118 (create) and 557 (verify/settle) and payment_test.go:314. CreateOrderPayment (line 43) resolves no rate yet. GetCommissionRate(database.DB) exists (services/commission.go:36).
- services/order_payout.go — Release/Reverse move whatever amount was SET at checkout (gated OFF). NO change.
- Test harnesses: services/earnings_test.go, handlers/payment_test.go (TestChefGrossPayout_ChefFunded line 288, TestOrderSettlements_* line 304 — assert GROSS today, MUST be rewritten to net), services/meal_plan_escrow_fees_test.go (conservation style).

Existing earnings_test.go cases that pass DeliveryFee + NO Tax and MUST be rewritten in Task 1 (RED) or Task 2's `go test -run TestComputeOrderEarnings` fails:
  - TestComputeOrderEarnings_FlatSixPercent_MoneyConserved (line 8)
  - TestComputeOrderEarnings_ChefFundedDiscount (line 96)
  - TestComputeOrderEarnings_InjectedRate (line 70)
  - TestComputeOrderEarnings_IntraState (line 131)  ← breaks: today asserts Gross 1070 / TDS 10.7 / Net 999.3 with DeliveryFee:50/no-Tax; under the new formula that input yields Gross 1020 / TDS 10.20 / Net 949.80.
SAFE (no Tax dependency, do NOT touch): _InterState (162, only checks GST), _DefaultRateIsSixPercent (58, only commission), _PlatformFundedLeavesChefWhole (123), TestEarningsTotals_AddAndRound (177).
</grounding>

<tasks>

<task type="auto" number="1" tdd="true">
  <name>Task 1 (RED): Encode the new formula + net-transfer + frozen-rate + conservation as failing tests</name>
  <files>apps/api/services/earnings_test.go, apps/api/handlers/payment_test.go</files>
  <behavior>
    services/earnings_test.go — rewrite the 4 Tax-dependent cases to pass Tax and NOT rely on DeliveryFee in gross:
    - TestComputeOrderEarnings_FlatSixPercent_MoneyConserved: Tax:50; gross=1070, commission=60, cgst/sgst=5.40, tds=10.70, net=999.30. Keep BOTH conservation assertions (commission + tds + net == gross; GST-on-commission NOT part of that identity — mirrors the existing correct lines 40/51).
    - New TestComputeOrderEarnings_GrossUsesTaxNotDelivery: Input{ItemRevenue:1000, Tax:50, DeliveryFee:70, ChefTip:20, rate 0.06} → gross==1070 (delivery 70 EXCLUDED), net==999.30; OrderEarnings.DeliveryFee==70 retained as display only.
    - TestComputeOrderEarnings_ChefFundedDiscount: Tax:50 (not delivery); itemRevenue=900, commission=54, gross=970, net=906.30.
    - TestComputeOrderEarnings_InjectedRate: Tax:50; commission=120, net=939.30.
    - TestComputeOrderEarnings_IntraState (line 131): Tax:50 instead of DeliveryFee:50; keep Gross 1070 / cgst 5.40 / sgst 5.40 / igst 0 / TDS 10.70 / Net 999.30 (now driven by Tax, GST-on-commission still off the commission).
    - Do NOT touch _InterState, _DefaultRateIsSixPercent, _PlatformFundedLeavesChefWhole, TestEarningsTotals_AddAndRound.
    handlers/payment_test.go — replace the gross-era tests:
    - TestChefGrossPayout_ChefFunded → TestChefNetPayout: Order{Subtotal:1000, Tax:50, ChefTip:20, CommissionRate:0.06} → chefNetPayout(order)==999.30; with ChefFundedDiscount:100 → net 906.30; over-discounted floored ≥0. Note chefNetPayout reads the FROZEN order.CommissionRate (no rate arg).
    - TestOrderSettlements_*: orderSettlements(order)[0].Amount == ToPaise(999.30) (NET); driver [1].Amount == ToPaise(DeliveryFee + DriverTip) UNCHANGED. orderSettlements reads order.CommissionRate.
    - New TestChefTransferEqualsStatementNetPayout (NON-tautological, W3): from ONE models.Order, build the payout-side EarningsInput exactly as chefNetPayout maps it, and the statement-side EarningsInput exactly as statement.go maps a scanned statementOrderRow (subtotal→ItemRevenue, tax→Tax, delivery_fee→DeliveryFee, chef_tip→ChefTip, chef_funded_discount→ChefFundedDiscount, delivery_address_state→DeliveryState, commission_rate→CommissionRate). Assert ComputeOrderEarnings(payoutInput).NetPayout == ComputeOrderEarnings(statementInput).NetPayout == chefNetPayout(order). Rate pinned from order.CommissionRate. This guards field-mapping drift between the two construction sites (e.g. Subtotal vs Total). Add a comment: promote to a live SQL round-trip if/when a handler test DB is wired.
    - New TestFrozenRateSurvivesRetune: an order with CommissionRate:0.06 persisted yields net 999.30 even though GetCommissionRate would return a different value — chefNetPayout must use the column, not the live setting.
    - New TestConservationExcludesGSTOnCommission (B1): delivered order 6%, assert
        NetPayout + (commission + TDS + serviceFee) + (deliveryFee + driverTip) + refunds == order.Total,
      and assert GST-on-commission is NOT added (it is absent from order.Total).
  </behavior>
  <action>Write the tests above. They MUST fail (EarningsInput has no Tax; Order has no CommissionRate; chefNetPayout undefined; orderSettlements reads a missing column). Commit RED.</action>
  <verify>go test ./services/ ./handlers/ fails to build for the expected reasons (unknown field Tax, unknown field CommissionRate, undefined chefNetPayout).</verify>
  <done>Tests express Tax-in/delivery-out gross, net transfer == statement, frozen rate, GST-excluded conservation, and non-tautological equality; they fail (RED) for the right reason.</done>
</task>

<task type="auto" number="2" tdd="true">
  <name>Task 2 (GREEN): earnings.go — add Tax field, gross = itemRevenue + Tax + ChefTip</name>
  <files>apps/api/services/earnings.go</files>
  <action>
    1. Add `Tax float64` to EarningsInput (after DeliveryFee) with a doc comment: the order's food GST — the chef receives it; it enters gross.
    2. Change line 118: gross := Round2(itemRevenue + in.Tax + in.ChefTip)  (DROP in.DeliveryFee from gross).
    3. Keep OrderEarnings.DeliveryFee populated (Round2(in.DeliveryFee)) as a display/context field only — MUST NOT enter gross or net. One-line comment saying so.
    4. Update the header doc comment (lines 96-103) and the file-level bullet (line 14): gross = itemRevenue + Tax + chefTip; TDS on gross; delivery is the driver's, not chef income; GST-on-commission is platform revenue not deducted from chef and not part of the customer's captured total.
    5. GST-on-commission (CGST/SGST/IGST) and TDS logic UNCHANGED. Keep the math pure (rate + tax injected). Func stays <50 lines.
  </action>
  <verify>cd apps/api && go test ./services/ -run TestComputeOrderEarnings — all green; ₹999.30 example passes.</verify>
  <done>ComputeOrderEarnings gross uses Tax not DeliveryFee; earnings_test cases green.</done>
</task>

<task type="auto" number="3" tdd="true">
  <name>Task 3 (B2): Freeze Order.CommissionRate — column + migration</name>
  <files>apps/api/models/order.go, apps/api/database/migrations (new)</files>
  <action>
    1. Add to models/order.go, next to TaxRate/TaxName (after line 86), mirroring their frozen-at-placement pattern:
         // CommissionRate freezes the platform commission rate applied when the order
         // was placed, so a later admin retune of the runtime rate cannot make the
         // settlement statement disagree with the Route transfer already sent (#390).
         // 0/unset for legacy rows → callers fall back to the live rate/default.
         CommissionRate float64 `gorm:"default:0" json:"commissionRate"`
    2. Add a golang-migrate migration (follow the repo's existing migration convention/numbering under apps/api/database/migrations): ALTER TABLE orders ADD COLUMN commission_rate double precision NOT NULL DEFAULT 0; down = DROP COLUMN. Legacy rows keep 0 → fallback path.
  </action>
  <verify>cd apps/api && go build ./... ; migration file present with up+down; grep shows CommissionRate on Order.</verify>
  <done>Order carries a frozen commission_rate column with a reversible migration; build passes.</done>
</task>

<task type="auto" number="4" tdd="true">
  <name>Task 4: All 4 callers pass Tax AND the per-row frozen rate</name>
  <files>apps/api/services/statement.go, apps/api/services/statement_pdf.go, apps/api/services/tds_certificate.go, apps/api/handlers/chef_earnings.go</files>
  <action>
    Row structs — add BOTH columns:
      - statementOrderRow (statement.go line 31): add `Tax float64 gorm:"column:tax"` and `CommissionRate float64 gorm:"column:commission_rate"`.
      - earningsOrderRow (chef_earnings.go line 39): add `Tax float64 gorm:"column:tax"` and `CommissionRate float64 gorm:"column:commission_rate"`.
    Queries — add both columns to the SELECT:
      - loadStatementOrderRows (statement.go): add `o.tax, o.commission_rate`. (covers statement.go + statement_pdf.go)
      - tds_certificate.go inline query (line 66): add `o.tax, o.commission_rate`.
      - chef_earnings.go query (line 103): add `tax, commission_rate`.
    Literals — pass Tax and the per-row frozen rate with a legacy fallback:
      - statement.go:85 — Tax: r.Tax; CommissionRate: rowRate(r.CommissionRate, flatRate) where the existing flatRate (line 69) is the fallback when the column is 0.
      - statement_pdf.go:55 — Tax: r.Tax; CommissionRate: rowRate(r.CommissionRate, commissionRate) (its line 49 GetCommissionRate result as fallback).
      - tds_certificate.go:92 — Tax: r.Tax; CommissionRate: r.CommissionRate (ComputeOrderEarnings already falls back to DefaultCommissionRate when 0; add a GetCommissionRate fallback only if the file already resolves one — it does not, so 0→DefaultCommissionRate is the intended legacy behaviour, note it in a comment).
      - chef_earnings.go:170 — Tax: row.Tax; CommissionRate: rowRate(row.CommissionRate, commissionRate) (line 122 fallback).
    Add a tiny unexported `rowRate(frozen, fallback float64) float64` helper (returns frozen when >0, else fallback) in each package that needs it (keep <10 lines). Do NOT remove DeliveryFee from any caller — it stays a display context field.
    Confirm every ComputeOrderEarnings caller now supplies Tax AND a rate sourced from the frozen column.
  </action>
  <verify>cd apps/api && go build ./... ; each of the 4 caller literals shows Tax: and CommissionRate:; go test ./services/ green.</verify>
  <done>All 4 callers pass Tax and the frozen per-order rate (legacy 0 → live/default fallback); both row structs + all 3 SELECTs carry tax + commission_rate.</done>
</task>

<task type="auto" number="5" tdd="true">
  <name>Task 5 (GREEN): payment.go — persist rate at checkout, net transfer at 3 sites, verify reads column</name>
  <files>apps/api/handlers/payment.go</files>
  <action>
    1. Replace chefGrossPayout(order) with a single shared helper that reads the FROZEN rate off the order:
         func chefNetPayout(order *models.Order) float64
       building EarningsInput{ ItemRevenue: order.Subtotal, Tax: order.Tax, ChefTip: order.ChefTip,
         DeliveryFee: order.DeliveryFee, ChefFundedDiscount: order.ChefFundedDiscount,
         DeliveryState: order.DeliveryAddressState, CommissionRate: order.CommissionRate }
       and returning services.ComputeOrderEarnings(in, order.Chef.State).NetPayout.
       (ComputeOrderEarnings falls back to DefaultCommissionRate when order.CommissionRate is 0 — legacy safety.) <50 lines, pure.
       NOTE: chef-state = order.Chef.State (same source statement/tds use). DeliveryState = order.DeliveryAddressState (FLAT column, models/order.go:112) — NOT a nested address.
    2. Persist + freeze the rate at checkout: in CreateOrderPayment (line 43), resolve
         rate := services.GetCommissionRate(database.DB)
       ONCE and stamp it before dispatching to the gateway path:
         database.DB.Model(&order).Update("commission_rate", rate)  (and set order.CommissionRate = rate on the in-memory struct so the same request uses it).
       Do this once for BOTH razorpay and stripe branches (before the switch). After this, order.CommissionRate is the single source for the split.
    3. Update the 3 payout sites to chefNetPayout(order) (all read order.CommissionRate — cannot drift):
       - FSSAI-withhold audit (line 107): chefAmount := chefNetPayout(order).
       - orderSettlements (line 198/208): keep signature orderSettlements(order) — it now reads order.CommissionRate internally via chefNetPayout(order). chef Settlement Amount = services.ToPaise(chefNetPayout(order)). Driver Settlement UNCHANGED (order.DeliveryFee + order.DriverTip). FSSAI account-clearing UNCHANGED.
       - Stripe transfer (line 336): chefAmount := chefNetPayout(order); applicationFee = totalMinor − chefMinor still holds. Update comment (333-335): chef receives NET (gross − commission − TDS); delivery is the driver's.
    4. verify/settle path (line 557): the order is reloaded with its persisted commission_rate, so orderSettlements(order) recomputes the IDENTICAL split (create==verify determinism — no re-resolve of the live rate). Add a comment stating the split is deterministic because the rate is frozen on the row.
    5. Update the doc comment on the old chefGrossPayout/orderSettlements block (179-196): single source of truth = chefNetPayout reading the frozen order.CommissionRate; the 3 sites share it so they cannot drift.
    order_payout.go release/reverse: NO change.
  </action>
  <verify>cd apps/api && go test ./handlers/ -run "TestChefNetPayout|TestOrderSettlements|TestChefTransferEqualsStatementNetPayout|TestFrozenRateSurvivesRetune" — all green.</verify>
  <done>All 3 chef-payout sites go through chefNetPayout(order) using the frozen rate; rate persisted once at checkout; verify recomputes the identical split; driver unchanged.</done>
</task>

<task type="auto" number="6" tdd="true">
  <name>Task 6 (B1 + W1): GST-excluded conservation test + migration note</name>
  <files>apps/api/handlers/payment_test.go, .planning/quick/260705-667-390-net-chef-payout-transfer-statement-n/MIGRATION-NOTE.md, apps/api/services/statement.go</files>
  <action>
    1. Conservation test (payment_test.go) — the identity EXCLUDES GST-on-commission (B1). Delivered order at 6% (₹1000 food / ₹50 tax / ₹70 delivery / ₹20 chef tip / any serviceFee):
         e = ComputeOrderEarnings(orderInput, chefState)
         platformRetained = e.PlatformCommission + e.TDS + order.ServiceFee     // NO CGST/SGST/IGST
         driver           = order.DeliveryFee + order.DriverTip
         refunds          = 0
         assert  e.NetPayout + platformRetained + driver + refunds == order.Total (captured)
       AND assert chefNetPayout(order) == e.NetPayout.
       Comment: GST-on-commission is a downstream carve-out of the platform's commission (a remittance obligation), NOT money the customer paid — it is absent from order.Total, so it must NOT appear in this identity. This matches earnings_test.go:40/51 which assert commission + tds + net == gross with GST excluded.
    2. MIGRATION-NOTE.md: (a) existing WeeklyStatement rows were computed with the old (delivery-in-gross, tax-out) formula; flags are OFF so no real money moved, but regenerating/re-displaying old statements now differs — do NOT rewrite historical rows in this slice. (b) W1 REGULATORY caveat: already-issued TDS Form-16A certificates (tds_certificate.go recomputes gross+TDS live) will report a DIFFERENT TDS on regeneration for historical orders — call this out explicitly as a regulatory follow-up; do not silently regenerate past certificates. (c) legacy orders have commission_rate=0 → fall back to live/default rate on statement compute. (d) OUT-OF-SCOPE follow-ups: meal-plan-day payout net (RefundDay/meal-plan escrow is a separate path — verify whether meal-plan day transfers also pay gross and need the same fix), vendor UI copy beyond DTO auto-reflection, TCS §52 (CA-pending).
    3. Add a short code comment above GenerateWeeklyStatements (statement.go) pointing at MIGRATION-NOTE.md (#390): pre-#390 rows used the old gross basis and legacy orders lack a frozen rate.
  </action>
  <verify>cd apps/api && go test ./handlers/ -run TestConservationExcludesGSTOnCommission — green; MIGRATION-NOTE.md exists with the historical-rows + TDS-cert caveats + follow-ups.</verify>
  <done>Captured-conservation identity (GST-on-commission EXCLUDED) holds at 6%; transfer==NetPayout asserted; migration + TDS-cert caveat + follow-ups documented.</done>
</task>

<task type="auto" number="7" tdd="false">
  <name>Task 7 (verify): full build, test, format</name>
  <files>apps/api</files>
  <action>gofmt on all touched files; go vet; full go build; go test for services + handlers.</action>
  <verify>cd apps/api && gofmt -l services handlers models | (! grep .) && go build ./... && go test ./services/ ./handlers/ — all pass.</verify>
  <done>gofmt clean, build clean, services + handlers tests green.</done>
</task>

</tasks>

<threat_model>
Trust boundary: customer payment capture → platform Route/Transfer split → chef & driver payout accounts.

| Threat ID | Category | Component | Disposition | Mitigation |
|-----------|----------|-----------|-------------|------------|
| T-390-01 | Tampering (I) | chef payout amount drifting from the reconciled statement | mitigate | single chefNetPayout(order) feeds all 3 sites; non-tautological TestChefTransferEqualsStatementNetPayout asserts payout-side == statement-side == chefNetPayout |
| T-390-02 | Repudiation | admin retunes the rate between checkout and statement → statement ≠ transfer already sent | mitigate | rate FROZEN on order.CommissionRate at checkout; verify + statement + tds + earnings all read the column; TestFrozenRateSurvivesRetune |
| T-390-03 | Information/financial-integrity | money not conserved against the captured total | mitigate | conservation test: net + (commission + TDS + serviceFee) + driver + refunds == order.Total. GST-on-commission EXCLUDED (downstream carve-out, not in order.Total) |
| T-390-04 | Elevation (of loss) | over-large ChefFundedDiscount driving a negative payout | mitigate | itemRevenue floored at 0 in ComputeOrderEarnings; floored-payout test |
| T-390-05 | Denial (of correct payout) | delivery fee wrongly paid to chef instead of driver | mitigate | gross drops DeliveryFee; driver Settlement unchanged; TestGrossUsesTaxNotDelivery + driver-unchanged assertion |

Conservation invariant (the load-bearing guard, GST-on-commission EXCLUDED):
chefNet + (commission + TDS + serviceFee) + (deliveryFee + driverTip) + refunds == capturedTotal (order.Total).
GST-on-commission is the platform's downstream remittance on its own commission — not part of what the customer paid.
</threat_model>

<verification>
- go build ./... clean; gofmt clean; go vet clean; migration up/down valid.
- services + handlers tests green, including: ₹999.30 worked example, GrossUsesTaxNotDelivery, ChefFundedDiscount (906.30), InjectedRate (939.30), IntraState (Tax-driven), ChefNetPayout, OrderSettlements net, ChefTransferEqualsStatementNetPayout (non-tautological), FrozenRateSurvivesRetune, ConservationExcludesGSTOnCommission.
- grep confirms all 4 ComputeOrderEarnings callers pass Tax + a frozen-rate source; all 3 payment.go sites route through chefNetPayout(order); Order has commission_rate column + migration.
</verification>

<success_criteria>
- Chef Razorpay Route transfer amount == ComputeOrderEarnings(order).NetPayout (transfer == statement).
- Gross = itemRevenue + Tax + ChefTip; delivery excluded from chef gross/net; driver transfer unchanged.
- Commission rate FROZEN on the order at checkout; verify + statement + tds + earnings read order.CommissionRate — mid-flight retune cannot break equality.
- GST-excluded conservation holds at 6%. Determinism between create and verify (frozen rate). No historical-row / past-cert rewrite (documented follow-up).
</success_criteria>

<output>
Branch feat/390-net-chef-payout. Commit sequence: RED (Task 1) → earnings.go (2) → Order.CommissionRate column + migration (3) → callers pass Tax + frozen rate (4) → payment.go persist rate + net transfer + verify reads column (5) → GST-excluded conservation + migration/TDS-cert note (6) → format/verify (7). Open a PR to main (do not push to main directly).
</output>
