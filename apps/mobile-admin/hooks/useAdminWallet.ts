import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api } from '../lib/api';
import type { WalletResponse } from '../lib/admin-types';

export function useCustomerWallet(userId: string) {
  return useQuery<WalletResponse>({
    queryKey: ['admin', 'wallet', userId],
    queryFn: () => api.get<WalletResponse>(`/admin/wallet/${userId}`).then((r) => r.data),
    enabled: !!userId,
  });
}

export function useAdjustWallet() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({
      userId,
      amount,
      reason,
      type,
    }: {
      userId: string;
      amount: number;
      reason: string;
      type: 'credit' | 'debit';
    }) => api.post(`/admin/wallet/${userId}/adjust`, { amount, reason, type }),
    onSuccess: (_data, vars) =>
      qc.invalidateQueries({ queryKey: ['admin', 'wallet', vars.userId] }),
  });
}
