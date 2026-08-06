import { describe, it, expect } from 'vitest';

// #1041: a cancelled plan showed struck-through meals and a bare total, never the refund. The
// only amount anywhere was a wallet line reading "refund (75%)" against a total that percentage
// does not apply to, leaving the customer ₹17 they could not account for.

import { mealPlanRefundSummary } from '../utils/meal-plan-refund-summary';

// The plan from the bug report: ₹240.27 paid, ₹162.89 back.
const plan = {
  total: 240.27,
  days: [{ refundAmount: 162.89 }, { refundAmount: undefined }],
};

describe('mealPlanRefundSummary', () => {
  it('states what was paid, what came back, and what did not', () => {
    expect(mealPlanRefundSummary(plan)).toEqual({
      paid: 240.27,
      refunded: 162.89,
      withheld: 77.38,
    });
  });

  it('sums a refund spread across several days', () => {
    const summary = mealPlanRefundSummary({
      total: 300,
      days: [{ refundAmount: 80 }, { refundAmount: 60 }, {}],
    });

    expect(summary?.refunded).toBe(140);
    expect(summary?.withheld).toBe(160);
  });

  // Nothing refunded yet — a live plan, or one every day of which was forfeited.
  it('returns nothing when no day was refunded', () => {
    expect(mealPlanRefundSummary({ total: 300, days: [{}, {}] })).toBeNull();
  });

  // Truncation means the days can land a paise under the total; never report a negative
  // withheld amount from rounding.
  it('never reports a negative withheld amount', () => {
    const summary = mealPlanRefundSummary({ total: 100, days: [{ refundAmount: 100.01 }] });

    expect(summary?.withheld).toBe(0);
  });
});
