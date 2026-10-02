import { useQuery } from '@tanstack/react-query';
import { api } from '../lib/api';
import { getMarket, type Market } from '../lib/market';

// The submitted kitchen's market, from its registered country. Undefined while
// loading so callers don't flash India-only UI at an AU/NZ chef.
export function useKitchenMarket(): Market | undefined {
  const { data } = useQuery({
    queryKey: ['chef', 'market'],
    queryFn: () =>
      api.get<{ country?: string }>('/chef/profile').then((r) => r.data),
    staleTime: 5 * 60_000,
  });
  return data ? getMarket(data.country) : undefined;
}
