import { describe, it, expect } from '@jest/globals';

import {
  buildPayoutPayload,
  emptyPayoutForm,
  summarisePayout,
  validatePayoutInput,
  payoutStatusChip,
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

// #1082 — the chip is the only thing telling a chef whether their money will
// arrive. It reads the server's plain-language verdict; it must not re-derive
// one from a gateway status string it does not own.
describe('payoutStatusChip', () => {
  it('shows direct settlement once the server says verified', () => {
    const chip = payoutStatusChip({ state: 'verified', message: 'Verified — payouts active' });
    expect(chip.tone).toBe('success');
    expect(chip.label).toBe('Verified — payouts active');
  });

  it('shows the server message while verification is pending', () => {
    const chip = payoutStatusChip({ state: 'pending', message: 'Pending verification' });
    expect(chip.tone).toBe('pending');
    expect(chip.label).toBe('Pending verification');
  });

  it('does not describe a failed registration as in progress', () => {
    const chip = payoutStatusChip({
      state: 'failed',
      message: "Couldn't verify — please check your details",
    });
    expect(chip.tone).toBe('error');
    expect(chip.label).toBe("Couldn't verify — please check your details");
  });

  // The wording is the server's, so it can be corrected without shipping an
  // app release. A locally-invented label would drift from it.
  it('renders the server wording rather than one of its own', () => {
    const chip = payoutStatusChip({ state: 'pending', message: 'Almost there' });
    expect(chip.label).toBe('Almost there');
  });

  // 'none' means no details on file yet — telling the chef to add them beats
  // an "Activation pending" that suggests someone else is working on it.
  it('passes on the "add your details" prompt', () => {
    const chip = payoutStatusChip({ state: 'none', message: 'Add your bank details' });
    expect(chip.tone).toBe('pending');
    expect(chip.label).toBe('Add your bank details');
  });

  it('is pending when the server sends no verdict at all', () => {
    const chip = payoutStatusChip(undefined);
    expect(chip.tone).toBe('pending');
    expect(chip.label).toBe('Activation pending');
  });
});
