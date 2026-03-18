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
    setCurrency(newCode);
    // Persist to server if logged in
    if (isAuthenticated) {
      apiClient.put('/customer/currency', { currencyCode: newCode }).catch(() => {});
    }
  };

  if (currencies.length === 0) return null;

  return (
    <Select value={code} onValueChange={handleChange}>
      <SelectTrigger
        variant="filled"
        size="sm"
        className="w-auto min-w-0 gap-1 border-none bg-transparent px-2 text-xs font-medium hover:bg-gray-100"
      >
        <SelectValue>
          <span className="flex items-center gap-1">
            <span>{symbol}</span>
            <span className="hidden sm:inline">{code}</span>
          </span>
        </SelectValue>
      </SelectTrigger>
      <SelectContent>
        {currencies.map((c) => (
          <SelectItem key={c.code} value={c.code}>
            {c.symbol} {c.code} — {c.name}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}
