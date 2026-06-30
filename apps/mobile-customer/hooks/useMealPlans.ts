import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api } from '../lib/api';

// useMealPlans — customer-side tiffin meal-plan data (#196). Consumes the APIs
// from #192/#194/#196:
//   GET  /chefs/:id/weekly-menu     (public published menu to book against)
//   POST /meal-plans                (book a multi-day calendar from one chef)
//   GET  /meal-plans                (the customer's own plans)
//   GET  /meal-plans/:id            (one plan)
//   PUT  /meal-plans/:id/approve    (accept the chef's trimmed set)
//   PUT  /meal-plans/:id/reject     (decline the trim → cancel)

export type MealSlot = 'lunch' | 'dinner';
export type MealVariant = 'veg' | 'nonveg';

export interface WeeklyMenuItem {
  id?: string;
  dayOfWeek: number; // 0=Sun .. 6=Sat
  slot: MealSlot;
  variant: MealVariant;
  name: string;
  description?: string;
  price: number;
  imageUrl?: string;
}

export interface WeeklyMenu {
  isPublished: boolean;
  publishedAt?: string | null;
  items: WeeklyMenuItem[];
}

export interface MealPlanDay {
  id: string;
  date: string;
  slot: MealSlot;
  variant: MealVariant;
  status: string;
  dishName?: string;
  price: number;
}

export interface MealPlan {
  id: string;
  mealPlanNumber: string;
  chefId: string;
  status: string;
  startDate: string;
  endDate: string;
  subtotal: number;
  total: number;
  currency?: string;
  days: MealPlanDay[];
  chef?: { businessName?: string; profileImage?: string } | null;
  chefRespondBy?: string | null;
  customerApproveBy?: string | null;
}

export interface CreateMealPlanDay {
  date: string; // YYYY-MM-DD
  slot: MealSlot;
  variant: MealVariant;
}

/** A chef's published weekly menu (the cells a customer books against). */
export function useChefWeeklyMenu(chefId: string | undefined) {
  return useQuery<WeeklyMenu>({
    queryKey: ['chef-weekly-menu', chefId],
    queryFn: () =>
      api.get<WeeklyMenu>(`/v1/chefs/${chefId}/weekly-menu`).then((r) => r.data),
    enabled: Boolean(chefId),
    staleTime: 5 * 60_000,
  });
}

/** The customer's own meal plans (any status). */
export function useMyMealPlans() {
  return useQuery<{ data: MealPlan[] }>({
    queryKey: ['meal-plans'],
    queryFn: () =>
      api.get<{ data: MealPlan[] }>('/v1/meal-plans').then((r) => r.data),
  });
}

/** One meal plan by id (scoped to the authed customer server-side). Polls while
 *  the plan is live so the per-day cooking status updates in near-real-time (#50). */
export function useMealPlan(id: string | undefined) {
  return useQuery<{ mealPlan: MealPlan }>({
    queryKey: ['meal-plans', id],
    queryFn: () =>
      api.get<{ mealPlan: MealPlan }>(`/v1/meal-plans/${id}`).then((r) => r.data),
    enabled: Boolean(id),
    refetchInterval: (query) => {
      const s = query.state.data?.mealPlan?.status;
      return s === 'confirmed' || s === 'active' ? 20000 : false;
    },
  });
}

/** Book a calendar of days from one chef. */
export function useCreateMealPlan() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: { chefId: string; days: CreateMealPlanDay[] }) =>
      api
        .post<{ mealPlan: MealPlan; escrowEnabled?: boolean }>(
          '/v1/meal-plans',
          body,
        )
        .then((r) => r.data),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['meal-plans'] }),
  });
}

/** Skip a confirmed day before its lead-time cutoff (refunded to wallet when escrow is on). */
export function useSkipMealPlanDay() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (vars: { planId: string; dayId: string }) =>
      api
        .put<{ mealPlan: MealPlan }>(
          `/v1/meal-plans/${vars.planId}/days/${vars.dayId}/skip`,
        )
        .then((r) => r.data),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['meal-plans'] }),
  });
}

/** Approve or reject the chef's revised (cherry-picked) plan. */
export function useFinalizeMealPlan() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (vars: { id: string; approve: boolean }) =>
      api
        .put<{ mealPlan: MealPlan }>(
          `/v1/meal-plans/${vars.id}/${vars.approve ? 'approve' : 'reject'}`,
        )
        .then((r) => r.data),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['meal-plans'] }),
  });
}
