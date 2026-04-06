import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api } from '../lib/api';

interface DeliveryStats {
  today: { deliveries: number; earnings: number };
  week: { deliveries: number; earnings: number };
  month: { deliveries: number; earnings: number };
  isOnline: boolean;
  rating: number;
  totalDeliveries: number;
}

export function useDriverDashboard() {
  return useQuery<DeliveryStats>({
    queryKey: ['driver', 'dashboard'],
    queryFn: () => api.get('/delivery/stats').then((r) => r.data),
    staleTime: 30_000,
  });
}

export function useToggleOnline() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (isOnline: boolean) => api.put('/delivery/online', { isOnline }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['driver', 'dashboard'] });
      queryClient.invalidateQueries({ queryKey: ['driver', 'available'] });
    },
  });
}
