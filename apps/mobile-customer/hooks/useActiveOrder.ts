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
 * Returns the most recent active order (if any). Polls every 15 s so the
 * home active-order card reflects chef-driven status changes (e.g. preparing
 * → ready → handed over) live, instead of going stale until the app is
 * relaunched. The 30 s staleTime alone never refetched while the customer sat
 * on the home screen; this adds an explicit interval scoped to this card only.
 */
const ACTIVE_ORDER_POLL_MS = 15_000;

export function useActiveOrder(): UseActiveOrderResult {
  const { data, isLoading } = useOrders(
    { limit: 20 },
    { refetchInterval: ACTIVE_ORDER_POLL_MS },
  );

  const order = useMemo<Order | undefined>(() => {
    if (!data?.data?.length) return undefined;
    // data.data comes back newest-first from the API. The first active order in
    // that list is the most recent one — but only PAID orders are live to track.
    // An unpaid/abandoned order (payment_status pending) isn't really in flight
    // (the chef never sees it), so it must not linger as the "active" card; the
    // customer can still complete it from their order history. (The backend
    // stale-order cron eventually cancels these.)
    return data.data.find(
      (o) => isActiveStatus(o.status) && o.paymentStatus === 'completed',
    );
  }, [data]);

  return { order, isLoading };
}
