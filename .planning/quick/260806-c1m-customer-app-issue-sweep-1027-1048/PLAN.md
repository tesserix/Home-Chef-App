---
type: quick
slug: customer-app-issue-sweep-1027-1048
created: 2026-08-06
---

# Customer app issue sweep (#1027–#1048)

Batch of client-side defects in `apps/mobile-customer` filed by the 2026-08-05/06
E2E sweep. All are display/transparency defects except #1038 (a hard crash).
No API changes — every figure needed is already on the wire.

## Tasks (one atomic commit each)

1. **#1038 — receipt screen crashes.** `lib/share-pdf.ts` imports `expo-sharing`
   and `expo-file-system/legacy` at module scope; `app/order/[id]/receipt.tsx`
   imports it at top level, so a binary without the native module takes down the
   whole route. Move to a dynamic `import()` inside the Share press handler.

2. **#1027 — receipt Total is the pre-wallet gross.** Add the wallet/loyalty
   credit lines and an "Amount paid" line, using `order.walletApplied` /
   `order.loyaltyApplied` (already on the model), matching order detail.

3. **#1048 / #1033 — cancelled order calls the withheld residual "Platform fee".**
   `app/order/[id]/index.tsx` prints `total − refund − vendorKept` under the
   platform-fee label, contradicting the platform-fee line four rows above.
   Split into the real platform fee and the withheld tax remainder.

4. **#1040 — meal plan list counts meals but says "days".**
   `components/meal-plan/MealPlanList.tsx` labels `plan.days.length` (booked
   meals) as days. Label it meals.

5. **#1039 — meal plan detail shows a bare Total.** Render
   `mealPlanAdvanceBreakdown(plan)` (already written for exactly this, used only
   by the approval hook) so food/GST/delivery reconcile to the total, and use its
   total for the "If approved" figure instead of the food-only sum.

6. **#1042 — subscription card omits next charge date and chef.** Use
   `currentPeriodEnd` and `chefId` (both on `MealSubscription`, both unused) so
   three identical-looking cards are distinguishable and a recurring charge
   states when it lands.

7. **#1047 — own review offers Report and Block yourself.** Branch
   `components/chef/ChefReviewList.tsx` on `review.customerId === currentUser.id`
   and suppress the moderation sheet on your own review. Edit/Delete needs
   backend endpoints that do not exist (`/v1/reviews` is POST + GET-by-order
   only) — out of scope here, noted on the issue.

## Verification

`npx tsc --noEmit`, jest, then run the app on the Android emulator
(`emulator-5554`, `com.tesserix.homechef.customer`) over Metro on 8082.
