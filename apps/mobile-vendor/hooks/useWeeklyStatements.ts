import { useQuery } from '@tanstack/react-query';
import { api } from '../lib/api';

// ---- API contract types -------------------------------------------------------
// Must match the backend shape exactly (GET /chef/statements/weekly).

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
}

interface WeeklyStatementsResponse {
  statements: WeeklyStatement[];
}

// ---- Hook --------------------------------------------------------------------

/** Fetch the chef's issued weekly settlement statements, newest first. */
export function useWeeklyStatements(limit = 12) {
  return useQuery<WeeklyStatement[]>({
    queryKey: ['chef', 'statements', 'weekly', limit],
    queryFn: () =>
      api
        .get<WeeklyStatementsResponse>(`/chef/statements/weekly?limit=${limit}`)
        .then((r) => r.data.statements ?? []),
    staleTime: 60_000,
  });
}
