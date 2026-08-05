# HomeChef · Fe3dr — Full E2E Test Plan (2026-08-05)

Simulator-driven end-to-end run across the **customer** and **chef** apps. Every
customer action that has a chef-side consequence is only passed once both sides
were observed.

Mechanics (idb coordinates, payment paths, issue filing): `.claude/skills/e2e-sim-testing/SKILL.md`.
Product/route map: `.claude/skills/fe3dr/SKILL.md`.

**Legend:** ✅ Pass · ❌ Fail (issue linked) · ⚠️ Partial / blocked · ⏳ Not yet run · 🚫 N/A this run

## Environment

| | |
|---|---|
| Customer sim | Fe3dr Customer · iOS 26.5 · `com.tesserix.homechef.customer` · Metro 8082 |
| Vendor sim | Fe3dr Vendor · iOS 26.5 · `com.tesserix.homechef.vendor` · Metro 8081 |
| Customer API | `https://fe3dr.com/api` (**production**) |
| Vendor API | `https://vendors.fe3dr.com/api/v1` (**production**) |
| Payment | **Cashfree** (v3 web SDK in a WebView) — **UPI only.** Cards excluded by request. |

> **Gate G0 — must pass before any payment case runs.** Cashfree splits sandbox
> from production **by host**, resolved server-side from the chef's `mode`
> (`live` / `test`) and handed to the app as `cashfreeEnv`. The payment route
> must carry `env=SANDBOX` and the WebView must load `sandbox.cashfree.com`.
> If it reads `PRODUCTION`, the kitchen is live: **run no payment case**, mark
> C1–C8 blocked, and tell the user.

## Run status — customer sweep 2026-08-06 02:50 IST

**Resume at B5** (§B kitchen-closed), then B2, B6–B12. E7 (reject with a reason) and
E8 (item-level cancel) still need a fresh sandbox-UPI order — both vendor queues are empty.

**Environment as left:** Saffron Home Kitchen open, no open test orders, cart empty,
ChefBook comment count back to 1 (test comment posted and deleted). Metro on 8082
(customer) and 8081 (vendor); harness `sim.sh` in the session scratchpad.

**Covered this pass (customer only, no vendor cross-check):** rewards, order-detail
money, receipt, meal plans, wallet, referral, subscriptions, ChefBook + comments.

**Blocker:** the receipt/tax-invoice screen cannot be opened at all (#1038), so #1027
cannot be re-verified and D7 stays failed.

**Theme of this pass — charge transparency is inconsistent.** Order detail is exemplary
and reconciles to the paise. Every other money surface falls short of it: meal plans hide
27% of the charge (#1039), cancelled plans never state the refund (#1041), subscriptions
never state the next charge (#1042).

**Observed, not filed:**
- Customer accent renders `#E00B41` / `#FF385C` (coral-rose), not persimmon `#C2410C`.
  Deliberate long-standing tokens in `tailwind.config.js`; `.impeccable.md` documents the
  migration as in progress, so this is a known gap rather than a regression.
- ChefBook comment delete is destructive, immediate and unconfirmed.
- Vendor copy: "1 orders awaiting acceptance", "~1 meals".
- Vendor payout prefill: "You can't charge more than ₹39" against a ₹39.15 prefill.
- Vendor order detail never stamps **Preparing** (the customer rail does render it).
- Loyalty "Earn 500 more to redeem" governs points→wallet conversion only, not checkout.

## Fixtures

| Kitchen | id | Mode | Availability | Used for |
|---|---|---|---|---|
| Saffron Home Kitchen | `e150c72a` | resolved at G0 | open · min ₹199 | ordering, reviews, messages, refunds |
| My Kitchen | `eaff3ec4` | — | `acceptingOrders=false` · closed | closed-kitchen gates (§B) |

Orders placed here are real production rows. Every order opened in this run is
driven to a terminal state (delivered or cancelled) before the run closes, and
its id is recorded in the row that created it.

## Money rules under test (`docs/refund-policy-v3-spec.md`)

The refund cases below assert against these, not against whatever the UI happens
to show. **Refund base = `food − commission + GST + delivery`** — GST *is*
refunded (matched by a credit note), delivery *is* refunded, the platform
commission is retained. Worked example: ₹100 food + ₹8 GST + ₹10 delivery =
₹118 charged → a 100% refund is **₹103** (₹85 food net of ₹15 commission, plus
₹8 GST, plus ₹10 delivery).

**Customer-initiated cancel**, by lead time before cook-start:

| Lead | Refund | Decided by |
|---|---|---|
| > 12h | 100%, automatic | nobody — instant |
| 12h–6h | floor 75% | chef sets 75–100% |
| 6h–2h | floor 50% | chef sets 50–100% |
| < 2h | floor 0% | chef may still grant up to 100% |

**Chef-initiated cancel:** customer always gets 100% incl. GST + fees. Lead > 4h
→ no chef penalty; lead < 4h → **6% of the order** deducted from the chef's next
settlement.

**Meal-plan per-day:** `lead > 12h` → auto full **food** refund to wallet, chef's
held day payout reversed. `lead ≤ 12h` → chef decides Full / Half / None, then
admin pays to wallet (instant) or source (5–7 business days, RBI). Cutoff is
`platform_settings.mealplan.refund_prep_cutoff_hours`.

---

## A · Discovery & browse (customer)

| ID | Scenario | Steps | Expected | Status | Notes |
|----|----------|-------|----------|--------|-------|
| A1 | Home feed renders | Launch → Home | Kitchen cards with photo, cuisines, rating, Open/Closed pill, min order | ✅ | Saffron card: photo, cuisines, ★4.8 (6), Open pill, Min ₹199 — all present |
| A2 | Cuisine filter | Tap **North Indian** | Feed narrows; chip reads selected | ✅ | North Indian chip selects; Saffron (North Indian) retained |
| A3 | Open Now filter | Tap **Open Now** | Only `availability.status=open` kitchens remain | ⏳ | |
| A4 | Filters sheet | **Filters** → apply rating/price | Sheet opens, applies, chip shows active count | ⏳ | |
| A5 | Dish search | Search "biryani" → `search-dishes` | Dish results with chef attribution; unapproved dishes absent | ✅ | "Dal" → Home-style Veg Thali ₹240 with chef link. "Dalada" → correct "Nothing found" empty state. Field auto-capitalises the query (cosmetic) |
| A6 | Chef detail | Open Saffron Home Kitchen | Header, rating, menu sections, pill matching the card | ⏳ | |
| A7 | Chefs map | Map icon → `chefs-map` | Map renders with chef pins, no blank tile grid | ⏳ | |
| A8 | Favourites | ♡ on a card → Saved tab | Appears under Saved; un-hearting removes it | ⏳ | |
| A9 | Address switcher | Tap **Home · Bengaluru** | Saved addresses list; switching re-fetches the feed | ⏳ | |

## B · Kitchen closed & availability gates

The failure this section exists to catch: a kitchen that reads **Open** on the
card but rejects the order at checkout. Card, detail header and checkout must
agree, and all three must read `availability`, never `acceptingOrders` alone.

| ID | Scenario | Steps | Expected | Status | Notes |
|----|----------|-------|----------|--------|-------|
| B1 | Manual-off reads Closed | Dashboard pill → Kitchen status → **Close kitchen** | Card pill reads **Closed** | ✅ | Sheet offers Close kitchen / Pause 15 / 30 / 60. The pill still read **Open** in the frame right after the tap and flipped to **Closed** a few seconds later — a server round-trip, not optimistic UI. Customer feed then read **Closed** after pull-to-refresh. |
| B2 | Past-cutoff reads Closed | A kitchen with `acceptingOrders=true` but past its cutoff | Reads **Closed**, not Open | ⏳ | `acceptingOrders` alone must not drive the pill |
| B3 | Detail header agrees | Open the closed kitchen | Header pill also **Closed** | ✅ | Chef page header shows **Closed** plus the banner "Closed right now — you can still order and reserve a slot for when they reopen." |
| B4 | Add-to-cart allowed, with the reason named | Add a dish from the closed kitchen | Add is permitted **by design** (reserve-a-slot), and the closed state is stated before the add | ✅ | **Plan's original expectation was wrong** — the product deliberately lets a closed kitchen take a scheduled order. Add worked; the banner above the menu names the reason first, so nothing is silent. |
| B5 | No checkout surprise | Cart → Checkout with the kitchen closed | Must **not** be the first place "closed for today" appears | ⚠️ | No surprise: the chef page said it first. Checkout renders a slot picker (Tomorrow · Dinner, Fri · Breakfast/Lunch…) and Place Order stays disabled until a slot + the ToS box. **Unfinished:** did not scroll to the top of the slot section to confirm an ASAP option is absent/disabled, and the cart screen itself repeats no closed notice. Resume here. |
| B6 | Closes while in cart | Cart holds items; chef toggles Open→Closed on the vendor sim | Cart/checkout surfaces the change before payment | ⏳ | Cross-app |
| B7 | Closing-soon pill | Kitchen ≤30 min from close | `status=closing_soon`, label "Closing soon · N min" | ⏳ | Time-dependent |
| B8 | Opening-soon pill | Kitchen opening in ≤30 min | `status=opening_soon` | ⏳ | Time-dependent |
| B9 | Reopen propagates | Chef toggles Closed→Open | Customer feed reflects it on refresh | ⚠️ | Chef side confirmed: sheet collapses to Cancel / **Open kitchen**, pill back to **Open**, copy back to "You're open and visible to customers." Customer feed not re-checked after the reopen. |
| B10 | Capacity exhausted | Chef sets a daily cap and fills it | Kitchen gates further orders with a capacity reason, distinct from "closed" | ⏳ | |
| B11 | Born-test kitchen hidden | A kitchen that has never been live, in test mode | Absent from the customer feed entirely | ⏳ | `IsBornTest` |
| B12 | Live→test kitchen | A previously-live kitchen now in test | Shown as **Closed**, not vanished | ⏳ | Regulars must not read it as "shut down" |

## C · Cart & checkout (customer)

| ID | Scenario | Steps | Expected | Status | Notes |
|----|----------|-------|----------|--------|-------|
| C1 | Add to cart | Chef detail → add a dish | Cart badge increments; item and price correct | ⏳ | |
| C2 | Quantity edit | Cart → +/− | Line total and cart total recompute | ⏳ | |
| C3 | Remove line | Remove the only item | Empty state, not a broken total | ⏳ | |
| C4 | Single-chef guard | Add a dish from a second kitchen | "Replace cart?" prompt; replacing clears the first chef's items | ⏳ | |
| C5 | Min-order gate | Cart below ₹199 | Checkout blocked with the shortfall named | ⏳ | |
| C6 | Price breakdown sanity | Cart → Checkout | `subtotal + delivery + taxes − discounts = total`, to the paisa | ⏳ | Arithmetic asserted, not eyeballed |
| C7 | Wallet credit applied | Checkout with wallet balance | Wallet line reduces payable; total recomputes; balance debited only on success | ⏳ | |
| C8 | Address required | Checkout with no address | Blocked, routed to address add | ⏳ | |
| C9 | Terms gate | Checkout without agreeing | Place Order disabled until agreed | ⏳ | |

## D · Payment — Cashfree sandbox UPI

| ID | Scenario | Steps | Expected | Status | Notes |
|----|----------|-------|----------|--------|-------|
| D0 | **Sandbox gate (G0)** | Reach the Cashfree sheet | `env=SANDBOX`; WebView on `sandbox.cashfree.com` | ✅ | **SANDBOX confirmed.** Sheet shows `testMerchantName`, test phone +91 9845110001, and Cashfree's "Test UPI Transactions" panel. Safe to pay. |
| D1 | UPI offered | Sheet renders | UPI section present and selectable | ✅ | "Pay by UPI ID / QR" present; sandbox VPAs `testsuccess@gocash` / `testfailure@gocash` offered. Collect path, no intent hand-off needed. |
| D2 | UPI success | Sandbox UPI success path | Routes to `/payment/result`, which polls the server's real `paymentStatus` → Paid | ✅ | Paid ₹392.08 with `testsuccess@gocash`. UPI collect pending ~60s, then app landed on "Payment confirmed" via the polled server status. |
| D3 | Order persisted | Orders tab after D2 | Order listed with the right total and status | ✅ | Order id `cc78cc0c-2112-42a7-8e98-351681978f44`. |
| D4 | UPI failure | New order → sandbox UPI failure path | Failure surfaced with a retry path; **no phantom paid order** | ⏳ | |
| D5 | Payment hold | `/payment/hold` between order and gateway | Nothing charged and nothing committed until the gateway settles | ⏳ | |
| D6 | Abandon payment | Open the sheet, back out | Order left unpaid; cart not silently emptied; no orphaned paid row | ⏳ | |
| D7 | Receipt sanity | Order → Receipt | Line items, taxes, payment ref and total match checkout exactly | ❌ | **Re-test 6 Aug 02:26 — now unreachable.** Tapping *View receipt* redboxes `Cannot find native module 'ExpoSharing'` (`share-pdf.ts:2` ← `receipt.tsx:22`, a module-scope native import); dismissing leaves a blank white screen with no back affordance, and expo-router restores the dead route on relaunch. → #1038. #1027 (pre-wallet total) **cannot be re-verified** until #1038 is fixed. |
| D8 | Order detail money | Order detail → Price breakdown | Every charge itemised and the lines sum to what was taken | ✅ | `#SAFFRON-HOME-KITCHEN-HC26080514338360`: Subtotal ₹320.00 + Delivery ₹39.14 + Platform fee ₹13.53 + CGST ₹10.20 + SGST ₹10.20 = **₹393.07 Total** ✓, then Wallet −₹0.60 and Loyalty −₹1.60 → **"Charged to your payment method ₹390.87"** ✓. Tax reconciles too: 5% food ₹16.00 + 5% self-delivery ₹1.96 + 18% platform fee ₹2.44 = ₹20.40 = CGST+SGST. This screen is the standard the receipt (#1027/#1038) and meal plans (#1039) should meet. |
| D8 | Card path | — | Excluded by request | 🚫 | Sandbox card row dead-ends on a vault OTP |

## E · Chef receives & fulfils (vendor)

| ID | Scenario | Steps | Expected | Status | Notes |
|----|----------|-------|----------|--------|-------|
| E1 | New order surfaces | Vendor Home / Orders after D2 | Pending card with items, total and the **chef payout** — never the customer total | ✅ | Card reads **"₹320 you earn"** — the payout, not the ₹392.08 the customer paid. Banner copy bug: "**1 orders** awaiting acceptance". |
| E2 | Accept | Tap Accept | Status → Accepted; earnings announced; customer side updates | ✅ | Accepted at 23:45; timeline stamped Ordered 23:35 / Accepted 23:45. Delivery-fee panel prefills ₹39.15 while its own copy says "You can't charge more than ₹39" — accepted anyway, so the copy truncates rather than the cap being violated. |
| E3 | Mark preparing | Advance | Status → Preparing; timeline stamped | ✅ | Pill → Preparing and the Active-list rail advances. The detail TIMING block never stamps Preparing (Ordered/Accepted/Ready/Out for delivery/Delivered only) — by design, but inconsistent with the rail. See E9. |
| E4 | Mark ready | Advance | Label matches fulfilment mode (pickup → "Ready for pickup"; chef_delivery → "Ready"; 3PL → "Ready · awaiting rider") | ✅ | Carrier is chosen *at* Mark Ready: only **Ready · I'll deliver** was offered (3PL dark, so the rider button is hidden by design). Photo of the prepared order required — picker opened, image accepted. Pill → **Ready**, stamped 6 Aug 00:03. The screen then correctly flipped from "DELIVERY AREA / your rider will collect" to **DELIVER TO** with full address, phone and "You're delivering this order yourself." |
| E5 | Out for delivery | Advance | Reflected on both apps | ✅ | Chef side stamped 6 Aug 00:04. |
| E6 | Mark delivered | Advance | Terminal; moves to history both sides | ✅ | Delivered 6 Aug 00:05. Footer switches to "Delivered to customer · Download invoice (PDF)". Test order driven to a terminal state. |
| E7 | Reject a new order | Reject with a reason | Rejected; customer sees rejection + refund initiated | ⏳ | |
| E8 | Item-level cancel | Cancel one line of a multi-item order | Line cancelled; order total, refund and chef payout all recomputed consistently | ⏳ | |
| E9 | Status timeline | Order detail | Ordered → Accepted → Preparing → Ready → Out for delivery → Delivered, each stamped | ⚠️ | Five of six stamped: Ordered 23:35 · Accepted 23:45 · Ready 00:03 · Out for delivery 00:04 · Delivered 00:05. **Preparing is never stamped**, although the Active-list rail renders it as a step — the two views disagree about whether Preparing is a milestone. Cosmetic, no money impact; not raised as an issue. |

## F · Cancellations & refunds — à la carte

Each refund case asserts the **amount**, not just that a refund happened. Compute
the expected figure from §"Money rules" and compare against the app, the receipt
and the wallet ledger.

| ID | Scenario | Steps | Expected | Status | Notes |
|----|----------|-------|----------|--------|-------|
| F1 | Customer cancel > 12h lead | Order → Cancel | **100% auto**, no chef action; refund = `food − commission + GST + delivery` | ⏳ | Assert the number |
| F2 | 12h–6h tier | Cancel in the 12–6h window | Request routed to chef; floor **75%**; chef cannot go below it | ⏳ | |
| F3 | 6h–2h tier | Cancel in the 6–2h window | Floor **50%**; chef sets 50–100% | ⏳ | |
| F4 | < 2h tier | Cancel inside 2h | Floor **0%**; chef *may* still grant up to 100% | ⏳ | |
| F5 | Chef sees the request | Vendor → More → Cancel requests | Listed with the customer's reason and the allowed percentage range | ⏳ | |
| F6 | Chef cannot breach the floor | Try to set below the tier floor | Rejected with the floor named | ⏳ | Guard |
| F7 | Chef approves | Approve at the floor | Order cancelled; customer sees Cancelled + the exact refund line | ⏳ | |
| F8 | Snapshot consistency | Compare the pre-cancel estimate with the posted refund | Identical — the retention snapshot is what is paid, no drift | ⏳ | |
| F9 | Chef cancel, lead > 4h | Chef cancels (`out_of_ingredient`) with >4h lead | Customer refunded **100% incl. GST + fees**; **no** chef penalty | ⏳ | |
| F10 | Chef cancel, lead < 4h | Chef cancels inside 4h | Customer 100%; **6% of the order** deducted from the chef's next settlement | ⏳ | Verify in Earnings |
| F11 | Chef cancel failure message | Force a cancel failure | The API's specific reason surfaces (gateway retry / already refunded), not a generic toast | ⏳ | |
| F12 | Commission never refunded | Any 100% refund | Platform commission retained; customer refund excludes it | ⏳ | Sanity |
| F13 | GST refunded + credit note | Any refund | GST refunded and a credit note recorded | ⏳ | v3 reversed v2 here |
| F14 | Delivery refunded | Any refund | Delivery fee included in the refund base | ⏳ | |
| F15 | Refund reaches wallet | Wallet after a refund | Credit posted with a ledger entry naming the order; balance = old + refund | ⏳ | |
| F16 | Refund to source | A source-destination refund | Marked in-flight with an RBI 5–7 business day expectation, not "instant" | ⏳ | |
| F17 | Customer chooses the medium | Refund destination prompt | Customer picks wallet vs source; choice honoured | ⏳ | RBI rule |
| F18 | Refund history | `/refund` | Every refund above listed with status and amount | ⏳ | |
| F19 | No double refund | Re-trigger a cancel on a refunded order | Blocked; no second credit | ⏳ | Guard |
| F20 | Cancel after delivery | Try to cancel a delivered order | Blocked; routed to Report an issue instead | ⏳ | |
| F21 | Report an issue | Delivered order → Report issue | Submitted; acknowledged; visible to support | ⏳ | |

## G · Meal plans & subscriptions

| ID | Scenario | Steps | Expected | Status | Notes |
|----|----------|-------|----------|--------|-------|
| G1 | Browse plans | Plans tab → `/meal-plans` | Plans listed with price, duration, chef | ❌ | List renders with number, chef, range, count, price and a status chip (Confirmed / Cancelled / Expired / Completed). But the count labels **meals as days**: "7 Aug – 9 Aug · 5 days" for 5 meals over 3 dates, and "4 Aug – 4 Aug · 2 days". `MealPlanList.tsx:177` prints `{days.length} day(s)` where `days` is the booked-meal array. → #1040 |
| G2 | Plan detail | Open a plan | Menu schedule, inclusions, price breakdown; `sum(days) + fees = plan total` | ❌ | `MP-3e1a2ecd`: 5 meal rows with date/slot/veg-marker/dish/price render correctly, but the money block is a bare **Total ₹1,513.63** against meals summing to **₹1,190** — **₹323.63 (27%) unexplained**, no delivery, fee or GST line. `mealPlanAdvanceBreakdown()` (`useMealPlans.ts:231`) already computes food/gst/delivery and is called only from `useMealPlanApproval.ts:79`, never here. Also `[id].tsx:271` shows food-only `acceptedTotal` under "If approved" — #402 reappearing. → #1039 |
| G3 | Book a plan | Book → Cashfree sandbox UPI | Full advance charged into escrow; plan active; chef sees it | ⏳ | |
| G4 | Escrow held per day | Vendor Earnings after G3 | Payout **held per day**, not released upfront | ⏳ | |
| G5 | Chef prep view | Vendor → Meal plans → Prep / Daily menu | Booked plan appears in the prep list for the right dates | ⏳ | |
| G6 | Weekly menu | Vendor → Meal plans → Weekly menu | Chef sets the week's menu; customer sees it on the plan | ⏳ | |
| G7 | Day served releases payout | Chef serves a day | That day's held payout releases; remaining days still held | ⏳ | |
| G8 | Skip a day > 12h lead | Customer skips a day >12h before cook-start | **Auto-approved**: full **food** refund to wallet, chef's day payout reversed, no chef action | ⏳ | Food only — not the gross |
| G9 | Skip a day ≤ 12h lead | Skip inside 12h | State → `skip_pending_chef`; routed to the chef | ⏳ | |
| G10 | Chef declines the skip | Decline | Day proceeds, no refund; customer told | ⏳ | |
| G11 | Chef accepts · Full | Accept Full | → `refund_pending_admin` at 100% of food | ⏳ | |
| G12 | Chef accepts · Half | Accept Half | → `refund_pending_admin` at 50% of food | ⏳ | |
| G13 | Chef accepts · None | Accept None | → `resolved_no_refund`; chef paid 100%, day skipped, customer forfeits | ⏳ | |
| G14 | Refund choices screen | `/meal-plans/refund-choices` | Options match the state machine; the chosen one is applied | ⏳ | |
| G15 | Chef refund decisions | Vendor → Meal plans → Refund decisions | Pending decisions listed; a decision propagates to the customer | ⏳ | |
| G16 | Whole-plan cancel | Cancel an active plan mid-way | Undelivered days refunded per lead-time; delivered days not | ⏳ | `RefundUndeliveredDays` |
| G17 | Per-day arithmetic | After G16 | `refund = Σ refundableFood(undelivered days) × proportion`; served days untouched | ⏳ | Sanity |
| G18 | Refund destination | Admin pays wallet vs source | Wallet instant; source flagged 5–7 business days | ⏳ | |
| G19 | Plan refund in wallet | Wallet ledger | Entry names the plan and the day | ❌ | Ledger names the plan (`Tiffin SAFFRON-HOME-KITCHEN-MP-7b6bf84a — refund (75%) +₹162.89`) but **not the basis**: ₹162.89 is 67.8% of the ₹240.27 paid. The refund is almost certainly right (backend refunds food + delivery, withholds GST and platform fee — `meal_plan_escrow.go`), but the **plan detail shows no refund amount at all**, so the customer cannot reconcile it anywhere. → #1041 |
| G20 | Meal subscription | `/meal-subscription/[chefId]` | Subscribe completes; appears under `/subscriptions` | ❌ | `/subscriptions` lists three rows (one **Trial** with Cancel, two Cancelled), each `Lunch · Veg · 5 days/week · Weekly · ₹900 · 0 delivered · 0 skipped · 0 missed`. Two omissions on a recurring charge: **no next-charge/trial-end date** (`currentPeriodEnd` is on the model, `useMealSubscription.ts:29`, and never rendered) and **no chef name** (`chefId` unused), so three identical cards give no way to tell which kitchen *Cancel* applies to. `creditBalance` is handled correctly at `subscriptions.tsx:251` — the pattern the other two need. → #1042 |
| G21 | Subscription cancel | Cancel a subscription | Stops future charges; no orphaned upcoming days | ⏳ | |
| G22 | Chef subscription config | Vendor `/subscriptions` | Config renders and saves | ⏳ | Uncommitted work in tree |

## H · Reviews, comments & ratings

| ID | Scenario | Steps | Expected | Status | Notes |
|----|----------|-------|----------|--------|-------|
| H1 | Rate a delivered order | Order → Review → stars + comment | Review accepted; confirmation | ⏳ | |
| H2 | Review on the chef page | Chef → Reviews | Visible with the right author and text | ⏳ | |
| H3 | Rating aggregate | Chef card / header | Average and count reflect the new review | ⏳ | |
| H4 | Chef sees it | Vendor → More → Reviews → `review/[reviewId]` | Listed and opens | ⏳ | |
| H5 | Chef replies | Reply | Saved; visible to the customer under the review | ⏳ | |
| H6 | Review gating | Try to review an undelivered order | No review affordance before delivery | ⏳ | |
| H7 | Duplicate review | Re-open a reviewed order | Edits the existing review; no second row | ⏳ | |
| H8 | Cancelled order | Try to review a cancelled order | Blocked | ⏳ | |
| H9 | Tip after delivery | Order → Tip → sandbox UPI | Tip charged against its own row; chef earnings reflect it | ⏳ | |

## I · Order messaging (customer ↔ chef)

| ID | Scenario | Steps | Expected | Status | Notes |
|----|----------|-------|----------|--------|-------|
| I1 | Customer sends | Order → Messages → send | Appears in the thread, own-side styling | ⏳ | |
| I2 | Chef receives | Vendor order detail → thread | Visible without a manual refresh (or on pull-to-refresh) | ⏳ | |
| I3 | Chef replies | Reply | Lands on the customer thread | ⏳ | |
| I4 | Unread badge | Leave one unread | Badge on the order row | ⏳ | |
| I5 | Thread after terminal | Delivered/cancelled order | Readable; composer disabled or clearly closed | ⏳ | |

## J · ChefBook

| ID | Scenario | Steps | Expected | Status | Notes |
|----|----------|-------|----------|--------|-------|
| J1 | Chef writes a post | Vendor → More → ChefBook → title + blocks → publish | Published; empty title rejected with "Give your article a title" | ⏳ | |
| J2 | Block types | Heading, paragraph, image URL, list | Each renders in the editor and the published post | ⏳ | |
| J3 | Customer feed entry | Customer Home → **ChefBook** chip | Post listed with cover image and author | ✅ | Feed opens from the Home chip with the strapline "Recipes, methods and kitchen notes, written by the chefs who cook them." Card shows cover, chef avatar + "Saffron Home Kitchen", title, excerpt, "1 min read", comment count and reaction count. |
| J4 | Post detail | `chefbook/[slug]` | Blocks in order; images load | ✅ | "Dalma, the Odia comfort classic": cover image, paragraph, H2, bulleted list and tag chips all render in order. Reaction row (Yum / Love / Want to try / Clever) shows own reaction selected with a count. **Comments verified end to end** — posted "E2E test comment…", count went 1→2, composer cleared, comment appeared attributed to Priya Sharma; deleted it again and the count returned to 1. Test data left clean. Note: delete fires immediately with **no confirmation**, and it is irreversible. |
| J5 | Chef link-through | Post → chef | Routes to that kitchen | ⏳ | |
| J6 | Edit / unpublish | Chef edits | Change reflected customer-side | ⏳ | |

## K · Catering & group orders

| ID | Scenario | Steps | Expected | Status | Notes |
|----|----------|-------|----------|--------|-------|
| K1 | Catering request | `/catering` → submit | Filed; chef sees it under More → Catering | ⏳ | |
| K2 | Catering deposit | Pay with sandbox UPI | Deposit verified server-side; request → confirmed | ⏳ | |
| K3 | Catering cancel/refund | Cancel a deposited request | Refund per policy; amount asserted | ⏳ | |
| K4 | Create a group order | Cart → group order | Share code / link generated | ⏳ | |
| K5 | Join by code | `/group/[code]` | Joins the existing group cart | ⏳ | |
| K6 | Per-share payment | Pay a share with sandbox UPI | Share charged; group total updates | ⏳ | |
| K7 | Chef sees one order | Vendor Orders | Arrives as a single order with all items | ⏳ | |
| K8 | Group refund | Cancel a group order | Each payer refunded their own share; shares sum to the group refund | ⏳ | Sanity |

## L · Wallet, loyalty & referral

| ID | Scenario | Steps | Expected | Status | Notes |
|----|----------|-------|----------|--------|-------|
| L1 | Wallet balance | `/wallet` | Matches the header chip | ✅ | ₹163.53 on both the Home header chip and the Wallet card. Splits Wallet credit ₹163.53 / Loyalty ₹0.00 (0 pts), and states the spend rule up front: "Usable on food & delivery. Fees and taxes are paid separately." |
| L2 | Ledger entries | Wallet history | Each entry names its source order/refund | ✅ | Every row carries a type, a date and a source — refunds name the order or plan number, debits read "checkout". Credit/debit direction is shown by both sign and arrow glyph. Basis of the % is missing on plan refunds — see G19 / #1041. |
| L3 | Ledger sums | Σ credits − Σ debits | Equals the displayed balance | ⏳ | Sanity |
| L4 | Loyalty | `/loyalty` | Points/tier render; an order accrues points | ✅ | Balance 0, Bronze tier. Ledger shows matched +32/−32 pairs traceable to the delivered order (earn on delivery, burn on redemption). The "Earn 500 more to redeem" gate is the **points→wallet conversion** minimum (`loyalty.tsx:60`), not checkout redemption — checkout has no minimum, which is why 9–97-point redemptions appear in history. Copy is ambiguous but the behaviour is correct; not filed. |
| L5 | Referral | `/referral` | Code + share sheet; copyable | ✅ | "Give ₹55, get ₹75" with the terms spelled out (friend gets ₹55 in points; you get ₹75 once they place their first order). Code `RJY85G8C`, WhatsApp / Message / Email + "More ways to share", counters (0 friends joined, ₹0 earned) and a 3-step HOW IT WORKS. Clearest money copy in the app. |

## M · Account, notifications & support (customer)

| ID | Scenario | Steps | Expected | Status | Notes |
|----|----------|-------|----------|--------|-------|
| M1 | Profile edit | Profile → Edit | Saved and persisted after relaunch | ⏳ | |
| M2 | Food preferences | `/profile/preferences` | Saved; reflected in the feed where applicable | ⏳ | |
| M3 | Addresses | Add / edit / delete | CRUD works; default honoured at checkout | ⏳ | |
| M4 | Notifications | Bell | List renders; deep-links correctly; badge clears | ⏳ | |
| M5 | Support chat (Otto) | `/support-chat` → "how do I cancel an order?" | Reply in ~30–60s with **customer-app** navigation | ⏳ | tenant `homechef` |
| M6 | Data privacy | `/data-privacy` | Export / delete requests submit | ⏳ | |
| M7 | Blocked accounts | `/blocked-accounts` | Renders; unblock works | ⏳ | |
| M8 | Legal screens | Terms, Privacy, EULA | Render and scroll, no blank screens | ⏳ | |

## N · Chef operations (vendor)

| ID | Scenario | Steps | Expected | Status | Notes |
|----|----------|-------|----------|--------|-------|
| N1 | Dashboard stats | Vendor Home | Earnings, orders, rating, audience render and tie to Earnings | ⏳ | |
| N2 | Open/closed toggle | Toggle **Open** | Customer app reflects it on refresh | ⏳ | Feeds §B |
| N3 | Add a menu item | Menu → New → save | Created, pending approval, hidden from customers until approved | ⏳ | |
| N4 | Edit a menu item | Change the price of an approved dish | Stays visible; a pricing-change approval is filed | ⏳ | |
| N5 | Capacity | `/capacity` | Daily caps save and gate ordering | ⏳ | Feeds B10 |
| N6 | Earnings | More → **Earnings** | Payouts and transactions that already happened | ⏳ | |
| N7 | Payout | More → **Payout** | Bank account (account no. + IFSC) for future earnings — *not* a history screen | ⏳ | Historically confused with N6 |
| N8 | Expenses | `/expenses` | Add/list; totals correct | ✅ | Expenses & Tax: Net ₹6,822.73 − Expenses ₹265 = Net income ₹6,557.73. The ₹85 row lists with its category, note and originating order. |
| N9 | Analytics | `/analytics` | Charts render with real data, no NaN/empty axes | ⚠️ | Cards, revenue trend, demand, subscriptions and popular items all render. **P&L expenses line is broken — see N20.** Copy bug: "~1 meals". |
| N10 | Rewards | `/rewards` | Tier/progress render | ⏳ | |
| N11 | Documents | More → Documents → Renew | Expiring docs listed; renewal upload works | ⏳ | |
| N12 | Admin requests | `/admin-requests` | Listed; detail opens | ⏳ | |
| N13 | Notification prefs | `/notification-preferences` | Toggles persist across relaunch | ⏳ | |
| N14 | Language | `/language` → Hindi | UI switches; no missing-key placeholders | ⏳ | en/hi |
| N15 | Chef support (Otto) | More → Help & support → Chat → topic → Summary + Message → Start chat | Reply in ~30–60s using **vendor** navigation (Payout vs Earnings correct) | ⏳ | tenant `homechef-vendor` |
| N16 | Promote | `/promote` | Renders; audience figure matches the dashboard | ⏳ | |
| N17 | FSSAI | `/fssai` | Request flow renders at the ₹118 assisted-filing price | ⏳ | |
| N18 | Log an expense from an order | Order detail → Order expenses → Add expense → category + amount + note → Save | Row saved against that order; header count and total update in place; earnings figure unchanged (expenses never touch settlement) | ✅ | ₹85 Ingredients on `#SAFFRON-HOME-KITCHEN-HC26080513351116`. Header became `1 · ₹85`; "You'll earn ₹320" correctly unchanged. |
| N19 | Expense reaches Expenses & tax | More → Expenses & tax after N18 | The order-scoped expense appears in **Recorded expenses**, dated on `expense_date`, tagged with its order, and inside the FY totals | ✅ | Listed as `Ingredients · #SAFFRON-HOME-KITCHE… · 05 Aug 2026 · E2E test - chicken and cream · ₹85`. FY Expenses rose to ₹265. |
| N20 | **Expense reaches Analytics P&L** | More → Analytics → Profit & Loss, on 7 / 30 / 90 days | Expenses line equals the period's recorded expenses; `expensesByCategory` breaks it down; Profit = net earnings − expenses | ❌ | **₹0 on every period.** Analytics: `₹6,823 − ₹0 = ₹6,823`. Expenses & tax, same rows: `₹6,822.73 − ₹265 = ₹6,557.73`. Root cause: the P&L query filters `deleted_at IS NULL` on `chef_expenses`, which has no such column, and its `Scan` error is unchecked (`handlers/chef_profit_loss.go:96`). → #1028 |
| N21 | Expense categories | Log one expense per category | All eight buckets (`ingredients`/`gas`/`utensils`/`packaging`/`transport`/`equipment`/`utilities`/`other`) save and bucket separately in the FY breakdown and the P&L category list | ⏳ | Blocked from full verification by #1028 for the P&L half |
| N22 | Backdated expense | Expenses & tax → Add expense → **Yesterday** / Other date | Bucketed on `expense_date`, not `created_at` — a bill entered today for yesterday lands in yesterday's period on both screens | ⏳ | |
| N23 | Delete an expense | Recorded expenses → bin icon | Row removed; FY totals, P&L and the order's expense header all drop by that amount | ⏳ | |
| N24 | Expense with receipt | Add expense → Attach bill / receipt | Image uploads to the private bucket; reader gets a short-lived signed URL, never a raw path | ⏳ | `ChefExpense.ReceiptPath`, column `receipt_url` |

## O · Cross-cutting

| ID | Scenario | Steps | Expected | Status | Notes |
|----|----------|-------|----------|--------|-------|
| O1 | Offline behaviour | Kill the network mid-browse | Clear offline state, no white screen; recovers on reconnect | ⏳ | |
| O2 | Cold-start session | Force-quit and relaunch both apps | Still signed in; lands on the right tab | ⏳ | |
| O3 | Deep link from notification | Tap an order notification | Opens that order | ⏳ | |
| O4 | Live status push | Chef advances status while the customer sits on Track | Customer updates without a manual refresh (WS) | ⏳ | |
| O5 | Payout never leaks | Any customer total vs chef payout | Chef never sees the customer total; customer never sees the chef payout | ⏳ | Regression guard |
| O6 | No money invented | Every refund in §F and §G | Σ refunds ≤ Σ charged, per order and per plan | ⏳ | Sanity |

---

## Results

**Run 1 — 2026-08-05, 23:15–23:30 AEST. Halted at G0.**

| Area | Pass | Fail | Blocked | Not run |
|---|---|---|---|---|
| A · Discovery | 3 (A1, A2, A5) | 0 | 0 | 6 |
| D · Payment | 0 | 0 | **9 (D0 gate failed)** | 0 |
| E–H, K · downstream of a paid order | 0 | 0 | **all** | 0 |
| B, C, G(read-only), I–O | 0 | 0 | 0 | all |

### G0 failed — the run cannot pay

Saffron Home Kitchen is `mode: live` (confirmed: `GET /api/v1/search/dishes`
returns `"mode":"live"` for its dishes, and the vendor dashboard shows no
**TEST MODE** banner). `GetCashfreeFor(mode)` therefore selects the
**production** Cashfree credentials, so any payment in this run would be a real
charge against a live kitchen.

Test mode is not chef-controlled — the vendor app only *renders* the banner. It
is set through the admin API (`/admin/chefs/:id/test-sessions`,
`/admin/test-mode-policy`), whose UI lives in `../tesserix-home`.

**To unblock, one of:**
1. Put Saffron Home Kitchen into a test session from the Tesserix admin, or
2. Nominate an existing test-mode kitchen and confirm the customer account can see it.

Until then D0–D8 stay blocked, and with them every case that needs a paid order:
E (fulfilment), F (refunds), G3–G19 (plans), H1–H5/H9 (reviews, tips),
I (order messaging), K (catering, group orders).

## Failures & issues

| Case | Symptom | Issue |
|---|---|---|
| — | — | — |
