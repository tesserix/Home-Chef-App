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
  // The single most-recent active order (back-compat for callers that show one).
  order: Order | undefined;
  // All in-flight orders, newest-first — the home card stacks these so a
  // customer with more than one order going sees every one, not just the latest.
  orders: Order[];
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

  const orders = useMemo<Order[]>(() => {
    if (!data?.data?.length) return [];
    // data.data comes back newest-first from the API. Only PAID orders are live
    // to track — an unpaid/abandoned order (payment_status pending) isn't really
    // in flight (the chef never sees it), so it must not linger as an active
    // card; the customer can still complete it from their order history. (The
    // backend stale-order cron eventually cancels these.)
    return data.data.filter(
      (o) => isActiveStatus(o.status) && o.paymentStatus === 'completed',
    );
  }, [data]);

  return { order: orders[0], orders, isLoading };
}
