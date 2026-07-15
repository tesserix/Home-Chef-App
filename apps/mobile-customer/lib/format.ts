// Display formatting helpers — one source of truth so the same value never
// renders inconsistently across screens (the cart total was showing as "₹780",
// "₹780.00", and "₹780" on three different surfaces).

/**
 * Format a rupee amount for display. Whole amounts show no decimals ("₹780"),
 * fractional amounts show paise ("₹780.50"), grouped Indian-style ("₹1,23,456").
 * Pair with a `tabular-nums` Text style so figures align column-to-column.
 */
export function formatMoney(amount: number | null | undefined): string {
  const n = typeof amount === 'number' && Number.isFinite(amount) ? amount : 0;
  const hasPaise = Math.round(n) !== n;
  return `₹${n.toLocaleString('en-IN', {
    minimumFractionDigits: hasPaise ? 2 : 0,
    maximumFractionDigits: 2,
  })}`;
}
