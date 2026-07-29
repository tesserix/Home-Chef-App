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
 * Every plan this chef has, newest first. The server returns the full history,
 * so callers filter to the states they care about rather than re-querying.
 */
export function usePlanRequests() {
  return useQuery<PlanRequest[]>({
    queryKey: ['chef', 'meal-plans'],
    queryFn: () =>
      apiClient.get<{ data: PlanRequest[] }>('/chef/meal-plans').then((r) => r?.data ?? []),
    staleTime: 30_000,
  });
}

/**
 * Only the plans actually waiting on this chef. `pending_chef` is the one state
 * where the chef is the blocker — everything else is with the customer, already
 * confirmed, or finished.
 */
export function usePendingPlanRequests() {
  const q = usePlanRequests();
  return { ...q, data: (q.data ?? []).filter((p) => p.status === 'pending_chef') };
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
