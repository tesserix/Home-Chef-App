import { useQuery } from '@tanstack/react-query';
import { api } from '../lib/api';
import type { Order, TrackingResponse } from '../types/customer';

/** 5-second poll for live driver location (D-05), stopping once the order
 * reaches a terminal state. Pure so it can be unit-tested directly. */
export function trackingRefetchInterval(
  status: Order['status'] | undefined,
): number | false {
  if (status === 'delivered' || status === 'cancelled') return false;
  return 5000;
}

export function useOrderTracking(orderId: string, enabled: boolean = true) {
  return useQuery<{ data: TrackingResponse }>({
    queryKey: ['order-tracking', orderId],
    queryFn: () => api.get(`/v1/orders/${orderId}/track`).then((r) => r.data),
    enabled: enabled && !!orderId,
    refetchInterval: (query) =>
      trackingRefetchInterval(query.state.data?.data?.status),
    refetchIntervalInBackground: false, // T-02-04-02: stop polling when app backgrounds
    staleTime: 4000, // just under polling interval to always show fresh data
  });
}
