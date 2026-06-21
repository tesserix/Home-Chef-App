import { useQuery } from '@tanstack/react-query';
import { api } from '../lib/api';
import type {
  DeliveryPartnerSummary,
  DeliveryRow,
  DeliveryStats,
  Paginated,
} from '../lib/admin-types';

export function useDeliveryStats() {
  return useQuery<DeliveryStats>({
    queryKey: ['admin', 'delivery', 'stats'],
    queryFn: () => api.get<DeliveryStats>('/admin/delivery/stats').then((r) => r.data),
  });
}

export function useDeliveryPartners(q: { status?: string; search?: string; page?: number } = {}) {
  const { status = '', search = '', page = 1 } = q;
  return useQuery<Paginated<DeliveryPartnerSummary>>({
    queryKey: ['admin', 'delivery', 'partners', { status, search, page }],
    queryFn: () =>
      api
        .get<Paginated<DeliveryPartnerSummary>>('/admin/delivery/partners', {
          params: { status, search, page, limit: 20 },
        })
        .then((r) => r.data),
  });
}

export function useDeliveryList(q: { status?: string; page?: number } = {}) {
  const { status = '', page = 1 } = q;
  return useQuery<Paginated<DeliveryRow>>({
    queryKey: ['admin', 'delivery', 'list', { status, page }],
    queryFn: () =>
      api
        .get<Paginated<DeliveryRow>>('/admin/delivery/list', {
          params: { status, page, limit: 20 },
        })
        .then((r) => r.data),
  });
}
