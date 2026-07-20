import { useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '../lib/api';

// Matches models.DeliveryFailureReason on the backend (apps/api/models/
// delivery_failure.go). Only the reasons that make sense for a chef delivering
// their OWN order are surfaced — `driver_no_show` is a 3PL-courier reason and is
// deliberately omitted here. The backend accepts any valid reason.
export type DeliveryFailureReason =
  | 'customer_unavailable'
  | 'customer_refused'
  | 'wrong_address'
  | 'food_damaged'
  | 'other';

// Stable display order for the reason picker (iOS action sheet + Android chain).
export const DELIVERY_FAILURE_REASONS: DeliveryFailureReason[] = [
  'customer_unavailable',
  'customer_refused',
  'wrong_address',
  'food_damaged',
  'other',
];

export const DELIVERY_FAILURE_REASON_LABEL: Record<DeliveryFailureReason, string> = {
  customer_unavailable: "Customer wasn't available",
  customer_refused: 'Customer refused the delivery',
  wrong_address: 'Address was wrong / unreachable',
  food_damaged: 'Food was damaged on the way',
  other: 'Other reason',
};

// useReportDeliveryFailure lets a self-delivery chef report they could not
// deliver an order (#393). The backend opens a pending delivery-failure review
// and freezes the payout hold to `disputed` WITHOUT moving money — an admin
// confirms fault and the refund/release outcome follows. We invalidate the
// order caches so the row flips out of the active list immediately.
export function useReportDeliveryFailure(orderId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (failureReason: DeliveryFailureReason) =>
      api.post(`/chef/orders/${orderId}/delivery-failed`, { failureReason }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['chef', 'orders'] });
      qc.invalidateQueries({ queryKey: ['chef', 'orders', 'detail', orderId] });
      qc.invalidateQueries({ queryKey: ['chef', 'dashboard'] });
    },
  });
}
