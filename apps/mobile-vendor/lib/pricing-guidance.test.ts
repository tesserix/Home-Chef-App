import { describe, it, expect } from '@jest/globals';

import { pricingHint, RESTAURANT_PARITY_PRICE } from './pricing-guidance';

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
