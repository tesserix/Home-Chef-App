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

describe('validatePayoutInput — bank transfer', () => {
  it('accepts a well-formed account', () => {
    expect(validatePayoutInput('bank_transfer', validBank)).toEqual([]);
  });

  it('reports every problem at once rather than one per submit', () => {
    const errors = validatePayoutInput('bank_transfer', emptyPayoutForm);
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
      const errors = validatePayoutInput('bank_transfer', form({ ...validBank, bankIFSC }));
      expect(errors.some((e) => e.field === 'bankIFSC')).toBe(true);
    }
  });

  it('accepts a lowercase IFSC, since the payload upper-cases it', () => {
    const errors = validatePayoutInput('bank_transfer', form({ ...validBank, bankIFSC: 'hdfc0001234' }));
    expect(errors).toEqual([]);
  });

  it('rejects account numbers outside 9-18 digits or containing letters', () => {
    for (const bankAccountNumber of ['12345678', '1234567890123456789', '12345678a']) {
      const errors = validatePayoutInput('bank_transfer', form({ ...validBank, bankAccountNumber }));
      expect(errors.some((e) => e.field === 'bankAccountNumber')).toBe(true);
    }
  });

  it('tolerates spaces in the account number, as printed on passbooks', () => {
    const errors = validatePayoutInput(
      'bank_transfer',
      form({ ...validBank, bankAccountNumber: '1234 5678 9012' }),
    );
    expect(errors).toEqual([]);
  });

  it('ignores the UPI field entirely', () => {
    const errors = validatePayoutInput('bank_transfer', form({ ...validBank, upiId: 'nonsense' }));
    expect(errors).toEqual([]);
  });
});

describe('validatePayoutInput — UPI', () => {
  it('accepts a well-formed VPA', () => {
    expect(validatePayoutInput('upi', form({ upiId: 'asha@okhdfcbank' }))).toEqual([]);
  });

  it('rejects a missing or malformed VPA', () => {
    for (const upiId of ['', 'asha', '@okhdfc', 'asha@', 'a@b']) {
      const errors = validatePayoutInput('upi', form({ upiId }));
      expect(errors.some((e) => e.field === 'upiId')).toBe(true);
    }
  });

  it('ignores bank fields entirely', () => {
    expect(validatePayoutInput('upi', form({ upiId: 'asha@okhdfcbank', bankIFSC: 'junk' }))).toEqual([]);
  });
});

describe('buildPayoutPayload', () => {
  it('sends only the chosen method fields', () => {
    // The backend writes every non-empty field to Secret Manager, so posting
    // a stale UPI ID alongside new bank details would leave a second, unused
    // payout destination on file.
    const payload = buildPayoutPayload('bank_transfer', form({ ...validBank, upiId: 'asha@okhdfcbank' }));
    expect(payload).toEqual({
      payoutMethod: 'bank_transfer',
      bankAccountName: 'Asha Menon',
      bankAccountNumber: '123456789012',
      bankIFSC: 'HDFC0001234',
    });
    expect(payload.upiId).toBeUndefined();
  });

  it('normalises before sending', () => {
    const payload = buildPayoutPayload(
      'bank_transfer',
      form({ bankAccountName: '  Asha Menon ', bankAccountNumber: '1234 5678 9012', bankIFSC: ' hdfc0001234 ' }),
    );
    expect(payload.bankAccountNumber).toBe('123456789012');
    expect(payload.bankIFSC).toBe('HDFC0001234');
    expect(payload.bankAccountName).toBe('Asha Menon');
  });

  it('sends only the VPA for UPI', () => {
    const payload = buildPayoutPayload('upi', form({ ...validBank, upiId: ' asha@okhdfcbank ' }));
    expect(payload).toEqual({ payoutMethod: 'upi', upiId: 'asha@okhdfcbank' });
  });
});

describe('summarisePayout', () => {
  it('never exposes the full account number', () => {
    const summary = summarisePayout('bank_transfer', validBank);
    expect(summary).toBe('Bank ••••9012');
    expect(summary).not.toContain('123456789012');
  });

  it('keeps the PSP visible but masks the handle', () => {
    // The PSP tells the chef which app receives the money; the handle is the
    // identifying part and stays hidden.
    const summary = summarisePayout('upi', form({ upiId: 'asha@okhdfcbank' }));
    expect(summary).toContain('okhdfcbank');
    expect(summary).not.toContain('asha@okhdfcbank');
  });
});
