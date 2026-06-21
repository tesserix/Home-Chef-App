import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api } from '../lib/api';
import type { ChefWithStats, Paginated } from '../lib/admin-types';

export interface ChefsQuery {
  search?: string;
  status?: string; // pending | verified | suspended | active
  page?: number;
  limit?: number;
}

export function useAdminChefs(q: ChefsQuery = {}) {
  const { search = '', status = '', page = 1, limit = 20 } = q;
  return useQuery<Paginated<ChefWithStats>>({
    queryKey: ['admin', 'chefs', { search, status, page, limit }],
    queryFn: () =>
      api
        .get<Paginated<ChefWithStats>>('/admin/chefs', {
          params: { search, status, page, limit },
        })
        .then((r) => r.data),
  });
}

/**
 * There is no GET /admin/chefs/:id endpoint — the list response carries the
 * full enriched chef. The detail screen reads the chef out of whichever
 * chefs-list query is cached (warm after navigating from the list).
 */
export function useAdminChefFromCache(id: string): ChefWithStats | undefined {
  const qc = useQueryClient();
  const entries = qc.getQueriesData<Paginated<ChefWithStats>>({ queryKey: ['admin', 'chefs'] });
  for (const [, data] of entries) {
    const found = data?.data?.find((ch) => ch.id === id);
    if (found) return found;
  }
  return undefined;
}

function invalidateChefs(qc: ReturnType<typeof useQueryClient>) {
  qc.invalidateQueries({ queryKey: ['admin', 'chefs'] });
  qc.invalidateQueries({ queryKey: ['admin', 'stats'] });
  qc.invalidateQueries({ queryKey: ['admin', 'approvals'] });
}

export function useVerifyChef() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (chefId: string) => api.put(`/admin/chefs/${chefId}/verify`),
    onSuccess: () => invalidateChefs(qc),
  });
}

export function useRejectChef() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ chefId, reason }: { chefId: string; reason: string }) =>
      api.put(`/admin/chefs/${chefId}/reject`, { reason }),
    onSuccess: () => invalidateChefs(qc),
  });
}

export function useSuspendChef() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (chefId: string) => api.put(`/admin/chefs/${chefId}/suspend`),
    onSuccess: () => invalidateChefs(qc),
  });
}
