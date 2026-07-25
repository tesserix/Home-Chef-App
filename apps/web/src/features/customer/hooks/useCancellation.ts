import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { apiClient } from '@/shared/services/api-client';

// Customer cancellation-with-arbitration (#475/#478). Requests a cancel, shows
// the vendor's decision + refund, and disputes it. This is the same API the
// vendor surfaces and the mobile app use — the legacy POST /orders/:id/cancel
// skips the policy and refund calculation entirely and must not be used.

export interface CancellationRequest {
  id: string;
  orderId: string;
  /** pending_vendor | auto_refunded | approved | disputed | admin_review | resolved */
  status: string;
  vendorReason?: string;
  refundDestination?: RefundDestination;
  refundTotalPaise: number;
  refundExecuted: boolean;
  vendorRespondBy?: string | null;
}

/**
 * Where a refund landed, as reported by the server. Not a customer choice —
 * the server derives it from the order's payment and refunds to the original
 * method. 'wallet' appears only on legacy rows, or where there is no gateway
 * payment to refund against.
 */
export type RefundDestination = 'wallet' | 'original';

/** The cancellation request for an order (null when none). Polls while pending. */
export function useCancellationRequest(orderId: string | undefined) {
  return useQuery<CancellationRequest | null>({
    queryKey: ['order', orderId, 'cancel-request'],
    queryFn: () =>
      apiClient
        .get<{ request: CancellationRequest }>(`/orders/${orderId}/cancel-request`)
        .then((r) => r.request)
        .catch(() => null), // 404 = no request yet
    enabled: Boolean(orderId),
    retry: false,
    refetchInterval: (query) =>
      query.state.data?.status === 'pending_vendor' ? 20000 : false,
  });
}

export function useRequestCancellation() {
  const queryClient = useQueryClient();
  return useMutation({
    // No refundDestination: the server decides (original payment method).
    mutationFn: (vars: { orderId: string; reason?: string }) =>
      apiClient.post(`/orders/${vars.orderId}/cancel-request`, {
        reason: vars.reason,
      }),
    onSuccess: (_data, vars) => {
      queryClient.invalidateQueries({ queryKey: ['order', vars.orderId] });
      queryClient.invalidateQueries({ queryKey: ['orders'] });
    },
  });
}

export function useDisputeCancellation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (vars: { orderId: string; reason?: string }) =>
      apiClient.post(`/orders/${vars.orderId}/cancel-request/dispute`, {
        reason: vars.reason,
      }),
    onSuccess: (_data, vars) =>
      queryClient.invalidateQueries({
        queryKey: ['order', vars.orderId, 'cancel-request'],
      }),
  });
}

/** Whether an order is at a stage the customer can still ask to cancel. */
export function orderCancellable(status: string): boolean {
  return ['pending', 'accepted', 'preparing'].includes(status);
}
