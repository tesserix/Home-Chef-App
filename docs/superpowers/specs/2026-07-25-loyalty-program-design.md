# HomeChef Loyalty Program — Design

**Goal:** A points-based loyalty + referral + streak program that retains customers and motivates chefs, built on the platform's existing (partially-wired) loyalty/referral/wallet/ledger infrastructure. Points convert to spendable wallet credit; customers spend on the app, chefs may withdraw to bank via admin approval.

**Guiding constraint:** points must not be an open-ended liability. Value-per-point is kept lean (₹0.05); rewards feel big by granting *more points*, not costlier points; friction (min-redeem, per-order cap, monthly cap, 1-year expiry with breakage) holds realized cost well below the headline rate. Every threshold is admin-tunable at runtime via `platform_settings` (no deploy).

**Release definition:** all four phases below built and E2E-tested (the loyalty program is the final pre-release requirement).

---

## 1. What already exists (leverage, don't rebuild)

Verified in the codebase — reuse these:
- **Points engine** — `models/loyalty.go`: `LoyaltyAccount` (per user: `Balance`, `LifetimePoints`, `Tier`, `CurrentStreak`, `LongestStreak`, `LastStreakDay`) and immutable `LoyaltyTransaction` ledger (`Type` credit/debit, `Source`, `Points`, `PointsAfter`, `OrderID`, `Reason`, `CreatedBy`, `IdempotencyKey`). Tiers bronze/silver/gold.
- **Order points earning — WIRED** — `services.AwardOrderLoyalty` fires from `handleOrderDelivered` (order-delivered consumer), idempotent on `loyalty:order:<id>`.
- **Redeem points → wallet — WIRED** — `services.RedeemLoyalty` debits points + credits wallet (`WalletSourceLoyalty`) in one tx, enforces `MinRedeem`, no partial/overdraw.
- **Referral — WIRED** — `ReferralCode` (8-char, per user), `Referral` (one per referee ever), reward on referee's **first** paid order (both wallets credited), device+IP fraud snapshot + monthly spend cap. Fired from 4 payment-capture sites.
- **Config** — runtime `platform_settings` keys `loyalty.*` and `referral.*`, read by `GetLoyaltyConfig` / referral config; admin GET/PUT endpoints exist.
- **Frontend** — mobile-customer `app/loyalty.tsx`, `app/referral.tsx`, hooks, register-with-code; web equivalents live. Mobile gated **off** by `REWARDS_ENABLED` / `REFERRAL_ENABLED` in `lib/features.ts`.
- **Wallet + double-entry ledger** — `CreditWallet` + ledger buckets `user_wallet_cashback` (loyalty), `user_wallet_referral` (referral). Approval system (`ApprovalRequest`) and chef Razorpay payout infra exist.

## 2. What is new or changes (gaps)

- Per-point value change (redeem rate) + earn on **food subtotal**.
- **Per-batch (dated) point expiry** with FIFO redemption + a daily expiry sweep (today there is only a single balance, no dated batches).
- **Refund reversal** of earned points; restore points spent on a refunded points-paid order.
- **Admin grant/adjust points** endpoint (`admin_adjustment` source exists in the model, unused).
- Referral trigger **1st → 5th** qualifying order; referrer + referee point bonuses (not wallet cash).
- **Order-streak engine** (non-subscription) with **80%-off / free-order vouchers** applied at checkout.
- **Chef rewards** — monthly rating + revenue-milestone point crons (chef-scoped, no chef loyalty exists today).
- **Chef cash-out** — points → wallet → bank withdrawal request → admin approval → Razorpay payout (new approval type `loyalty_payout`).
- Enable mobile screens; add a **tesserix-home admin config UI**.

---

## 3. Points economics (LOCKED)

| Setting | Value | Config key |
|---|---|---|
| Earn | **1 pt per ₹10 of food subtotal** (excludes GST, platform fee, delivery) | `loyalty.points_per_rupee` = 0.1 |
| Point value | **1 pt = ₹0.05** (100 pts = ₹5; 1,000 = ₹50; 2,000 = ₹100) | `loyalty.redeem_rate` = 0.05 |
| Effective base | **~0.5% cashback** | — |
| Min redemption | **500 pts (= ₹25)** | `loyalty.min_redeem` = 500 |
| Max redemption / order | **10% of food subtotal** | `loyalty.max_redeem_pct` = 0.10 (new) |
| Monthly redemption cap | **₹300 / customer** | `loyalty.monthly_redeem_cap` = 300 (new) |
| Expiry | **1 year from each earn date**, per-batch FIFO | `loyalty.expiry_days` = 365 (new) |
| Credited when | Order **Delivered + auto-confirm/refund window elapsed** | — |
| On refund | Reverse that order's earned points; restore points spent on a refunded points-paid order | — |
| Customer withdraw / transfer | **No** — spend on the app only | — |

**Points→₹ reference table** (1 pt = ₹0.05): 100 = ₹5 · 500 = ₹25 (min) · 1,000 = ₹50 · 2,000 = ₹100 · 6,000 = ₹300 (monthly cap).

---

## 4. Data model changes

All schema (SQL) lives in **tesserix-k8s** `db-schema-bootstrap/schemas/homechef/homechef/`; app repo carries only GORM models.

- **`LoyaltyEarnBatch`** (new) — dated point lots for expiry: `id, user_id, source, points, points_remaining, earned_at, expires_at, order_id, idempotency_key`. Every credit writes a batch; redemptions and expiry consume `points_remaining` FIFO by `expires_at`. `LoyaltyAccount.Balance` stays the fast running total (sum of `points_remaining`), reconciled by the sweep.
- **`LoyaltyAccount`** — add `Kind` (`customer` | `chef`, default customer) so chef rows are distinguishable (same table, keyed by user id).
- **`Referral`** — add `RefereeOrderCount` (int, qualifying orders so far) and keep single `rewarded` transition at count == threshold.
- **`StreakVoucher`** (new) — `id, user_id, kind` (`discount_80` | `free`), `earned_at, expires_at, max_discount, status` (`active` | `used` | `expired`), `used_order_id`. At most one `active` per user.
- **Order-streak** — add **new** `LoyaltyAccount` columns `OrderStreakCount`, `LongestOrderStreak`, `LastOrderStreakDay` for the ≥₹500 paid-order streak. This is kept **separate** from the existing meal-subscription streak (`CurrentStreak/LastStreakDay`, a points bonus) so the two mechanics never clobber each other.
- **`ApprovalRequest`** — new type `loyalty_payout` (chef withdrawal), reuses the existing approval + audit + Razorpay payout plumbing.

## 5. Earning, expiry, redemption

- **Earn (Phase 1):** `AwardOrderLoyalty` computes on **food subtotal** (not total). Writes a `LoyaltyEarnBatch` (`expires_at = earned_at + expiry_days`). Idempotent per order. Fires only after Delivered + the auto-confirm/refund window (so a later refund needn't claw back — but reversal still handled defensively).
- **Refund reversal:** on order refund/cancel, debit the points earned for that order (idempotent, from that order's batch); if the order was paid partly with points, credit those points back into a fresh short-expiry batch.
- **Expiry sweep:** daily cron expires `points_remaining` in batches past `expires_at` (debit txn `source=expiry`), decrements the account balance, emits a "points expired" notification for large amounts.
- **Redeem:** `RedeemLoyalty` extended — enforce `max_redeem_pct` (per order, computed against food subtotal at checkout) and `monthly_redeem_cap` (rolling 30-day sum of redeemed ₹). FIFO-consume batches. Still one tx, no partial/overdraw, credits wallet `WalletSourceLoyalty`.

## 6. Referral — first-5-orders, E2E (Phase 2)

- On each referee **paid, non-cancelled order with food subtotal ≥ ₹200**, increment `Referral.RefereeOrderCount`. When it reaches **5** (`referral.reward_after_orders`, default 5), grant **both** point bonuses once (idempotent `referral:reward:<referral_id>`), set `rewarded`.
- **Amounts (config):** referrer `referral.referrer_points` = **1000** (₹50), referee `referral.referee_points` = **500** (₹25). Granted as **points** (into earn batches), not wallet cash.
- Keep existing guards: one referral per referee ever, self-referral blocked, device+IP snapshot, monthly spend cap.
- **Frontend:** enable `REFERRAL_ENABLED`; referral screen shows code/link + **"N of 5 orders"** progress and reward state.

## 7. Streaks — 80%-off / free vouchers (Phase 3)

- **Qualifying day** = a calendar day (IST) with ≥1 paid, non-cancelled order whose **food subtotal ≥ ₹500** (`streak.min_order` = 500). One order/day counts; a day with no qualifying order **resets** the streak to 0.
- **Milestones:** at `OrderStreakCount == 10` → issue an **80%-off** `StreakVoucher`; at `== 20` → issue a **free** voucher, then reset `OrderStreakCount` to 0. (`streak.tier1_days` = 10, `streak.tier2_days` = 20.)
- **Voucher:** valid **7 days** (`streak.voucher_days`), one order, **max ₹500 off** (`streak.max_discount`), at most one active voucher at a time. Chef is paid **in full**; the platform funds the discount.
- **Checkout:** the voucher applies via the existing discount plumbing (like a promo) — reduces the customer charge, order records the platform-funded discount for reconciliation.
- **Advance point:** `OrderStreakCount` advances from the same `handleOrderDelivered` consumer, now also for ordinary orders (guarded by the ≥₹500 food-subtotal rule); a calendar day with no qualifying order resets it.

## 8. Chef rewards + cash-out (Phase 4)

- **Reuse the points ledger** with `LoyaltyAccount.Kind = chef` and chef sources `chef_rating`, `chef_revenue`.
- **Monthly rating cron:** for each active chef, if last-month avg rating ≥ **4.5** (`chef.rating_min`) over ≥ **10** rated orders (`chef.rating_min_orders`) → award `chef.rating_points` = **500** (₹25). Idempotent per (chef, month).
- **Monthly revenue cron:** last-month **food revenue** ≥ ₹10k → **1000 pts**; ≥ ₹25k → **2500**; ≥ ₹50k → **5000** (highest tier reached; `chef.revenue_tiers` config). Idempotent per (chef, month).
- **Cash-out flow:** chef redeems points → wallet (same `RedeemLoyalty`, chef account) → **requests a bank withdrawal** (amount ≤ wallet balance) → creates an `ApprovalRequest{type: loyalty_payout}` in the admin queue → admin approves (tesserix-home) → Razorpay payout to the chef's linked account (reuse the payout release path); admin reject returns the amount to the wallet. **Only chefs can withdraw; only via approval.**

## 9. Admin & config

- All thresholds are `platform_settings` keys under `loyalty.*`, `referral.*`, `streak.*`, `chef.*`, read through the config services (cached, invalidated on write). Extend the existing admin GET/PUT config endpoints.
- **tesserix-home admin UI** (`apps/web/app/admin/apps/homechef/loyalty/…`): view/edit config, loyalty analytics (existing endpoint), and the **loyalty-payout approval** rows (in the existing Approvals queue, new type badge).

## 10. Feature flags & rollout

- Server master switches (config.go env, default off until each phase ships): `LOYALTY_ENABLED`, `REFERRAL_ENABLED`, `STREAK_ENABLED`, `CHEF_REWARDS_ENABLED` — gate earning/redemption/crons so a phase can be dark-launched, verified, then turned on.
- Mobile: flip `REWARDS_ENABLED` / `REFERRAL_ENABLED` in `features.ts` per phase; ship in the batched mobile build.
- Money-touching earn/redeem paths must be idempotent and safe with flags off.

## 11. Testing strategy

- **Unit (Go, sqlite harness):** points math (earn on subtotal, FIFO redemption, expiry), refund reversal, referral 5-order counter + idempotent reward, streak day-counting + milestone voucher issuance + reset, voucher checkout application + cap, chef rating/revenue cron thresholds + idempotency, cash-out → approval → payout/reject.
- **E2E (emulator + Razorpay test card, as done for wallet):** earn on a real order → redeem to wallet → spend; referral: referee 5 orders → both bonuses; streak: qualifying days → voucher → 80%-off checkout; chef: seed a milestone → chef withdraw → admin approve (tesserix-home) → payout.
- Each phase ships behind its flag, is E2E-verified, then enabled.

## 12. Build order (4 SDD cycles) + non-goals

1. **Points core** — economics change, earn-on-subtotal, per-batch expiry + sweep, refund reversal, admin grant/adjust, enable mobile loyalty. *Ships base loyalty.*
2. **Referral E2E** — 5-order milestone + bonuses + progress UI + enable.
3. **Streak vouchers** — order-streak engine + checkout application + caps.
4. **Chef rewards + cash-out** — rating/revenue crons + withdrawal → admin approval → payout + admin UI.

Each cycle = its own implementation plan + subagent-driven build + tests, shipped and E2E-tested before the next.

**Non-goals (YAGNI):** paid-tier subscriptions, gamified badges beyond bronze/silver/gold, point gifting/transfer between users, partner/brand rewards, driver loyalty (separate `DriverReferral` exists, out of scope), points on non-food charges.
