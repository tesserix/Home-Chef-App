import { describe, it, expect } from '@jest/globals';

import {
  chefPayoutAmount,
  isPayoutEstimated,
  payoutHeadlineLabel,
  type ChefPayout,
} from './chefPayout';

// The order from the report: the customer was billed ₹745.06, the kitchen earns
// ₹679.16. Every chef surface must show the second number.
const payout: ChefPayout = {
  foodAmount: 640,
  deliveryFee: 39.16,
  chefTip: 0,
  penalty: 0,
  netPayout: 679.16,
  currency: 'INR',
  status: 'estimated',
};

describe('chefPayoutAmount', () => {
  it('shows what the chef earns, never the customer total', () => {
    expect(chefPayoutAmount(payout, 745.06)).toBe(679.16);
  });

  it('renders netPayout as served — the client never re-adds the lines', () => {
    // A payout whose lines would not re-sum (a penalty the client can't see)
    // must still render the server's figure.
    expect(chefPayoutAmount({ ...payout, netPayout: 600 }, 745.06)).toBe(600);
  });

  it('falls back to the customer total only when the API served no payout', () => {
    // An app running against an API older than the payout field. Showing the old
    // number beats showing nothing.
    expect(chefPayoutAmount(undefined, 745.06)).toBe(745.06);
    expect(chefPayoutAmount(null, 745.06)).toBe(745.06);
  });

  it('renders a zero payout rather than falling back', () => {
    // A penalty can wipe an order's payout to ₹0. That is a real figure, not a
    // missing one — falling back here would show the customer's total instead.
    expect(chefPayoutAmount({ ...payout, netPayout: 0 }, 745.06)).toBe(0);
  });
});

describe('payoutHeadlineLabel', () => {
  it('never calls money paid before it has moved', () => {
    // `pending` means computed and owed — it settles on the weekly payout. Only
    // `released` is money in the chef's account.
    expect(payoutHeadlineLabel({ ...payout, status: 'pending' })).toBe("You'll be paid");
    expect(payoutHeadlineLabel({ ...payout, status: 'released' })).toBe('You were paid');
  });

  it('reads as a projection before delivery', () => {
    expect(payoutHeadlineLabel(payout)).toBe("You'll earn");
  });

  it('names a reversal rather than implying a payment', () => {
    expect(payoutHeadlineLabel({ ...payout, status: 'reversed' })).toBe('Reversed');
  });
});

describe('isPayoutEstimated', () => {
  it('is true before delivery and false once the row is written', () => {
    expect(isPayoutEstimated(payout)).toBe(true);
    expect(isPayoutEstimated({ ...payout, status: 'pending' })).toBe(false);
    expect(isPayoutEstimated({ ...payout, status: 'released' })).toBe(false);
    expect(isPayoutEstimated(undefined)).toBe(false);
  });
});
