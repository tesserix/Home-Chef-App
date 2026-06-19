import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api } from '../lib/api';

// Chef subscription + premium tier hooks (#44). The premium prices and perks all
// come from the API (admin-configurable) — never hardcode them in the UI.

export type SubscriptionTier = 'standard' | 'premium';
export type SubscriptionStatusValue =
  | 'trial'
  | 'active'
  | 'past_due'
  | 'suspended'
  | 'cancelled'
  | 'expired';

export interface Subscription {
  id: string;
  tier: SubscriptionTier;
  status: SubscriptionStatusValue;
  billingInterval: 'monthly' | 'quarterly' | 'yearly';
  planAmount: number;
  currency: string;
  trialEndsAt?: string;
  currentPeriodEnd?: string;
}

export interface SubscriptionResponse {
  subscription: Subscription | null;
  status: string;
  earnings?: {
    total: number;
    threshold: number;
    thresholdMet: boolean;
    currency: string;
  };
}

export function useChefSubscription() {
  return useQuery<SubscriptionResponse>({
    queryKey: ['chef', 'subscription'],
    queryFn: () => api.get<SubscriptionResponse>('/chef/subscription').then((r) => r.data),
    staleTime: 60_000,
  });
}

export interface PlanOption {
  interval: 'monthly' | 'quarterly' | 'yearly';
  amount: number;
  currency: string;
  savingsPercent?: number;
}

export interface PlansResponse {
  plans: PlanOption[];
  premiumPlans: PlanOption[];
  premiumPerks: string[];
  premiumCommissionRate: number;
  standardCommissionRate: number;
  currency: string;
  trialDays: number;
}

export function useSubscriptionPlans() {
  return useQuery<PlansResponse>({
    queryKey: ['chef', 'subscription', 'plans'],
    queryFn: () => api.get<PlansResponse>('/chef/subscription/plans').then((r) => r.data),
    staleTime: 5 * 60_000,
  });
}

/** Upgrade to / downgrade from the premium tier. */
export function useChangeTier() {
  const queryClient = useQueryClient();
  return useMutation<unknown, Error, SubscriptionTier>({
    mutationFn: (tier: SubscriptionTier) =>
      api.put('/chef/subscription/tier', { tier }).then((r) => r.data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['chef', 'subscription'] });
      void queryClient.invalidateQueries({ queryKey: ['chef', 'dashboard'] });
    },
  });
}

// ── Advanced analytics (premium-gated, #44) ──────────────────────────────────

export interface AdvancedAnalytics {
  windowDays: number;
  customers: number;
  repeatCustomers: number;
  repeatRate: number;
  avgRevenuePerCustomer: number;
  avgOrdersPerCustomer: number;
  bestDay: { day: string; orders: number };
  currency: string;
}

/** Returns advanced analytics, or `locked` when the chef isn't premium (402). */
export function useAdvancedAnalytics() {
  return useQuery<{ data: AdvancedAnalytics | null; locked: boolean }>({
    queryKey: ['chef', 'analytics', 'advanced'],
    queryFn: async () => {
      try {
        const r = await api.get<AdvancedAnalytics>('/chef/analytics/advanced');
        return { data: r.data, locked: false };
      } catch (err: unknown) {
        const status = (err as { response?: { status?: number } })?.response?.status;
        if (status === 402) return { data: null, locked: true };
        throw err;
      }
    },
    staleTime: 60_000,
    retry: false,
  });
}
