import { describe, it, expect } from '@jest/globals';
import { deliveryFeeLabel } from './delivery-fee-label';

describe('deliveryFeeLabel', () => {
  it.each(['AUD', 'NZD'])('uses %s for delivery fees', (currency) => {
    expect(deliveryFeeLabel(5, true, true, currency)).toBe('$5 delivery');
    expect(deliveryFeeLabel(5, false, true, currency)).toBe('Delivery from $5');
  });
  it('preserves legacy, free and pickup labels', () => {
    expect(deliveryFeeLabel(5, true, true)).toBe('₹5 delivery');
    expect(deliveryFeeLabel(0, true, true, 'AUD')).toBe('Free delivery');
    expect(deliveryFeeLabel(0, false, true, 'NZD')).toBe('Free delivery nearby');
    expect(deliveryFeeLabel(5, true, false, 'AUD')).toBe('Pickup only');
    expect(deliveryFeeLabel(undefined, true, true, 'NZD')).toBeUndefined();
  });
});
