import { describe, it, expect } from '@jest/globals';

import {
  pricingHint,
  RESTAURANT_PARITY_PRICE,
  BAKERY_PARITY_PRICE,
} from './pricing-guidance';

// Why: a home chef's edge over a restaurant is price for comparable food. If a
// chef prices at restaurant parity the customer has no reason to choose them —
// they'll just order from a restaurant instead. Chefs setting prices in
// isolation have no visibility of that, so the form should tell them at the
// moment they type the number.

describe('pricingHint', () => {
  it('says nothing while the price is still empty or unparseable', () => {
    expect(pricingHint('')).toBeNull();
    expect(pricingHint('   ')).toBeNull();
    expect(pricingHint('abc')).toBeNull();
  });

  it('says nothing for a comfortably home-priced dish', () => {
    expect(pricingHint('180')).toBeNull();
    expect(pricingHint('320')).toBeNull();
  });

  it('warns once the price reaches restaurant parity', () => {
    const hint = pricingHint(String(RESTAURANT_PARITY_PRICE));
    expect(hint).not.toBeNull();
    expect(hint!.tone).toBe('warn');
    expect(hint!.message).toMatch(/restaurant/i);
  });

  it('keeps warning above parity', () => {
    expect(pricingHint('750')?.tone).toBe('warn');
  });

  it('stays quiet just below parity, so the warning means something', () => {
    expect(pricingHint(String(RESTAURANT_PARITY_PRICE - 1))).toBeNull();
  });

  it('ignores a zero or negative price — that is validation, not guidance', () => {
    expect(pricingHint('0')).toBeNull();
    expect(pricingHint('-50')).toBeNull();
  });

  it('tolerates spacing and decimals a chef might type', () => {
    expect(pricingHint(' 500.00 ')?.tone).toBe('warn');
  });
});

// A whole celebration cake is not a plated meal: ₹1400 for a two-kilo truffle
// cake is the going rate, and telling a baker to stay under ₹500 is advice they
// can only ignore — which teaches them to ignore the next warning too.
describe('pricingHint for a bakery item', () => {
  it('stays quiet at a normal celebration-cake price', () => {
    expect(pricingHint('1450', { isBakery: true })).toBeNull();
  });

  it('warns only once the price passes what a bakery would charge', () => {
    const hint = pricingHint(String(BAKERY_PARITY_PRICE), { isBakery: true });
    expect(hint?.tone).toBe('warn');
    expect(hint!.message).toMatch(/bakery/i);
  });

  it('does not mention restaurants to a baker', () => {
    expect(pricingHint('5000', { isBakery: true })!.message).not.toMatch(/restaurant/i);
  });

  it('still warns a cooked-food kitchen at the lower line', () => {
    expect(pricingHint('1450', { isBakery: false })?.tone).toBe('warn');
  });
});

// The parity lines are rupee figures; applying them to dollars would flag
// every ordinary AU/NZ dish, so other currencies get no hint until they have
// their own.
describe('pricingHint outside India', () => {
  it('stays silent for AUD and NZD', () => {
    expect(pricingHint('600', { currency: 'AUD' })).toBeNull();
    expect(pricingHint('3000', { currency: 'NZD', isBakery: true })).toBeNull();
  });

  it('still warns for an explicit INR kitchen', () => {
    expect(pricingHint('600', { currency: 'INR' })?.message).toMatch(/₹600/);
  });
});
