import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { apiClient } from '@/shared/services/api-client';

// Admin arbitration of cancellation requests (#480) — disputes + vendor timeouts.
// Same CANCEL_REASONS tiers as the vendor surfaces, so the whole flow is consistent.

export interface AdminCancellationRequest {
  id: string;
  orderId: string;
  status: string;
  customerReason?: string;
  vendorReason?: string;
  disputeReason?: string;
  disputed: boolean;
  refundTotalPaise: number;
  vendorKeptPaise: number;
  platformKeptPaise: number;
  refundExecuted: boolean;
  createdAt: string;
}

export function useAdminCancellations(status = '') {
  return useQuery({
    queryKey: ['admin', 'cancel-requests', status],
    queryFn: () =>
      apiClient.get<{ data: AdminCancellationRequest[] }>(
        `/admin/cancel-requests${status ? `?status=${status}` : ''}`,
      ),
    refetchInterval: 30_000,
  });
}

export function useResolveCancellation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (vars: { id: string; reason: string; note?: string }) =>
      apiClient.post(`/admin/cancel-requests/${vars.id}/resolve`, {
        reason: vars.reason,
        note: vars.note,
      }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['admin', 'cancel-requests'] }),
  });
}

// Shared verbatim with the vendor + mobile surfaces.
export const CANCEL_REASONS = [
  { value: 'not_started', label: 'Not started yet', hint: 'Customer gets most of it back (~90%)' },
  { value: 'materials_purchased', label: 'Ingredients bought', hint: 'Materials covered — ~40% back' },
  { value: 'in_preparation', label: 'Already cooking', hint: 'Preparation started — no refund' },
  { value: 'ready', label: 'Already made', hint: 'Food is ready — no refund' },
] as const;
