import { describe, it, expect } from '@jest/globals';

import { paymentSecuredByLine } from './payment-provider';

describe('paymentSecuredByLine', () => {
  // The live defect: checkout said "Razorpay" while the Cashfree sheet took the
  // money seconds later in the same flow.
  it('names Cashfree when Cashfree will take the payment', () => {
    expect(paymentSecuredByLine('cashfree')).toBe('Payments secured by Cashfree (RBI-licensed).');
  });

  it('still names Razorpay when Razorpay is the resolved gateway', () => {
    expect(paymentSecuredByLine('razorpay')).toBe('Payments secured by Razorpay (RBI-licensed).');
  });

  it('does not claim RBI licensing for Stripe', () => {
    expect(paymentSecuredByLine('stripe')).toBe('Payments secured by Stripe.');
  });

  it('is case- and whitespace-insensitive, matching NormalizeProvider', () => {
    expect(paymentSecuredByLine('  CashFree ')).toBe('Payments secured by Cashfree (RBI-licensed).');
  });

  it('never guesses a brand when the provider is missing', () => {
    const generic = 'Payments are processed by an RBI-licensed payment aggregator.';
    expect(paymentSecuredByLine(undefined)).toBe(generic);
    expect(paymentSecuredByLine(null)).toBe(generic);
    expect(paymentSecuredByLine('')).toBe(generic);
    expect(paymentSecuredByLine('some-new-gateway')).toBe(generic);
  });
});
