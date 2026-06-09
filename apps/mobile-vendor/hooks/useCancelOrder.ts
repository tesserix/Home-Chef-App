import { useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '../lib/api';

// Matches the enum on the backend — see apps/api/models/order.go
// CancelReason. Surface labels live in the UI; the backend rejects
// anything outside this set with a 400.
export type CancelReason =
  | 'out_of_ingredient'
  | 'equipment_failure'
  | 'customer_request'
  | 'other';

export const CANCEL_REASON_LABEL: Record<CancelReason, string> = {
  out_of_ingredient: "I'm out of an ingredient",
  equipment_failure: 'Equipment failure',
  customer_request: 'Customer asked to cancel',
  other: 'Other',
};

// useCancelOrder triggers a whole-order cancel + refund. On success
// we invalidate the per-order detail cache + the orders queues
// (pending / in-progress / history) so the chef immediately sees the
// flipped status without a manual pull-to-refresh.
export function useCancelOrder(orderId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (reason: CancelReason) =>
      api.post(`/chef/orders/${orderId}/cancel`, { reason }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['chef', 'order-detail', orderId] });
      qc.invalidateQueries({ queryKey: ['chef', 'orders'] });
      qc.invalidateQueries({ queryKey: ['vendor', 'pending-orders'] });
    },
  });
}
