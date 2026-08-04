# Test execution — 4 Aug 2026, post per-supply-GST release

> ## ▶ RESUME HERE (new session)
>
> **This file is the log. Record every finding here — do not open a second doc.**
> Plan/scenario definitions live in `.planning/TEST-EXECUTION-PLAN.md`.
>
> Standing instruction from the user: **record findings, do not fix them mid-run**
> unless asked. Money is asserted in Postgres, never read off a screen.

## Environment

| | |
|---|---|
| Customer sim | `226082A0-856A-4FE0-B000-B4DD891CA6E6` (right) |
| Vendor sim | `43FCB3B9-54C2-4A61-BB76-CB8667DE011E` (left) |
| Metro | customer **8082**, vendor **8081** — `npx expo start --port <p>` from each app dir |
| API | prod, `https://fe3dr.com/api` (vendor: `https://vendors.fe3dr.com/api/v1`) |
| DB | `kubectl exec -n homechef homechef-postgres-26 -c postgres -- psql -U postgres -d homechef_db` |
| Harness | `/tmp/hctest/snap.sh` (money snapshot), `/tmp/hctest/ui.sh` (`tap`/`text`/`shot`) |
| Test accounts | `customer01@fe3dr.com`, `vendor@fe3dr.com`. Email OTP is **111000**. |

**Postgres primary is `homechef-postgres-26`.** `snap.sh` still points at `-29`
(a replica — same data, but fix it if writes ever need verifying).

## Harness gotchas that cost time before

1. **`idb` reports off-screen elements with content coordinates.** Tapping their
   reported centre hits whatever is actually at that pixel. Scroll first, then
   re-read the tree. Swipe needs `--duration`: `idb ui swipe --udid X --duration 0.4 220 700 220 250`.
   A control that "does nothing" is almost always off-screen, not inert.
2. **`describe-all` can return a stale tree** right after a tap. Re-read before
   concluding a navigation failed.
3. **Cashfree sandbox parks card payments at `PENDING`** until the simulated bank
   page resolves, and the "Pay Now" favourite tile closes the WebView before that
   happens — so the order sits `pending` and used to auto-cancel (that auto-cancel
   was D-14, now fixed). **Do not fight the OTP page.** Force the outcome:
   ```bash
   APP=$(gcloud secrets versions access latest --secret=prod-homechef-cashfree-app-id --project=tesseracthub-480811)
   SEC=$(gcloud secrets versions access latest --secret=prod-homechef-cashfree-secret-key --project=tesseracthub-480811)
   # cf_payment_id from: GET https://sandbox.cashfree.com/pg/orders/<razorpay_order_id>/payments
   curl -s -X POST https://sandbox.cashfree.com/pg/simulate \
     -H "x-api-version: 2023-08-01" -H "Content-Type: application/json" \
     -H "x-client-id: $APP" -H "x-client-secret: $SEC" \
     -d '{"entity":"PAYMENTS","entity_id":"<cf_payment_id>","entity_simulation":{"payment_status":"SUCCESS"}}'
   ```
   The order settles within ~10s. Note the `entity_simulation` **nesting** — the
   flat form is rejected, and `entity` must be `PAYMENTS`, not `PAYMENT`.
4. **The vendor app loses its session on relaunch** and needs a manual sign-in
   (cause unverified — both builds are unsigned with no keychain entitlement, so
   it is NOT the entitlement difference I first assumed). **Do not terminate the
   vendor app.** All current code is hot-reloadable through Metro.
5. Dev error overlay intercepts taps. Dismiss it before driving the UI.

## Remaining scenarios

**CAN-02 is done and passed** (4 Aug, `HC26080403544039`) — see Results. Remaining,
money-critical, in the order worth running:

| ID | Scenario | Why it matters |
|---|---|---|
| CAN-03/04/06 | other cancellation tiers | same splitter, other percentages |
| CHF-02 | chef rejects → full refund | |
| REF-02/03/05 | partial refund via report-issue; reconciliation | REF-03 was mid-flight when the run paused |
| WAL-03/04/05 | wallet top-up, expiry, insufficient balance | |
| LOY-02/04/06 | loyalty expiry, reversal, cap | earn/redeem already pass |
| TIP-01 | tip in gross, no commission | previously mis-marked blocked — retestable |
| POU-04/05 | payout release + reversal | |
| ORD-04 | order below ₹199 rejected | no payment needed |
| REFR-01 | referral credit | no payment needed |
| GRP-01/02 | group orders | |
| MPL-01/02/03 | meal plans | |
| PAY-05 | payment reconciliation | needs the CRON |

Owner-only (cannot run unattended): **CAN-05**, **POU-06** (admin auth), and the
**POU-03 auto-path** (k8s CronJob).

### CAN-02 — prediction already recorded, assert against it

₹320 food + ₹39.12 delivery + ₹13.53 fee, tax 20.40 (food 16.00 / svc 2.44 /
dlv 1.96), total 393.05, at the `materials_purchased` **40%** tier:

| | paise | ₹ |
|---|---|---|
| Food refund | 32000 × 40% | 128.00 |
| Delivery refund (not dispatched) | 3912 | 39.12 |
| Tax refund | 1600×12800/32000 + 196 | **8.36** |
| **Customer receives** | 17548 | **175.48** |
| Vendor keeps | 19200 | 192.00 |
| Platform keeps | 2557 | 25.57 |

Platform-keep decomposes as fee 13.53 + svc GST 2.44 + unrefunded food GST 9.60.
**The old proportional model would refund ₹9.15 of tax** — handing back ₹0.79 of
GST on a fee that was kept. That ₹0.79 is what this release fixed; it is the
number to watch.

Flow: customer requests cancellation → **vendor approves and picks the reason**
(the reason selects the tier) → refund executes.

## Open defects

| | Status |
|---|---|
| 🟥 **D-01** "Free delivery" advertised, ₹39.12 charged | open |
| 🟥 **D-10** retry offered on an already-cancelled order | open |
| 🟨 **D-12** order detail shows Total ₹393.05 while the customer was charged ₹391.45 — the loyalty credit shown at checkout is missing from the receipt | open, found 4 Aug |
| ✅ D-09 chef GST over-credit | fixed #987, **verified live** (₹955.40) |
| ✅ D-03 GST head on unknown state | fixed #988 |
| ✅ D-11 real-time dead on mobile | fixed #983/#984, verified |
| 🟨 PAY-03 auto-cancel contradicts the written criterion | needs a product call |
| ✅ **D-13** analytics/dashboard revenue ≠ earnings | fixed, `fix/chef-revenue-reconciliation` |
| ✅ **D-14** order cancelled under a live gateway payment | fixed, same branch |

Everything above is deployed. API was on `main-1a98c35` at handoff.

---

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

### ✅ D-14 · new · CRITICAL · an order is cancelled out from under a live charge — FIXED

Found while three consecutive CAN-02 attempts stranded at `pending`. It looked
like a harness problem. It was not.

**What the gateway actually said.** Querying Cashfree directly for
`HC26080403544039`:

```
cf_payment_id 5114933571626 | PENDING | debit_card | ₹393.05 | is_captured: false
```

The API was right to answer 400 — there was no captured payment. The defect is
what happens next.

**The hole.** `stale_order_cron.go` asked the gateway one question — *is there a
CAPTURED payment?* — via `SuccessfulPayment`, which returns `nil` for PENDING
exactly as it does for "no attempt was ever made". Two opposite answers, one
value, and the sweep cancels on both. The same hole existed on the Razorpay leg,
where `capturedPaymentFor` returns `""` for **`authorized`** — money already held
on the customer's card.

PENDING is not a sandbox curiosity. It is where a card sits while the bank's
OTP/3DS page is open and where a UPI collect sits while the payer decides.

**Why it reaches money.** `order_payment_reconcile_cron.go` is forward-only *by
design* — its own header documents that it must never touch the set the stale
cron wrongly cancelled, because those need a **refund**, not a settle. So the
sequence is:

1. payment goes PENDING at the gateway
2. 30 min passes, the sweep reads "not captured", cancels the order
3. the payment resolves to SUCCESS
4. nothing recovers it — customer charged, order cancelled, `refund_amount = 0`

Both of today's earlier stranded orders (`…00184814`, `…00303019`) were cancelled
in exactly this state.

**Fix.** The probe is now tri-state — captured / in-flight / dead — and only
*dead* cancels. Any status neither gateway has shipped yet reads as in-flight:
the two mistakes are not symmetric, and waiting one more tick on a genuinely dead
attempt is free.

**Hardening on top (Temporal + NATS).** Whether an order got settled depended on
which of two sweeps happened to look at it and when. Added
`PaymentResolutionWorkflow`: a durable per-order poll that asks the gateway on a
backoff until the answer is terminal, settles through the *same* shared core, and
expires only on the gateway's own "dead". Its horizon is deliberately longer than
the stale cron's threshold and running out of it hands the order back **still
pending** — a timer must never be what cancels an order. Gated behind
`PAYMENT_RESOLUTION_ENABLED`, default off; the sweeps stay authoritative until
ops enables it. A payment unresolved past 10 min raises `payments.stalled` on the
transactional outbox (ops signal, not a customer notification — staged rather
than published direct so an activity retry cannot double-publish).

11 tests, verified to fail against the pre-fix code.

**It has already happened — twice.** I ran the audit rather than leaving it as a
follow-up. Of 31 orders matching the wrongly-cancelled shape (`cancelled` /
`failed` / `cancel_reason='payment not completed'` / gateway order id present /
`refund_amount = 0`), 25 are Cashfree. Asking Cashfree about each:

| Order | Gateway says | Our side |
|---|---|---|
| `HC26073102484828` | **SUCCESS ₹757.63** (credit_card, 31 Jul 08:19) | cancelled · failed · refund **0** |
| `HC26073113579923` | **SUCCESS ₹600.00** (debit_card, 31 Jul 19:28) | cancelled · failed · refund **0** |

The other 23 are genuine — `NONE` (no attempt) or `USER_DROPPED`. So the sweep
was right 23 times and wrong twice: **8% of cancellations landed on a payment
the gateway had taken.**

**No real customer money is involved.** Both rows are `mode=live`, but the live
Cashfree slot currently holds a **TEST** key (app id `TEST11…`), so these are
sandbox captures. That is the only reason this is a test-run finding rather than
an incident. Under production credentials the identical code path takes real
money and leaves it unrefunded, and the reconcile cron will not touch these rows
by design.

Today's two stranded orders (`…00184814`, `…00303019`) show `USER_DROPPED` — I
abandoned those sheets, so cancelling them was correct. The live one was
`HC26080403544039`, which sat at PENDING and would have been cancelled at the
30-minute mark had I not resolved it first.

**Harness note.** The Cashfree sandbox leaves card payments PENDING until the
simulated bank page resolves, and the "Pay Now" favourite tile closes the WebView
before that happens. Force the outcome instead of fighting the UI:

```
POST https://sandbox.cashfree.com/pg/simulate
{"entity":"PAYMENTS","entity_id":"<cf_payment_id>","entity_simulation":{"payment_status":"SUCCESS"}}
```

### ✅ D-13 · new · the chef's own screens report three different numbers — FIXED

Found by asking the obvious question the run had not yet asked: does the vendor
Analytics screen agree with the Earnings screen? It does not, and the gap is
structural, not a rounding artefact.

**The dashboard hero is labelled "Total earnings" and taps through to payouts,
but it summed `total − refund_amount` — the CUSTOMER's order value.** That carries
the delivery fee (the driver's), the platform fee, and the platform's own GST on
both. On the order this run already reconciled to the paisa:

| | |
|---|---|
| Customer charged (`total`) | ₹393.05 |
| Chef's gross earnings | **₹336.00** (320 food + 16.00 food GST) |
| Hero reported | ₹393.05 |

Three independent causes, all now fixed:

1. **Wrong basis.** `chefCountedRevenueExpr` is now the SQL twin of
   `ComputeOrderEarnings`' gross — food revenue net of chef-funded discount, plus
   `ChefTaxOf` (food GST only, with the pre-split fallback), plus the tip.
2. **Cancelled orders paid the chef the platform's kept share.** `total − refund`
   on the observed 40%-tier cancellation is ₹217.57, which is the chef's ₹192.00
   *plus the platform's ₹25.57*. The expression now reads `vendor_kept_paise` —
   the share epic #475 already persists and `ComputeCancellationEntitlement`
   already pays out — capped at what the platform still holds for the order. That
   cap is the same solvency test, and it matters: production carries a
   fully-refunded order whose snapshot still claims ₹40.53 for the vendor.
3. **Advanced analytics bypassed the shared scope entirely** — `SUM(total)` over
   `chef_id` alone, so sandbox orders and soft-deleted rows fed revenue-per-customer
   and best-day.

**And the Earnings screen itself was missing two guards the weekly statement has
always had:** it was unscoped by `mode` (a sandbox order counted toward a live
chef's earnings) and counted refunded orders (`status` stays `delivered` on the
order-issue refund path — filtering on status alone does not catch it, the same
trap #927 fixed in `statement.go`).

Effect on the test kitchen, measured in Postgres: lifetime hero **₹8,706.92 →
₹7,491.38**. This week's earnings gross is unchanged at **₹955.40** — the figure
already reconciled against the statement when D-09 was verified, which is the
point: the fix moves the wrong number onto the right one, not the other way.

Nine tests added to `chef_counted_orders_test.go`, each pinned to an order this
run actually observed.

### 🟨 D-12 · new · receipt cannot be reconciled against the charge

Order `HC26080400184814`: checkout displayed **Credits applied −₹1.60** and the
gateway correctly asked for **₹391.45**. The order detail's Price Breakdown then
shows **Total ₹393.05** with no loyalty line at all.

A customer comparing their bank charge to the receipt cannot make them agree.
The money is right everywhere — 32 loyalty points at ₹0.05 = ₹1.60, applied as a
payment RAIL rather than a discount (`loyalty_applied` is its own column, and
`total` is deliberately pre-rail) — but the receipt omits the rail it was paid
on. The wallet leg is shown on other orders; loyalty is not.

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
| **CAN-02** | 🟩 **pass** | policy % refund, sum reconciles | `HC26080403544039`, `materials_purchased` 40% tier. **Exact match to the prediction recorded before the run**: refund **17548** · vendor kept **19200** · platform kept **2557**. Σ = 39305 = the order total. `refund_executed=t`, order `cancelled/refunded`, `refund_amount 175.48` |
| **CAN-02 tax** | 🟩 **pass** | per-supply, not proportional | Refund 175.48 = food 128.00 (40% × 320) + delivery 39.12 + tax **8.36**. The old proportional model refunds ₹9.15 — **the ₹0.79 of GST on a retained fee is the thing this release fixed, and it is fixed** |
| **POU-04** | 🟩 **pass** | entitlement matches the snapshot | `chef_bonuses`: `cancellation_retained` **₹192**, `pending`, `source_key=cancelkept:38dd64e1…` — equals `vendor_kept_paise` exactly, paid via the statement, not by relaxing a payout-hold guard |
| CAN-03/04/06 | ⬜ not run | | re-run of the 3 Aug passes, post-release |
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
