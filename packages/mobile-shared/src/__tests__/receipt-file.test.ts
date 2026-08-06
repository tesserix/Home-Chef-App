import { describe, it, expect } from 'vitest';

import { receiptFileName } from '../utils/receipt-file';

describe('receiptFileName', () => {
  it('names a tax invoice after the order it belongs to', () => {
    expect(receiptFileName('HC-2026-0042', true)).toBe('Fe3dr-tax-invoice-HC-2026-0042.pdf');
  });

  it('names a payment receipt separately, so the two are never confused', () => {
    expect(receiptFileName('HC-2026-0042', false)).toBe('Fe3dr-receipt-HC-2026-0042.pdf');
  });

  it('strips characters a file system would choke on', () => {
    expect(receiptFileName('HC/2026 #42', false)).toBe('Fe3dr-receipt-HC-2026-42.pdf');
  });

  it('falls back to a generic name when the order number is missing', () => {
    expect(receiptFileName('', false)).toBe('Fe3dr-receipt.pdf');
  });
});
