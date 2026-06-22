import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api } from '../lib/api';
import type { Paginated, UserWithStats } from '../lib/admin-types';

export interface UsersQuery {
  search?: string;
  role?: string;
  page?: number;
  limit?: number;
}

export function useAdminUsers(q: UsersQuery = {}) {
  const { search = '', role = '', page = 1, limit = 20 } = q;
  return useQuery<Paginated<UserWithStats>>({
    queryKey: ['admin', 'users', { search, role, page, limit }],
    queryFn: () =>
      api
        .get<Paginated<UserWithStats>>('/admin/users', {
          params: { search, role, page, limit },
        })
        .then((r) => r.data),
  });
}

function invalidateUsers(qc: ReturnType<typeof useQueryClient>) {
  qc.invalidateQueries({ queryKey: ['admin', 'users'] });
  qc.invalidateQueries({ queryKey: ['admin', 'stats'] });
}

export function useSuspendUser() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (userId: string) => api.put(`/admin/users/${userId}/suspend`),
    onSuccess: () => invalidateUsers(qc),
  });
}

export function useActivateUser() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (userId: string) => api.put(`/admin/users/${userId}/activate`),
    onSuccess: () => invalidateUsers(qc),
  });
}
