import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { apiClient } from '@/shared/services/api-client';

// Pending tiffin plan requests awaiting the chef's answer.
//
// A customer pre-books a week from the published weekly menu, which lands as a
// `pending_chef` plan. The chef either takes every day or cherry-picks the ones
// they can cook; only then does the customer approve and pay the advance.
//
// The portal never called these endpoints, so a chef working on web could not see
// — let alone accept — a single plan request. The request just sat there until the
// 12h chefRespondBy window lapsed and the expiry sweep voided it, which reads to
// the customer as the chef ignoring them. Mobile has had this since the feature
// shipped; this is the web twin of apps/mobile-vendor/app/meal-plans/index.tsx
// plus its review screen.
//
// Backed by GET /chef/meal-plans and POST /chef/meal-plans/:id/respond.

/** Mirrors models.MealPlanDay (apps/api/models/meal_plan.go). */
export interface PlanRequestDay {
  id: string;
  date: string;
  slot: 'lunch' | 'dinner';
  variant: 'veg' | 'nonveg';
  status: string;
  dishName?: string;
  price: number;
}

/** Mirrors models.MealPlan projected for the chef (ProjectForChef). */
export interface PlanRequest {
  id: string;
  mealPlanNumber: string;
  status: string;
  startDate: string;
  endDate: string;
  subtotal: number;
  tax: number;
  total: number;
  days?: PlanRequestDay[];
  /** Minimised customer view — never carries email/phone for a chef. */
  customer?: { firstName?: string; lastName?: string };
}

/**
 * Plans in the given states, newest first.
 *
 * The server filters by status — it does NOT return the full history — so the
 * states you want must be asked for explicitly.
 */
function usePlansByStatus(statuses: string[]) {
  const key = statuses.join(',');
  return useQuery<PlanRequest[]>({
    queryKey: ['chef', 'meal-plans', key],
    queryFn: () =>
      apiClient
        .get<{ data: PlanRequest[] }>(`/chef/meal-plans?status=${encodeURIComponent(key)}`)
        .then((r) => r?.data ?? []),
    staleTime: 30_000,
  });
}

/**
 * Plans actually waiting on this chef. `pending_chef` is the one state where the
 * chef is the blocker — everything else is with the customer, already agreed, or
 * finished.
 */
export function usePendingPlanRequests() {
  return usePlansByStatus(['pending_chef']);
}

/**
 * Plans the chef has agreed to cook but hasn't finished.
 *
 * Without this the kitchen had no view of its own commitments: a plan vanished
 * from every chef-facing surface the moment it was accepted, and only
 * reappeared as individual orders 12h before each meal. A week booked in advance
 * was invisible to the person who agreed to cook it.
 */
export function useUpcomingPlans() {
  return usePlansByStatus(['confirmed', 'active']);
}

/** One plan by id, looked up across the states a chef can act on or watch. */
export function usePlanRequest(id: string | undefined) {
  const pending = usePendingPlanRequests();
  const upcoming = useUpcomingPlans();
  const all = [...(pending.data ?? []), ...(upcoming.data ?? [])];
  return {
    isLoading: pending.isLoading || upcoming.isLoading,
    plan: id ? all.find((p) => p.id === id) : undefined,
  };
}

interface RespondInput {
  planId: string;
  /** Take the whole plan. When false, `acceptedDayIds` decides. */
  acceptAll: boolean;
  /** Ignored by the server when acceptAll is true. */
  acceptedDayIds?: string[];
}

export function useRespondToPlanRequest() {
  const qc = useQueryClient();
  return useMutation<unknown, unknown, RespondInput>({
    mutationFn: ({ planId, acceptAll, acceptedDayIds }) =>
      apiClient.post(`/chef/meal-plans/${planId}/respond`, {
        acceptAll,
        acceptedDayIds: acceptAll ? [] : (acceptedDayIds ?? []),
      }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['chef', 'meal-plans'] });
    },
  });
}
