import { useMutation, useQueryClient } from '@tanstack/react-query';

import { api } from '../lib/api';
import type { PayoutHoldStatus } from '../lib/payout-hold';
import type { Order } from '../types/customer';

// Customer fulfilment confirmation for the escrow dual-approval (#617/#387). The
// customer confirms they received a delivered order / meal-plan day, advancing its
// payout hold awaiting_customer_confirmation -> release_eligible (or -> disputed
// when they have an open issue). All endpoints are owner-scoped and idempotent, and
// inert while the escrow flags are off (the hold never reaches `awaiting`, so the
// calling surfaces never render the CTA). The server returns customer-safe copy in
// `message` — show it directly.

export interface ConfirmReceiptResult {
  payoutHoldStatus: PayoutHoldStatus;
  customerConfirmedAt?: string;
  message: string;
}

/** Confirm receipt of a delivered order.
 *
 *  Writes the server's result straight into the `['order', id]` cache rather
 *  than relying on invalidation alone (#868). A delivered order has already
 *  stopped polling (`useOrder`'s refetchInterval returns false once the status
 *  is terminal), so an invalidate is only as good as the refetch it triggers —
 *  and when that refetch doesn't run, the screen keeps rendering the
 *  pre-confirmation snapshot until the user manually pulls to refresh. The
 *  response carries exactly the two fields the UI reads, so applying it
 *  directly makes the update unconditional. The refetch below then reconciles
 *  anything else the confirmation changed server-side. */
export function useConfirmOrderReceived() {
  const qc = useQueryClient();
  return useMutation<ConfirmReceiptResult, Error, string>({
    mutationFn: (orderId) =>
      api.post<ConfirmReceiptResult>(`/v1/orders/${orderId}/confirm-received`).then((r) => r.data),
    onSuccess: (result, orderId) => {
      qc.setQueryData<{ data: Order }>(['order', orderId], (prev) =>
        prev
          ? {
              data: {
                ...prev.data,
                payoutHoldStatus: result.payoutHoldStatus,
                customerConfirmedAt: result.customerConfirmedAt,
              },
            }
          : prev,
      );
      // `refetchQueries`, not `invalidateQueries`: the detail query is no
      // longer polling, and refetch forces it regardless of staleness.
      void qc.refetchQueries({ queryKey: ['order', orderId] });
      void qc.invalidateQueries({ queryKey: ['orders'] });
    },
  });
}

/** Confirm receipt of a single meal-plan day. Invalidating `['meal-plans']`
 *  prefix-matches the per-plan detail query too. */
export function useConfirmMealPlanDayReceived() {
  const qc = useQueryClient();
  return useMutation<ConfirmReceiptResult, Error, { planId: string; dayId: string }>({
    mutationFn: ({ planId, dayId }) =>
      api
        .post<ConfirmReceiptResult>(`/v1/meal-plans/${planId}/days/${dayId}/confirm-received`)
        .then((r) => r.data),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['meal-plans'] }),
  });
}

/** Confirm receipt of a delivered group/office order (host only, #649/#456).
 *  Refreshes the group-order detail cache. */
export function useConfirmGroupOrderReceived() {
  const qc = useQueryClient();
  return useMutation<ConfirmReceiptResult, Error, string>({
    mutationFn: (groupId) =>
      api
        .post<ConfirmReceiptResult>(`/v1/group-orders/${groupId}/confirm-received`)
        .then((r) => r.data),
    onSuccess: (_d, groupId) => qc.invalidateQueries({ queryKey: ['group-order', groupId] }),
  });
}

export interface ConfirmTiffinResult {
  confirmed: number;
}

/** Bulk-confirm all of today's delivered-and-awaiting tiffin days. */
export function useConfirmTodaysTiffin() {
  const qc = useQueryClient();
  return useMutation<ConfirmTiffinResult, Error, void>({
    mutationFn: () => api.post<ConfirmTiffinResult>('/v1/tiffin/confirm-today').then((r) => r.data),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['meal-plans'] }),
  });
}
