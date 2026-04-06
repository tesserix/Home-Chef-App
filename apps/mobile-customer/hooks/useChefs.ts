import { useQuery } from '@tanstack/react-query';
import { api } from '../lib/api';
import type { Chef, MenuItem } from '../types/customer';

export interface ChefFilters {
  search?: string;
  cuisine?: string;
  dietary?: string;
  rating?: number;
  isOpen?: boolean;
  sort?: 'rating' | 'orders' | 'newest' | 'price';
  page?: number;
  limit?: number;
}

export function useChefs(filters: ChefFilters = {}) {
  return useQuery<{ data: Chef[]; meta: { total: number; page: number; limit: number } }>({
    queryKey: ['chefs', filters],
    queryFn: () => api.get('/v1/chefs', { params: filters }).then((r) => r.data),
    staleTime: 1000 * 60 * 2, // 2 minutes — chef list changes infrequently
  });
}

export function useChef(id: string) {
  return useQuery<{ data: Chef }>({
    queryKey: ['chef', id],
    queryFn: () => api.get(`/v1/chefs/${id}`).then((r) => r.data),
    enabled: !!id,
  });
}

export function useChefMenu(chefId: string) {
  return useQuery<{ data: MenuItem[] }>({
    queryKey: ['chef-menu', chefId],
    queryFn: () => api.get(`/v1/chefs/${chefId}/menu`).then((r) => r.data),
    enabled: !!chefId,
    staleTime: 1000 * 60 * 5, // 5 minutes — menu changes rarely
  });
}
