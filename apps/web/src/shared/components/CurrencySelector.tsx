import { useEffect } from 'react';
import { useCurrencyStore } from '@/app/store/currency-store';
import { useAuth } from '@/app/providers/AuthProvider';
import { apiClient } from '@/shared/services/api-client';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/shared/components/ui/Select';

export function CurrencySelector() {
  const { isAuthenticated } = useAuth();
  const code = useCurrencyStore((s) => s.code);
  const symbol = useCurrencyStore((s) => s.symbol);
  const currencies = useCurrencyStore((s) => s.currencies);
  const setCurrency = useCurrencyStore((s) => s.setCurrency);
  const fetchCurrencies = useCurrencyStore((s) => s.fetchCurrencies);
  const fetchRates = useCurrencyStore((s) => s.fetchRates);

  useEffect(() => {
    fetchCurrencies();
    fetchRates();
  }, [fetchCurrencies, fetchRates]);

  const handleChange = (newCode: string) => {
    const previous = code;
    setCurrency(newCode);
    if (isAuthenticated) {
      apiClient.put('/customer/currency', { currencyCode: newCode }).catch((err) => {
        // Revert local state so the UI matches what the server believes.
        // Silent retry isn't right here — the next page load would override.
        setCurrency(previous);
        console.warn('Failed to persist currency preference', err);
      });
    }
  };

  // No currency list yet (still loading or fetch failed) — show the current
  // selection as a non-interactive label rather than disappearing entirely.
  // Disappearing controls confuse users and hide that the app supports
  // currency switching at all.
  if (currencies.length === 0) {
    return (
      <span
        className="inline-flex items-center gap-1 px-2 text-xs font-medium text-muted-foreground"
        aria-label={`Currency: ${code}`}
      >
        <span aria-hidden="true">{symbol}</span>
        <span className="hidden sm:inline">{code}</span>
      </span>
    );
  }

  return (
    <Select value={code} onValueChange={handleChange}>
      <SelectTrigger
        variant="filled"
        size="sm"
        aria-label="Change display currency"
        className="w-auto min-w-0 gap-1 border-none bg-transparent px-2 text-xs font-medium hover:bg-mist"
      >
        <SelectValue>
          <span className="flex items-center gap-1">
            <span aria-hidden="true">{symbol}</span>
            <span className="hidden sm:inline">{code}</span>
          </span>
        </SelectValue>
      </SelectTrigger>
      <SelectContent>
        {currencies.map((c) => (
          <SelectItem key={c.code} value={c.code}>
            {c.symbol} {c.code}
            {c.name ? ` — ${c.name}` : ''}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}
