import { useQuery } from '@tanstack/react-query';
import { api } from '../lib/api';
import type { PayoutHoldStatus } from '../lib/payout-hold';

// ---- API contract types -------------------------------------------------------
// Must match the backend shape exactly (GET /chef/earnings/breakdown).

export interface EarningsBreakdownRates {
  platformCommission: number;
  gst: number;
  tds: number;
}

export interface EarningsBreakdownTotals {
  grossRevenue: number;
  platformCommission: number;
  cgst: number;
  sgst: number;
  igst: number;
  tds: number;
  netPayout: number;
  ordersCount: number;
  // Escrow split (#617): net payout still in escrow vs already released. Both 0
  // while the escrow flags are off, so the vendor screen hides the split.
  held: number;
  released: number;
}

export interface EarningsBreakdownOrder {
  orderId: string;
  orderNumber: string;
  completedAt: string;
  itemRevenue: number;
  deliveryFee: number;
  tip: number;
  gross: number;
  platformCommission: number;
  cgst: number;
  sgst: number;
  igst: number;
  tds: number;
  netPayout: number;
  // Escrow hold lifecycle (#617). Absent (undefined) when there is no hold —
  // escrow flags off — so the per-order pill hides.
  payoutHoldStatus?: PayoutHoldStatus;
}

export interface EarningsBreakdownResponse {
  cycleStart: string;
  cycleEnd: string;
  currency: string;
  rates: EarningsBreakdownRates;
  totals: EarningsBreakdownTotals;
  orders: EarningsBreakdownOrder[];
}

export type BreakdownPeriod = 'week' | 'month' | 'cycle';

// ---- Hook --------------------------------------------------------------------

/**
 * Fetch the earnings breakdown for a given period.
 * "cycle" is used for the "All" tab (full payout cycle).
 */
export function useEarningsBreakdown(period: BreakdownPeriod) {
  return useQuery<EarningsBreakdownResponse>({
    queryKey: ['chef', 'earnings', 'breakdown', period],
    queryFn: () =>
      api
        .get<EarningsBreakdownResponse>(
          `/chef/earnings/breakdown?period=${period}`,
        )
        .then((r) => r.data),
    staleTime: 60_000,
  });
}
