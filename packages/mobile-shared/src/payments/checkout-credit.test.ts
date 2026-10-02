import { describe, expect, it } from 'vitest';
import { checkoutCreditIntent } from './checkout-credit';

describe('checkoutCreditIntent', () => {
  const intent = { useWallet: true, useLoyalty: true, walletAmount: 25, loyaltyPoints: 100 };
  it('disables unsupported credits for Stripe without forwarding INR amounts', () => {
    expect(checkoutCreditIntent('stripe', intent)).toEqual({ useWallet: false, useLoyalty: false });
    expect(intent.walletAmount).toBe(25);
  });
  it('preserves the customer credit choices for Cashfree', () => {
    expect(checkoutCreditIntent('cashfree', intent)).toEqual(intent);
  });
  it.each([undefined, '', 'unknown'])('requires a resolved gateway before placing an order (%s)', (provider) => {
    expect(() => checkoutCreditIntent(provider, intent)).toThrow('Please wait for your payment quote and try again.');
  });
});
