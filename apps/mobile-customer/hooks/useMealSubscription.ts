import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api } from '../lib/api';

// Customer meal subscription (tiffin, #2/#3/#283). Browse a chef's offer, preview
// the price, subscribe, and manage (pause/resume/skip/cancel) + see fulfillment.

export interface MealChefOffer {
  available: boolean;
  slots?: string[];
  cadences?: string[];
  perMealPrice?: number;
  deliveryFee?: number;
  cutoffTime?: string;
  trialEnabled?: boolean;
  trialPrice?: number;
  trialDurationDays?: number;
}

export interface MealSubscription {
  id: string;
  chefId: string;
  slots: string[];
  days: number[];
  variant: string;
  cadence: string;
  cycleAmount: number;
  currency: string;
  status: 'trialing' | 'active' | 'paused' | 'past_due' | 'cancelled';
  currentPeriodEnd?: string;
  creditBalance: number;
}

export interface MealFulfillment {
  id: string;
  date: string;
  slot: string;
  dishName: string;
  price: number;
  status: 'scheduled' | 'placed' | 'delivered' | 'missed' | 'skipped';
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
  addressId?: string;
}

/** A chef's tiffin offer (or { available:false }). */
export function useMealChefOffer(chefId?: string) {
  return useQuery<MealChefOffer>({
    queryKey: ['meal-offer', chefId],
    queryFn: async () => (await api.get<MealChefOffer>(`/v1/chefs/${chefId}/subscription`)).data,
    enabled: !!chefId,
  });
}

/** Live per-cycle price preview for a selection. */
export function usePreviewMealPrice() {
  return useMutation<{ cycleAmount: number; currency: string; deliveryFee: number }, Error, MealSelection>({
    mutationFn: async (sel) => (await api.post('/v1/meal-subscriptions/preview', sel)).data,
  });
}

export function useSubscribeMeal() {
  const qc = useQueryClient();
  return useMutation<{ subscription: MealSubscription }, Error, MealSelection>({
    mutationFn: async (sel) => (await api.post('/v1/meal-subscriptions', sel)).data,
    onSuccess: () => qc.invalidateQueries({ queryKey: ['meal-subscriptions'] }),
  });
}

export function useMealSubscriptions() {
  return useQuery<{ data: MealSubscription[]; count: number }>({
    queryKey: ['meal-subscriptions'],
    queryFn: async () => (await api.get('/v1/meal-subscriptions')).data,
  });
}

export function useMealFulfillments(id?: string) {
  return useQuery<{ data: MealFulfillment[]; adherence: MealAdherence }>({
    queryKey: ['meal-fulfillments', id],
    queryFn: async () => (await api.get(`/v1/meal-subscriptions/${id}/fulfillments`)).data,
    enabled: !!id,
  });
}

/** Lifecycle action: pause | resume | skip | cancel. */
export function useMealSubAction() {
  const qc = useQueryClient();
  return useMutation<unknown, Error, { id: string; action: 'pause' | 'resume' | 'cancel' | 'skip'; date?: string; reason?: string }>({
    mutationFn: async ({ id, action, date, reason }) =>
      (await api.post(`/v1/meal-subscriptions/${id}/${action}`, action === 'skip' ? { date } : { reason })).data,
    onSuccess: () => qc.invalidateQueries({ queryKey: ['meal-subscriptions'] }),
  });
}
