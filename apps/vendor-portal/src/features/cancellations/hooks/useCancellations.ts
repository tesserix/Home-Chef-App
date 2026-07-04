import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { apiClient } from '@/shared/services/api-client';

// Vendor arbitration of cancellation requests (#475). The web twin of the
// mobile-vendor hook — SAME API + SAME CANCEL_REASONS so web and mobile are
// consistent.

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
  return useQuery({
    queryKey: ['chef', 'cancel-requests', status],
    queryFn: () =>
      apiClient.get<{ data: CancellationRequest[] }>(`/chef/cancel-requests?status=${status}`),
    refetchInterval: 30_000,
    staleTime: 10_000,
  });
}

/** Confirm a cancellation with a reason → the API computes + issues the tiered refund. */
export function useConfirmCancellation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (vars: { id: string; reason: string }) =>
      apiClient.post(`/chef/cancel-requests/${vars.id}/confirm`, { reason: vars.reason }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['chef', 'cancel-requests'] });
      qc.invalidateQueries({ queryKey: ['chef', 'orders'] });
      qc.invalidateQueries({ queryKey: ['chef', 'dashboard'] });
    },
  });
}

// Shared verbatim with mobile-vendor (apps/mobile-vendor/hooks/useCancellations.ts)
// so the two vendor surfaces present identically.
export const CANCEL_REASONS = [
  { value: 'not_started', label: 'Not started yet', hint: 'Customer gets most of it back (~90%)' },
  { value: 'materials_purchased', label: 'Ingredients bought', hint: 'Materials covered — customer gets ~40% back' },
  { value: 'in_preparation', label: 'Already cooking', hint: 'Preparation started — no refund' },
  { value: 'ready', label: 'Already made', hint: 'Food is ready — no refund' },
] as const;

export type CancelReasonValue = (typeof CANCEL_REASONS)[number]['value'];
