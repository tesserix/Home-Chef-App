import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api } from '../lib/api';

// Customer cancellation-with-arbitration (#475/#478). Requests a cancel, shows
// the vendor's decision + refund, and disputes it. Consumes the same API the
// vendor surfaces do.

export interface CancellationRequest {
  id: string;
  orderId: string;
  status: string; // pending_vendor | auto_refunded | approved | disputed | admin_review | resolved
  vendorReason?: string;
  refundDestination?: string;
  refundTotalPaise: number;
  refundExecuted: boolean;
  vendorRespondBy?: string | null;
}

export type RefundDestination = 'wallet' | 'original';

/** The cancellation request for an order (null when none). Polls while pending. */
export function useCancellationRequest(orderId: string | undefined) {
  return useQuery<CancellationRequest | null>({
    queryKey: ['order', orderId, 'cancel-request'],
    queryFn: () =>
      api
        .get<{ request: CancellationRequest }>(`/v1/orders/${orderId}/cancel-request`)
        .then((r) => r.data.request)
        .catch(() => null), // 404 = no request
    enabled: Boolean(orderId),
    retry: false,
    refetchInterval: (query) =>
      query.state.data?.status === 'pending_vendor' ? 20000 : false,
  });
}

export function useRequestCancellation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (vars: { orderId: string; reason?: string; refundDestination: RefundDestination }) =>
      api
        .post(`/v1/orders/${vars.orderId}/cancel-request`, {
          reason: vars.reason,
          refundDestination: vars.refundDestination,
        })
        .then((r) => r.data),
    onSuccess: (_d, vars) => {
      qc.invalidateQueries({ queryKey: ['order', vars.orderId] });
      qc.invalidateQueries({ queryKey: ['orders'] });
    },
  });
}

export function useDisputeCancellation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (vars: { orderId: string; reason?: string }) =>
      api
        .post(`/v1/orders/${vars.orderId}/cancel-request/dispute`, { reason: vars.reason })
        .then((r) => r.data),
    onSuccess: (_d, vars) =>
      qc.invalidateQueries({ queryKey: ['order', vars.orderId, 'cancel-request'] }),
  });
}

/** Whether an order is at a stage the customer can still ask to cancel. */
export function orderCancellable(status: string): boolean {
  return ['pending', 'accepted', 'preparing'].includes(status);
}
