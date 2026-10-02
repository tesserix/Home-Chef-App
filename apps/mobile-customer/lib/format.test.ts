import { describe, it, expect } from '@jest/globals';
import { currencyForCountry, currencySymbol, formatMoney } from './format';

describe('formatMoney', () => {
  it('preserves Indian grouping for existing callers', () => {
    expect(formatMoney(123456)).toBe('₹1,23,456');
    expect(formatMoney(780.5)).toBe('₹780.50');
  });

  it.each(['AUD', 'NZD'])('renders %s in its home-market style without converting', (currency) => {
    expect(formatMoney(123456.5, currency)).toBe('$123,456.50');
  });

  it('normalizes server currency codes and handles missing amounts', () => {
    expect(formatMoney(undefined, ' aud ')).toBe('$0');
    expect(formatMoney(Number.NaN, 'NZD')).toBe('$0');
    expect(formatMoney(5, 'usd')).toBe('USD 5');
  });

  it('gives an input-prefix symbol per currency', () => {
    expect(currencySymbol('AUD')).toBe('$');
    expect(currencySymbol(undefined)).toBe('₹');
  });
});

describe('currencyForCountry', () => {
  it('maps each market to its currency and defaults to INR', () => {
    expect(currencyForCountry('AU')).toBe('AUD');
    expect(currencyForCountry('nz')).toBe('NZD');
    expect(currencyForCountry('IN')).toBe('INR');
    expect(currencyForCountry(undefined)).toBe('INR');
    expect(currencyForCountry('US')).toBe('INR');
  });
});
