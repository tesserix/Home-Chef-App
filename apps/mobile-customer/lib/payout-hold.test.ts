import { describe, it, expect } from '@jest/globals';

import { canConfirmReceipt, payoutHoldMeta, type PayoutHoldStatus } from './payout-hold';

describe('canConfirmReceipt', () => {
  it('is true only for a delivered order awaiting confirmation', () => {
    expect(
      canConfirmReceipt({ status: 'delivered', payoutHoldStatus: 'awaiting_customer_confirmation' }),
    ).toBe(true);
  });

  it('is false when the order is not yet delivered (even if awaiting)', () => {
    // Defensive: the hold is only parked on delivery, but never show the CTA pre-delivery.
    expect(
      canConfirmReceipt({ status: 'delivering', payoutHoldStatus: 'awaiting_customer_confirmation' }),
    ).toBe(false);
  });

  it('is false once confirmed / disputed / terminal', () => {
    for (const s of ['release_eligible', 'released', 'disputed', 'withheld', 'reversed'] as PayoutHoldStatus[]) {
      expect(canConfirmReceipt({ status: 'delivered', payoutHoldStatus: s })).toBe(false);
    }
  });

  it('is false with no hold — the escrow flags are off (empty / undefined)', () => {
    expect(canConfirmReceipt({ status: 'delivered', payoutHoldStatus: '' })).toBe(false);
    expect(canConfirmReceipt({ status: 'delivered' })).toBe(false);
  });
});

describe('payoutHoldMeta', () => {
  it('labels the confirmed states (release_eligible/released) as Received', () => {
    expect(payoutHoldMeta('release_eligible').label).toBe('Received');
    expect(payoutHoldMeta('released').label).toBe('Received');
  });

  it('labels a disputed hold as under review', () => {
    expect(payoutHoldMeta('disputed').label).toBe('Issue under review');
  });

  it('renders no pill for awaiting (that state shows the CTA), terminal, or no-hold', () => {
    for (const s of ['awaiting_customer_confirmation', 'withheld', 'reversed', ''] as PayoutHoldStatus[]) {
      expect(payoutHoldMeta(s).label).toBe('');
    }
    expect(payoutHoldMeta(undefined).label).toBe('');
  });
});
