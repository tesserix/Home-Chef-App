import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api } from '../lib/api';

// ---- API contract types -------------------------------------------------------
// Must match GET /chef/rewards (handlers/chef_rewards.go) exactly.

export interface ChefReferralProgress {
  kitchenName: string;
  status: 'pending' | 'rewarded' | 'rejected';
  deliveredOrders: number;
  milestoneOrders: number;
  reward: number;
  createdAt: string;
  rewardedAt?: string;
}

export interface ChefRewards {
  referral: {
    enabled: boolean;
    code: string;
    link: string;
    referrerAmount: number;
    refereeAmount: number;
    milestoneOrders: number;
    currency: string;
    totalEarned: number;
    referrals: ChefReferralProgress[];
  };
  loyalty: {
    enabled: boolean;
    points: number;
    lifetimePoints: number;
    earnRate: number;
    redeemRate: number;
    minConvertPoints: number;
    canConvert: boolean;
    convertValue: number;
  };
  pendingPayoutCredit: number;
}

export interface ConvertRewardsResponse {
  convertedPoints: number;
  cashback: number;
  currency: string;
  status: string;
  message: string;
}

/** Referral code + milestone progress, points balance + conversion terms. */
export function useChefRewards() {
  return useQuery<ChefRewards>({
    queryKey: ['chef', 'rewards'],
    queryFn: () => api.get<ChefRewards>('/chef/rewards').then((r) => r.data),
    staleTime: 30_000,
  });
}

/** Convert the full points balance into cashback on the next weekly payout. */
export function useConvertRewards() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () =>
      api.post<ConvertRewardsResponse>('/chef/rewards/convert').then((r) => r.data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['chef', 'rewards'] });
    },
  });
}
