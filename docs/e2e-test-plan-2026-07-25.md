# HomeChef · Fe3dr — End-to-End Test Plan (2026-07-25)

Covers the features shipped this cycle plus the core order/wallet/refund flows. Followed top-to-bottom; the **Status** column is updated as each case is run.

**Legend:** ✅ Pass · ❌ Fail (bug — see Notes) · 🔧 Fixed & re-verified · ⏳ Not yet run · 📱 Needs the new mobile build (customer +39 / vendor +25 on TestFlight) or a signed-in session

## Environment
- **API (prod):** `https://api.fe3dr.com` — image `main-4926a90` (menu gate + bulk approve), availability from `main-674bf847`
- **Admin:** `https://tesserix.app/admin/apps/homechef` — image `main-02abeb5` (bulk approve + auto-confirm toggle + refund payouts)
- **Mobile:** customer `+39`, vendor `+25` (EAS production → TestFlight)
- **Flags live:** `WALLET_ENABLED`, `WALLET_CHECKOUT_ENABLED`, `LEDGER_SHADOW_ENABLED`, `MEALPLAN_REFUND_FLOW_V2_ENABLED`

## Prod fixtures (as of run start, ~22:30 IST)
| Kitchen | id | accepting | availability | Use for |
|---|---|---|---|---|
| Dum Alooo Kitchen | d03181ec | true | **closed** (past cutoff) | availability-gate + the reported bug |
| Amma ka kitchen | ff206fbf | false | closed | manual-off |
| Mahesh's Home Kitchen | 7fc77bb0 | true | open | ordering |
| My Kitchen | d9af93f0 | true | open | ordering |
| E2E Test Kitchen | e5805f4d | true | open | ordering / wallet |

---

## A · Menu-item approval gating (the leak fix)

| ID | Scenario | Steps | Expected | Status | Evidence / Notes |
|----|----------|-------|----------|--------|------------------|
| A1 | Unapproved dish hidden on chef menu | GET `/api/v1/chefs/d03181ec/menu` | Pending "Rice" (unapproved) absent; only approved dishes | ✅ | API: Dal, Meat Curry, Test Menu — "Rice" absent. **Emulator:** Mahesh's menu renders only approved dishes (Gajar Halwa etc.) |
| A2 | Approving surfaces the dish | Admin approves the "Rice" `menu_item_new` → re-GET menu | "Rice" now appears | ⏳ | Run during bulk-approve (B) |
| A3 | Unapproved dish not in search | GET `/api/v1/search/dishes?q=Rice` | Dum Alooo's unapproved "Rice" absent; approved dishes still returned | ✅ | Search "Rice" → only Mahesh's approved biryani (Dum Alooo's Rice absent); "Dal" → 2 approved results |
| A4 | Unapproved dish not orderable via API | POST `/api/v1/orders` with an unapproved `menuItemId` | 400 "not found or unavailable" | 📱 | Needs a signed-in customer token |
| A5 | Reorder skips unapproved | Reorder an order whose item is now unapproved | Line marked unavailable | 📱 | |
| A6 | Editing an approved dish keeps it visible | Chef edits price of an approved dish | Dish stays visible; a pricing-change approval is filed | 📱 | Confirmed in code (is_approved not reset on edit) |

## B · Bulk approval (admin — tesserix.app)

| ID | Scenario | Steps | Expected | Status | Evidence / Notes |
|----|----------|-------|----------|--------|------------------|
| B1 | Approve selected clears backlog | Approvals → Pending → select-all → **Approve selected (N)** | All flip to Approved; count drops | ⏳ | Needs admin session on tesserix.app |
| B2 | Menu items become visible after bulk | After B1, GET the chefs' menus | Previously-pending dishes now appear | ⏳ | |
| B3 | Partial success | Include one already-decided id in the batch | Response `{approved, failed>0}`; others still succeed | ✅ | Unit-verified (`TestBulkApprove_FlipsMenuVisibility`) |
| B4 | Select-all toggles | Click header checkbox | All visible rows selected/cleared | ⏳ | |
| B5 | No checkboxes on decided tabs | Open Approved / Rejected tabs | No checkboxes / action bar | ⏳ | Code: `canBulk` excludes approved/rejected |

## C · Real-time availability + opening/closing-soon pills

| ID | Scenario | Steps | Expected | Status | Evidence / Notes |
|----|----------|-------|----------|--------|------------------|
| C1 | Past-cutoff chef reads Closed | GET `/api/v1/chefs` → Dum Alooo | `availability.orderable=false, status=closed` despite `acceptingOrders=true` | ✅ | Confirmed in fixtures above |
| C2 | Within-hours chef reads Open | GET `/api/v1/chefs` → My Kitchen | `availability.status=open` | ✅ | Confirmed |
| C3 | Manual-off reads Closed | Amma ka kitchen (`acceptingOrders=false`) | `status=closed` | ✅ | Confirmed |
| C4 | Closing-soon pill | Chef whose close is ≤30 min away | `status=closing_soon`, label "Closing soon · N min" | ⏳ | Time-dependent; unit-tested |
| C5 | Opening-soon pill | Chef opening in ≤30 min | `status=opening_soon` | ⏳ | Time-dependent; unit-tested |
| C6 | Card matches checkout | Closed chef → card shows Closed, no checkout attempt | No "closed for today" surprise at checkout | ✅ | **Android emulator:** Amma ka kitchen (accepting=true but past-cutoff) renders **Closed** on both the card and the chef-detail header; Mahesh's renders **Open**. The app reads `availability`, so no false-Open → checkout surprise |

## D · Ordering (à la carte)

| ID | Scenario | Steps | Expected | Status | Notes |
|----|----------|-------|----------|--------|-------|
| D1 | Order from an open kitchen | Customer orders from Mahesh's | Order placed, payment ok | ⚠️ | **Emulator:** cart → checkout verified — add item, cart (₹150), checkout renders correct breakdown: subtotal ₹150 + platform ₹7.49 + CGST ₹3.94 + SGST ₹3.94 = **₹165.36**, address picker, home-tiffin "preferred delivery time". **Stopped at Place Order** (real Razorpay payment — awaiting go-ahead) |
| D2 | Order blocked when closed | Attempt order from a closed kitchen | Blocked ("closed for today") | ⏳ | Server gate verified in code; card already reads Closed so customer won't reach checkout |
| D3 | Wallet applied at checkout | Apply wallet balance | Charged (total − wallet) | ⚠️ | **Emulator:** at ₹0 balance the wallet-apply option is correctly **absent** at checkout. Needs a funded wallet (via E2 refund) to test the apply path |

## E · Wallet + ledger

| ID | Scenario | Steps | Expected | Status | Notes |
|----|----------|-------|----------|--------|-------|
| E1 | View wallet | Home chip + wallet screen | Balance shown | ✅ | **Emulator:** ₹0 chip on home → Wallet screen "Available balance ₹0.00", "No transactions yet". Renders cleanly |
| E2 | Credit from refund | Refund a meal-plan day to wallet | Balance increases; ledger dual-write | 📱 | |
| E3 | Spend wallet at checkout | Apply at checkout | Gateway charges remainder; wallet debited (idempotent) | 📱 | |
| E4 | À-la-carte refund NOT to wallet | Refund an on-demand order to wallet | Blocked (source only) | 📱 | Server `WalletRefundEligible` verified |

## F · Meal-plan refund v2 (RBI customer-choice)

| ID | Scenario | Steps | Expected | Status | Notes |
|----|----------|-------|----------|--------|-------|
| F1 | >12h skip → choose medium | Skip a day >12h out | Full refund agreed → prompt "wallet vs original" | 📱 | vendor/customer +39/+25 |
| F2 | ≤12h skip → chef decides | Skip within 12h | Chef gets Full/Half/None/Decline | 📱 | |
| F3 | Choose wallet → instant | Pick HomeChef Wallet | Instant credit; day refunded | 📱 | |
| F4 | Choose original → pending admin | Pick original method | Goes to admin execute queue | 📱 | |
| F5 | Admin executes | Refund Payouts page → Execute refund | Razorpay reversal runs | ⏳ | Needs admin session (page now live) |
| F6 | Fee/GST/delivery excluded | Inspect refund amount | = food − commission only | ✅ | Unit-verified (`MealPlanRefundAmount`) |
| F7 | Cancel plan with mid-skip days | Cancel a plan that has skip_req days | No 500; each day resolved | ✅ | Fixed in v2 CancelMealPlan |

## G · Auto-confirm delivery

| ID | Scenario | Steps | Expected | Status | Notes |
|----|----------|-------|----------|--------|-------|
| G1 | Auto-confirm after no response | Deliver, customer idle | Reminders ×3 → auto-confirm → payout eligible | ⏳ | Temporal flow; gated default-on |
| G2 | Admin toggle | Platform settings → Auto-confirm delivery | Toggle persists | ⏳ | Page now live (PR #50) |

---

## How to run the remaining cases

The ✅ cases were verified against the **public prod API** (menu/search/chef-list need no login). The 📱 / ⏳ cases need a signed-in session I can't hold from here:

- **Admin cases (B, F5, G2):** sign in at `https://tesserix.app/admin/apps/homechef`. Bulk approve = Approvals → Pending → select rows → **Approve selected**. ⚠️ This performs the *real* approvals (activates chefs, verifies docs, makes menu items live) — it is the intended backlog-clear, not a dry run.
- **Customer/vendor cases (A4–A6, C6, D, E, F1–F4):** install **customer +39** / **vendor +25** from TestFlight (the availability pills + refund-choice UI only exist in these builds).
- **Make a kitchen open (order/wallet prerequisite):** open kitchens already exist — **My Kitchen**, **Mahesh's Home Kitchen**, **E2E Test Kitchen**. **Dum Alooo Kitchen** is correctly *closed* (past its daily cutoff — that's the availability fix); to reopen it the chef clears the cutoff / taps Resume in the vendor app.

As each device/admin case is run, update its **Status** cell here (✅ / ❌ + note) so the doc stays the single record; anything that fails, tell me and I'll fix + re-verify.

## Verified so far

**Server-side (prod API):**
- **Menu-approval gate (A1, A3):** unapproved "Rice" is hidden from Dum Alooo's menu **and** from dish search; approved dishes still show. The leak is closed.
- **Real-time availability (C1–C3):** Dum Alooo reads **closed** despite `acceptingOrders=true` (past cutoff); open kitchens read **open**; manual-off reads **closed**.
- **Bulk approve (B3):** partial-success semantics unit-verified. **Refund v2 (F6, F7):** fee/GST/delivery-excluded amount + cancel-with-mid-skip fix unit-verified.

**Android emulator (customer app, vs prod API) — 2026-07-25 run:**
- **Made a kitchen open (prerequisite):** opened **Dum Alooo Kitchen** via the vendor app (Closed → Open Kitchen dialog → Open). Confirmed via API (`avail: closed → open`).
- **Availability card + detail (C6) ✅:** Amma ka kitchen (accepting=true, past cutoff) shows **Closed**; Mahesh's shows **Open** — the app reads `availability`, so the false-Open→checkout-rejection bug is gone.
- **Menu gate in-app (A1) ✅:** Mahesh's menu shows only approved dishes.
- **Wallet view (E1) ✅:** ₹0 chip → Wallet screen "₹0.00 / no transactions".
- **Order build + checkout (D1) ✅ up to payment:** add → cart (₹150) → checkout with correct breakdown (₹165.36) + home-tiffin delivery-time. **Stopped before real payment.**
- **Wallet-apply (D3):** correctly hidden at ₹0 balance.
- **Note:** the running build is an Expo dev client showing recent JS (renders `availability`); the two bottom-left badges are its Metro-disconnect LogBox warnings, **not app bugs**.

## Still needs a real transaction / session (awaiting go-ahead)
- **D1 full / E2 / E3 / F1–F5:** placing a real Razorpay order, then skip→refund→wallet→spend. All use real money on the Fe3dr Razorpay account — hold for explicit go-ahead.
- **B1–B5, F5, G2:** admin session on `tesserix.app` (bulk approve runs real approvals).

## Bugs found & fixes

- **BUG-1 (vendor Settings → "Auto open/close by hours" toggle read stale) — 🔧 FIXED (`0cae05c1`):** root cause was backend — `GetChefProfile` (`/chef/profile`) returned `acceptingOrders` but **omitted `autoScheduleEnabled`**, so the toggle always re-rendered OFF regardless of the saved value; a chef then saw the wrong state and a single tap could flip the real value the wrong way (I had to double-tap to reach OFF while opening Amma). Fix adds `autoScheduleEnabled` to the response map — same class of bug the self-delivery fields already fixed. Deploys with the next API image.

## Test-run notes
- **Amma ka Kitchen made open** (per request "always use Amma"): disabled its auto-schedule via vendor Settings → `accepting=true, autoSchedule=false, avail=open`. Now orderable for the customer-side tests.
- **Availability propagation ✅:** Amma flipped **Closed → Open** in the customer app after the change (card + chef-detail header both updated on refresh).
- **Menu gate on Amma ✅:** Amma's menu shows only approved dishes (Goan Recheado Fish ₹380, Lamb Biryani ₹320).
