// Each market renders money the way its own shoppers read it; amounts are never converted.
const CURRENCIES: Record<string, { symbol: string; locale: string }> = {
  INR: { symbol: '₹', locale: 'en-IN' },
  AUD: { symbol: '$', locale: 'en-AU' },
  NZD: { symbol: '$', locale: 'en-NZ' },
};

function currencyCode(currency: string | null | undefined): string {
  return (currency ?? '').trim().toUpperCase() || 'INR';
}

/** The symbol to show beside an amount input, e.g. "₹" or "$". */
export function currencySymbol(currency?: string | null): string {
  const code = currencyCode(currency);
  return CURRENCIES[code]?.symbol ?? code;
}

/** Formats major units in the transaction currency: "₹1,23,456", "$780.50". */
export function formatMoney(amount: number | null | undefined, currency?: string | null): string {
  const n = typeof amount === 'number' && Number.isFinite(amount) ? amount : 0;
  const code = currencyCode(currency);
  const known = CURRENCIES[code];
  return `${known ? known.symbol : `${code} `}${n.toLocaleString(known?.locale ?? 'en-US', {
    minimumFractionDigits: Math.round(n) !== n ? 2 : 0,
    maximumFractionDigits: 2,
  })}`;
}
