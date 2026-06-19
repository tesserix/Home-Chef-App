import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api } from '../lib/api';

// useMealPlans — chef-side tiffin meal-plan data (#195). Wraps the weekly-menu
// CRUD (#192) and the request → accept/cherry-pick handshake (#194/#196):
//   GET  /chef/weekly-menu          (incl. draft)
//   PUT  /chef/weekly-menu          (replace-all + publish)
//   GET  /chef/meal-plans?status=   (the chef's requests, scoped server-side)
//   POST /chef/meal-plans/:id/respond  (accept all / cherry-pick a subset)

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
  menuItemId?: string | null;
}

export interface WeeklyMenu {
  isPublished: boolean;
  publishedAt?: string | null;
  items: WeeklyMenuItem[];
}

export interface MealPlanDay {
  id: string;
  date: string; // ISO
  slot: MealSlot;
  variant: MealVariant;
  status: string;
  dishName?: string;
  price: number;
}

export interface MealPlan {
  id: string;
  mealPlanNumber: string;
  customerId: string;
  chefId: string;
  status: string;
  startDate: string;
  endDate: string;
  subtotal: number;
  total: number;
  currency?: string;
  days: MealPlanDay[];
  customer?: { firstName?: string; lastName?: string; email?: string } | null;
}

/** The authed chef's weekly menu (published or draft). */
export function useWeeklyMenu() {
  return useQuery<WeeklyMenu>({
    queryKey: ['chef', 'weekly-menu'],
    queryFn: () => api.get<WeeklyMenu>('/chef/weekly-menu').then((r) => r.data),
    staleTime: 60_000,
  });
}

/** Replace-all save + publish toggle for the weekly menu. */
export function useSaveWeeklyMenu() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: { isPublished: boolean; items: WeeklyMenuItem[] }) =>
      api.put<WeeklyMenu>('/chef/weekly-menu', body).then((r) => r.data),
    onSuccess: () =>
      qc.invalidateQueries({ queryKey: ['chef', 'weekly-menu'] }),
  });
}

/** The chef's meal-plan requests for a given status (default: awaiting their response). */
export function useChefMealPlanRequests(status: string = 'pending_chef') {
  return useQuery<{ data: MealPlan[] }>({
    queryKey: ['chef', 'meal-plans', status],
    queryFn: () =>
      api
        .get<{ data: MealPlan[] }>(`/chef/meal-plans?status=${status}`)
        .then((r) => r.data),
    refetchInterval: 30_000, // requests are time-boxed (24h) — keep the inbox fresh
    staleTime: 10_000,
  });
}

// ── Bulk subscription prep view (#50) ──────────────────────────────────────

export interface PrepManifestLine {
  slot: MealSlot;
  variant: MealVariant;
  dishName: string;
  total: number;
  prepared: number;
}

export interface PrepPackingRow {
  dayId: string;
  slot: MealSlot;
  variant: MealVariant;
  dishName: string;
  status: string;
  planNumber: string;
  customerName: string;
}

export interface PrepTotals {
  lunch: number;
  dinner: number;
  total: number;
  prepared: number;
}

export interface PrepManifest {
  date: string; // YYYY-MM-DD
  manifest: PrepManifestLine[];
  packingList: PrepPackingRow[];
  totals: PrepTotals;
}

/** The day's prep manifest (counts by dish) + packing list. date = YYYY-MM-DD. */
export function usePrepManifest(date: string) {
  return useQuery<PrepManifest>({
    queryKey: ['chef', 'prep', date],
    queryFn: () => api.get<PrepManifest>(`/chef/prep?date=${date}`).then((r) => r.data),
    refetchInterval: 30_000,
    staleTime: 10_000,
  });
}

/** Mark a whole dish (date+slot+variant+dish) or explicit days prepared. */
export function useMarkPrepared() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: {
      date?: string;
      slot?: MealSlot;
      variant?: MealVariant;
      dishName?: string;
      dayIds?: string[];
    }) => api.post<{ prepared: number }>('/chef/prep/mark', body).then((r) => r.data),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['chef', 'prep'] }),
  });
}

/** Accept all days, or cherry-pick a subset (the rest are declined). */
export function useRespondMealPlan() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (vars: {
      id: string;
      acceptAll: boolean;
      acceptedDayIds: string[];
    }) =>
      api
        .post(`/chef/meal-plans/${vars.id}/respond`, {
          acceptAll: vars.acceptAll,
          acceptedDayIds: vars.acceptedDayIds,
        })
        .then((r) => r.data),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['chef', 'meal-plans'] }),
  });
}
