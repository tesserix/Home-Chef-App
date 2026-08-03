import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api } from '../lib/api';

// Vendor arbitration of cancellation requests (#475). Consumes the same API the
// web vendor-portal does (GET /chef/cancel-requests, POST /chef/cancel-requests/
// :id/confirm) and shares the CANCEL_REASONS labels so web + mobile read identically.

export interface CancellationRequest {
  id: string;
  orderId: string;
  status: string;
  customerReason?: string;
  vendorReason?: string;
  refundTotalPaise: number;
  vendorKeptPaise: number;
  platformKeptPaise: number;
  refundExecuted: boolean;
  createdAt: string;
  vendorRespondBy?: string | null;
}

/** The chef's cancellation requests for a status (default: awaiting their confirm). */
export function useCancellationRequests(status: string = 'pending_vendor') {
  return useQuery<{ data: CancellationRequest[] }>({
    queryKey: ['chef', 'cancel-requests', status],
    queryFn: () =>
      api
        .get<{ data: CancellationRequest[] }>(`/chef/cancel-requests?status=${status}`)
        .then((r) => r.data),
    refetchInterval: 30_000, // requests are time-boxed — keep the queue fresh
    staleTime: 10_000,
  });
}

/** Confirm a cancellation with a reason → the API computes + issues the tiered refund. */
export function useConfirmCancellation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (vars: { id: string; reason: string }) =>
      api
        .post(`/chef/cancel-requests/${vars.id}/confirm`, { reason: vars.reason })
        .then((r) => r.data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['chef', 'cancel-requests'] });
      qc.invalidateQueries({ queryKey: ['chef', 'orders'] });
      qc.invalidateQueries({ queryKey: ['chef', 'dashboard'] });
    },
  });
}

// The reason tiers the vendor chooses — SHARED verbatim with the web vendor-portal
// so the two surfaces are consistent. The % is the default; the API resolves the
// live percentage from PlatformSettings.
//
// The percentages apply to the FOOD only. Delivery and its share of tax are still
// refunded whenever no driver has been dispatched (`ComputeCancellationRefund`:
// `deliveryRefund = deliveryPaise` unless `orderDispatched`), so the bottom two
// tiers are not "no refund" — they are no refund ON THE FOOD.
//
// That wording was wrong in a way that mattered: a chef picking "Already cooking"
// on a live order was told the customer got nothing, and the customer's screen
// then read "₹41.07 refunded" — the delivery fee and its tax. Neither party had
// been told the same thing about the same cancellation (#946).
export const CANCEL_REASONS = [
  { value: 'not_started', label: 'Not started yet', hint: 'Customer gets most of the food cost back (~90%)' },
  { value: 'materials_purchased', label: 'Ingredients bought', hint: 'Materials covered — customer gets ~40% of the food cost back' },
  { value: 'in_preparation', label: 'Already cooking', hint: 'Preparation started — you keep the food cost, the delivery fee is still refunded' },
  { value: 'ready', label: 'Already made', hint: 'Food is ready — you keep the food cost, the delivery fee is still refunded' },
] as const;

export type CancelReasonValue = (typeof CANCEL_REASONS)[number]['value'];
