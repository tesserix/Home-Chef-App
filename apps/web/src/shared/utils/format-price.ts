import { useCurrencyStore } from '@/app/store/currency-store';

/**
 * Converts an INR amount to the user's selected currency and formats it.
 * Uses exchange rates from the currency store.
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
 * React hook that returns a reactive price formatter.
 * Re-renders when currency or rates change.
 */
export function useFormatPrice(): (inrAmount: number) => string {
  const code = useCurrencyStore((s) => s.code);
  const symbol = useCurrencyStore((s) => s.symbol);
  const decimals = useCurrencyStore((s) => s.decimals);
  const rates = useCurrencyStore((s) => s.rates);

  return (inrAmount: number): string => {
    if (code === 'INR') {
      return `${symbol}${inrAmount.toFixed(decimals)}`;
    }

    const rate = rates[code];
    if (!rate || rate <= 0) {
      return `₹${inrAmount.toFixed(2)}`;
    }

    const converted = inrAmount * rate;
    return `${symbol}${converted.toFixed(decimals)}`;
  };
}
