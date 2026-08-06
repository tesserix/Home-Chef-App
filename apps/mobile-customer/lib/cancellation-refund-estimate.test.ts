import { describe, it, expect } from '@jest/globals';
import {
  estimateCancellationRefund,
  toPaise,
  MAX_FOOD_REFUND_PCT,
  type CancellationRefundEstimateInput,
} from './cancellation-refund-estimate';

// #1032 — the pre-cancellation refund estimate. Money logic, so every branch is
// pinned: an overstated bound here is a promise the platform cannot keep, which
// is the failure this feature exists to prevent.

/** The real order from #1032: ₹404.07 paid, ₹387.60 refunded, ₹16.47 retained
 *  (platform fee ₹13.96 + its 18% GST ₹2.51). */
function order(
  over: Partial<CancellationRefundEstimateInput> = {},
): CancellationRefundEstimateInput {
  return {
    status: 'accepted',
    subtotalPaise: 30000,
    discountPaise: 0,
    deliveryFeePaise: 4000,
    platformFeePaise: 1396,
    taxPaise: 2000,
    totalPaise: 37396,
    alreadyRefundedPaise: 0,
    ...over,
  };
}

describe('toPaise', () => {
  it('rounds a rupee float to whole paise', () => {
    expect(toPaise(404.07)).toBe(40407);
    expect(toPaise(0.005)).toBe(1);
  });

  it('treats a missing or non-finite amount as zero', () => {
    expect(toPaise(undefined)).toBe(0);
    expect(toPaise(null)).toBe(0);
    expect(toPaise(Number.NaN)).toBe(0);
  });
});

describe('estimateCancellationRefund', () => {
  it('quotes an exact figure on a pending order, where no chef tier applies', () => {
    const e = estimateCancellationRefund(order({ status: 'pending' }));
    expect(e.exact).toBe(true);
    // Food in full + delivery in full; only the platform fee (and its GST) stays.
    expect(e.minRefundPaise).toBe(34000);
    expect(e.maxRefundPaise).toBe(34000);
  });

  it('quotes a range on an accepted order, floored at delivery only', () => {
    const e = estimateCancellationRefund(order({ status: 'accepted' }));
    expect(e.exact).toBe(false);
    // Worst tier is 0% of the food, so only the delivery fee is guaranteed.
    expect(e.minRefundPaise).toBe(4000);
    // Best tier is 90% of ₹300 = ₹270, plus delivery.
    expect(e.maxRefundPaise).toBe(27000 + 4000);
  });

  it('applies the same range to a preparing order', () => {
    const e = estimateCancellationRefund(order({ status: 'preparing' }));
    expect(e.minRefundPaise).toBe(4000);
    expect(e.maxRefundPaise).toBe(31000);
  });

  it('never counts the platform fee as refundable', () => {
    const e = estimateCancellationRefund(order({ status: 'pending' }));
    expect(e.platformFeeKeptPaise).toBe(1396);
    expect(e.maxRefundPaise).toBeLessThanOrEqual(
      e.foodPaise + e.deliveryRefundPaise,
    );
  });

  it('excludes GST from both bounds so the estimate can only resolve upward', () => {
    const e = estimateCancellationRefund(order({ status: 'pending', taxPaise: 2000 }));
    expect(e.taxPaise).toBe(2000);
    // 34000, not 34000 + any share of the 2000 tax.
    expect(e.maxRefundPaise).toBe(34000);
  });

  it('applies the tier to the food the customer paid, not the list price (#962)', () => {
    const e = estimateCancellationRefund(
      order({ status: 'accepted', subtotalPaise: 30000, discountPaise: 10000 }),
    );
    expect(e.foodPaise).toBe(20000);
    expect(e.maxRefundPaise).toBe(18000 + 4000); // 90% of ₹200
  });

  it('floors the food base at zero when the discount exceeds the subtotal', () => {
    const e = estimateCancellationRefund(
      order({ status: 'pending', subtotalPaise: 5000, discountPaise: 9000 }),
    );
    expect(e.foodPaise).toBe(0);
    expect(e.minRefundPaise).toBe(4000); // delivery only
  });

  it('refunds the delivery fee in full at every cancellable status', () => {
    for (const status of ['pending', 'accepted', 'preparing']) {
      expect(estimateCancellationRefund(order({ status })).deliveryRefundPaise).toBe(4000);
    }
  });

  it('stops refunding delivery once a driver is carrying the order', () => {
    for (const status of ['picked_up', 'delivering', 'delivered']) {
      const e = estimateCancellationRefund(order({ status }));
      expect(e.deliveryRefundPaise).toBe(0);
      expect(e.minRefundPaise).toBe(0);
    }
  });

  it('caps both bounds at what is still owed after a prior partial refund (#642)', () => {
    const e = estimateCancellationRefund(
      order({ status: 'pending', alreadyRefundedPaise: 35000 }),
    );
    expect(e.minRefundPaise).toBe(2396); // 37396 − 35000
    expect(e.maxRefundPaise).toBe(2396);
    expect(e.exact).toBe(true);
  });

  it('quotes nothing once the order has been refunded in full', () => {
    const e = estimateCancellationRefund(
      order({ status: 'accepted', alreadyRefundedPaise: 40000 }),
    );
    expect(e.minRefundPaise).toBe(0);
    expect(e.maxRefundPaise).toBe(0);
  });

  it('honours an admin-configured top tier', () => {
    const e = estimateCancellationRefund(order({ status: 'accepted', maxFoodRefundPct: 50 }));
    expect(e.maxRefundPaise).toBe(15000 + 4000);
  });

  it('clamps an out-of-range top tier rather than over-refunding', () => {
    expect(estimateCancellationRefund(order({ maxFoodRefundPct: 500 })).maxRefundPaise).toBe(
      30000 + 4000,
    );
    expect(estimateCancellationRefund(order({ maxFoodRefundPct: -20 })).maxRefundPaise).toBe(4000);
  });

  it('truncates the tier share the way the Go model does, never rounding up', () => {
    // 90% of 33333 paise = 29999.7 → the server pays 29999.
    const e = estimateCancellationRefund(
      order({ status: 'accepted', subtotalPaise: 33333, deliveryFeePaise: 0 }),
    );
    expect(e.maxRefundPaise).toBe(29999);
  });

  it('handles a pickup order with no delivery fee', () => {
    const e = estimateCancellationRefund(
      order({ status: 'accepted', deliveryFeePaise: 0 }),
    );
    expect(e.minRefundPaise).toBe(0);
    expect(e.maxRefundPaise).toBe(27000);
    expect(e.exact).toBe(false);
  });

  it('treats an unknown status like an accepted one — chef decides, nothing promised', () => {
    const e = estimateCancellationRefund(order({ status: 'something_new' }));
    expect(e.minRefundPaise).toBe(4000);
    expect(e.maxRefundPaise).toBe(31000);
  });

  it('never quotes more than the top tier allows', () => {
    expect(MAX_FOOD_REFUND_PCT).toBe(90);
    const e = estimateCancellationRefund(order({ status: 'accepted' }));
    expect(e.maxRefundPaise).toBeLessThan(e.foodPaise + e.deliveryRefundPaise);
  });
});
