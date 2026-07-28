import { describe, expect, it } from 'vitest';
import { canConfirmReceipt, payoutHoldMeta } from './payout-hold';

// The gate must match apps/mobile-customer/lib/payout-hold.ts exactly — a
// customer who ordered on the phone and opened the web app (or vice versa) has
// to see the same thing, and the confirm endpoint only accepts the one state.

describe('canConfirmReceipt', () => {
  it('shows the CTA only on a delivered order awaiting confirmation', () => {
    expect(
      canConfirmReceipt({ status: 'delivered', payoutHoldStatus: 'awaiting_customer_confirmation' }),
    ).toBe(true);
  });

  it('stays hidden before the order is delivered', () => {
    for (const status of ['pending', 'accepted', 'preparing', 'ready', 'picked_up']) {
      expect(
        canConfirmReceipt({ status, payoutHoldStatus: 'awaiting_customer_confirmation' }),
      ).toBe(false);
    }
  });

  it('stays hidden once the hold has moved on', () => {
    for (const held of ['release_eligible', 'released', 'disputed', 'withheld', 'reversed'] as const) {
      expect(canConfirmReceipt({ status: 'delivered', payoutHoldStatus: held })).toBe(false);
    }
  });

  it('stays hidden when there is no hold at all', () => {
    // A non-gateway-charged order never parks a hold, so the CTA must not appear
    // — the endpoint would reject it.
    expect(canConfirmReceipt({ status: 'delivered' })).toBe(false);
    expect(canConfirmReceipt({ status: 'delivered', payoutHoldStatus: '' })).toBe(false);
  });
});

describe('payoutHoldMeta', () => {
  it('labels the confirmed states', () => {
    expect(payoutHoldMeta('release_eligible').label).toBe('Received');
    expect(payoutHoldMeta('released').label).toBe('Received');
  });

  it('treats a dispute as under review, not an error', () => {
    expect(payoutHoldMeta('disputed').label).toBe('Issue under review');
  });

  it('renders nothing while the CTA owns the state, or when there is no hold', () => {
    expect(payoutHoldMeta('awaiting_customer_confirmation').label).toBe('');
    expect(payoutHoldMeta(undefined).label).toBe('');
    expect(payoutHoldMeta('').label).toBe('');
    expect(payoutHoldMeta('withheld').label).toBe('');
    expect(payoutHoldMeta('reversed').label).toBe('');
  });
});
