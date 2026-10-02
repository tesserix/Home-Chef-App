import { describe, it, expect } from '@jest/globals';
import { formatMoney } from './format';

describe('formatMoney', () => {
  it('preserves Indian grouping for existing callers', () => {
    expect(formatMoney(123456)).toBe('₹1,23,456');
    expect(formatMoney(780.5)).toBe('₹780.50');
  });

  it.each(['AUD', 'NZD'])('labels %s amounts without converting them', (currency) => {
    expect(formatMoney(123456.5, currency)).toBe(`${currency} 123,456.50`);
  });

  it('normalizes server currency codes and handles missing amounts', () => {
    expect(formatMoney(undefined, ' aud ')).toBe('AUD 0');
    expect(formatMoney(Number.NaN, 'NZD')).toBe('NZD 0');
  });
});
