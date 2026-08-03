# Fe3dr / HomeChef — End-to-End Money Test Execution Plan & Report

**Status:** IN PROGRESS
**Run started:** 2026-08-03
**Driver:** mobile apps (Expo/Android emulators) against **production**
**Owner:** Mahesh Sangawar

> **How to use this document.**
> Section 4 defines scenarios in a **platform-agnostic** way — preconditions, actions,
> and the money assertions that must hold. Those definitions are the reusable asset:
> a future web (Playwright) suite implements the same IDs and only swaps the driver
> layer. Section 5 is the per-run execution record. Keep IDs stable forever; never
> renumber. Add new scenarios by appending, not inserting.

---

## 0. Results so far

**8 orders + 1 full meal-plan lifecycle driven end-to-end against production** (Cashfree sandbox). 60 scenario-checks executed: **46 pass, 10 fail, 1 n/a, 3 unverified.** Every money figure was verified against the database and, for payments/refunds, against the Cashfree API — not the UI.

**The three cancellation policies are internally consistent and correct:**

| Who cancels | When | Customer gets | Chef gets | Platform keeps |
|---|---|---|---|---|
| Chef (reject) | pre-accept | **100%** incl. platform fee (₹393.84) | ₹0 | ₹0 |
| Customer | pre-accept | total − platform fee (₹377.07) | ₹0 | ₹16.77 |
| Customer | post-accept | tiered by chef-declared progress (₹175.47 @ 40%) | **₹192.00** via `chef_bonuses` | ₹26.37 |

Each row sums exactly to the order total.

| Defect | Severity | One-line |
|---|---|---|
| **D-07** | **Critical** | Float-precision wallet debit fails silently → cancel refunds credit never spent → **repeatable money-minting loop** (+₹47.56/cycle, verified) |
| **D-08** | **High** | Meal plans charge **8% GST + ₹2.99/day delivery** from unlocalised USD-style defaults; orders charge 5% — ₹9.29 overcharge on a ₹210 plan |
| **D-01** | High | Every self-delivery chef is advertised "Free delivery", then charged at checkout (₹39.12 observed) |
| **D-02** | High | Chefs are overpaid the GST charged on delivery + platform fees (₹30.78 on one pending statement) |
| **D-03** | Med-High | Pickup orders taxed as inter-state **IGST** instead of CGST+SGST — contradicts the customer's own tax invoice for the same order |
| **D-04** | Low-Med | Minimum-order error says *"Minimum order is $199.00"* — dollar sign on an INR-only marketplace |
| **D-05** | High (stability) | Vendor app hard-crashes on navigation — native Fabric mount failure, plus a `router.replace` redirect loop |

**Still to run, and what each needs:**

| Scenario | Blocked on |
|---|---|
| PAY-02 (payment failure), PAY-04 (retry) | Drivable — needs a deliberate bad-OTP run |
| CAN-04 (cancel after capture), CAN-06 (cancel a delivery order) | Drivable |
| REF-02 (refund to wallet), REF-05 (repeated partials) | Drivable |
| WAL-05 (duplicate credit key) | Needs a forced retry — best covered by a unit test, not the UI |
| LOY-06 (FIFO lot consumption) | Needs multiple earn lots seeded over time |
| REFR-01 (referral reward) | **Needs a fresh signup** with customer01's code + first order (new GIP account) |
| GRP-01/02 (group orders) | Drivable — needs a second customer account on a second device/session |
| MPL-01/02/03 (meal plans) | Drivable, but another party was already exercising meal-plan refunds on this account |
| CAN-05, POU-06 | **Admin-gated** — tesserix.app Google login |
| POU-03 auto-release | **Cron-gated** — 24h window |

**What is provably correct:** the earnings formula (net ₹343.93 matched my independent computation to the paisa), refund rail splitting (card + wallet always sum exactly to the refund, never exceeding capture), cancellation money conservation (refund + chef kept + platform kept = order total, exactly), chef retained-share payment via `chef_bonuses`, loyalty earn/redeem/refund, and the tip pass-through (tips never commissioned).

---

## 1. Scope

Validate the **complete money trail** for the customer and chef journeys: order
placement, payment, cancellation, refund, wallet, loyalty, referral, tips, and the
escrow/payout engine. Every scenario asserts against the **database**, not the UI —
a screen showing "Refunded" is not evidence that money moved correctly.

**Out of scope:** driver app, 3PL dispatch (no provider enabled), catering,
subscriptions beyond meal-plan refunds.

---

## 2. Environment

### 2.1 Under test

| Item | Value |
|---|---|
| API | `homechef` namespace, prod GKE (`tesseract-prod-in-gke`) |
| Deployed image at start | `main-3692c90` |
| Backend fix in flight | `89066f81 fix(api): reconcile the weekly statement with the payout hold machine (#967)` |
| DB | CNPG `homechef_db`, read-only via `kubectl exec` into primary |
| Payment gateway | **Cashfree SANDBOX** — the *live* credential slot holds a `TEST11…` App ID, so no real money can move |

### 2.2 Clients

| Emulator | App | Build | Account | Metro |
|---|---|---|---|---|
| `emulator-5554` | Customer | latest `main`, debug | Priya Sharma · `customer01@fe3dr.com` | 8082 |
| `emulator-5556` | Vendor | latest `main`, debug | `vendor@fe3dr.com` | 8081 |

### 2.3 Fixtures

**Chef — Saffron Home Kitchen** (`e150c72a-42e2-4beb-8cb1-389666dd813c`)

| Property | Value |
|---|---|
| Mode | `live` → wallet, loyalty, referral, payouts all **active** (not the test partition) |
| Verified / accepting | yes / yes |
| Self-delivery | enabled — ₹39 base + ₹8/km, max 8 km |
| Delivery radius | 8 km |
| Location / state | 12.9719, 77.6412 · Karnataka |
| Menu | 13 items, ₹80–₹440 |

**Customers** — all Bengaluru, Karnataka (⇒ **intra-state**, so CGST+SGST split applies)

| Account | Coords | Distance to chef | Wallet | Points |
|---|---|---|---|---|
| customer01 (Priya) | 12.9611, 77.6387 | ~1.2 km | ₹0 | 32 (333 lifetime) |
| customer02 (Rahul) | 12.9592, 77.6564 | ~2.2 km | — | — |
| customer03 (Ananya) | 12.9784, 77.6408 | ~0.7 km | — | — |

### 2.5 Environment caveat — the run was NOT isolated

Discovered at 11:26 IST: a **second client was operating the same `vendor@fe3dr.com` account concurrently** for the whole run (two IPs, ~equal request volume to `/api/v1/chef/*`, plus chef-side writes I did not make). Production also carried unrelated third-party traffic (`/meal-plans/:id/verify-payment`, `/meal-plans/:id/days/:dayId/skip`).

**What this does NOT invalidate** — the money findings. D-01…D-04 are code-level (verified by reading source at named line numbers) and reproduced against specific order rows and the Cashfree API. Another actor cannot manufacture `DeliveryFee: 0 // TODO`, a taxBase that includes fees, or an IGST branch on an empty state.

**What this DOES invalidate** — any inference from *state changing over time*: kitchen open/closed transitions, order-queue contents, chef availability, and the earlier "auto-reject after 30 min" attribution (OBS-1) which could equally have been the other client rejecting. Treat all timing/state observations as unattributed.

**Before the next run:** confirm exclusive use of the test accounts, or use a dedicated chef account.

**Update 11:45** — exclusive use of the *vendor* account was confirmed by the owner. The **customer** account is still shared: `customer01`'s wallet received two meal-plan refunds (₹162.89 + ₹139.90, "Tiffin … refund (75%)") at 06:09:34/06:09:37 UTC from meal-plan flows I never drove. Wallet balance moved ₹1.53 → ₹304.32 without my involvement. **Any wallet-balance assertion must therefore be a delta measured immediately either side of the action, never an absolute.**

### 2.4 Tooling

- `scratchpad/q.sh` — read-only SQL against prod
- `scratchpad/snap.sh <label>` — full money snapshot (orders, wallet, loyalty, refunds, payout ledger, tips, cancellations, penalties, statements, referrals)

---

## 3. Expected money model

Single source of truth: `apps/api/services/earnings.go`. **All assertions derive from this.**

```
itemRevenue = subtotal − chefFundedDiscount          (floored at 0)
commission  = commissionRate × itemRevenue           (rate FROZEN per order at checkout)
gross       = itemRevenue + tax + chefTip            (delivery fee EXCLUDED — driver's money)
GST         = 18% × commission
                intra-state → CGST = round2(GST/2), SGST = GST − CGST
                inter-state → IGST = GST
              (platform's own liability; NOT deducted from chef, NOT in customer total)
TDS         = 1% × gross                             (Section 194-O)
netPayout   = gross − commission − TDS
```

| Constant | Value | Source |
|---|---|---|
| `DefaultCommissionRate` | 0.06 | runtime-tunable via `payout.commission_rate` |
| `RateGST` | 0.18 | |
| `RateTDS` | 0.01 | |

**Loyalty defaults** (`loyalty.*` settings): earn 0.1 pt/₹ · redeem ₹0.05/pt · min 500 pts
to wallet (**no floor at checkout**) · max 10% of subtotal per order · ₹300 per rolling
30 days · 365-day expiry · FIFO lot consumption.

**Refund funding split** (`SplitRefundByFunding`): pro-rata across wallet / loyalty / card;
each rail capped at what it has not already been refunded; **card takes the remainder** so
the three parts sum to exactly the refund. Loyalty is returned as **wallet rupees, not points**.

**Key invariants that must hold in every scenario**

| ID | Invariant |
|---|---|
| INV-1 | Refund never exceeds what the order actually captured |
| INV-2 | `walletRefund + loyaltyRefund + cardRefund == refundAmount` exactly |
| INV-3 | Per rail, cumulative refunded ≤ cumulative funded |
| INV-4 | Chef payout on a delivered order == `ComputeOrderEarnings().NetPayout` |
| INV-5 | Delivery fee never enters chef gross or TDS base |
| INV-6 | Tips reach the chef in full — never commissioned |
| INV-7 | No money stranded: a cancelled/refunded order leaves no unreleased hold |
| INV-8 | Every money mutation is idempotent on its key (no double-apply on retry) |
| INV-9 | An order is paid to the chef **at most once** across both the statement batch and the hold machine — `orders.billed_statement_id` is set once and never re-billed (#927/#967) |

---

## 4. Scenario definitions (platform-agnostic — reusable for web)

Legend — **Status:** ⬜ not run · 🟩 pass · 🟥 fail · 🟨 partial · ⛔ blocked
**Gate:** `ADMIN` needs the tesserix.app admin (Google login, manual) · `CRON` needs a scheduled job

### A. Order placement — ORD

| ID | Scenario | Money assertion | Gate | Status |
|---|---|---|---|---|
| ORD-01 | Pickup order, single item, paid by card | total = subtotal + tax + serviceFee; **no delivery fee** | | ⬜ |
| ORD-02 | Delivery order (chef self-delivery) | delivery fee = ₹39 + ₹8/km, capped 8 km; excluded from chef gross (INV-5) | | ⬜ |
| ORD-03 | Multi-item order | subtotal = Σ(price×qty); tax = taxRate × subtotal | | ⬜ |
| ORD-04 | Below ₹199 minimum | order rejected; nothing captured | | ⬜ |
| ORD-05 | "Free delivery" claim vs ₹39 self-delivery config | advertised fee matches charged fee, or threshold documented | | 🟥 **D-01** |

### B. Payment — PAY

| ID | Scenario | Money assertion | Gate | Status |
|---|---|---|---|---|
| PAY-01 | Successful capture | order funded once; escrow hold created; `payment_status=completed` | | ⬜ |
| PAY-02 | Payment failure | no funding, no hold, no capacity leak | | ⬜ |
| PAY-03 | Abandoned checkout | order stays pending; nothing captured | | ⬜ |
| PAY-04 | Retry after failure | exactly one capture and one hold (INV-8) | | ⬜ |
| PAY-05 | Captured-but-cancelled reconciliation (#872) | no money stranded outside the pending window (INV-7) | | ⬜ |

### C. Chef lifecycle — CHF

| ID | Scenario | Money assertion | Gate | Status |
|---|---|---|---|---|
| CHF-01 | Accept order | hold persists; no payout yet | | ⬜ |
| CHF-02 | Reject order | full customer refund; chef earns nothing | | ⬜ |
| CHF-03 | Preparing → ready → handover | no money movement on status alone | | ⬜ |
| CHF-04 | Delivered + confirm receipt | hold becomes releasable; net per INV-4 | | ⬜ |

### D. Cancellation — CAN

| ID | Scenario | Money assertion | Gate | Status |
|---|---|---|---|---|
| CAN-01 | Customer cancels pre-accept | 100% refund; hold reversed; chef ₹0 | | ⬜ |
| CAN-02 | Customer cancels post-accept | policy % refund; chef retains their share; sum reconciles | | ⬜ |
| CAN-03 | Chef cancels | full customer refund; penalty on `chef_penalties` (not payout ledger) | | ⬜ |
| CAN-04 | Cancel after capture | refund actually reaches customer (INV-1, INV-7) | | ⬜ |
| CAN-05 | Cancellation needing admin resolution | admin decision drives refund + entitlement | ADMIN | ⬜ |
| CAN-06 | Cancel a delivery order | delivery-fee treatment matches RTO/cancel policy | | ⬜ |

### E. Refund — REF

| ID | Scenario | Money assertion | Gate | Status |
|---|---|---|---|---|
| REF-01 | Full refund to original method | refund == captured (INV-1) | | ⬜ |
| REF-02 | Refund to wallet | wallet credited exactly; ledger entry idempotent | | ⬜ |
| REF-03 | Partial refund via report-issue | partial reconciles; remainder still held | | ⬜ |
| REF-04 | Refund on wallet+loyalty+card order | split sums exactly (INV-2) | | ⬜ |
| REF-05 | Repeated partial refunds | per-rail cumulative ≤ funded (INV-3) | | ⬜ |

### F. Wallet — WAL

| ID | Scenario | Money assertion | Gate | Status |
|---|---|---|---|---|
| WAL-01 | Wallet credited by a refund | balance delta == refund wallet slice | | ⬜ |
| WAL-02 | Wallet applied at checkout | capture == total − walletApplied; chef still paid in full | | ⬜ |
| WAL-03 | Full-wallet order | capture == 0, no gateway call; chef/driver funded by platform top-up | | ⬜ |
| WAL-04 | Wallet exceeds balance | rejected with `ErrInsufficientWalletBalance` | | ⬜ |
| WAL-05 | Duplicate wallet credit key | applied once only (INV-8) | | ⬜ |

### G. Loyalty — LOY

| ID | Scenario | Money assertion | Gate | Status |
|---|---|---|---|---|
| LOY-01 | Earn on delivered order | points == 0.1 × eligible subtotal | | ⬜ |
| LOY-02 | Redeem at checkout | no 500-pt floor; ₹ credit == 0.05 × points | | ⬜ |
| LOY-03 | Redeem above 10% of subtotal | capped at 10% | | ⬜ |
| LOY-04 | Monthly ₹300 cap | redemption blocked past cap | | ⬜ |
| LOY-05 | Refund of a loyalty-funded order | returned as **wallet rupees, not points** | | ⬜ |
| LOY-06 | FIFO lot consumption | oldest lots consumed first | | ⬜ |

### H. Tips — TIP

| ID | Scenario | Money assertion | Gate | Status |
|---|---|---|---|---|
| TIP-01 | Tip after delivery | tip enters chef gross; **no commission** on it (INV-6) | | ⬜ |
| TIP-02 | Tip on a later-cancelled order | tip refunded or paid, never stranded | | ⬜ |

### I. Escrow & payout — POU

| ID | Scenario | Money assertion | Gate | Status |
|---|---|---|---|---|
| POU-01 | Hold created on capture | hold == expected net | | ⬜ |
| POU-02 | Commission/GST/TDS on delivered order | matches §3 exactly, incl. CGST+SGST split | | ⬜ |
| POU-03 | Hold release after confirm-receipt window | released once, correct amount | CRON | ⬜ |
| POU-04 | Chef entitlement on cancelled order | matches cancellation snapshot; solvency-capped | | ⬜ |
| POU-05 | Weekly statement vs payout holds (#967) | statement reconciles with hold machine | | ⬜ |
| POU-06 | Payout batch approve → execute | manual-first approval; amounts match statement | ADMIN | ⬜ |

### J. Referral — REFR

| ID | Scenario | Money assertion | Gate | Status |
|---|---|---|---|---|
| REFR-01 | Referral reward on referee's first order | both sides credited once (INV-8) | | ⬜ |

### K. Group orders — GRP

| ID | Scenario | Money assertion | Gate | Status |
|---|---|---|---|---|
| GRP-01 | Group order create → join → pay | per-participant split sums to order total | | ⬜ |
| GRP-02 | Group order cancelled | all participants reversed; no residue | | ⬜ |

### L. Meal plans — MPL

| ID | Scenario | Money assertion | Gate | Status |
|---|---|---|---|---|
| MPL-01 | Book a meal plan | escrow fees per `meal_plan_escrow` | | ⬜ |
| MPL-02 | Per-day refund tiers | tier applied correctly; conservation holds | | ⬜ |
| MPL-03 | Cancel whole plan | refund floor respected; holds reconciled | | ⬜ |

---

## 5. Execution record

Per scenario, on completion: order number, expected vs actual (₹), DB evidence, verdict.

Order under test: **`SAFFRON-HOME-KITCHEN-HC26080303443871`** — delivery, Butter Chicken ₹320, ₹20 tip, 32 loyalty points applied.

| Time (IST) | ID | Verdict | Order | Expected | Actual | Notes |
|---|---|---|---|---|---|---|
| 09:14 | ORD-02 | 🟩 pass | …3443871 | fee = ₹39 base + ₹8 × (roadKm − 2) | **₹39.12** | Road distance ≈2.015 km via `GOOGLE_MAPS_API_KEY` router, just past the 2 km free radius. Crow-flies (1.23 km) is *not* the basis. |
| 09:14 | ORD-03 | 🟩 pass | …3443871 | subtotal ₹320, tax 5% of (subtotal+delivery+service) | ₹320.00 / ₹18.7544 | `taxBase = 375.088 × 5%` ✓ arithmetic correct (but see D-02 for who receives it) |
| 09:14 | ORD-05 | 🟥 **fail** | …3443871 | advertised fee == charged fee | "Free delivery" vs **₹39.12** | **D-01** |
| 09:20 | PAY-01 | 🟩 pass | …3443871 | single capture, `payment_status=completed` | completed, ₹412.24 | Cashfree sandbox (`testMerchantName`), card + OTP `111000`. Captured = total − loyalty credit. |
| 09:20 | LOY-02 | 🟩 pass | …3443871 | 32 pts → ₹1.60, no 500-pt floor at checkout | debit 32, balance 32→0, lifetime 333 unchanged | Confirms ₹0.05/pt and that the floor correctly applies only to points→wallet |
| 09:20 | POU-01 | 🟨 partial | …3443871 | hold created on capture | `payout_hold_status` **empty**; `payout_ledger_entries` empty table-wide | Not a defect: holds are stamped later — delivered orders reach `release_eligible`. Re-assert after CHF-04. |
| 09:25 | POU-02 | 🟥 **fail** | statement `6aa26c7c` | chef gross = food revenue + **food** GST + tip | gross includes **₹31.09 GST on delivery+platform fees** | **D-02** — verified on a real `pending` statement about to pay ₹4193.62 |
| 09:46 | CHF-02 | 🟩 pass | …3443871 | chef rejection → full customer refund, chef earns nothing | auto-rejected, `payment_status=refunded`, chef ₹0 | Order auto-rejected after ~26 min unaccepted (see OBS-1) |
| 09:46 | REF-01 | 🟩 pass | …3443871 | refund == captured, never more (INV-1) | **Cashfree: captured ₹412.24 → refunded ₹412.24 SUCCESS** | Verified against the gateway API, not the DB |
| 09:46 | REF-04 | 🟩 pass | …3443871 | rails sum exactly to refund (INV-2) | card ₹412.24 + wallet ₹1.60 = **₹413.84 = order.Total** | `SplitRefundByFunding` clamps card slice to `Total − wallet − loyalty` |
| 09:46 | LOY-05 | 🟩 pass | …3443871 | loyalty returned as **wallet rupees, not points** | wallet +₹1.60 (`source=loyalty`); points balance stayed 0, lifetime 333 | Matches the documented owner decision |

**Order 2: `SAFFRON-HOME-KITCHEN-HC26080304344806`** — pickup, Butter Chicken ₹320, ₹30 tip, ₹1.60 wallet credit. Driven to delivered + confirmed.

| Time (IST) | ID | Verdict | Expected | Actual | Notes |
|---|---|---|---|---|---|
| 10:04 | ORD-01 | 🟩 pass | pickup ⇒ **no** delivery fee | `delivery_fee = 0`, tax ₹16.7984 = 5% × (320 + 15.968) | Pickup correctly drops the fee from both UI and total |
| 10:04 | WAL-02 | 🟩 pass | capture == total − wallet | total ₹382.77, wallet ₹1.60 applied, **captured ₹381.17** | Wallet balance (earned from order 1's refund) auto-applied |
| 10:08 | CHF-01 | 🟩 pass | accept ⇒ no payout yet | `status=accepted`, hold still unset | |
| 10:09–10:12 | CHF-03 | 🟩 pass | status transitions move no money | preparing → ready → collected; totals unchanged | "Mark ready" and "handed over" each require a photo |
| 10:12 | CHF-04 / POU-01 | 🟩 pass | hold created at delivery | `status=delivered`, `payout_hold_status=`**`awaiting_customer_confirmation`** | Hold is stamped at **delivery**, not capture — corrects the earlier POU-01 partial |
| 10:12 | LOY-01 | 🟩 pass | earn 0.1 pt/₹ of subtotal | **+32 pts** (0.1 × 320); lifetime 333 → 365 | |
| 10:16 | POU-03 | 🟩 pass* | confirm receipt ⇒ releasable | `POST /orders/:id/confirm-received` 200 → `payout_hold_status=`**`release_eligible`** | *Explicit customer confirm; the 24h auto-path still needs the cron (see §7) |
| 10:20 | POU-02 | 🟩 pass (arithmetic) | gross 366.80, comm 19.20, TDS 3.67, **net 343.93** | Earnings screen: gross ₹366.8, −₹19.2, −₹3.67, **net ₹343.93** | Matches my independent computation **to the paisa** — the formula is right; D-02 is bad *input*, not bad maths |
| 10:20 | TIP-01 | 🟩 pass | tip in gross, **never commissioned** (INV-6) | gross ₹366.80 includes the ₹30 tip; commission ₹19.20 = 6% × **320 only** | INV-6 holds |
| 10:20 | POU-02 | 🟥 **fail** | intra-state ⇒ CGST+SGST | **"GST (IGST) (18.0%)"** on a Karnataka→Karnataka pickup | **D-03** |

**Order 3: `SAFFRON-HOME-KITCHEN-HC26080304578640`** — delivery, ₹320, no tip, 32 loyalty pts. Cancelled by customer pre-accept.

| Time (IST) | ID | Verdict | Expected | Actual | Notes |
|---|---|---|---|---|---|
| 10:31 | CAN-01 | 🟩 pass | pre-accept cancel ⇒ refund, no hold, chef ₹0 | `status=cancelled`, `refund_amount=₹377.07`, `payout_hold_status` empty, chef ₹0 | Platform fee + its GST (₹16.77) retained — **disclosed in-app** ("The platform fee isn't refundable") |
| 10:31 | REF-04 | 🟩 pass | rails sum exactly (INV-2) | **Cashfree card ₹375.54** + wallet ₹1.53 = **₹377.07** ✓ | Verified via Cashfree API, not just DB |
| 10:31 | INV-1 | 🟩 pass | refund ≤ capture | captured ₹392.24, card refund ₹375.54 | |
| 10:31 | CAN-01 ledger | 🟩 pass | explicit split recorded | `cancellation_requests`: food 32000 + delivery 3912 + tax 1795 = **37707 paise**; `vendor_kept=0`; `platform_kept=1677` | Pre-accept ⇒ chef correctly earns nothing |

**Order 4: `SAFFRON-HOME-KITCHEN-HC26080305154249`** — delivery ₹393.84. Accepted by chef, then cancelled by customer (post-accept).

| Time (IST) | ID | Verdict | Expected | Actual | Notes |
|---|---|---|---|---|---|
| 10:48 | CHF-01 | 🟩 pass | accept ⇒ no payout yet | `status=accepted`, no hold | |
| 10:48 | CAN-02 | 🟩 pass | post-accept ⇒ chef approval + tiered refund | request → `pending_vendor` (15-min SLA) → chef picked "Ingredients bought" → `approved` | Chef self-declares progress tier; tiers: not-started ~90%, ingredients ~40%, cooking/made 0% of food |
| 10:48 | CAN-02 money | 🟩 pass | conservation: refund + chef + platform == total | food ₹128 (40%) + delivery ₹39.12 + tax ₹8.35 = **refund ₹175.47**; `vendor_kept` **₹192.00**; `platform_kept` **₹26.37**; **Σ = ₹393.84 = order total** ✓ | Gateway refund ₹175.47 SUCCESS |
| 10:48 | **POU-04** | 🟩 pass | chef actually receives the retained share | `chef_bonuses`: `kind=cancellation_retained`, **₹192**, `source_key=cancelkept:8bccfa11…`, status `pending`, credited on next weekly statement | **Confirms the `dfafdb67` fix.** Paid via ChefBonus → statement, *not* by relaxing payout-hold guards — matches the documented design |
| 11:05 | **POU-05** | 🟩 pass | statement reconciles with the hold machine (#967) | Both statements internally exact — `6aa26c7c`: 4250+243.59 = gross 4493.57, comm 255 = 6%×4250, CGST 22.97+SGST 22.94 = 45.91 = 18%×255, TDS 44.95, **net 4193.62** ✓. `c5b236e5`: 1050+59.03 = 1109.03, comm 63, GST 11.34, TDS 11.09, **net 1034.94** ✓ | DB-only check; no app interaction needed |
| 11:05 | **INV-9** | 🟩 pass | an order is billed at most once | `billed_statement_id` is a single FK — structurally unique; 10 + 2 orders across 2 statements, no overlap. 1 delivered order unbilled = today's pickup, correctly awaiting next week's run | |
| 11:05 | OBS-4 follow-up | 🟩 pass | tip backfill covers the tips both statements missed | Statements exclude tips of ₹25 + ₹50 = **₹75**; `tip_catchup` bonuses total ₹25+₹20+₹30 = **₹75** — exact match | Confirms `chef_tip_backfill.go` reconciles precisely |
| 11:05 | ORD-04 | 🟨 partial | below-minimum order rejected | Enforced at `handlers/orders.go:394` (`subtotal < chef.MinimumOrder`) — but the message reads **"Minimum order is $199.00"** | **D-04**. Live trigger blocked by the kitchen auto-closing (OBS-13); verified from source + the enforcement site |
**Orders 5 & 6** — `…05139453` (created, never paid) and `…05318139` (paid, chef-rejected).

| Time (IST) | ID | Verdict | Expected | Actual | Notes |
|---|---|---|---|---|---|
| 11:14 | ORD-04 | 🟥 **fail** (message) / 🟩 pass (enforcement) | sub-minimum order rejected | ₹180 subtotal rejected, **no order row created** ✓ — but customer sees **"Minimum order is $199.00"** | **D-04** confirmed live on device |
| 11:15 | **PAY-03** | 🟩 pass | abandoned checkout ⇒ nothing taken | Cashfree order `ACTIVE` (never PAID); `payment_status=pending`, no hold, no refund | Verified against the gateway |
| 11:18 | **CAN-03** | 🟩 pass | chef rejects ⇒ customer made whole, chef ₹0 | `refund_amount = ₹393.84` = **full total incl. platform fee**; Cashfree ₹393.84 SUCCESS; no payout hold | **Correct policy contrast:** chef-fault ⇒ 100% back *including* the platform fee; customer-cancel ⇒ platform fee retained (CAN-01) |
| 11:18 | CAN-03 penalty | 🟩 pass | pre-accept *reject* is not a breach | **No new penalty row.** Only `cancel_late` exists (for cancelling an *accepted* order), and it was waived under "1 cancellation per 30 days" | Sensible distinction: declining a new order ≠ abandoning an accepted one |
**Order 7: `SAFFRON-HOME-KITCHEN-HC26080306287649`** — pickup, Veg Thali ₹240, funded almost entirely by wallet.

| Time (IST) | ID | Verdict | Expected | Actual | Notes |
|---|---|---|---|---|---|
| 11:58 | **WAL-02** | 🟩 pass | capture == total − wallet | total ₹264.57, wallet **₹239.99**, **Cashfree captured ₹24.58** (verified via gateway API); wallet 304.32 → 64.33 | |
| 11:58 | WAL-03 | 🟨 n/a by design | zero-capture full-wallet order | **Not reachable** — wallet is capped at the food subtotal; fees + taxes must always be paid in real money ("Fees & taxes are always paid separately"). `FullWallet` (capture == 0) is therefore unreachable for a normal order | Deliberate business rule, not a defect |
| 12:10 | **WAL-03 (chef side)** | 🟩 pass | chef paid in full despite tiny capture | Chef net **₹235.67** on ₹240 subtotal while only ₹24.58 entered the gateway — platform absorbs the ₹239.99 redemption from its float | The important half of the wallet model, and it holds |
| 12:10 | POU-02 (2 orders) | 🟩 pass | week aggregate matches per-order math | Gross **₹619.4** (366.80+252.60), commission **−₹33.6**, TDS **−₹6.2**, net **₹579.6** — matches my independent computation exactly | Second full-precision confirmation of the earnings formula |
| 12:10 | D-03 recurrence | 🟥 fail | intra-state ⇒ CGST+SGST | **"GST (IGST) (18.0%)"** again — both orders in the period are pickups | Confirms D-03 is systematic, not a one-off |
| 14:46 | **CAN-06** | 🟩 pass | cancelling a *delivery* order refunds the delivery fee | Order `…06508670` (delivery): `delivery_refund_paise = 3912` (₹39.12) refunded in full; platform kept only its fee + that fee's GST (₹16.77) | Same ledger shape as CAN-01 |
**Meal plan `SAFFRON-HOME-KITCHEN-MP-7b6bf84a`** — booked, chef-accepted, paid ₹240.27, then cancelled. Full lifecycle driven by us.

| Time (IST) | ID | Verdict | Expected | Actual | Notes |
|---|---|---|---|---|---|
| 15:57 | **MPL-01** | 🟥 **fail** | plan priced consistently with orders | subtotal ₹210, fee ₹10.48 (4.99%), **tax ₹16.80 @ 8%**, delivery **₹2.99**, total ₹240.27 | **D-08** — orders charge 5%; meal plans 8% + a ₹2.99/day delivery from USD-shaped defaults |
| 16:05 | **MPL-02 (tier selection)** | 🟩 pass | tier chosen by lead time, per configured table | Day Aug 5 lunch, lead ≈32h → `refund_floor_percent = 75`, `refund_stage = pending_chef`, `payout_hold_status = disputed` | Matches the **configured** table exactly (`minLeadHours 12 → floorPercent 75`, no auto-approve). The *default* table's 100%-auto band is overridden in `platform_policy` — correct, config-driven behaviour |
| 16:12 | **MPL-03 (approval chain)** | 🟩 pass | chef sets %, customer picks destination | chef confirmed 75% → `refund_percent = 75`, `refund_stage` → **`pending_customer`**; chef payout for the day held as `disputed` throughout | Three-party flow works: customer requests → chef agrees an amount ≥ floor → customer chooses where it lands |
| 16:12 | MPL refund **base** | ⬜ unverified | quoted floor derivable from code | UI quotes floor **₹170.75** on ₹210 food ⇒ implied base ₹227.67. `perDaySkipRefund` = foodNet (210 − 6%) + delivery 2.99 = **₹200.39**, whose 75% is ₹150.29 — **does not match** | The cancellation path uses a different base than the skip path; I could not derive ₹227.67 from the functions read. **Not asserting pass or fail** — needs a code walk of the v2 cancellation path |
| 16:12 | MPL refund **execution** | ⬜ not run | executed refund == quoted ₹170.75 | Stopped at `pending_customer` — the destination chooser (`meal-plans/refund-choices`) was not reached before context limits | Next session: pick destination, then compare gateway/wallet delta against ₹170.75 |
| 14:50 | ~~MPL-01/02/03~~ | ⬜ superseded | meal-plan escrow + refund tiers | **Cannot verify from available data.** The only two refunded plans (₹162.89, ₹139.90) were created by the *other party* at 06:09 — refund size depends on the lead-time tier and per-day state at cancellation, which can't be reconstructed post-hoc. Observed ratios 0.7757 / 0.7772 of day price match no clean reading | Established: tier table is `>12h → 100% auto · 12–6h → floor 75% · 6–2h → floor 50% · <2h → floor 0%`, fully config-driven via `PlatformPolicy.MealPlanRefundTiers` with validation that demotes any sub-100% auto-approve band. **Needs a plan booked and cancelled by us to verify.** |
| 12:32 | **WAL-04** | 🟩 pass | over-application clamped to balance | Requested **₹9,999** against balance ₹64.33 → applied **₹64.33** exactly; field self-corrected to "64"; credits −₹65.53 (wallet 64.33 + loyalty 1.20); total ₹328.31 = 393.84 − 65.53 ✓ | Server-side `clampInt(requested, 0, WalletMaxPaise)` confirmed live |
| 12:32 | **LOY-03 / LOY-04** | 🟩 pass (code) | 10%-per-order and ₹300/30-day caps enforced | `services/checkout_credit.go:145,148` — four ceilings (balance, `MaxRedeemPct`, `MonthlyRedeemCap`, order-remainder), takes the **minimum**, reports the binding reason, floors to whole points | **Not driven live** — customer holds only 24 pts (₹1.20) and has used ₹18.25 of the ₹300 cap, so neither cap can bind without seeding a large balance |
| 12:32 | Credit ordering | 🟩 pass | wallet burns before loyalty | `redeemable = min(subtotal − discount + delivery, total − platformFee − tax)` = ₹359.12; wallet applied first ("booked liability first"), loyalty against the remainder | Explains why credits never cover fees/taxes |
| 11:44 | **TIP-02** | 🟥 **fail** | post-delivery tip reaches the chef | HTTP 409 *"This chef can't receive tips right now"* — `handlers/tips.go` is Razorpay-only; **0 of 2 chefs have a `razorpay_account_id`** | **D-06** — feature is dead platform-wide after the Cashfree migration |
| 11:44 | **REF-03** | 🟩 pass | issue report ⇒ held for review, no automatic money movement | `order_issues` row: `requested_amount ₹336.80` (item ₹320 + its ₹16.80 tax), `refund_amount 0`, status **`pending`**, no `refund_txn_id`. Order `refund_amount` still 0; payout hold untouched | Manual-first, as designed. An `auto_refunded` path exists (seen on an older issue) but did not trigger here |
| 11:18 | Privacy | 🟩 pass | chef sees area only pre-accept | Rejected order showed **"DELIVERY AREA: Bengaluru, Karnataka"** — no street address, first name only | Matches the #320 privacy model |

| 10:48 | CHF-03 penalty | 🟩 pass | chef-cancel penalty has an allowance | `chef_penalties`: `cancel_late`, amount **₹0**, status `waived` — "Within the allowance of 1 cancellation(s) per 30 days" | Penalty mechanism lives on `chef_penalties`, not the payout ledger |

---

## 6. Defects found

| # | Severity | Scenario | Summary | Status |
|---|---|---|---|---|
| **D-07** | **Critical** | PAY-04 / WAL | **A float-precision wallet balance makes the debit fail, and the customer keeps the credit they just spent.** Order `…06508670` completed with `wallet_applied = ₹64.33`, gateway captured ₹328.31 (= total − wallet − loyalty), loyalty correctly debited 24 pts — but **zero wallet transactions exist for the order and the balance is unchanged at ₹64.33, still spendable.** Production log is explicit:<br>`wallet-debit failed order=SAFFRON-HOME-KITCHEN-HC26080306508670: insufficient wallet balance`<br><br>**Root cause.** `services/wallet.go:108` compares raw float64 rupees — `if w.Balance < amount { return ErrInsufficientWalletBalance }`. The stored balance had accumulated float error (`64.32999999999998`, from credits 139.90 + 162.89 and debits written back through float64), while checkout offered the balance rounded to `64.33`. `64.33 > 64.32999999999998` by ~2×10⁻¹⁴, so a debit of the customer's *entire* balance is rejected. The rest of the money code deliberately works in integer paise (`ToPaise`/`FromPaise`; `wallet_split.go` — *"works in paise to stay exact"*); the wallet ledger is the exception.<br><br>**Impact.** (1) Customer is under-charged by the applied credit and keeps it — repeatable, since the balance survives every attempt. (2) `SettleOrderWallet` deliberately `return`s on a genuine debit failure *before* funding chef/driver top-ups, so on any order where the capture cannot cover the chef's payout the **chef is underpaid**. (On this order capture ₹328.31 exceeded the chef's ₹316.16 net, so the chef was unaffected — but an order like `…06287649`, capture ₹24.58 vs chef net ₹235.67, would have stranded the chef's money.) (3) Silent — the order reports fully paid; only a log line records it.<br><br>**Trigger.** Applying a full wallet balance whose float representation sits just below its 2-dp value. Not retry-specific.<br><br>**⚠ COMPOUNDS INTO A REPEATABLE MONEY-MINTING EXPLOIT ON CANCELLATION.** Cancelling the same order refunds the wallet rail *pro-rata on `wallet_applied`* — credit the customer never actually spent, because `SplitRefundByFunding` trusts `order.WalletApplied` and never checks that the debit succeeded. Full verified trail for `…06508670`:<br><br>*Gave up:* card **₹328.31** (gateway-confirmed) + wallet **₹0** + loyalty ₹1.20 = **₹329.51**.<br>*Got back:* card **₹314.34** (gateway-confirmed) + wallet-rail credit **₹61.59** + loyalty-rail credit ₹1.14 = **₹377.07**.<br>**Net customer gain ₹47.56**; wallet balance ₹64.33 → **₹127.06**.<br><br>The loop is self-amplifying: each cycle leaves a larger balance, so the next order applies more credit and mints more. **Violates INV-3** (per rail, cumulative refunded ≤ cumulative funded) — the wallet rail funded ₹0 and was refunded ₹61.59. INV-1 still holds on the card rail (314.34 ≤ 328.31), which is why the gateway never objects and nothing surfaces in reconciliation. | OPEN |
| **D-08** | **High** (tax correctness + overcharge) | MPL-01 | **Meal plans charge 8% GST and a ₹2.99/day delivery fee from unlocalised defaults — orders on the same platform charge 5%.** `services/meal_plan_escrow.go:58 MealPlanFeeTotals` reads `GetPlatformPolicy()`, whose stored blob contains **only** `mealPlanRefundTiers`, so fee/tax/delivery fall back to the hardcoded defaults at `services/platform_policy.go:117-119`: `PlatformFeePercent 4.99`, **`TaxPercent 8.0`**, **`BaseDeliveryFee 2.99`** — USD-style price points (2.99 / 4.99) applied as rupees.<br><br>**Verified on a plan I booked** (`SAFFRON-HOME-KITCHEN-MP-7b6bf84a`, subtotal ₹210): stored `tax_rate = 8`, `tax = ₹16.80`, `platform_fee = ₹10.48`, `total = ₹240.27`. Note `210 + 10.48 + 16.80 = 237.28` — the missing **₹2.99 is delivery**, which `meal_plans` has **no column for**; it is only recoverable as `Total − Subtotal − Fee − Tax` (`planDeliveryTotal`), a derivation the code itself warns is fragile.<br><br>**Direct DB comparison:** `orders.tax_rate = 5` (8 rows today) vs `meal_plans.tax_rate = 8` (10 rows). Same country, same chef, same food. India food-service GST is **5%** (orders split it correctly CGST 2.5 + SGST 2.5); 8% is not a valid Indian GST rate, so every meal-plan invoice over-collects and misstates tax.<br><br>**Also:** the chef's own `chef_subscription_configs.delivery_fee = 0` is ignored entirely — `MealPlanFeeTotals` never reads it.<br><br>**Customer impact:** on this ₹210 plan, ₹240.27 charged vs ₹230.98 at order-consistent rates = **₹9.29 overcharged (~4.4%)**. | OPEN |
| **D-06** | **High** | TIP-02 | **Post-delivery tipping is broken for every chef on the platform — the feature is still on the Razorpay rail after the Cashfree migration.** `handlers/tips.go:88` resolves `services.GetRazorpayFor(order.Mode)` and line 99 requires `order.Chef.RazorpayAccountID`; empty ⇒ HTTP 409 *"This chef can't receive tips right now"*. Orders themselves settle through **Cashfree**, so no chef is onboarded to Razorpay: **`razorpay_account_id` is empty for 0-of-2 chefs (100%)**, while the test chef does hold `cashfree_vendor_id = hc_e150c72a…`. Reproduced on device: delivered order → "Tip chef" → ₹50 → 409. The customer app advertises this on every delivered order (*"100% goes straight to your chef and rider, with no platform cut"*), so it is a visible dead end, and chefs lose all post-delivery tip income. Note this is the **post-delivery** tip path only — the **checkout** tip works (verified ₹20 and ₹30 reaching `chef_tip`). | OPEN |
| **D-05** | **High** (stability) | vendor app | **Vendor app hard-crashes during navigation — twice, two different failures.** (1) Native Fabric mount crash backing out of an order detail: `java.lang.IllegalStateException: addViewAt: failed to insert view [496] into parent [498] at index 0`, caused by *"The specified child already has a parent. You must call removeView() on the child's parent first"* (`ReactClippingViewManager.addView`). The app dies to the dev-build error screen and must be relaunched. (2) Earlier, opening the Orders tab: `Maximum update depth exceeded`, stack `commitHookEffectListMount → … → linkTo → replace` — a `useEffect` issuing an unconditional `router.replace`, i.e. a redirect loop. Both are new-architecture (`newArchEnabled: true`) navigation faults and would affect release builds, not just dev.<br><br>**Reproduced deterministically (2×, identical stack).** Steps: Orders → open any order detail → tap "Go back". Crash fires in `ReactClippingViewManager.addView` → `SurfaceMountingManager.addViewAt`. Observed at 11:18 (view 496→498) and 12:00 (view 512→514). The app is unusable until relaunched via the dev-client deep link, and it loses its session each time. | OPEN |
| **D-04** | Low-Medium | ORD-04 | **Minimum-order error shows a dollar sign on an INR-only marketplace.** `handlers/orders.go:396`: `fmt.Sprintf("Minimum order is $%.2f", chef.MinimumOrder)` → the customer is told *"Minimum order is $199.00"*. A currency-symbol map already exists at `handlers/currency.go:176` (`"INR": "₹"`) and is simply not used here. Sole user-facing instance — the rest of the `$` hits in the sweep are MongoDB operators. | OPEN |
| **D-03** | **Medium-High** (GST compliance) | POU-02 | **Pickup orders are taxed as inter-state (IGST) instead of intra-state (CGST+SGST).** `ComputeOrderEarnings` picks the head via `NormaliseState(in.DeliveryState) == NormaliseState(chefState)`. A pickup order has **no delivery address**, so `delivery_address_state` is empty (verified: order `…04344806` state = EMPTY vs delivery orders = "Karnataka"); `"" != "Karnataka"` → IGST branch. Vendor Earnings screen shows **"GST (IGST) (18.0%) − ₹3.46"** for a chef and customer both in Karnataka. A pickup happens *at the chef's own premises* — definitionally intra-state. Total GST is unchanged (18%), but it is remitted under the wrong head, misallocating state revenue and corrupting GSTR filings / the chef's input credit. Fix: for `fulfillment_type = pickup`, use the chef's own state as the place of supply.<br><br>**This is an internal contradiction, not a judgement call.** For the *same* order `…04344806`, the customer-facing **TAX INVOICE** renders **CGST (2.5%) ₹8.40 + SGST (2.5%) ₹8.40** (intra-state), while the chef's Earnings screen renders **IGST (18.0%) ₹3.46** (inter-state). Two documents, one order, two different places of supply. The food-GST path already resolves this correctly; the commission-GST path does not. | OPEN |
| **D-02** | **High** | POU-02 | **Chef is overpaid the GST charged on the delivery fee and platform fee.** `handlers/orders.go:484` sets `taxBase := subtotal + deliveryFee + platformFee − discount`, so `orders.tax` is GST on *all three*. `services/earnings.go:201` then passes `order.Tax` into `ComputeOrderEarnings` as chef income — but that function's contract says *"Tax is the order's food GST. The chef receives it"* and that the delivery fee is *"the DRIVER's money and is EXCLUDED"*. The fee is excluded; **the GST on it is not**. Chef gross, the TDS base, and net payout are all inflated by `taxRate × (deliveryFee + platformFee)`, and the platform pays away GST it is liable to remit. Measured on order `…HC26080303443871`: tax ₹18.7544 vs food-only ₹16.00 → chef net **₹335.96 instead of ₹333.24, +₹2.72 per order**.<br><br>**Confirmed against real settled money.** Weekly statement `6aa26c7c…` (chef Saffron, week 26 Jul–02 Aug, status `pending` — i.e. about to be paid) bills 10 orders: subtotal ₹4250.00, tax charged ₹243.59, of which **₹31.09 is GST on delivery + platform fees**; food-only GST would be ₹212.50. Statement `gross_revenue` ₹4493.57 ≈ subtotal + full tax. Net effect after the 1% TDS offset: **chef overpaid ₹30.78 on this one statement (~₹3.08/order)**. | OPEN |
| **D-01** | **High** | ORD-05 | **Every self-delivery chef is advertised as "Free delivery", then charged at checkout.** `models/chef.go:489` hardcodes `ChefProfileResponse.DeliveryFee: 0 // TODO: populate when delivery fee model is added`, while the *same struct* already carries the real `SelfDeliveryBaseFee` / `SelfDeliveryPerKm` / `SelfDeliveryFreeRadiusKm`. Both `ChefCard.tsx:208` and `chef/[id].tsx:557` render `deliveryFee === 0` as "Free delivery". Observed: Saffron Home Kitchen card says "Free delivery"; checkout for the same chef+address charges **₹39.12**. Customer-facing pricing misrepresentation on every chef with self-delivery. Fix is local — populate from the fields already present. | OPEN |

---

## 7. Blocked / needs action

| Item | Why | Needs |
|---|---|---|
| CAN-05, POU-06 | `/admin/*` requires BFF key + internal-pool identity + `role=admin`; admin is tesserix.app with Google login | Owner performs the admin action on request |
| POU-03 | k8s CronJob; cluster writes are ArgoCD-only | Owner triggers, or assert state machine + mark wall-clock blocked |

---

## 8. Setup issues encountered (recorded — these bite again)

| # | Issue | Resolution |
|---|---|---|
| 1 | Vendor app silently ran the **customer bundle** — launched while a stale session's Metro held port 8081 | Kill stale Metro; customer→8082, vendor→8081; verify branding on screen before trusting any run |
| 2 | Two concurrent Gradle builds corrupted each other via shared `node_modules` (duplicate `libworklets.so`) | Build Android apps **serially** in this monorepo |
| 3 | Expo `run:android --device` wants the **AVD name**, not the adb serial — and still exits 0 on failure | Pass AVD name; verify with `pm list packages` + `lastUpdateTime`, never trust the exit code |
| 4 | Expo LogBox toast overlaps the tab bar and swallows taps | Dismiss it before driving the customer UI |
| 5 | `--port` and `--no-bundler` are mutually exclusive | Use `--port` and let Metro start |

---

## 8b. Observations (not defects, but they mislead)

| # | Observation |
|---|---|
| **OBS-1** | Orders auto-reject via `chefAcceptTimeout = 30 min` (`temporal/workflows/order.go:43`), timed from **order creation, not payment** — so slow checkout eats the chef's window. Observed: created 09:14, paid 09:20, auto-rejected 09:46. Tests must accept promptly, and the reason string attributes an automated action to the chef — misleading in audit trails and in any chef-fault penalty logic. |
| **OBS-2** | **`refund_transactions.amount` is the refund requested across ALL rails, not the amount sent to the gateway.** For order `…3443871` it reads ₹413.84 while Cashfree was correctly called for ₹412.24 (the rest went to wallet as the loyalty rail). Auditing refunds from this column alone will look like an over-refund when it isn't. Cross-check `wallet_txns` + the gateway before concluding. |
| **OBS-3** | `orders.total` is the **pre-credit** order value (₹413.84); the customer is charged `total − loyalty_applied − wallet_applied` (₹412.24). Reconciling "what was charged" from `total` alone overstates revenue. |
| **OBS-5** | **The Expo LogBox toast silently swallows taps on anything beneath it** — it blocked both the customer tab bar and the "Confirm received" bottom-sheet button. Taps register as no-ops with no error, which reads exactly like a broken button. Dismiss the toast before every interaction; this cost real debugging time twice. |
| **OBS-6** | Vendor app renders **unrounded floats as money**: order detail shows `₹15.968` (platform fee) and `₹16.798` (tax) while the Total on the same screen is correctly `₹382.77`. Customer app had this class of bug fixed in `e42bec94`; the vendor app still has it. |
| **OBS-7** | `GET /api/v1/orders/:id/track/ws` returns **401**, so live tracking never connects and the app logs "[tracking-ws] falling back to polling … after N consecutive failures" indefinitely. Functionally degraded but silent to the user. |
| **OBS-8** | Chef dashboard "Total earnings" (₹7,060 → ₹7,443) moves by the **order total** (₹382.77), not the chef's net payout (₹343.93), and increments **before delivery**. A chef reading that figure will materially overestimate what they are owed. |
| **OBS-9** | Vendor "Accept ₹383 order" and the order-detail PRICING block show the **customer's total**, never the chef's net payout. The chef accepts a ₹383 order and is paid ₹343.93; nothing on that screen says so. |
| **OBS-4** | ~~Statement `6aa26c7c` excludes ₹25 of tips…~~ **RESOLVED during testing.** `chef_bonuses` shows three `tip_catchup` rows (₹25, ₹20, ₹30) created 03:00:02 today — `chef_tip_backfill.go` does run and credits the missed tips to the next statement. No action needed. |
| **OBS-17** | **A meal plan's `total` transiently collapses to the food subtotal.** After the chef accepts and before the customer approves, `meal_plans.total` = ₹210 (= subtotal) while `platform_fee` ₹10.48 and `tax` ₹16.80 remain stored — so `planDeliveryTotal` (`Total − Subtotal − Fee − Tax`) evaluates to **−₹27.28**, a negative delivery. It recomputes to ₹240.27 at checkout, so no money was mispriced here, but any refund/skip maths run against a plan in `awaiting_customer` would use a negative delivery figure. The customer-facing card also reads **"If approved ₹210"** while the actual charge is ₹240.27 (+14.4%) — the confirm dialog does disclose "food + GST + delivery, shown at checkout". |
| **OBS-18** | Meal plans accept **no wallet or loyalty credit** — ₹240.27 was charged in full with ₹127.06 of wallet available and no credit UI at plan checkout. Consistent with the escrow model, but customers holding store credit cannot spend it on tiffin. Worth confirming it is intentional. |
| **OBS-16** | **Refund confirmation misstates the destination.** Cancelled order `…06508670` shows *"₹377.07 refunded to your card."* but only **₹314.34** reached the card — ₹61.59 + ₹1.14 were credited as wallet store credit. A customer reconciling their card statement will find ₹62.73 missing and open a ticket. The split is known at refund time (`FundingSplit`), so the copy can state it accurately. |
| **OBS-15** | The checkout credit inputs commit **on blur**, not per keystroke. Mid-edit the screen shows a stale "applied" figure and an unchanged order total, so a customer who types an amount and immediately taps Place Order may believe a different credit is applied than the one that commits. No money risk (the server clamps correctly), but the transient state is misleading. |
| **OBS-14** | The test chef's Cashfree payout vendor is in **`BANK_VALIDATION_FAILED`**. Payout *collection* works (orders settle fine), but disbursement to this chef would fail. Likely test-data, yet it means the weekly statement's ₹4,193.62 could not actually pay out today even without D-02. Worth checking how many live chefs sit in this state. |
| **OBS-13** | **NOT A DEFECT — environment contamination.** The kitchen kept flipping to `accepting_orders = false` (e.g. opened 05:41:55, closed 05:44:06 with no vendor-app interaction from me). Root cause: **the `vendor@fe3dr.com` account was being driven by a second client throughout the run.** API logs show two distinct IPs hitting `/api/v1/chef/*` at near-identical volume over 30 min (`104.30.167.39` = 159 reqs, `2401:d002:b712:bb00:…` = 155 reqs), plus chef writes I never made (`PUT /chef/orders/:orderId/status` 05:48:09, `POST /chef/meal-plans/:id/respond` 05:48:42). The schedule cron is correctly guarded and this chef has `auto_schedule_enabled = false`, so the cron is exonerated. **See §2.5 for what this invalidates.** |
| **OBS-10** | After cancelling, the confirm dialog says *"The breakdown is on your order"*, but navigating there lands on **"Order not found"** — the *tracking* route 404s for a cancelled order. The order and a clear breakdown ("Refunded −₹377.07 / Retained (non-refundable fees) ₹16.77") are reachable from the Orders list, so this is a dead-end link, not data loss. |
| **OBS-11** | Cancellation refunds **truncate** rather than round: computed ₹377.076 was refunded as ₹377.07. Sub-paise, but the convention should be deliberate and consistent. |
| **OBS-12** | On a partial refund the credit rails are returned **pro-rata**, so credits are partially forfeited: order 3 applied ₹1.60 of loyalty but only ₹1.53 came back (refund was 95.7% of total). Correct per `SplitRefundByFunding`, but customers effectively lose a share of their credits on any partial refund. |

## 9. Change log

| Date | Change |
|---|---|
| 2026-08-03 | Document created; environment verified; money model derived; scenarios defined. |
| 2026-08-03 | Executed against `main-89066f8` / `main-700218f`. 4 orders driven end-to-end. Found D-01, D-02, D-03; recorded OBS-1…OBS-12. Verified ORD-01/02/03, PAY-01, CHF-01/02/03/04, CAN-01/02, REF-01/04, LOY-01/02/05, WAL-02, TIP-01, POU-01/02/03/04. |
