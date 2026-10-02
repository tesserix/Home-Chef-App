import { currencySymbol } from './format';

// Buckets are sized to what a single dish costs in each market, not converted.
const BUCKETS: Record<string, number[]> = {
  INR: [100, 250, 500],
  AUD: [15, 25, 40],
  NZD: [15, 25, 40],
};

export function priceFilterOptions(
  currency: string,
): { label: string; value: number | undefined }[] {
  const symbol = currencySymbol(currency);
  return [
    { label: 'Any price', value: undefined },
    ...(BUCKETS[currency] ?? BUCKETS.INR!).map((v) => ({ label: `< ${symbol}${v}`, value: v })),
  ];
}
