// Derives the single most-recent active order from the order list.
// An "active" order has a status in the set below — it's in flight
// and the customer may want to track it. Returns undefined when there
// is no active order (caller renders nothing).

import { useMemo } from 'react';
import { useOrders } from './useOrderHistory';
import type { Order } from '../types/customer';

export const ACTIVE_STATUSES: ReadonlyArray<Order['status']> = [
  'pending',
  'accepted',
  'preparing',
  'ready',
  'picked_up',
  'delivering',
];

function isActiveStatus(status: Order['status']): boolean {
  return (ACTIVE_STATUSES as ReadonlyArray<string>).includes(status);
}

export interface UseActiveOrderResult {
  order: Order | undefined;
  isLoading: boolean;
}

/**
 * Returns the most recent active order (if any). The underlying query
 * refreshes every 30 s (useOrders staleTime) so the card stays current
 * without extra plumbing — the user can also pull-to-refresh the home
 * list and this query piggy-backs on the same cache entry.
 */
export function useActiveOrder(): UseActiveOrderResult {
  const { data, isLoading } = useOrders({ limit: 20 });

  const order = useMemo<Order | undefined>(() => {
    if (!data?.data?.length) return undefined;
    // data.data comes back newest-first from the API. The first active
    // order in that list is therefore the most recent one.
    return data.data.find((o) => isActiveStatus(o.status));
  }, [data]);

  return { order, isLoading };
}
