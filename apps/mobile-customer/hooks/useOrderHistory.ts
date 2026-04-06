import { useQuery } from '@tanstack/react-query';
import { api } from '../lib/api';
import type { Order } from '../types/customer';

export interface OrderListParams {
  page?: number;
  limit?: number;
  status?: string;
}

export interface OrderListResponse {
  data: Order[];
  meta: { total: number; page: number; limit: number };
}

export function useOrders(params: OrderListParams = {}) {
  return useQuery<OrderListResponse>({
    queryKey: ['orders', params],
    queryFn: () =>
      api
        .get('/v1/orders', { params })
        .then((r) => r.data as OrderListResponse),
    staleTime: 1000 * 30, // 30 seconds — orders change frequently
  });
}

export function useOrder(id: string) {
  return useQuery<{ data: Order }>({
    queryKey: ['order', id],
    queryFn: () =>
      api.get(`/v1/orders/${id}`).then((r) => r.data as { data: Order }),
    enabled: !!id,
  });
}
