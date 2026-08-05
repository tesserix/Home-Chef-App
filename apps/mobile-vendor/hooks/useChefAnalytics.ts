import { useQuery } from '@tanstack/react-query';
import { api } from '../lib/api';

// Chef analytics hooks (#49) — sales summary + trends, subscription health, and
// the tomorrow's-demand forecast. One module so the analytics screen stays thin.

export type AnalyticsPeriod = '7d' | '30d' | '90d';

export interface AnalyticsSummary {
  orders: number;
  /** Orders that settled in the window — the ones `revenue` came from. */
  settledOrders: number;
  /** Net payout, on the same settlement basis as the Earnings screen (#1030). */
  revenue: number;
  aov: number;
  repeatRate: number; // %
  prevRevenue: number;
}

/** The calendar week and month the platform pays out on, so this screen and the
 *  Earnings screen quote the same figure rather than two rolling windows. */
export interface SettledEarnings {
  currency: string;
  week: number;
  month: number;
}
export interface Trend {
  labels: string[];
  data: number[];
}
export interface PopularItem {
  name: string;
  orders: number;
  percentage?: number;
}
export interface AnalyticsResponse {
  summary: AnalyticsSummary;
  earnings: SettledEarnings;
  orderTrends: Trend;
  revenueTrends: Trend;
  popularItems: PopularItem[];
}

export function useChefAnalytics(period: AnalyticsPeriod) {
  return useQuery<AnalyticsResponse>({
    queryKey: ['chef', 'analytics', period],
    queryFn: () => api.get<AnalyticsResponse>(`/chef/analytics?period=${period}`).then((r) => r.data),
    staleTime: 60_000,
  });
}

export interface SubscriptionMetrics {
  activePlans: number;
  subscribers: number;
  churnRate: number; // %
  adherenceRate: number; // %
  deliveredDays: number;
  skippedDays: number;
}

export function useSubscriptionMetrics() {
  return useQuery<SubscriptionMetrics>({
    queryKey: ['chef', 'analytics', 'subscriptions'],
    queryFn: () =>
      api.get<SubscriptionMetrics>('/chef/analytics/subscriptions').then((r) => r.data),
    staleTime: 60_000,
  });
}

export interface DemandForecast {
  date: string;
  subscriptionMeals: number;
  subscriptionLunch: number;
  subscriptionDinner: number;
  alaCarteForecast: number;
  totalExpected: number;
  likelyDishes: { name: string; expected: number }[];
  basis: string;
}

export function useDemandForecast() {
  return useQuery<DemandForecast>({
    queryKey: ['chef', 'analytics', 'forecast'],
    queryFn: () => api.get<DemandForecast>('/chef/analytics/forecast').then((r) => r.data),
    staleTime: 60_000,
  });
}

// Profit & loss for the same window as the cards above (#pl). Net earnings come
// from the server's payout calculator, not from revenue minus a guess, so this
// card and the Earnings screen never disagree about money.
export interface ProfitLoss {
  period: AnalyticsPeriod;
  from: string;
  to: string;
  orders: number;
  grossSales: number;
  deductions: { platformCommission: number; gst: number; tds: number };
  netEarnings: number;
  expenses: number;
  expensesByCategory: { category: string; amount: number }[];
  netProfit: number;
  marginPercent: number;
}

export function useProfitLoss(period: AnalyticsPeriod) {
  return useQuery<ProfitLoss>({
    queryKey: ['chef', 'analytics', 'pl', period],
    queryFn: () => api.get<ProfitLoss>(`/chef/analytics/pl?period=${period}`).then((r) => r.data),
    staleTime: 60_000,
  });
}
