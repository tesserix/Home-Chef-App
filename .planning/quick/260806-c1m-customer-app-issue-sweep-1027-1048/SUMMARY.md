---
type: quick
slug: customer-app-issue-sweep-1027-1048
status: complete
completed: 2026-08-06
---

# Customer app issue sweep (#1027–#1048) — summary

Seven atomic commits, all in `apps/mobile-customer`, no API changes.

| Issue | Commit | Change |
|---|---|---|
| #1038 | `ebaf2013` | `lib/share-pdf.ts` loads `expo-file-system` / `expo-sharing` via dynamic `import()` inside the call. Nothing native is touched at module scope, so importing the module can no longer take down the receipt route on a binary that predates the dependency — the OTA hazard the issue called out. |
| #1027 | `6868cdd4` | Receipt headline and totals now show wallet/loyalty credit and the amount actually charged, from `walletApplied`/`loyaltyApplied` — the same fields and arithmetic order detail uses. |
| #1048, #1033 | `a883a9c7` | The cancelled-order residual is itemised into platform fee, withheld delivery and withheld GST, derived from the refund snapshot (`foodRefundPaise` / `deliveryRefundPaise` / `taxRefundPaise`, always sent, previously undeclared on the client type). The fee line now equals the fee line above it; tax is taken as the remainder so the lines always sum to the retained total. |
| #1040 | `e9d89bf4` | Plan rows say "5 meals", not "5 days" — `plan.days` is the booked-meal array. |
| #1039 | `db03349b` | Plan detail itemises food / delivery / GST above the total via a new `mealPlanSubsetBreakdown()`, which also prices the "If approved" figure (was the food-only sum — #402 on this screen). Unit-tested. |
| #1042 | `8e031aba` | Subscription cards name the kitchen (`useChef(sub.chefId)`) and the next charge (`currentPeriodEnd`), via a tested `lib/subscription-format.ts`. |
| #1047 | `67b39037` | Your own review shows a "Your review" marker instead of the report/block overflow — no more reporting yourself or blocking yourself. |

Verification: `tsc --noEmit` clean (one pre-existing unrelated error in
`app/(tabs)/profile.tsx:401`, the untyped `/feedback` route); `jest` 115/115 green
(17 suites, 10 new tests).

## Deliberately not done — needs backend work

- **#1047 Edit / Delete review.** `/v1/reviews` is `POST` + `GET /order/:orderId`
  only. There is no update or delete endpoint, so the constructive half of the
  issue needs an API change. The harmful half (self-report, self-block) is fixed.
- **#1041 cancelled meal plan never shows the refund amount.** `MealPlan` carries
  no refund figure at all — refunds live per-day on the server
  (`RefundPercent`, `RefundStage`). Disclosing the amount and its base needs the
  plan response to expose the escrow split first.
- **#1027 payment reference on the receipt.** No gateway reference (Cashfree
  order id / UTR) exists on the client order model; needs an API field.
