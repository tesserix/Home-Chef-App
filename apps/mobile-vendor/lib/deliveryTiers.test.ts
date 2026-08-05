import { describe, it, expect } from '@jest/globals';

import {
  DEFAULT_DELIVERY_FEE_CAP,
  bandCeiling,
  rowsFromTiers,
  tiersFromRows,
  validateTierRows,
} from './deliveryTiers';

// Why: the ladder a chef publishes here is the fee the customer pays and can
// never be raised afterwards, so a band that the server would reject has to be
// caught on the keypad — not after the chef saves and walks away.

const cap = { ...DEFAULT_DELIVERY_FEE_CAP, baseFee: 30, perKm: 12, maxFee: 300 };

describe('bandCeiling', () => {
  it('is the platform maximum for that distance', () => {
    expect(bandCeiling(cap, 5)).toBe(90);
    expect(bandCeiling(cap, 10)).toBe(150);
  });

  it('never exceeds the absolute maximum', () => {
    expect(bandCeiling(cap, 40)).toBe(300);
  });
});

describe('rowsFromTiers / tiersFromRows', () => {
  it('round-trips a published ladder', () => {
    const rows = rowsFromTiers([
      { upToKm: 5, fee: 75 },
      { upToKm: 10, fee: 150 },
    ]);
    expect(rows).toEqual([
      { km: '5', fee: '75' },
      { km: '10', fee: '150' },
    ]);
    expect(tiersFromRows(rows)).toEqual([
      { upToKm: 5, fee: 75 },
      { upToKm: 10, fee: 150 },
    ]);
  });

  it('drops a half-typed band rather than saving a zero-km one', () => {
    expect(tiersFromRows([{ km: '5', fee: '75' }, { km: '', fee: '' }])).toEqual([
      { upToKm: 5, fee: 75 },
    ]);
  });

  it('reads a free band as free, not as a blank row', () => {
    expect(tiersFromRows([{ km: '3', fee: '0' }])).toEqual([{ upToKm: 3, fee: 0 }]);
  });
});

describe('validateTierRows', () => {
  it('accepts an empty ladder — the chef simply hasn’t published one', () => {
    expect(validateTierRows([], cap)).toBeNull();
    expect(validateTierRows([{ km: '', fee: '' }], cap)).toBeNull();
  });

  it('accepts a rising ladder inside the platform ceiling', () => {
    expect(
      validateTierRows([{ km: '5', fee: '75' }, { km: '10', fee: '150' }], cap),
    ).toBeNull();
  });

  it('rejects a band above the platform ceiling for its distance', () => {
    expect(validateTierRows([{ km: '5', fee: '91' }], cap)).toMatch(/₹90/);
  });

  it('rejects a distance that does not increase', () => {
    expect(
      validateTierRows([{ km: '5', fee: '50' }, { km: '5', fee: '60' }], cap),
    ).toBeTruthy();
  });

  it('rejects a farther band that costs less than a nearer one', () => {
    expect(
      validateTierRows([{ km: '5', fee: '75' }, { km: '10', fee: '60' }], cap),
    ).toBeTruthy();
  });

  it('rejects a band beyond the platform distance limit', () => {
    expect(validateTierRows([{ km: String(cap.maxKm + 1), fee: '10' }], cap)).toBeTruthy();
  });

  it('rejects a half-typed band — a distance with no fee is not a price', () => {
    expect(validateTierRows([{ km: '5', fee: '' }], cap)).toBeTruthy();
  });
});
