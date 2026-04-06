import { useQuery } from '@tanstack/react-query';
import { api } from '../lib/api';
import type { TrackingResponse } from '../types/customer';

export function useOrderTracking(orderId: string, enabled: boolean = true) {
  return useQuery<{ data: TrackingResponse }>({
    queryKey: ['order-tracking', orderId],
    queryFn: () => api.get(`/v1/orders/${orderId}/track`).then((r) => r.data),
    enabled: enabled && !!orderId,
    refetchInterval: (query) => {
      const status = query.state.data?.data?.status;
      if (status === 'delivered' || status === 'cancelled') return false;
      return 5000; // D-05: 5-second polling for live driver location
    },
    refetchIntervalInBackground: false, // T-02-04-02: stop polling when app backgrounds
    staleTime: 4000, // just under polling interval to always show fresh data
  });
}
