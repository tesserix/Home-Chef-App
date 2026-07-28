import { useMutation, useQueryClient } from '@tanstack/react-query';
import { apiClient } from '@/shared/services/api-client';
import type { PayoutHoldStatus } from '@/shared/types';

// Customer fulfilment confirmation for the escrow dual-approval (#617/#387).
// Web counterpart of apps/mobile-customer/hooks/useConfirmReceived.ts — same
// owner-scoped, idempotent endpoint.
//
// This is the customer's half of the handshake: the chef marks the order
// delivered (or hands it over on a pickup order), which parks the payout at
// `awaiting_customer_confirmation` and starts a durable reminder flow. Confirming
// here advances the hold to `release_eligible` and ends that flow early; ignoring
// it lets the flow auto-confirm after its reminders are exhausted. Either way the
// chef gets paid — this just lets the customer close it out themselves.

export interface ConfirmReceiptResult {
  payoutHoldStatus: PayoutHoldStatus;
  customerConfirmedAt?: string;
  /** Customer-safe copy from the server — show it directly. */
  message: string;
}

/** Confirm receipt of a delivered order. Refreshes the order detail + list
 *  caches (the detail query stops polling once delivered, so it must be
 *  invalidated for the confirmed state to appear). */
export function useConfirmOrderReceived() {
  const queryClient = useQueryClient();
  return useMutation<ConfirmReceiptResult, unknown, string>({
    mutationFn: (orderId) =>
      apiClient.post<ConfirmReceiptResult>(`/orders/${orderId}/confirm-received`),
    onSuccess: (_data, orderId) => {
      queryClient.invalidateQueries({ queryKey: ['order', orderId] });
      queryClient.invalidateQueries({ queryKey: ['orders'] });
    },
  });
}
