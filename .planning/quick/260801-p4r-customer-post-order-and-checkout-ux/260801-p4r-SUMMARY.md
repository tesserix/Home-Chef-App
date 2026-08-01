---
gsd_summary_version: 1.0
quick_id: 260801-p4r
slug: customer-post-order-and-checkout-ux
status: complete
date: 2026-08-01
issues: [874, 868, 871]
branch: fix/post-order-nav-and-confirm-refresh
commits:
  - 50024d0e
  - 512cd050
---

# Quick Task 260801-p4r — Summary

Four customer-app fixes: three tracked issues plus one spacing defect the owner
spotted during live testing.

## What changed

**`50024d0e`**

- **#874 — back from order detail landed in the cart.** `payment/result.tsx`
  used `router.replace`, which swaps only the current screen and leaves
  checkout → chef → cart intact underneath. Now pops to the tabs root
  (`canDismiss()` → `dismissAll()`) and pushes, so the stack is Home → Order.
  Applied to both the "View order" and "My orders" CTAs.
- **#868 — order status needed a manual refresh after "Confirm received".**
  `useConfirmOrderReceived` now writes the mutation result straight into the
  `['order', id]` cache with `setQueryData`, then `refetchQueries` (not
  `invalidateQueries`) to reconcile. A delivered order has already stopped
  polling, so invalidation was only as good as a refetch that wasn't running.

**`512cd050`**

- **#871 — slot grid collapsed behind an affordance.** While "As soon as ready"
  is selected the chips are replaced by a single "Choose a specific time" row;
  picking a time (or tapping the row) reveals the grid, and re-selecting ASAP
  collapses it and clears the pick.
- **Terms card spacing** (owner-reported, no issue filed). The consent card had
  `mt-4` and no bottom margin, so it sat flush against the next section's top
  hairline. Now `mb-4` to match.

## Verification

- **tsc:** 1 error before and after — the known pre-existing
  `lib/payment.ts(114,7)` expo-router path typing. Zero new.
- **#874 — verified end-to-end by the owner** on the Android emulator through a
  real sandbox checkout (Cashfree `testMerchantName`, card + OTP): View order →
  back → dashboard. The checkout stack is gone.
- **#871 + terms spacing — verified by the owner** on the emulator after reload.
- **Regression:** orders list and order-detail screens render correctly; no new
  JS errors in logcat.

## NOT verified — read before trusting #868

**#868's fix was never exercised.** It requires a *delivered* order sitting in
`awaiting_customer_confirmation`, and no such order exists on the test account —
every historical order is `Cancelled`, and the order placed during this session
is only `Confirmed`. The change is reasoned from the code path and typechecks,
but nobody has watched the status pill update.

That is precisely the shape of the bug caught in #877 (drag-to-dismiss looked
correct, typechecked, and did nothing). Treat #868 as shipped-but-unproven; the
next real delivered order confirms or refutes it.

## Notes for next time

- The emulator talks to **production** (`https://fe3dr.com/api`). Test orders
  are real rows a real chef sees; payments are Cashfree *sandbox*, so no real
  money moves.
- Fast-refresh silently stopped applying at one point — the file was correct on
  disk but the screen wasn't. `am force-stop` + relaunch fixed it. If a change
  "isn't showing", check the file first, then restart the app before debugging
  the code.
