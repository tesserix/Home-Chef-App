import { describe, it, expect } from '@jest/globals';

import {
  buildPayoutPayload,
  emptyPayoutForm,
  summarisePayout,
  validatePayoutInput,
  type PayoutFormValues,
} from './payout';

function form(overrides: Partial<PayoutFormValues>): PayoutFormValues {
  return { ...emptyPayoutForm, ...overrides };
}

const validBank = form({
  bankAccountName: 'Asha Menon',
  bankAccountNumber: '123456789012',
  bankIFSC: 'HDFC0001234',
});

// UPI is not an accepted payout method (#767): Route settles to a bank account
// only, so the payout lib is bank-transfer only.
describe('validatePayoutInput — bank transfer', () => {
  it('accepts a well-formed account', () => {
    expect(validatePayoutInput(validBank)).toEqual([]);
  });

  it('reports every problem at once rather than one per submit', () => {
    const errors = validatePayoutInput(emptyPayoutForm);
    expect(errors.map((e) => e.field).sort()).toEqual([
      'bankAccountName',
      'bankAccountNumber',
      'bankIFSC',
    ]);
  });

  it('rejects a malformed IFSC', () => {
    // A typo here does not surface as a form error — it surfaces as a payout
    // that silently fails days after the chef already cooked and delivered.
    const cases = ['HDFC1001234', 'HDF00001234', 'HDFC000123', 'hdfc0001234 x'];
    for (const bankIFSC of cases) {
      const errors = validatePayoutInput(form({ ...validBank, bankIFSC }));
      expect(errors.some((e) => e.field === 'bankIFSC')).toBe(true);
    }
  });

  it('accepts a lowercase IFSC, since the payload upper-cases it', () => {
    const errors = validatePayoutInput(form({ ...validBank, bankIFSC: 'hdfc0001234' }));
    expect(errors).toEqual([]);
  });

  it('rejects account numbers outside 9-18 digits or containing letters', () => {
    for (const bankAccountNumber of ['12345678', '1234567890123456789', '12345678a']) {
      const errors = validatePayoutInput(form({ ...validBank, bankAccountNumber }));
      expect(errors.some((e) => e.field === 'bankAccountNumber')).toBe(true);
    }
  });

  it('tolerates spaces in the account number, as printed on passbooks', () => {
    const errors = validatePayoutInput(form({ ...validBank, bankAccountNumber: '1234 5678 9012' }));
    expect(errors).toEqual([]);
  });
});

describe('buildPayoutPayload', () => {
  it('always sends the bank_transfer method and its fields', () => {
    const payload = buildPayoutPayload(validBank);
    expect(payload).toEqual({
      payoutMethod: 'bank_transfer',
      bankAccountName: 'Asha Menon',
      bankAccountNumber: '123456789012',
      bankIFSC: 'HDFC0001234',
    });
  });

  it('normalises before sending', () => {
    const payload = buildPayoutPayload(
      form({ bankAccountName: '  Asha Menon ', bankAccountNumber: '1234 5678 9012', bankIFSC: ' hdfc0001234 ' }),
    );
    expect(payload.bankAccountNumber).toBe('123456789012');
    expect(payload.bankIFSC).toBe('HDFC0001234');
    expect(payload.bankAccountName).toBe('Asha Menon');
  });
});

describe('summarisePayout', () => {
  it('never exposes the full account number', () => {
    const summary = summarisePayout(validBank);
    expect(summary).toBe('Bank ••••9012');
    expect(summary).not.toContain('123456789012');
  });
});
