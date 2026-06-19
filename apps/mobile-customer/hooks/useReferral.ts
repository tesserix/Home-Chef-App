import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api } from '../lib/api';

// Referral program hooks (#38). Reward amounts come from the API (admin-
// configurable); never hardcode them. The reward lands in the store-credit wallet.

export interface ReferralStats {
  rewardedCount: number;
  pendingCount: number;
  totalEarned: number;
}

export interface ReferralInfo {
  code: string;
  link: string;
  enabled: boolean;
  referrerReward: number;
  refereeReward: number;
  currency: string;
  stats: ReferralStats;
}

export interface ReferralHistoryItem {
  refereeName: string;
  status: 'pending' | 'rewarded' | 'rejected';
  reward: number;
  createdAt: string;
}

export function useReferral() {
  return useQuery<ReferralInfo>({
    queryKey: ['referral'],
    queryFn: () => api.get('/v1/customer/referral').then((r) => r.data as ReferralInfo),
    staleTime: 60_000,
  });
}

export function useReferralHistory() {
  return useQuery<ReferralHistoryItem[]>({
    queryKey: ['referral', 'history'],
    queryFn: () =>
      api.get('/v1/customer/referral/history').then((r) => (r.data?.data ?? []) as ReferralHistoryItem[]),
    staleTime: 60_000,
  });
}

/** Apply a referral code (call once, right after signup). Idempotent server-side. */
export function useAcceptReferral() {
  const queryClient = useQueryClient();
  return useMutation<{ status: string; code: string }, Error, string>({
    mutationFn: (code: string) =>
      api.post('/v1/customer/referral/accept', { code }).then((r) => r.data as { status: string; code: string }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['referral'] });
    },
  });
}
