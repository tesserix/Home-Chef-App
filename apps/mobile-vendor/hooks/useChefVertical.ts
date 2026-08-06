import { useQuery } from '@tanstack/react-query';
import { api } from '../lib/api';

// Whether this kitchen sells bakes at all (#1065) — as a bakery, or as a meals
// kitchen that also bakes. Only then does the cake configurator appear on a menu
// item; the backend rejects a spec from anyone else, so the editor stays hidden
// rather than failing on save.
export function useOffersBakery(): boolean {
  const { data } = useQuery({
    queryKey: ['chef', 'vertical'],
    queryFn: () =>
      api
        .get<{ vertical?: string; sellsBakery?: boolean }>('/chef/profile')
        .then((r) => r.data),
    staleTime: 5 * 60_000,
  });
  return data?.vertical === 'bakery' || data?.sellsBakery === true;
}
