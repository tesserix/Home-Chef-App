// chefPayout.ts — the one figure every chef-facing surface shows for an order.
//
// The customer's total carries the platform fee and the GST the platform
// accounts for, so a kitchen reading ₹745.06 on an order it earns ₹679.16 for
// can never reconcile a payout. Every card, row and detail screen renders
// `netPayout` from the API instead, and none of them recompute it — the server
// runs one formula for the estimate a chef accepts on and the row written at
// delivery. See .planning/CHEF-PAYOUT-VIEW-DESIGN.md.

/** What the chef is paid for one order, served by the API. */
export interface ChefPayout {
  foodAmount: number;
  /** 0 unless the chef carried the leg and it was charged; the UI then omits the row. */
  deliveryFee: number;
  chefTip: number;
  penalty: number;
  /** The exact sum of the lines above, to the paise. Render this, never a local sum. */
  netPayout: number;
  currency: string;
  /** `estimated` before delivery (the order can still change), then `pending` →
   *  `released` / `reversed` once the row is written and settled. */
  status: string;
}

/** True while the figure is still a projection of an order in flight. */
export function isPayoutEstimated(payout?: ChefPayout | null): boolean {
  return payout?.status === 'estimated';
}

/**
 * The headline label for the payout total, in the tense the status warrants.
 *
 * Money that has not moved must never be described as paid: `pending` is owed
 * and settles on the weekly payout, and only `released` is in the chef's
 * account.
 */
export function payoutHeadlineLabel(payout?: ChefPayout | null): string {
  switch (payout?.status) {
    case 'released':
      return 'You were paid';
    case 'reversed':
      return 'Reversed';
    case 'estimated':
      return "You'll earn";
    default:
      return "You'll be paid";
  }
}

/**
 * The amount a chef-facing card should show for an order.
 *
 * Falls back to the customer total only when the API served no payout at all —
 * an app running against an older API. Showing nothing would be worse than
 * showing the old number, but every current build gets the payout.
 */
export function chefPayoutAmount(
  payout: ChefPayout | undefined | null,
  fallbackTotal: number,
): number {
  return payout ? payout.netPayout : fallbackTotal;
}
