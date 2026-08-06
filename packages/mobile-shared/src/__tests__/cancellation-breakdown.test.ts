import { describe, it, expect } from 'vitest';
import { splitRetainedAmount } from '../utils/cancellation-breakdown';

// #1048: the cancelled-order breakdown printed the whole platform-side residual
// as "Platform fee (non-refundable)" — ₹25.57 — four lines under a "Platform
// fee ₹13.53" row from the same block. The extra ₹12.04 was withheld GST,
// disclosed nowhere, on the screen that carries the "Dispute the refund
// amount" button.

describe('splitRetainedAmount', () => {
  // The order from the bug report, to the paise.
  it('separates the withheld tax from the platform fee', () => {
    const split = splitRetainedAmount({
      totalAmount: 393.06,
      refundAmount: 175.49,
      vendorKept: 192.0,
      platformFee: 13.53,
    });

    expect(split.platformFee).toBe(13.53);
    expect(split.taxWithheld).toBe(12.04);
    expect(split.platformFee + split.taxWithheld).toBeCloseTo(
      393.06 - 175.49 - 192.0,
      2,
    );
  });

  it('withholds no tax when the residual is only the fee', () => {
    const split = splitRetainedAmount({
      totalAmount: 200,
      refundAmount: 176.47,
      vendorKept: 10,
      platformFee: 13.53,
    });

    expect(split.platformFee).toBe(13.53);
    expect(split.taxWithheld).toBe(0);
  });

  // A refund that reaches into the fee itself must not print a fee larger than
  // what was actually kept, or the lines stop summing to the residual.
  it('caps the fee at the residual when part of the fee was refunded', () => {
    const split = splitRetainedAmount({
      totalAmount: 200,
      refundAmount: 190,
      vendorKept: 4,
      platformFee: 13.53,
    });

    expect(split.platformFee).toBe(6);
    expect(split.taxWithheld).toBe(0);
  });

  it('is empty when the refund returned everything', () => {
    const split = splitRetainedAmount({
      totalAmount: 200,
      refundAmount: 200,
      vendorKept: 0,
      platformFee: 13.53,
    });

    expect(split.platformFee).toBe(0);
    expect(split.taxWithheld).toBe(0);
  });

  it('treats a missing platform fee as no fee, never as NaN', () => {
    const split = splitRetainedAmount({
      totalAmount: 200,
      refundAmount: 100,
      vendorKept: 80,
      platformFee: undefined,
    });

    expect(split.platformFee).toBe(0);
    expect(split.taxWithheld).toBe(20);
  });
});
