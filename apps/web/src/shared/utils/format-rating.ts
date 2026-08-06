/**
 * One decimal, always — the API averages reviews without rounding, so a raw
 * value reaches the page as 4.857142857142857.
 *
 * An unrated chef gets an em dash rather than "0.0", which would read as a
 * terrible kitchen instead of a new one.
 */
export function formatRating(value: number | null | undefined): string {
  if (typeof value !== 'number' || !Number.isFinite(value) || value <= 0) return '—';
  return value.toFixed(1);
}
