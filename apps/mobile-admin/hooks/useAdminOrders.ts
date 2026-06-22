import { useQuery } from '@tanstack/react-query';
import { api } from '../lib/api';
import type { OrderDetail, OrderRow, Paginated } from '../lib/admin-types';

export interface OrdersQuery {
  search?: string;
  status?: string;
  page?: number;
  limit?: number;
}

export function useAdminOrders(q: OrdersQuery = {}) {
  const { search = '', status = '', page = 1, limit = 20 } = q;
  return useQuery<Paginated<OrderRow>>({
    queryKey: ['admin', 'orders', { search, status, page, limit }],
    queryFn: () =>
      api
        .get<Paginated<OrderRow>>('/admin/orders', {
          params: { search, status, page, limit },
        })
        .then((r) => r.data),
  });
}

export function useAdminOrder(id: string) {
  return useQuery<OrderDetail>({
    queryKey: ['admin', 'order', id],
    queryFn: () => api.get<OrderDetail>(`/admin/orders/${id}`).then((r) => r.data),
    enabled: !!id,
  });
}
