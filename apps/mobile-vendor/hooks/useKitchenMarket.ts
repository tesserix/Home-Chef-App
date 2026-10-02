import { useQuery } from '@tanstack/react-query';
import { api } from '../lib/api';
import { formatMoney } from '../lib/format';
import { getMarket, type Market } from '../lib/market';

function useKitchenMarketQuery() {
  return useQuery({
    queryKey: ['chef', 'market'],
    queryFn: () =>
      api.get<{ country?: string }>('/chef/profile').then((r) => r.data),
    staleTime: 5 * 60_000,
  });
}

// The submitted kitchen's market, from its registered country. Undefined while
// loading so callers don't flash India-only UI at an AU/NZ chef.
export function useKitchenMarket(): Market | undefined {
  const { data } = useKitchenMarketQuery();
  return data ? getMarket(data.country) : undefined;
}

// For screens whose checklist depends on the market and must not read a failed
// lookup as "nothing required".
export function useKitchenMarketState(): {
  market: Market | undefined;
  isError: boolean;
  refetch: () => void;
} {
  const { data, isError, refetch } = useKitchenMarketQuery();
  return { market: data ? getMarket(data.country) : undefined, isError, refetch: () => void refetch() };
}

// Formats an amount in the kitchen's own currency.
export function useKitchenMoney(): (amount: number | null | undefined) => string {
  const currency = useKitchenMarket()?.currency;
  return (amount) => formatMoney(amount, currency);
}
