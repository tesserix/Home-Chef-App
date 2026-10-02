import { describe, it, expect } from '@jest/globals';
import { priceFilterOptions } from './price-filter';

describe('priceFilterOptions', () => {
  it('keeps the rupee buckets for India', () => {
    expect(priceFilterOptions('INR').map((o) => o.label)).toEqual([
      'Any price',
      '< ₹100',
      '< ₹250',
      '< ₹500',
    ]);
  });

  it('uses dollar buckets sized for an AU or NZ dish', () => {
    expect(priceFilterOptions('AUD')).toEqual([
      { label: 'Any price', value: undefined },
      { label: '< $15', value: 15 },
      { label: '< $25', value: 25 },
      { label: '< $40', value: 40 },
    ]);
    expect(priceFilterOptions('NZD').map((o) => o.value)).toEqual([undefined, 15, 25, 40]);
  });
});
