import { describe, it, expect } from '@jest/globals';

import { formatINR, formatCount, formatDate, formatRelative, titleCase, errorMessage } from './format';

// These render every money figure an admin reads before acting on it — a payout
// queue, a refund arbitration, a wallet adjustment. Indian grouping is not what
// `toLocaleString('en-US')` produces, and getting it wrong misreads a lakh as a
// hundred thousand at a glance.

describe('formatINR', () => {
  it('groups by lakh and crore, not by thousand', () => {
    expect(formatINR(123456.78)).toBe('₹1,23,456.78');
    expect(formatINR(10000000)).toBe('₹1,00,00,000.00');
  });

  it('leaves amounts under a thousand ungrouped', () => {
    expect(formatINR(999.5)).toBe('₹999.50');
    expect(formatINR(0)).toBe('₹0.00');
  });

  it('puts the sign before the symbol on a negative', () => {
    expect(formatINR(-1500)).toBe('-₹1,500.00');
  });

  it('renders zero rather than NaN for a missing or broken amount', () => {
    // An admin screen showing "₹NaN" beside an Approve button is worse than one
    // showing ₹0.00 — the first invites a guess.
    expect(formatINR(null)).toBe('₹0.00');
    expect(formatINR(undefined)).toBe('₹0.00');
    expect(formatINR(Number.NaN)).toBe('₹0.00');
    expect(formatINR(Number.POSITIVE_INFINITY)).toBe('₹0.00');
  });

  it('rounds to two decimals', () => {
    expect(formatINR(1.005)).toBe('₹1.00');
    expect(formatINR(1.006)).toBe('₹1.01');
  });
});

describe('formatCount', () => {
  it('formats a count and defaults a missing one to zero', () => {
    expect(formatCount(1234)).toBe('1,234');
    expect(formatCount(null)).toBe('0');
    expect(formatCount(Number.NaN)).toBe('0');
  });
});

describe('date formatting', () => {
  it('formats an ISO timestamp', () => {
    expect(formatDate('2026-06-21T16:32:00.000Z')).toMatch(/^2[12] Jun 2026$/);
  });

  it('falls back to a dash rather than "Invalid Date"', () => {
    expect(formatDate(null)).toBe('—');
    expect(formatDate('not-a-date')).toBe('—');
    expect(formatRelative(undefined)).toBe('—');
  });
});

describe('titleCase', () => {
  it('turns a snake or kebab enum into readable words', () => {
    expect(titleCase('payout_hold_released')).toBe('Payout Hold Released');
    expect(titleCase('meal-plan')).toBe('Meal Plan');
  });

  it('returns an empty string for nothing', () => {
    expect(titleCase(null)).toBe('');
    expect(titleCase('')).toBe('');
  });
});

describe('errorMessage', () => {
  it('reads the API envelope in both of its shapes', () => {
    expect(errorMessage({ response: { data: { error: { message: 'Chef not found' } } } })).toBe(
      'Chef not found',
    );
    expect(errorMessage({ response: { data: { error: 'Refund already issued' } } })).toBe(
      'Refund already issued',
    );
    expect(errorMessage({ response: { data: { message: 'Forbidden' } } })).toBe('Forbidden');
  });

  it('prefers the envelope over the axios message', () => {
    const err = {
      response: { data: { error: { message: 'Below the refund floor' } } },
      message: 'Request failed with status code 422',
    };
    expect(errorMessage(err)).toBe('Below the refund floor');
  });

  it('falls back to the axios message, then to a generic line', () => {
    expect(errorMessage({ message: 'Network Error' })).toBe('Network Error');
    expect(errorMessage(null)).toBe('Something went wrong. Please try again.');
    expect(errorMessage('a bare string')).toBe('Something went wrong. Please try again.');
  });
});
