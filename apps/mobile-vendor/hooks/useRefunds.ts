import { useQuery } from '@tanstack/react-query';
import { api } from '../lib/api';

// ---- API contract types -------------------------------------------------------
// Must match the backend shape exactly (GET /chef/refunds).

export interface RefundItem {
  name: string;
  amount: number;
  reason?: string;
}

export interface RefundEntry {
  orderId: string;
  orderNumber: string;
  amount: number;
  reason?: string;
  initiatedBy?: string;
  refundedAt: string;
  items?: RefundItem[];
}

interface RefundsResponse {
  refunds: RefundEntry[];
}

// ---- Hook --------------------------------------------------------------------

/** Fetch the chef's refund history, newest first (one entry per refunded order). */
export function useRefunds(limit = 50) {
  return useQuery<RefundEntry[]>({
    queryKey: ['chef', 'refunds', limit],
    queryFn: () =>
      api
        .get<RefundsResponse>(`/chef/refunds?limit=${limit}`)
        .then((r) => r.data.refunds ?? []),
    staleTime: 60_000,
  });
}
