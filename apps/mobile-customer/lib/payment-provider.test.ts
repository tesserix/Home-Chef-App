import { describe, it, expect } from '@jest/globals';

import { paymentSecuredByLine } from './payment-provider';

describe('paymentSecuredByLine', () => {
  // The live defect: checkout named one brand while the Cashfree sheet took the
  // money seconds later in the same flow.
  it('names Cashfree when Cashfree will take the payment', () => {
    expect(paymentSecuredByLine('cashfree')).toBe('Payments secured by Cashfree (RBI-licensed).');
  });

  // Inverted by #1086. SelectCheckoutGateway can no longer resolve to the retired
  // gateway, so this value only reaches the client from a stale cache or a replayed
  // response — and naming a processor that will not touch the money is the defect
  // above, exactly.
  it('does not name the retired gateway, which takes no payment', () => {
    expect(paymentSecuredByLine('razorpay')).toBe(
      'Payments are processed by an RBI-licensed payment aggregator.',
    );
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
