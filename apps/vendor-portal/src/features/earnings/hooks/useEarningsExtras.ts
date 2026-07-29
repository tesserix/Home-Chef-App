import { useQuery } from '@tanstack/react-query';
import { apiClient } from '@/shared/services/api-client';

// The parts of the chef's money picture the web Earnings page was missing
// entirely: refund history, issued weekly settlement statements, and the annual
// TDS certificate. All three already existed on the API and on mobile
// (apps/mobile-vendor/hooks/useRefunds.ts + useWeeklyStatements.ts); web showed
// only the rate/tax breakdown, so a chef on the portal could see that money had
// gone missing but never what was refunded or which statement covered it.

// ---- Refunds -----------------------------------------------------------------
// Must match GET /chef/refunds exactly.

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

/** The chef's refund history, newest first (one entry per refunded order). */
export function useRefunds(limit = 50) {
  return useQuery<RefundEntry[]>({
    queryKey: ['chef', 'refunds', limit],
    queryFn: () =>
      apiClient
        .get<{ refunds: RefundEntry[] }>(`/chef/refunds?limit=${limit}`)
        .then((r) => r?.refunds ?? []),
    staleTime: 60_000,
  });
}

// ---- Weekly settlement statements --------------------------------------------
// Must match GET /chef/statements/weekly exactly.

export interface WeeklyStatement {
  id: string;
  weekStart: string; // YYYY-MM-DD (Monday, IST)
  weekEnd: string; // YYYY-MM-DD (following Monday, exclusive)
  currency: string;
  ordersCount: number;
  grossRevenue: number;
  platformCommission: number;
  cgst: number;
  sgst: number;
  igst: number;
  tds: number;
  netPayout: number;
  /** `paid` once the platform has settled the statement. */
  status: 'pending' | 'paid';
  paidAt?: string;
  payoutRef?: string;
}

/** The chef's issued weekly settlement statements, newest first. */
export function useWeeklyStatements(limit = 12) {
  return useQuery<WeeklyStatement[]>({
    queryKey: ['chef', 'statements', 'weekly', limit],
    queryFn: () =>
      apiClient
        .get<{ statements: WeeklyStatement[] }>(`/chef/statements/weekly?limit=${limit}`)
        .then((r) => r?.statements ?? []),
    staleTime: 60_000,
  });
}

/**
 * The Indian financial year label for today, e.g. "FY 2026-27".
 *
 * The FY starts in April, so anything in Jan-Mar belongs to the year that
 * started the PREVIOUS April — getting this wrong would label a March
 * certificate with next year's FY.
 */
export function currentFyLabel(d: Date = new Date()): string {
  const y = d.getFullYear();
  const startYear = d.getMonth() >= 3 ? y : y - 1;
  return `FY ${startYear}-${String((startYear + 1) % 100).padStart(2, '0')}`;
}

/**
 * Download a PDF the API serves behind auth.
 *
 * A plain <a href> cannot be used: these endpoints need the session credentials
 * the api-client attaches, and a bare navigation would land on a 401 page
 * instead of a file. Fetch as a blob, then hand the browser an object URL.
 */
export async function downloadPdf(path: string, filename: string): Promise<void> {
  const blob = await apiClient.getBlob(path);
  const url = URL.createObjectURL(blob);
  try {
    const a = document.createElement('a');
    a.href = url;
    a.download = filename;
    document.body.appendChild(a);
    a.click();
    a.remove();
  } finally {
    // Revoking immediately would race the download in some browsers; a tick is
    // enough for the click to have been dispatched.
    setTimeout(() => URL.revokeObjectURL(url), 0);
  }
}
