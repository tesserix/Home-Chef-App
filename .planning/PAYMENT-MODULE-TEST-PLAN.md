# Payment Module — Manual Test Plan

**App:** Home Chef (Customer + Vendor mobile, Admin = tesserix-home)
**Backend:** `main` @ `38902521` · **Payments:** Razorpay (test mode) + wallet
**Escrow flags:** `ORDER_PAYOUT_AUTO_RELEASE_ENABLED` / `MEAL_PLAN_ESCROW_ENABLED` — **currently OFF**

## How to read this

| Tag | Meaning |
|-----|---------|
| 🟢 **NOW** | Testable today with escrow flags **OFF** — the normal money paths. This is your current pass. |
| 🔒 **ESCROW** | Requires escrow flags **ON** (sandbox / post-#25). The hold→confirm→release→payout lifecycle. Run during #218/#200. |
| 👁 **OBSERVE** | Backend/cron behaviour — verify via DB/logs/admin, not a screen tap. |

**Key flags-OFF expectation:** with escrow OFF, a delivered order **never parks a payout hold** (`payout_hold_status` stays empty). So: ordering + all refunds work normally; the **"Confirm received" CTAs do not appear** (no hold to confirm); vendor earnings show **₹0 held / ₹0 released**. That is correct, not a bug.

## Preconditions / setup

- [ ] Test Razorpay keys active; use Razorpay **test cards** (e.g. success `4111 1111 1111 1111`, and a card that forces failure).
- [ ] At least one **customer** account, one **chef/vendor** account, one **admin** (tesserix-home).
- [ ] Chef has ≥2 menu items with capacity; a delivery address on the customer with coordinates (so the chef is in-range).
- [ ] Ability to view the DB or admin panels for the 👁 OBSERVE checks (order `payment_status`, `refund_amount`, `refunded_at`, wallet ledger).
- [ ] Note the **customer wallet balance** before each refund test.

---

## Suite A — Order payment & capture (🟢 NOW)

| ID | Case | Steps | Expected |
|----|------|-------|----------|
| A1 | Happy-path pay | Place an order → pay via Razorpay checkout (WebView) with a success test card | Payment captured; order → `accepted`/`paid`; `payment_status=completed`; chef gets a "new order" notification **exactly once** |
| A2 | Payment failure | Start checkout → use a failing test card / cancel the sheet | Order stays unpaid/`pending`; no chef notification; customer sees a friendly error (no raw gateway code); can retry |
| A3 | Verify vs webhook (no double-effect) | Complete a payment normally | Only **one** `order.completed` effect (one chef notification, one wallet settlement) even though both the client `verify` and the `payment.captured` webhook run — 👁 confirm no duplicate |
| A4 | Amount integrity | (Advanced) Attempt to pay less than the order total via a tampered client amount | Server rejects — capture is validated against the **stored** amount, not the client value |
| A5 | Wallet applied at checkout (🔒 gated by `WALLET_CHECKOUT_ENABLED`) | If that flag is on: apply wallet credit + pay the remainder | Gateway captures only `total − walletApplied`; wallet debited once; chef/rider top-ups settle |

---

## Suite B — Refunds & cancellations (🟢 NOW — the core regression pass)

> These are **escrow-independent** and fully live today. This is the highest-value area after this session's work.

| ID | Case | Steps | Expected |
|----|------|-------|----------|
| B1 | Customer cancels a paid order | Place + pay → customer cancels (before accept/prep) with a reason | Full refund issued in-flight (< ~10s); `payment_status→refunded`, `refunded_at` set, `status=cancelled`; `order.cancelled` event; customer notified |
| B2 | Chef rejects a paid order | Place + pay → chef **rejects** via the vendor app | Full refund to the customer; order → `cancelled`/`rejected`; refund reflected |
| B3 | Per-line "can't fulfill" (partial) | Order with 2+ items → chef cancels **one line** | Only that line refunded; order **subtotal/tax/total recompute**; the other item continues; `refunded_at` stays **NULL** (still partly live); remaining balance still refundable |
| B4 | Goodwill partial refund | Delivered order → chef/admin issues a **partial** goodwill refund (`/chef/orders/:id/refund`) | Customer refunded the partial amount; order stays `completed`; refund shows in earnings/refund history; `refunded_at` **NULL** |
| B5 | Full refund via refund endpoint | Admin/chef full-refunds a completed order | `payment_status=refunded`, `refunded_at` set, `status=refunded` |
| B6 | Refund to **wallet** vs original method | Issue a refund with `toWallet=true`, and another with `toWallet=false` | Wallet path credits store credit (no gateway round-trip); original-method path reverses the Razorpay charge; both idempotent |
| B7 | **Double-refund prevention** | Issue a refund, then **immediately re-submit the identical refund** (double-tap / retry) | Second attempt returns "duplicate ignored" — **no second real refund**; `refund_amount` bumped once |
| B8 | Over-refund guard | After a partial refund, attempt to refund **more than the remaining** balance | Rejected — "cannot exceed the remaining refundable amount" |
| B9 | Cumulative full refund | Partial-refund part of an order in-app, then refund the remainder | Once cumulative = total, `refunded_at` gets stamped; the order is fully refunded |
| B10 | Cancel after a partial refund | Partial-refund an order, then cancel the whole order | Cancellation refunds only the **remaining** (capped at RemainingRefundable) — cumulative never exceeds the order total |

---

## Suite C — Meal plans / tiffin (mixed)

| ID | Tag | Case | Steps | Expected |
|----|-----|------|-------|----------|
| C1 | 🟢 | Advance breakdown shown | Book a meal plan; open the pre-checkout confirm dialog | The amount shown = **food + GST + delivery** (the full server total), not food-only; footer relabelled |
| C2 | 🔒 | Advance capture & hold | (Escrow ON) Book + pay the plan advance | Advance captured to platform; per-day chef payouts created **on hold** |
| C3 | 🟢 | Skip a future day | Confirmed plan → skip a day before its lead-time cutoff | Day refunded to wallet (escrow-gated on the money side; the state change works either way) |
| C4 | 🟢 | Whole-plan cancel before start | Cancel a confirmed plan **before any day is served** (`PUT /meal-plans/:id/cancel`) | Plan → `cancelled`; full refund of any advance; chef notified; you can rebook (duplicate-plan guard released). **Mid-week (after a day served) is rejected** here — that's the #411 penalty flow |
| C5 | 🟢/🔒 | Per-day quality issue | On a **delivered** meal-plan day → "Report an issue" | Report accepted only for **delivered** days; opens an order issue tied to that day. Money reconcile (freeze/clawback) is escrow-gated |
| C6 | 🔒 | Per-day release on delivery | (Escrow ON) A day is delivered + confirmed | That day's held payout becomes release-eligible → released; other days stay held |

---

## Suite D — Group orders (mostly 🔒 ESCROW)

| ID | Tag | Case | Steps | Expected |
|----|-----|------|-------|----------|
| D1 | 🟢 | Multi-payer charge | Create a group order; participants each pay their share (Razorpay) | Each participant's share captured; amounts validated against the **captured** amount + signature |
| D2 | 🔒 | Consolidation → single held transfer | (Escrow ON) All participants paid | One consolidated held chef transfer |
| D3 | 🔒 | Cancel → reverse + refunds | (Escrow ON) Cancel the group order | Held transfer reversed; each participant refunded to wallet |

---

## Suite E — Escrow confirmation surfaces (🔒 ESCROW; verify **inert** now)

> With flags **OFF**, verify these CTAs are **absent/inert** (no hold parked). With flags **ON**, verify they function.

| ID | Case | Flags OFF (now) | Flags ON (sandbox) |
|----|------|-----------------|--------------------|
| E1 | Order "Confirm received" | CTA **not shown** on a delivered order | Shown on delivered order; tapping advances the hold `awaiting → release_eligible`; "Leave a review" demotes to secondary; confirmed → green pill |
| E2 | Meal-plan day confirm | No per-day confirm link | Per-day "Confirm received" works; disputed day → neutral pill |
| E3 | Tiffin "Confirm today" banner | Banner absent | Bulk-confirms today's awaiting days |
| E4 | Group host confirm | No host confirm CTA | Host-only "Confirm received" on the group; 403 for non-hosts |
| E5 | Vendor earnings escrow state | **Held ₹0 / Last payout –**; no per-order hold pills | Held/released totals populate; per-order pills (awaiting=amber, released=green, disputed=blue, withheld/reversed=red); statement paid/pending pills |

---

## Suite F — Wallet (🟢 NOW)

| ID | Case | Steps | Expected |
|----|------|-------|----------|
| F1 | Refund credits wallet | Do a refund-to-wallet (B6) | Wallet balance increases by exactly the refund; one ledger entry; idempotent on retry |
| F2 | Wallet-only order refund | Pay an order fully from wallet, then refund it | Entire refund returns as store credit; no gateway call |
| F3 | Repeated partial wallet refunds | Two sequential partial goodwill refunds to wallet | Each credits once (per-instance idempotency); no double credit, no missed credit |

---

## Suite G — Payment integrity / security (👁 OBSERVE + 🟢)

| ID | Case | How to check | Expected |
|----|------|--------------|----------|
| G1 | Duplicate payment-id blocked (#395·1) | 👁 On the deploy carrying #660, check API startup logs | **No** `post-migration ALERT: payment-id unique index not created`. (If present → a pre-existing duplicate gateway id to dedup before go-live.) The partial unique indexes now exist on `orders`/`meal_plans` |
| G2 | Signature enforcement | Attempt to verify a payment with a bad/missing signature (incl. tips & group shares) | Rejected on all money entry points |
| G3 | Webhook idempotency | 👁 Replay a `payment.captured` / `refund.processed` webhook | No duplicate side effects (single completion, single refund stamp) |
| G4 | Typed-escrow order refund guard | Try the generic order-refund endpoint on a **meal-plan-day / group-spawned** order | Rejected (422) — routed to the correct typed flow, not a second refund |

---

## Suite H — Reconciliation crons (👁 OBSERVE — backend safety nets)

> Not screen-tappable; verify via logs/DB. These are the automated backstops shipped this session. Mostly no-ops while flags are OFF (empty scan sets), but the refund-side ones run regardless.

| ID | Case | What to look for |
|----|------|------------------|
| H1 | Stuck-refund reconcile | An order left `payment_status=refunded, refunded_at NULL` for >15 min is auto-finalized (partial→`completed`, full→stamped). Cancelled orders keep `status=cancelled` |
| H2 | Cumulative-refund reconcile (#640) | The daily reconcile flags/stamps an order fully refunded **in aggregate** at the gateway (incl. an out-of-band dashboard refund) that wasn't stamped locally |
| H3 | Escrow conservation ledger (#398) | 🔒 With flags ON: a `PaymentDrift` row is written if an aggregate is refunded while its chef transfer is still settled at the gateway ("chef paid AND customer refunded"). Empty while flags OFF |
| H4 | Payout-reconcile sweep | 🔒 With flags ON: a released/reversed-but-unsettled hold is re-driven; a stranded cancelled/refunded aggregate is clawed back |

---

## Exit criteria (for the flags-OFF pass)

- [ ] Suites A, B, F fully pass (order pay + every refund/cancel path, wallet) — **no** double-refund, **no** over-refund, **no** stranded/short refund.
- [ ] C1, C3, C4, D1 pass; C/D/E escrow items confirmed **inert** (not broken) with flags OFF.
- [ ] G1 startup log is clean (no dup-payment-id alert).
- [ ] Every refund reconciles: customer refund amount == what moved; chef never paid on a fully-refunded order (verify in earnings).

## For the escrow-ON pass (later — #218 sandbox → #25 flip)

Re-run the 🔒 rows with `MEAL_PLAN_ESCROW_ENABLED` + `ORDER_PAYOUT_AUTO_RELEASE_ENABLED` on against Razorpay Route **test** accounts, verifying capture → hold → release-on-delivery → reverse-on-refund, the dual-confirmation surfaces (E1–E5), and the conservation ledger (H3). Include the #646 idempotency-conflict check.
