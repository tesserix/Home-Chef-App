import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api } from '../lib/api';

// Mirrors the Go API (#40): GET /v1/customer/loyalty returns the points balance,
// tier, streak and the live program config; /transactions returns the standard
// { data, pagination } envelope; POST /redeem converts points to wallet credit.

export interface LoyaltyConfig {
  enabled: boolean;
  redeemRate: number; // ₹ of wallet credit per 1 point
  minRedeem: number; // minimum points per redemption
  streakThreshold: number;
  streakBonus: number;
  tierSilverAt: number;
  tierGoldAt: number;
}

export interface LoyaltyAccount {
  balance: number;
  lifetimePoints: number;
  tier: 'bronze' | 'silver' | 'gold';
  currentStreak: number;
  longestStreak: number;
  config: LoyaltyConfig;
}

export interface LoyaltyTransaction {
  id: string;
  type: 'credit' | 'debit';
  source: string;
  points: number;
  pointsAfter: number;
  orderId?: string;
  reason?: string;
  createdAt: string;
}

export interface RedeemResult {
  pointsRedeemed: number;
  pointsBalance: number;
  walletCredited: number;
  walletBalance: number;
}

const EMPTY_CONFIG: LoyaltyConfig = {
  enabled: true,
  redeemRate: 0.1,
  minRedeem: 100,
  streakThreshold: 7,
  streakBonus: 50,
  tierSilverAt: 1000,
  tierGoldAt: 5000,
};

export function useLoyalty() {
  return useQuery<LoyaltyAccount>({
    queryKey: ['loyalty'],
    queryFn: async () => {
      const r = await api.get('/v1/customer/loyalty');
      return (r.data ?? {
        balance: 0,
        lifetimePoints: 0,
        tier: 'bronze',
        currentStreak: 0,
        longestStreak: 0,
        config: EMPTY_CONFIG,
      }) as LoyaltyAccount;
    },
  });
}

export function useLoyaltyTransactions() {
  return useQuery<LoyaltyTransaction[]>({
    queryKey: ['loyalty-transactions'],
    queryFn: async () => {
      const r = await api.get('/v1/customer/loyalty/transactions', {
        params: { page: 1, limit: 50 },
      });
      return (r.data?.data ?? []) as LoyaltyTransaction[];
    },
  });
}

/** Redeem points to wallet credit. Invalidates loyalty + wallet caches on success. */
export function useRedeemLoyalty() {
  const qc = useQueryClient();
  return useMutation<RedeemResult, Error, number>({
    mutationFn: async (points) => {
      const r = await api.post('/v1/customer/loyalty/redeem', { points });
      return r.data as RedeemResult;
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['loyalty'] });
      qc.invalidateQueries({ queryKey: ['loyalty-transactions'] });
      qc.invalidateQueries({ queryKey: ['wallet'] });
      qc.invalidateQueries({ queryKey: ['wallet-transactions'] });
    },
  });
}

/** Extract the API's error message for a failed redeem. */
export function loyaltyErrorMessage(err: unknown): string {
  const data = (err as { response?: { data?: { error?: unknown } } })?.response?.data?.error;
  if (typeof data === 'string') return data;
  if (data && typeof data === 'object' && 'message' in data) {
    return String((data as { message?: unknown }).message ?? 'Could not redeem points');
  }
  return 'Could not redeem points';
}
