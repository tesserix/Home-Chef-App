# Test execution — 4 Aug 2026, post per-supply-GST release

Driven on the **iOS simulators** (HC-Customer / HC-Vendor, iOS 26.5) against the **prod API**
`main-c8504d1`, Cashfree **test** environment. Scenarios from `TEST-EXECUTION-PLAN.md`.

**Every money figure was verified in Postgres**, not read off a screen. UI automation via `idb`.

No fixes were applied — this is a record only.

---

## Headline

**One critical defect found, and it is a regression introduced by this release.**

> 🟥 **D-09 — chefs are still credited the platform's own GST.** The D-02 fix shipped in
> #980 is only effective on 2 of 7 earnings paths. The weekly statement — which chefs are
> actually paid on — is among the 5 still wrong.

Also confirmed: **D-01 and D-03 from the July run are unchanged**, and one new UX dead-end
(**D-10**) was found in the payment-retry flow.

The **pricing and refund model itself held up perfectly** — every order, tax split, wallet
allocation and hold transition matched prediction to the paisa.

---

## Defects

### ✅ D-09 · FIXED AND VERIFIED IN PRODUCTION (#987)

The vendor earnings screen now reads **Gross revenue ₹955.40** on the same
period that showed ₹959.80. The ₹4.40 over-credit is gone.

Six raw queries were affected, not the two I identified by hand — a static
guard (`TestRawOrderSelectsCarryPerSupplyTax`) found `fy_statement.go`,
`tds_certificate.go` and `chef_profit_loss.go`, which I had wrongly assumed
shared the statement loader.

Original finding below.

### 🟥 D-09 · CRITICAL · chefs credited the platform's GST (regression, this release)

Proved on the vendor earnings screen, THIS WEEK, 3 delivered orders:

| | |
|---|---|
| Screen showed gross | **₹959.80** |
| Correct (chef tax = food GST) | ₹955.40 |
| Wrong (chef tax = whole order tax) | **₹959.80** ← matches |

Over-credit on `HC26080323440359` is **₹4.40** = `tax_service 2.44 + tax_delivery 1.96`.

**Root cause.** The fix added `TaxFood`/`TaxService` to the scan structs and routed them
through `ChefTaxOf(...)`, but two **raw SQL queries never select the columns**, so they scan
as `0` and the helper falls back to the whole tax:

```sql
-- services/statement.go  loadStatementOrderRows
SELECT o.id, o.order_number, o.delivered_at, o.subtotal, o.tax, ...      -- no o.tax_food

-- handlers/chef_earnings.go
SELECT id, order_number, delivered_at, subtotal, tax, ...                 -- no tax_food
```

**Blast radius — 5 of 7 paths still wrong**, because `loadStatementOrderRows` is shared:

| Path | Effective? |
|---|---|
| `payout_release.go` — payout queue | ✅ explicit `Select` includes the columns |
| `statement_catchup.go` | ✅ explicit `Select` includes the columns |
| `statement.go` — **weekly statement, drives real payouts** | ❌ |
| `statement_pdf.go` | ❌ same loader |
| `fy_statement.go` | ❌ same loader |
| `tds_certificate.go` — **reports inflated gross** | ❌ same loader |
| `chef_earnings.go` — vendor screen | ❌ |

**This reaches money, not just display.** Severity is proportional to the fee+delivery GST on
every delivered order — currently ~1.4% of a typical order's gross.

**Why the tests missed it:** `observed_orders_test.go` exercises `ChefAttributableTax` directly,
never through a query. Adding a field to a scan struct is invisible to a test that never runs
the SQL.

### ✅ D-03 · ROOT-CAUSED AND FIXED (#988)

The head selection was a bare string equality, so an order with a **blank**
`delivery_address_state` never matched the chef's state and fell through to
IGST. Two of the three orders on the observed statement had no delivery state
at all. Verified in the DB: chef Karnataka; orders Karnataka, blank, blank.

No money moves — the total is 18% either way — but the head is what gets
reported and filed. Unknown now means intra, which is the right default for a
hyper-local service where the customer is near the kitchen by construction.

Original finding below.

### 🟥 D-03 · recurrence · intra-state supply labelled IGST

Vendor earnings shows **`GST (IGST) (18.0%) − ₹6.05`**. Chef state = Karnataka; all orders in
the period are Karnataka or blank (blank ⇒ intra by design). Should read **CGST + SGST**.
Unchanged from the July run.

### 🟥 D-01 · recurrence · "Free delivery" advertised, ₹39.12 charged

The chef card still reads **"Free delivery"** while the delivery mode charges **₹39.12**.
Unchanged from the July run.

### 🟥 D-10 · new · payment retry offered on an already-cancelled order

After a payment fails, the platform auto-cancels the order (`cancel_reason='payment not
completed'`) **but the app still offers "Retry payment"**. The retry cannot succeed — a
cancelled order can never be paid — and returns to the same screen. Infinite dead-end.

No money at risk (nothing is captured). Either suppress the CTA once cancelled, or have
retry mint a fresh order.

### ✅ D-11 · RESOLVED · real-time was dead on mobile, root cause found and fixed

Filed as #982, fixed in #983 + #984, deployed. Three independent defects stacked:

1. **Auth** — `useOrderTrackingWS` opened a socket with no credential at all; the
   sibling hook's fix was never applied to it. Every other client dialled
   `/api/v1/**`, the BFF path, which cannot carry a WS upgrade.
2. **Transport** — the one that actually mattered on device. React Native's
   WebSocket does TLS through SocketRocket/CFStream, and that handshake **fails
   outright** against our edge: close `1006`, `OSStatus -9836` (errSSLProtocol).
   No HTTP request is ever issued, which is why these attempts appear nowhere in
   the API logs. Mobile now uses SSE — same NATS stream, over NSURLSession.
3. **Render churn** — every caller passed `getToken` as an inline arrow, so
   `connect`'s `useCallback` was rebuilt each render and the effect tore the
   stream down and reopened it. ~19 stream opens per minute for one idle user.

Measured on the simulators against prod, both apps signed in, per 45s:

| | before | after |
|---|---|---|
| stream opens | 19 | **2** (one per app) |
| ticket mints | 14 | **0** |
| steady state | constant storm | **zero traffic** — streams held open |

`#892`, `#909`, `#910` and `#928` were four previous attempts, all tuning the
backoff curve. None could have worked: the socket failed at TLS before any of
that logic ran. The close code named the cause the whole time.

### 🟨 Observations (not filed)

| | |
|---|---|
| **`[tracking-ws]` failed 33 consecutive times** on one order before falling back to polling, and kept retrying in the background. `[order-ws]` and `[notif-ws]` likewise. 48 console errors accumulated in one session. | Order tracking silently degrades to polling. Verified from the in-app error log, not inferred. May be a simulator-to-prod networking artefact rather than a product defect — **confirm on a real device before filing**. Source: `useOrderTrackingWS.ts:92`. |
| Home header briefly showed a stale wallet balance (₹248.60 vs ₹237.66) | Corrects on navigation. Cosmetic. |
| **Vendor app cannot persist its session on this simulator build.** SecureStore/keychain access fails ("A required entitlement isn't present"), so the chef is signed out on every relaunch. | Simulator entitlement limitation, not a product defect — but it blocks unattended vendor-side test automation, which needs a manual sign-in per launch. |
| Terms checkbox is exposed to accessibility as `Slider` | Blocks assistive tech and UI automation; likely a missing `accessibilityRole` on the RN component. |

---

## Results

| ID | Verdict | Expected | Actual |
|---|---|---|---|
| ORD-01 | 🟩 pass | pickup: no delivery fee | `HC26080316331487` sub 320 · delivery **0** · fee 13.53 · tax 18.44 (food 16.00 + svc 2.44 + dlv **0**) · total 351.97. Pickup drops the fee *and* its ₹1.96 tax |
| ORD-02 | 🟩 pass | ₹39 + ₹8/km beyond 2 km | `HC26080323440359` delivery **39.12** = 39 + 8 × 0.015 km |
| ORD-03 | 🟩 pass | subtotal = Σ(price×qty) | Verified on the 4-item ₹1200 order earlier in the session |
| ORD-04 | ⬜ not run | below ₹199 rejected | |
| ORD-05 | 🟥 fail | advertised == charged | **D-01** |
| PAY-01 | 🟩 pass | funded once | `payment_status=completed`; gateway asked for **₹114.31** = total − wallet, exactly |
| PAY-02 | 🟩 pass | no funding, no hold, no leak | Failed payment ⇒ `cancelled/failed`, no hold, **no wallet debit**, balance unchanged |
| PAY-03 | 🟨 deviation | "order stays **pending**" | Order **auto-cancelled** instead. Nothing captured. Arguably better than the plan states; flagged because it contradicts the written criterion |
| PAY-04 | 🟥 fail | exactly one capture and one hold | **D-10** |
| PAY-05 | ⬜ not run | reconciliation | Needs the CRON |
| CHF-01 | 🟩 pass | hold persists, no payout | `accepted`, no hold, commission frozen at **0.06** |
| CHF-02 | ⬜ not run | reject ⇒ full refund | |
| CHF-03 | 🟩 pass | no money on status alone | preparing → ready → picked_up, total **393.05** unchanged, no hold. Mark-ready required a photo (stored) |
| CHF-04 | 🟩 pass | delivered ⇒ releasable | hold **awaiting_customer_confirmation** stamped at delivery |
| CAN-01 | 🟩 pass | 100% refund, chef ₹0 | Verified earlier: `HC26080316029688` refunded **1301.08**, retained **59.88**, conserves to 1360.96 |
| CAN-02..06 | ⬜ not run | | |
| REF-01 | 🟩 pass | refund == captured | Cashfree partial refund **₹1,063.42** = the card leg exactly |
| REF-04 | 🟩 pass | split sums exactly (INV-2) | card 1063.42 + wallet 237.66 = **1301.08** ✓ |
| REF-02/03/05 | ⬜ not run | | |
| WAL-01 | 🟩 pass | wallet delta == refund slice | wallet credited **237.66** |
| WAL-02 | 🟩 pass | capture == total − wallet | 393.05 − 237.66; wallet 237.66 → **0**, one debit txn with idempotency key |
| WAL-03..05 | ⬜ not run | | |
| LOY-01 | 🟩 pass | 0.1 × subtotal | **+32 points** at delivery (0.1 × 320) |
| LOY-05 | 🟩 pass | loyalty returned as wallet rupees | Confirmed in `wallet_txns`: `refund-loyalty:cancel:…` credited as wallet |
| LOY-02/03/04/06 | ⬜ not run | | |
| TIP-01 | 🟨 blocked | tip in gross, no commission | Could not reach the tip screen — three CTAs overlap at the same y with adjacent x, tap did not navigate |
| POU-01 | 🟩 pass | hold created | Stamped at **delivery**, not capture — matches the July run's own correction to this criterion |
| POU-02 | 🟥 fail | commission/GST/TDS per §3 | **D-09** and **D-03** |
| POU-03 | 🟩 pass | released after confirm | Confirm ⇒ **release_eligible**, `customer_confirmed_at` stamped |
| POU-04/05/06 | ⬜ not run | | POU-06 needs admin |
| REFR-01, GRP-*, MPL-* | ⬜ not run | | |
| CAN-05, POU-06 | ⬛ blocked | admin auth | Owner-only per the plan |
| POU-03 (auto) | ⬛ blocked | k8s CronJob | Owner-triggered |

---

## What is provably correct

The **pricing and refund model introduced by this release is sound**. Verified against live
transactions, predicted in advance and matched to the paisa:

- per-supply tax splits on every order (`food + delivery + fee` always sums to `tax`)
- Option B: all-in platform fee shown net, GST backed out of it
- delivery rate follows the **carrier** and is frozen at checkout — confirmed on
  `HC26080323440359`: `tax_delivery=1.96`, `tax_rate_delivery=5`, `tax_delivery_by_platform=f`
  held correctly through the chef's Mark-Ready self-delivery choice
- refunds attributed per supply, not proportionally (retained = fee + its GST, exactly)
- wallet/card rail split summing exactly to the refund
- money conservation on every order and refund examined
- two decimals everywhere

The defects found are in **reporting and settlement paths** (D-09, D-03) and **UX** (D-01,
D-10) — not in the pricing engine.

---

## Method and its limits

- Money assertions were read from **Postgres**, never from a screen.
- UI driven by `idb` against the two booted simulators.
- **Synthetic taps were unreliable on this app.** Several controls (the terms checkbox, "Tip
  your chef", "Back to home", "Report an issue") did not respond to taps that landed on their
  reported centre, while adjacent controls did. Where a control did not respond I have marked
  the scenario **blocked**, not failed — I cannot distinguish an inert control from a tap that
  did not register, and it would be wrong to file the former on the evidence for the latter.
- **The earlier "blocked" verdicts were a harness fault, not the app.** `idb`
  reports off-screen elements with their content coordinates, so taps landed on
  whatever was at that pixel. Adding `--duration` to the swipe fixed scrolling
  and the terms checkbox then reported `checkbox, checked` first try. TIP-01 and
  REF-03 are retestable, not blocked.
- The dev error overlay stacked 48 console errors and intercepted taps; it had to be cleared
  repeatedly. This is a development-build artefact and does not affect release builds.

---

## Recommended order of work

1. **D-09** — add `tax_food`/`tax_service` to both raw SQL column lists, and add a test that
   goes through the query rather than the helper. Chef payouts are wrong until this lands.
2. **D-03** — intra/inter resolution on the earnings GST head.
3. **D-01** — reconcile the "Free delivery" claim with the charged fee.
4. **D-10** — suppress retry on a cancelled order.

Unrelated but still open from earlier: `GatewayFeeLevyEnabled` is off, and the §9(5) invoice
attribution still names the chef as supplier of record for the food.
