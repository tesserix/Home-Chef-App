import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api } from '../lib/api';

export interface AvailableDelivery {
  id: string;
  orderId: string;
  pickupAddress: string;
  dropoffAddress: string;
  distance: number;
  payout: number;
  estimatedTime: number;
  chefName: string;
  itemCount: number;
}

export interface ActiveDelivery {
  id: string;
  orderId: string;
  status: 'assigned' | 'at_pickup' | 'picked_up' | 'in_transit' | 'at_dropoff' | 'delivered' | 'cancelled';
  pickup: {
    address: string;
    lat: number;
    lng: number;
    chefName: string;
    chefPhone: string;
    instructions?: string;
  };
  dropoff: {
    address: string;
    lat: number;
    lng: number;
    customerName: string;
    customerPhone: string;
    instructions?: string;
  };
  order: {
    id: string;
    items: { name: string; quantity: number }[];
    total: number;
    specialInstructions?: string;
  };
  payout: number;
  acceptedAt: string;
}

interface AvailableDeliveriesResponse {
  deliveries: AvailableDelivery[];
  message?: string;
}

export function useAvailableDeliveries() {
  return useQuery<AvailableDeliveriesResponse>({
    queryKey: ['driver', 'available'],
    queryFn: () => api.get('/delivery/available').then((r) => r.data),
    refetchInterval: 30_000,
    refetchIntervalInBackground: false,
  });
}

export function useCurrentDelivery() {
  return useQuery<ActiveDelivery | null>({
    queryKey: ['driver', 'current'],
    queryFn: async () => {
      try {
        const r = await api.get('/delivery/current');
        return r.data ?? null;
      } catch (e: unknown) {
        if (
          e !== null &&
          typeof e === 'object' &&
          'response' in e &&
          (e as { response?: { status?: number } }).response?.status === 404
        ) {
          return null;
        }
        throw e;
      }
    },
    refetchInterval: 15_000,
    refetchIntervalInBackground: false,
  });
}

export function useAcceptDelivery() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (deliveryId: string) =>
      api.post(`/delivery/${deliveryId}/accept`, {}),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['driver', 'available'] });
      queryClient.invalidateQueries({ queryKey: ['driver', 'current'] });
    },
  });
}

export function useUpdateDeliveryStatus() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, status }: { id: string; status: string }) =>
      api.put(`/delivery/${id}/status`, { status }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['driver', 'current'] });
      queryClient.invalidateQueries({ queryKey: ['driver', 'dashboard'] });
    },
  });
}
