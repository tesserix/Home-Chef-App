import { describe, it, expect } from '@jest/globals';

import { weekSummary } from './week-summary';

describe('weekSummary', () => {
  it('pairs the money with the count that produced it', () => {
    // Every order placed this week has also been delivered, so one count says
    // everything.
    expect(weekSummary({ revenue: 1200, settledOrders: 5, orders: 5 })).toEqual({
      key: 'dashboard.weekSummary',
      params: { amount: '1,200', count: 5 },
    });
  });

  it('names both counts when some orders are still in the kitchen', () => {
    // The defect: "₹1,200 · 5 orders" when only 3 delivered — the money and the
    // count came from different populations and the arithmetic did not work.
    expect(weekSummary({ revenue: 1200, settledOrders: 3, orders: 5 })).toEqual({
      key: 'dashboard.weekSummaryPartial',
      params: { amount: '1,200', settled: 3, count: 5 },
    });
  });

  it('reports the week as activity, not ₹0, before anything is delivered', () => {
    expect(weekSummary({ revenue: 0, settledOrders: 0, orders: 4 })).toEqual({
      key: 'dashboard.weekSummaryPending',
      params: { count: 4 },
    });
  });

  it('shows nothing for a week with no orders at all', () => {
    expect(weekSummary({ revenue: 0, settledOrders: 0, orders: 0 })).toBeNull();
  });

  it('groups the amount in the Indian system and rounds to the rupee', () => {
    expect(weekSummary({ revenue: 123456.78, settledOrders: 9, orders: 9 })?.params).toEqual({
      amount: '1,23,457',
      count: 9,
    });
  });

  it('survives a response that predates weekSettledOrders', () => {
    // An older API build omits the field; treating undefined as "all of them"
    // keeps the line honest rather than claiming nothing was delivered.
    expect(weekSummary({ revenue: 800, settledOrders: undefined, orders: 3 })).toEqual({
      key: 'dashboard.weekSummary',
      params: { amount: '800', count: 3 },
    });
  });

  it('never claims more delivered than placed', () => {
    expect(weekSummary({ revenue: 900, settledOrders: 7, orders: 3 })).toEqual({
      key: 'dashboard.weekSummary',
      params: { amount: '900', count: 3 },
    });
  });
});
