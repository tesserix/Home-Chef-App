import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api } from '../lib/api';

// Customer meal subscription (tiffin, #2/#3/#283) — MANAGEMENT ONLY since
// #1035: list, pause/resume/skip/cancel, and see fulfilment. New signups were
// removed from the UI, so nothing here browses a chef's offer or subscribes.

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

// `MealChefOffer` / `MealSelection`, the chef's-offer read, the price preview
// and the subscribe mutation lived here to serve /meal-subscription/:chefId —
// the signup screen. #1035 removed new recurring signups from the UI, so that
// screen and its only entry point are gone and these had no callers left.
// GET /v1/chefs/:id/subscription and POST /v1/meal-subscriptions{,/preview}
// remain on the server; restore from git history if signup ever comes back.

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
    onSuccess: (_data, { id }) => {
      void qc.invalidateQueries({ queryKey: ['meal-subscriptions'] });
      // Skip changes a DAY, not the subscription — without this the day list keeps
      // rendering the skipped meal as "scheduled" and the customer taps Skip again
      // on a day that is already skipped. Invalidating only the subscription (which
      // is all this did) leaves the very list the action was taken from stale.
      void qc.invalidateQueries({ queryKey: ['meal-fulfillments', id] });
    },
  });
}
