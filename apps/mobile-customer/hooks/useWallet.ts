import { useQuery } from '@tanstack/react-query';
import { api } from '../lib/api';

// Mirrors the Go API (#33): GET /v1/customer/wallet returns a flat balance,
// /transactions returns the standard { data, pagination } envelope.
export interface WalletBalance {
  balance: number;
  currency: string;
}

export interface WalletTransaction {
  id: string;
  type: 'credit' | 'debit';
  source: string;
  amount: number;
  balanceAfter: number;
  currency: string;
  orderId?: string;
  reason?: string;
  createdAt: string;
}

export function useWallet() {
  return useQuery<WalletBalance>({
    queryKey: ['wallet'],
    queryFn: async () => {
      const r = await api.get('/v1/customer/wallet');
      return (r.data ?? { balance: 0, currency: 'INR' }) as WalletBalance;
    },
  });
}

export function useWalletTransactions() {
  return useQuery<WalletTransaction[]>({
    queryKey: ['wallet-transactions'],
    queryFn: async () => {
      const r = await api.get('/v1/customer/wallet/transactions', {
        params: { page: 1, limit: 50 },
      });
      return (r.data?.data ?? []) as WalletTransaction[];
    },
  });
}
