import { useQuery } from '@tanstack/react-query';
import { api } from '../lib/api';
import type { MealPlanRow, Paginated } from '../lib/admin-types';

export interface MealPlansQuery {
  status?: string;
  page?: number;
  limit?: number;
}

export function useAdminMealPlans(q: MealPlansQuery = {}) {
  const { status = '', page = 1, limit = 20 } = q;
  return useQuery<Paginated<MealPlanRow>>({
    queryKey: ['admin', 'meal-plans', { status, page, limit }],
    queryFn: () =>
      api
        .get<Paginated<MealPlanRow>>('/admin/meal-plans', {
          params: { status, page, limit },
        })
        .then((r) => r.data),
  });
}

export function useAdminMealPlan(id: string) {
  return useQuery<{ mealPlan: Record<string, unknown> }>({
    queryKey: ['admin', 'meal-plan', id],
    queryFn: () =>
      api.get<{ mealPlan: Record<string, unknown> }>(`/admin/meal-plans/${id}`).then((r) => r.data),
    enabled: !!id,
  });
}
