# HomeChef money-path test report — 4 August 2026

Post per-supply-GST release. Driven end to end on the iOS simulators against the
**production** API, Cashfree sandbox credentials.

**Every money figure in this report was asserted in Postgres or against the
payment gateway.** Nothing is read off a screen. Where a number could not be
verified, it is marked as not verified rather than rounded up into a claim.

| | |
|---|---|
| API at start | `main-1a98c35` |
| API at end | `main-1e5a428` (3 releases shipped during the run) |
| Orders driven | 6, end to end |
| Scenarios closed | 21 |
| Defects fixed and shipped | 5 |
| Defects found and left open | 4 |

---

## Headline

**Two defects reached real money, and one of them had already happened twice in
production.**

> 🟥 **An order could be cancelled out from under a payment the gateway had
> taken.** The abandoned-order sweep asked "is there a *captured* payment?" and
> cancelled on anything else — including a card sitting on the bank's OTP page and
> a Razorpay `authorized` hold, which is money already on the customer's card.
> The reconcile cron is forward-only *by design*, so nothing recovered those
> orders: customer charged, order cancelled, `refund_amount = 0`.

> 🟥 **Post-delivery tipping did not work for a single chef on the platform.** The
> flow required a Razorpay Route account. Zero chefs have one — payouts moved to
> Cashfree and the tip path never moved with them. An entire surface promising
> *"100% goes straight to your chef, with no platform cut"* returned HTTP 409 to
> everyone who used it.

Both are fixed and shipped.

**The pricing and refund engine itself held up exactly.** Every order, tax split,
refund allocation and hold transition matched a prediction written *before* the
transaction ran. The defects are in reporting, settlement and routing — not in the
money model.

---

## What was fixed and shipped

### D-14 · an order cancelled under a live charge — CRITICAL

`stale_order_cron.go` collapsed two opposite gateway answers into one value:
"every attempt is dead" and "an attempt is still running" both read as *not
captured*, and both cancelled.

Cashfree parks a card at `PENDING` while the bank's OTP page is open and a UPI
collect while the payer decides. Razorpay's `authorized` is money already held.
All of them cancelled the order at the 30-minute mark.

**The audit is the part that matters.** Of 31 production orders matching the
wrongly-cancelled shape, 25 are Cashfree. Asking Cashfree about each:

| Order | Gateway says | Our side |
|---|---|---|
| `HC26073102484828` | **SUCCESS ₹757.63** | cancelled · failed · refund **0** |
| `HC26073113579923` | **SUCCESS ₹600.00** | cancelled · failed · refund **0** |

The other 23 were genuine. So the sweep was right 23 times and wrong twice —
**8% of cancellations landed on a payment the gateway had taken.**

**No real customer money is involved**: the live Cashfree slot currently holds a
TEST key, so these are sandbox captures. That is the only reason this is a test
finding and not an incident. Under production credentials the identical path takes
real money and leaves it unrefunded.

*Fix:* the probe is now tri-state — captured / in-flight / dead — and only *dead*
cancels. Shipped in #989.

*Still needs a decision:* those two rows need a refund call. They are outside the
reconcile cron's reach by design.

### D-16 · the fix for D-14 had its own bug — caught by a canary, not by review

After #989 deployed I deliberately left a checkout unpaid to watch the sweep
correctly decline to cancel it. Cashfree reported `NOT_ATTEMPTED` — a session
created, never started.

#989 routed every unrecognised status to *in-flight*. That asymmetry is right, but
the classification was incomplete: `NOT_ATTEMPTED` is the ordinary abandoned
checkout, and the sweep must still cancel it to release the chef's reserved daily
capacity. Under #989 alone that capacity was held until the gateway session
expired — Cashfree's **30-day** default, because `order_expiry_time` is defined on
our request struct and never populated.

*Fix:* `NOT_ATTEMPTED`, `VOID` and `CANCELLED` joined the terminal set. Shipped in
#990.

*Open question for the team:* setting `order_expiry_time` would bound the
unknown-status case **structurally** — the gateway kills its own session, the
status goes terminal, and we stop depending on having enumerated every state
Cashfree will ever ship. Not done, because it shortens the customer's payment
window and that is a product decision.

### D-18 · post-delivery tipping was dead platform-wide

`handlers/tips.go` demanded `order.Chef.RazorpayAccountID`. In production:

| | |
|---|---|
| Chefs with `razorpay_account_id` | **0** |
| Chefs with `cashfree_vendor_id` | 1 of 2 profiles |

Every tip answered 409 *"This chef can't receive tips right now"* — which read as
a chef-data problem and was actually a routing one. Previous runs recorded TIP-01
as "blocked on a tap that wouldn't land"; the screen was reachable all along.

*Fix:* the tip now rides the same gateway as the order it thanks. On Cashfree it
is charged as its own order whose `order_splits` allocate the **whole** tip to the
chef's vendor account — no fee subtracted, because a tip carries no commission and
no tax. Rider tips on Cashfree now fail with an honest message instead of blaming
the chef. Shipped in #991.

### D-13 · the chef's own screens reported three different numbers

The vendor dashboard hero is labelled **"Total earnings"** and taps through to
payouts, but summed `total − refund_amount` — the **customer's** order value,
carrying the delivery fee (the driver's), the platform fee, and the platform's GST
on both. On a cancelled order it also handed the chef the share the **platform**
retained.

| | Before | After |
|---|---|---|
| Test kitchen lifetime hero | ₹8,706.92 | **₹7,491.38** |
| That week's earnings gross | ₹955.40 | **₹955.40** (unchanged) |

The week's gross holding still is the point: that figure was already reconciled
against the settlement statement. The fix moves the wrong number onto the right
one, not the other way. Verified live after deploy at **₹7,683.38 / 37 orders**,
matching the screen exactly.

Also fixed: the Earnings screen was missing two guards the weekly statement has
always had — `mode` scoping (a sandbox order counted toward a live chef's
earnings) and `refunded_at IS NULL`. Shipped in #989.

### D-04 · a dollar sign on an INR marketplace

A customer under a chef's threshold was told *"Minimum order is $199.00"*. Fixed
at both rejection sites; verified live. Shipped in #990.

---

## Defects found and left open

### D-19 · a food complaint asks for the platform's GST — and can auto-refund it

Reporting a missing item on a ₹320 food order requested **₹340.40**; the correct
figure is **₹336.00**. The ₹4.40 gap is `tax_service + tax_delivery` — the same
figure as the earlier D-09 over-credit, in a path the D-09 fix never touched.

```go
// handlers/order_issue.go:177
requested := services.ComputeIssueRefund(order.Subtotal, order.Tax, ...)
//                                                       ^^^^^^^^^^ whole order tax
```

What makes it more than a display bug is twenty lines below:

```go
// handlers/order_issue.go:252
if services.ShouldAutoRefund(cfg, requested) && riskDecision.AutoRefundAllowed {
    services.RefundIssueToWallet(database.DB, &issue, requested, "system", nil)
```

An **automatic** refund keyed on the same inflated figure. The issue filed during
this run stayed `pending`, but an older row on that table shows `auto_refunded` —
the path is live, not theoretical.

Not fixed here: it is a money path, three releases were already in flight, and the
one-argument change needs its own tests over `ComputeIssueRefund`'s apportionment.

### D-15 · ₹25 on a frozen statement that can never reach the chef

Statement `6aa26c7c` is frozen at ₹4,493.57; the same population recomputes today
to ₹4,518.59. The gap is one order's ₹25.00 `chef_tip`.

It is unrecoverable because the order is stamped `billed_statement_id`, and
`reconcileStatementCatchup` selects `WHERE billed_statement_id IS NULL` — it can
see orders that were *never billed*, not an order billed for **less than it is now
worth**.

**Deliberately not root-caused.** Either the tip landed after the statement froze,
or the builder dropped it. The schema does not record when a tip was written,
which is itself the fixable thing underneath.

### D-17 · the customer is shown a raw database value

> "We're sorry — this order was cancelled **out_of_ingredient**."

The chef picked a labelled button; the customer gets the enum. The human string
already exists in the vendor app.

### D-10 · retry offered on an already-cancelled order

Reproduced on `HC26080405111133` — cancelled, ₹393.05 already refunded, and the
app still offered "Retry payment". The retry cannot succeed.

---

## Scenarios

**Passed (21):** CAN-02, CAN-03, CHF-02, CHF-03, CHF-04, ORD-01, ORD-02, ORD-03,
PAY-01, PAY-02, PAY-05, POU-01, POU-03, POU-04, REF-03 (behaviour), REF-04,
REV-01, LOY-05, LOY-06, WAL-02, WAL-05.

**Partial:** POU-05 — population exact (statement bills 10 orders, recomputed
payable set is also 10), money ₹25.02 short → D-15.

**Failed:** ORD-05 (D-01), PAY-04 (D-10), POU-02 (D-09/D-03, both since fixed),
TIP-01 (D-18, now fixed).

**Not run:** CAN-04, CAN-06, REF-02, REF-05, WAL-03, WAL-04, LOY-02, LOY-03,
LOY-04, MPL-01/02/03.

**Blocked on account provisioning:** REFR-01 needs a fresh GIP signup, GRP-01/02
need a second customer account. Neither can be created unattended.

**Owner-only:** CAN-05, POU-06 (admin auth), POU-03 auto-path (CronJob).

### Two results worth reading twice

**PAY-05 proved itself on a real failure, not by inspection.** A payment was
captured at the gateway with the client no longer polling — nobody ever called
verify. The reconcile cron settled it 11 minutes after the order and one tick
after the capture, inside the documented ≤10 min worst case.

**CAN-02 matched a prediction written before the transaction ran**, to the paisa:

| | paise |
|---|---|
| Customer refund | **17548** |
| Vendor kept | **19200** |
| Platform kept | **2557** |
| Σ | 39305 = the order total |

The tax refund was **₹8.36**, not the old proportional model's ₹9.15. That ₹0.79
of GST on a retained fee is precisely what the release under test set out to fix,
and it is fixed.

### One near-miss worth naming

CHF-02's gateway refund read **₹392.00** against a ₹393.05 total. That looks like
a ₹1.05 shortfall and is not — `loyalty_applied` was ₹1.05 and each rail refunds
separately: 392.00 + 1.05 = 393.05 exactly. **Checking only the gateway would have
produced a false defect report.** Money conservation has to be asserted across all
rails, never on one.

---

## Method, and what it cannot tell you

- Money is asserted in Postgres and against the Cashfree API. Screens are used to
  drive the app, never to establish a figure.
- Predictions were written **before** the transaction where the scenario allowed
  it, so a pass means "matched a stated expectation", not "looked plausible
  afterwards".
- **Sandbox, not production money.** The live Cashfree slot holds a TEST key. The
  code paths are the production ones; the money is not real. Every finding here
  would behave identically with live credentials — that is the point of D-14 —
  but no customer was charged during this run.
- The canary technique earned its keep: D-16 was a bug in a fix that had already
  passed review and CI, found only by deliberately provoking the state in
  production and watching what the system did.
- Two-thirds of a full pass remains. The unrun scenarios are not lower risk, only
  lower reach — several need account provisioning that cannot be automated.

---

## Recommended order of work

1. **D-19** — one argument, but it can auto-refund the platform's own GST. Needs
   tests over the apportionment, not a one-line edit.
2. **The two charged-and-cancelled orders** — decide the refund.
3. **`order_expiry_time`** — a product call that closes the last structural gap in
   the D-14 family.
4. **D-15** — and record when a tip is written, so the next one is diagnosable.
5. **D-17, D-10** — customer-facing, cheap.

Still open from earlier runs and unaddressed here: **D-01** (free-delivery claim
vs charged fee), **D-08** (meal plans charge 8% GST and a ₹2.99 delivery from
unlocalised defaults), **D-12** (receipt omits the loyalty rail), and the PAY-03
auto-cancel criterion, which needs a product decision rather than a fix.
