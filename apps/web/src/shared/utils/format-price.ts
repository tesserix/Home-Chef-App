import { useCurrencyStore } from '@/app/store/currency-store';

// Symbol + decimal metadata for the currencies Stripe Connect supports.
// The user's preferred-currency store supplies these for display-conversion
// mode, but for "render this amount exactly in this currency" mode we need
// a lookup that doesn't depend on the store (since the store's current
// currency may differ from the order's).
const CURRENCY_META: Record<string, { symbol: string; decimals: number }> = {
  INR: { symbol: '₹', decimals: 2 },
  USD: { symbol: '$', decimals: 2 },
  GBP: { symbol: '£', decimals: 2 },
  EUR: { symbol: '€', decimals: 2 },
  CAD: { symbol: 'C$', decimals: 2 },
  AUD: { symbol: 'A$', decimals: 2 },
  NZD: { symbol: 'NZ$', decimals: 2 },
  SGD: { symbol: 'S$', decimals: 2 },
  HKD: { symbol: 'HK$', decimals: 2 },
  AED: { symbol: 'د.إ', decimals: 2 },
  JPY: { symbol: '¥', decimals: 0 },
  KRW: { symbol: '₩', decimals: 0 },
  CHF: { symbol: 'CHF', decimals: 2 },
  SEK: { symbol: 'kr', decimals: 2 },
  NOK: { symbol: 'kr', decimals: 2 },
  DKK: { symbol: 'kr', decimals: 2 },
  MYR: { symbol: 'RM', decimals: 2 },
  THB: { symbol: '฿', decimals: 2 },
};

/**
 * Format an amount that's already in a specific ISO-4217 currency — no
 * FX conversion applied. Use when rendering values taken directly from
 * an order, chef, or invoice whose currency is carried alongside the
 * numbers. The old INR→user-currency conversion path lives below.
 */
export function formatAmount(amount: number, currency: string): string {
  const meta = CURRENCY_META[currency.toUpperCase()] ?? { symbol: currency + ' ', decimals: 2 };
  return `${meta.symbol}${amount.toFixed(meta.decimals)}`;
}

/**
 * Legacy display-currency formatter: treats the input as INR and converts
 * to the user's preferred currency using the current exchange rate. Kept
 * for menu cards and other places where the amount's source currency
 * isn't explicit on the data.
 */
export function formatPrice(inrAmount: number): string {
  const { code, symbol, decimals, rates } = useCurrencyStore.getState();

  if (code === 'INR') {
    return `${symbol}${inrAmount.toFixed(decimals)}`;
  }

  const rate = rates[code];
  if (!rate || rate <= 0) {
    return `₹${inrAmount.toFixed(2)}`;
  }

  const converted = inrAmount * rate;
  return `${symbol}${converted.toFixed(decimals)}`;
}

/**
 * React hook variant of formatPrice. The returned formatter accepts an
 * optional `currency` override — when present, the amount is rendered as-is
 * in that currency and no FX is applied. That's the right behavior when
 * rendering an Order whose `currency` field is the source of truth.
 */
export function useFormatPrice(): (
  amount: number,
  opts?: { currency?: string }
) => string {
  const code = useCurrencyStore((s) => s.code);
  const symbol = useCurrencyStore((s) => s.symbol);
  const decimals = useCurrencyStore((s) => s.decimals);
  const rates = useCurrencyStore((s) => s.rates);

  return (amount: number, opts?: { currency?: string }): string => {
    // Explicit source currency: the amount already is in that currency,
    // just format. Never FX-convert — caller is telling us "this is the
    // real charge amount".
    if (opts?.currency) {
      return formatAmount(amount, opts.currency);
    }

    // Legacy path: assume INR, convert to user preference via FX.
    if (code === 'INR') {
      return `${symbol}${amount.toFixed(decimals)}`;
    }
    const rate = rates[code];
    if (!rate || rate <= 0) {
      return `₹${amount.toFixed(2)}`;
    }
    const converted = amount * rate;
    return `${symbol}${converted.toFixed(decimals)}`;
  };
}
