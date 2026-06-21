import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api } from '../lib/api';
import type { FSSAILockResponse } from '../lib/admin-types';

export function useFSSAILocked() {
  return useQuery<FSSAILockResponse>({
    queryKey: ['admin', 'fssai-locked'],
    queryFn: () =>
      api.get<FSSAILockResponse>('/admin/chefs/fssai-locked').then((r) => r.data),
  });
}

export function useOverrideFSSAILock() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({
      chefId,
      reason,
      days,
    }: {
      chefId: string;
      reason: string;
      days: number;
    }) => api.post(`/admin/chefs/${chefId}/fssai-override`, { reason, days }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['admin', 'fssai-locked'] });
      qc.invalidateQueries({ queryKey: ['admin', 'chefs'] });
    },
  });
}

export function useClearFSSAIOverride() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (chefId: string) => api.delete(`/admin/chefs/${chefId}/fssai-override`),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['admin', 'fssai-locked'] });
      qc.invalidateQueries({ queryKey: ['admin', 'chefs'] });
    },
  });
}
