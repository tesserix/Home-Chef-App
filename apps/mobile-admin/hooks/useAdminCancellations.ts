import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api } from '../lib/api';

// Admin arbitration of cancellation requests (#480) — the mobile twin of the
// admin-portal queue. Same CANCEL_REASONS tiers as every other surface.

export interface AdminCancellationRequest {
  id: string;
  orderId: string;
  status: string;
  customerReason?: string;
  vendorReason?: string;
  disputeReason?: string;
  refundTotalPaise: number;
  vendorKeptPaise: number;
  refundExecuted: boolean;
  createdAt: string;
}

export function useAdminCancellations(status = '') {
  return useQuery<{ data: AdminCancellationRequest[] }>({
    queryKey: ['admin', 'cancel-requests', status],
    queryFn: () =>
      api
        .get<{ data: AdminCancellationRequest[] }>(
          `/admin/cancel-requests${status ? `?status=${status}` : ''}`,
        )
        .then((r) => r.data),
    refetchInterval: 30_000,
  });
}

export function useResolveCancellation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, reason, note }: { id: string; reason: string; note?: string }) =>
      api.post(`/admin/cancel-requests/${id}/resolve`, { reason, note: note ?? '' }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['admin', 'cancel-requests'] }),
  });
}

// Shared verbatim with the vendor + web surfaces.
export const CANCEL_REASONS = [
  { value: 'not_started', label: 'Not started yet', hint: 'Customer gets most of it back (~90%)' },
  { value: 'materials_purchased', label: 'Ingredients bought', hint: 'Materials covered — ~40% back' },
  { value: 'in_preparation', label: 'Already cooking', hint: 'Preparation started — no refund' },
  { value: 'ready', label: 'Already made', hint: 'Food is ready — no refund' },
] as const;
