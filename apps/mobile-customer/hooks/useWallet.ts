import { useQuery } from '@tanstack/react-query';
import { api } from '../lib/api';
import { useAuthStore } from '../store/auth-store';

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

// A wallet belongs to an account, so neither of these may run for someone
// browsing without one — signed out they are guaranteed 401s, and the home
// screen would render a ₹0 wallet chip built from the failure.
function useHasAccount(): boolean {
  return useAuthStore((s) => s.isAuthenticated);
}

export function useWallet() {
  const hasAccount = useHasAccount();
  return useQuery<WalletBalance>({
    queryKey: ['wallet'],
    enabled: hasAccount,
    queryFn: async () => {
      const r = await api.get('/v1/customer/wallet');
      return (r.data ?? { balance: 0, currency: 'INR' }) as WalletBalance;
    },
  });
}

export function useWalletTransactions() {
  const hasAccount = useHasAccount();
  return useQuery<WalletTransaction[]>({
    queryKey: ['wallet-transactions'],
    enabled: hasAccount,
    queryFn: async () => {
      const r = await api.get('/v1/customer/wallet/transactions', {
        params: { page: 1, limit: 50 },
      });
      return (r.data?.data ?? []) as WalletTransaction[];
    },
  });
}
