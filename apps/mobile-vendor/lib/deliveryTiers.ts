// The chef's published distance→fee ladder. This is the price the customer pays
// at checkout and it is never re-priced afterwards, so every rule the server
// enforces (models/delivery_fee_tier.go) is mirrored here to catch a bad band on
// the keypad rather than on save.

import { currencySymbol } from './format';

export interface DeliveryTier {
  upToKm: number;
  fee: number;
}

/** One band as the chef is typing it. Both fields are free text until saved. */
export interface TierRow {
  km: string;
  fee: string;
}

/** The platform's ceiling on what a chef may charge, served with the profile. */
export interface DeliveryFeeCap {
  baseFee: number;
  perKm: number;
  maxFee: number;
  maxBands: number;
  maxKm: number;
}

// Mirrors the server's shipped defaults, used until the profile arrives.
export const DEFAULT_DELIVERY_FEE_CAP: DeliveryFeeCap = {
  baseFee: 30,
  perKm: 12,
  maxFee: 300,
  maxBands: 6,
  maxKm: 50,
};

/** The most a chef may charge for a band reaching `km`. */
export function bandCeiling(cap: DeliveryFeeCap, km: number): number {
  return Math.min(cap.baseFee + cap.perKm * km, cap.maxFee);
}

export function rowsFromTiers(tiers: DeliveryTier[] | undefined): TierRow[] {
  return (tiers ?? []).map((t) => ({ km: String(t.upToKm), fee: String(t.fee) }));
}

/** Parsed ladder, ignoring rows the chef has not finished typing. */
export function tiersFromRows(rows: TierRow[]): DeliveryTier[] {
  const out: DeliveryTier[] = [];
  for (const row of rows) {
    const km = parseFloat(row.km);
    const fee = parseFloat(row.fee);
    if (!Number.isFinite(km) || !Number.isFinite(fee)) continue;
    out.push({ upToKm: km, fee });
  }
  return out;
}

function isBlank(row: TierRow): boolean {
  return row.km.trim() === '' && row.fee.trim() === '';
}

/**
 * The first thing wrong with the ladder, phrased for the chef — or null when it
 * is publishable. An entirely blank ladder is valid: it just means the chef
 * prices delivery the old way.
 */
export function validateTierRows(
  rows: TierRow[],
  cap: DeliveryFeeCap,
  currency?: string | null,
): string | null {
  const filled = rows.filter((r) => !isBlank(r));
  if (filled.length === 0) return null;
  if (filled.length > cap.maxBands) {
    return `You can set up to ${cap.maxBands} distance bands.`;
  }

  let prevKm = 0;
  let prevFee = -1;
  for (const row of filled) {
    const km = parseFloat(row.km);
    const fee = parseFloat(row.fee);
    if (!Number.isFinite(km) || !Number.isFinite(fee)) {
      return 'Every band needs both a distance and a fee.';
    }
    if (km <= 0) return 'A band distance must be more than 0 km.';
    if (km > cap.maxKm) return `A band can't reach beyond ${cap.maxKm} km.`;
    if (fee < 0) return "A band fee can't be negative.";
    if (km <= prevKm) return 'Each band must reach farther than the one above it.';
    if (fee < prevFee) return "A farther band can't cost less than a nearer one.";
    const ceiling = bandCeiling(cap, km);
    if (fee > ceiling) {
      return `The most you can charge for ${formatKm(km)} km is ${currencySymbol(currency)}${formatMoneyish(ceiling)}.`;
    }
    prevKm = km;
    prevFee = fee;
  }
  return null;
}

function formatKm(km: number): string {
  return Number.isInteger(km) ? String(km) : km.toFixed(1);
}

function formatMoneyish(n: number): string {
  return Number.isInteger(n) ? String(n) : n.toFixed(2);
}
