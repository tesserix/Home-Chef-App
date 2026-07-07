import { useMutation, useQueryClient } from '@tanstack/react-query';

import { api } from '../lib/api';
import type { PayoutHoldStatus } from '../lib/payout-hold';

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

/** Confirm receipt of a delivered order. Refreshes the order detail + list caches
 *  (the detail query stops polling once delivered, so it must be invalidated). */
export function useConfirmOrderReceived() {
  const qc = useQueryClient();
  return useMutation<ConfirmReceiptResult, Error, string>({
    mutationFn: (orderId) =>
      api.post<ConfirmReceiptResult>(`/v1/orders/${orderId}/confirm-received`).then((r) => r.data),
    onSuccess: (_d, orderId) => {
      qc.invalidateQueries({ queryKey: ['order', orderId] });
      qc.invalidateQueries({ queryKey: ['orders'] });
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
