import { describe, it, expect, jest } from '@jest/globals';
jest.mock('../lib/api', () => ({ api: {} }));
import { refundAtPercent, type RefundDecisionDay } from './useMealPlans';

describe('refund percentage preview', () => {
  it('matches the server floor to paise instead of rounding up', () => {
    expect(refundAtPercent({ fullRefund: 222.93 } as RefundDecisionDay, 75)).toBe(167.19);
  });
  it('retains the exact full refund at 100 percent', () => {
    expect(refundAtPercent({ fullRefund: 222.93 } as RefundDecisionDay, 100)).toBe(222.93);
  });
});
