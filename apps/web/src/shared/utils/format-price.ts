import { useCurrencyStore } from '@/app/store/currency-store';

// Symbol + decimal metadata for the currencies Stripe Connect supports.
// Used as a fallback when Intl can't render the currency (older browsers, exotic codes)
// and as the source of decimal counts for the legacy FX path that already has a
// symbol from the store.
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

// Em-dash for NaN/Infinity. Never render "NaN" or "Infinity" to a user.
const UNFORMATTABLE = '—';

/**
 * Locale-aware number formatter for the legacy store-driven path. Returns the
 * digits + separators only — caller prefixes the store's symbol so user-chosen
 * symbol overrides (e.g. "Rs." instead of "₹") survive.
 */
function formatNumberWithGrouping(amount: number, decimals: number): string {
  return new Intl.NumberFormat(undefined, {
    minimumFractionDigits: decimals,
    maximumFractionDigits: decimals,
  }).format(amount);
}

/**
 * Format an amount that's already in a specific ISO-4217 currency — no
 * FX conversion applied. Use when rendering values taken directly from
 * an order, chef, or invoice whose currency is carried alongside the
 * numbers. Delegates to Intl.NumberFormat so:
 *   - thousands/decimal separators follow the browser locale
 *   - currency symbol position is locale-correct
 *   - JPY/KRW zero-decimal currencies render integer
 *   - negative values format properly ("-$1,234.56" not "$-1234.56")
 */
export function formatAmount(amount: number, currency: string): string {
  if (!Number.isFinite(amount)) return UNFORMATTABLE;

  const code = currency.toUpperCase();
  try {
    return new Intl.NumberFormat(undefined, {
      style: 'currency',
      currency: code,
      currencyDisplay: 'narrowSymbol',
    }).format(amount);
  } catch {
    // Unknown currency code: fall back to local symbol table, then ISO prefix.
    const meta = CURRENCY_META[code];
    if (meta) {
      return `${meta.symbol}${formatNumberWithGrouping(amount, meta.decimals)}`;
    }
    return `${code} ${formatNumberWithGrouping(amount, 2)}`;
  }
}

/**
 * Legacy display-currency formatter: treats the input as INR and converts
 * to the user's preferred currency using the current exchange rate. Kept
 * for menu cards and other places where the amount's source currency
 * isn't explicit on the data.
 *
 * Symbol comes from the store (so a user-customized glyph wins over the
 * locale default); grouping/decimals go through Intl so 1,234,567.89
 * renders correctly even though we prefix the symbol ourselves.
 */
export function formatPrice(inrAmount: number): string {
  if (!Number.isFinite(inrAmount)) return UNFORMATTABLE;

  const { code, symbol, decimals, rates } = useCurrencyStore.getState();

  if (code === 'INR') {
    return `${symbol}${formatNumberWithGrouping(inrAmount, decimals)}`;
  }

  const rate = rates[code];
  if (!rate || rate <= 0) {
    // FX missing — render the source amount as INR so the user still sees
    // a real number rather than a broken "0.00".
    return `₹${formatNumberWithGrouping(inrAmount, 2)}`;
  }

  const converted = inrAmount * rate;
  return `${symbol}${formatNumberWithGrouping(converted, decimals)}`;
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
    if (!Number.isFinite(amount)) return UNFORMATTABLE;

    if (opts?.currency) {
      return formatAmount(amount, opts.currency);
    }

    if (code === 'INR') {
      return `${symbol}${formatNumberWithGrouping(amount, decimals)}`;
    }
    const rate = rates[code];
    if (!rate || rate <= 0) {
      return `₹${formatNumberWithGrouping(amount, 2)}`;
    }
    const converted = amount * rate;
    return `${symbol}${formatNumberWithGrouping(converted, decimals)}`;
  };
}
