/** Formats major units in the transaction currency; never converts amounts. */
export function formatMoney(amount: number | null | undefined, currency = 'INR'): string {
  const n = typeof amount === 'number' && Number.isFinite(amount) ? amount : 0;
  const code = currency.trim().toUpperCase() || 'INR';
  const locale = code === 'INR' ? 'en-IN' : code === 'NZD' ? 'en-NZ' : 'en-AU';
  const prefix = code === 'INR' ? '₹' : `${code} `;
  return `${prefix}${n.toLocaleString(locale, {
    minimumFractionDigits: Math.round(n) !== n ? 2 : 0,
    maximumFractionDigits: 2,
  })}`;
}
