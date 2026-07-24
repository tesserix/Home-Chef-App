import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '../lib/api';

export interface DashboardData {
  todayOrders: number;
  todayEarnings: number;
  /** Rolling 7-day order count (IST week window). Optional: a client can outrun
   * the API deploy that added it. */
  weekOrders?: number;
  /** Rolling 7-day captured revenue (IST week window). */
  weekRevenue?: number;
  /** Lifetime captured revenue across all time. The hero shows this all-time
   * total; optional so a client can outrun the API deploy that added it. */
  totalEarnings?: number;
  /** Lifetime order count (chef.TotalOrders). */
  totalOrders?: number;
  rating: number;
  totalReviews: number;
  acceptingOrders: boolean;
  /** ISO timestamp the kitchen auto-reopens, when temporarily paused. */
  pausedUntil?: string | null;
  /** True when the chef's FSSAI licence has lapsed and they're locked out (#92). */
  fssaiLocked?: boolean;
  recentOrders: RecentOrder[];
  /**
   * The kitchen queue (#695) — every live order, scoped by STATUS and ordered
   * oldest-first (most urgent at the top).
   *
   * Read THIS for in-flight work, never a client-side filter of `recentOrders`:
   * that is a 10-row recency window across all statuses, so a rush silently
   * evicted the order the chef was actually cooking. Optional because a client
   * can outrun the API deploy; callers fall back to filtering recentOrders.
   */
  activeOrders?: RecentOrder[];
}

/** Durations offered by the "Back in X min" pause control. */
export type PauseMinutes = 15 | 30 | 60;

export interface RecentOrder {
  id: string;
  customerName: string;
  total: number;
  status: string;
  createdAt: string;
  // 'pickup' orders show a pickup stepper + chef handover action on the
  // dashboard in-flight card. Legacy orders default to delivery.
  fulfillmentType?: 'delivery' | 'chef_delivery' | 'pickup';
}

export function useVendorDashboard() {
  return useQuery<DashboardData>({
    queryKey: ['chef', 'dashboard'],
    queryFn: () => api.get<DashboardData>('/chef/dashboard').then((r) => r.data),
    staleTime: 30_000,
    // This query owns EVERY in-flight card, and nothing else refreshes them:
    // pending orders poll (useVendorOrders) but the kitchen queue did not, so a
    // driver marking an order picked_up left the chef's card reading "Ready ·
    // awaiting pickup" until they manually pulled to refresh — and two devices in
    // one kitchen never converged. 15s while foregrounded; RN pauses timers in the
    // background, where push is the signal.
    refetchInterval: 15_000,
  });
}

export function useToggleAcceptingOrders() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (acceptingOrders: boolean) =>
      api.put('/chef/settings', { acceptingOrders }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['chef', 'dashboard'] }),
  });
}

/** Temporarily pause receiving for {15,30,60} min; backend auto-resumes. */
export function usePauseReceiving() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (minutes: PauseMinutes) =>
      api.post('/chef/availability/pause', { minutes }),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: ['chef', 'dashboard'] }),
  });
}

/** Reopen the kitchen immediately, cancelling any pause timer. */
export function useResumeReceiving() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () => api.post('/chef/availability/resume', {}),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: ['chef', 'dashboard'] }),
  });
}
