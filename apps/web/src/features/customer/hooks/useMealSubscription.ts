import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { apiClient } from '@/shared/services/api-client';

// Customer meal subscription (tiffin, #283) — web. Paths resolve under /v1 via the
// shared api-client (the meal-subscription routes are not under /customer).

export interface MealChefOffer {
  available: boolean;
  slots?: string[];
  cadences?: string[];
  perMealPrice?: number;
  deliveryFee?: number;
  trialEnabled?: boolean;
  trialPrice?: number;
}

export interface MealSubscription {
  id: string;
  chefId: string;
  slots: string[];
  days: number[];
  variant: string;
  cadence: string;
  cycleAmount: number;
  status: 'trialing' | 'active' | 'paused' | 'past_due' | 'cancelled';
  creditBalance: number;
}

export interface MealAdherence {
  scheduled: number;
  delivered: number;
  missed: number;
  skipped: number;
}

export interface MealSelection {
  chefId: string;
  slots: string[];
  days: number[];
  variant: string;
  cadence: string;
}

export function useMealChefOffer(chefId?: string) {
  return useQuery({
    queryKey: ['meal-offer', chefId],
    queryFn: () => apiClient.get<MealChefOffer>(`/chefs/${chefId}/subscription`),
    enabled: !!chefId,
  });
}

export function usePreviewMealPrice() {
  return useMutation({
    mutationFn: (sel: MealSelection) =>
      apiClient.post<{ cycleAmount: number; currency: string; deliveryFee: number }>('/meal-subscriptions/preview', sel),
  });
}

export function useSubscribeMeal() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (sel: MealSelection) => apiClient.post<{ subscription: MealSubscription }>('/meal-subscriptions', sel),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['meal-subscriptions'] }),
  });
}

export function useMealSubscriptions() {
  return useQuery({
    queryKey: ['meal-subscriptions'],
    queryFn: () => apiClient.get<{ data: MealSubscription[]; count: number }>('/meal-subscriptions'),
  });
}

export function useMealSubAction() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, action, reason }: { id: string; action: 'pause' | 'resume' | 'cancel'; reason?: string }) =>
      apiClient.post(`/meal-subscriptions/${id}/${action}`, { reason }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['meal-subscriptions'] }),
  });
}
