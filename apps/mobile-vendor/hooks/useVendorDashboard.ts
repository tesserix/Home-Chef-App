import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '../lib/api';

export interface DashboardData {
  todayOrders: number;
  todayEarnings: number;
  rating: number;
  totalReviews: number;
  acceptingOrders: boolean;
  recentOrders: RecentOrder[];
}

export interface RecentOrder {
  id: string;
  customerName: string;
  total: number;
  status: string;
  createdAt: string;
}

export function useVendorDashboard() {
  return useQuery<DashboardData>({
    queryKey: ['chef', 'dashboard'],
    queryFn: () => api.get<DashboardData>('/chef/dashboard').then((r) => r.data),
    staleTime: 30_000,
  });
}

export function useToggleAcceptingOrders() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (acceptingOrders: boolean) =>
      api.put('/chef/settings', { acceptingOrders }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['chef', 'dashboard'] }),
  });
}
