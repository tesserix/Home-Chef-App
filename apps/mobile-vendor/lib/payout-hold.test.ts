import { describe, it, expect } from '@jest/globals';

import { payoutHoldMeta, statementStatusMeta, type PayoutHoldStatus } from './payout-hold';

describe('payoutHoldMeta', () => {
  it('maps each hold state to a distinct label key', () => {
    const cases: Record<Exclude<PayoutHoldStatus, ''>, string> = {
      awaiting_customer_confirmation: 'earnings.holdAwaiting',
      release_eligible: 'earnings.holdConfirmed',
      released: 'earnings.holdReleased',
      disputed: 'earnings.holdDisputed',
      withheld: 'earnings.holdWithheld',
      reversed: 'earnings.holdReversed',
    };
    for (const [status, labelKey] of Object.entries(cases)) {
      expect(payoutHoldMeta(status as PayoutHoldStatus).labelKey).toBe(labelKey);
    }
  });

  it('renders no pill with no hold (escrow flags off)', () => {
    expect(payoutHoldMeta('').labelKey).toBe('');
    expect(payoutHoldMeta(undefined).labelKey).toBe('');
  });
});

describe('statementStatusMeta', () => {
  it('distinguishes paid from pending, with different colors', () => {
    expect(statementStatusMeta('paid').labelKey).toBe('earnings.statementPaid');
    expect(statementStatusMeta('pending').labelKey).toBe('earnings.statementPending');
    expect(statementStatusMeta('paid').fg).not.toBe(statementStatusMeta('pending').fg);
  });
});
