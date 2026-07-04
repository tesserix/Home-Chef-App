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

// Per-date dynamic menu (#405/#406): a date can carry MULTIPLE dishes per slot,
// and a dish can be a combo (bundle) with its set price + component names. The
// label ("Thali"/"Combo") is localized client-side via comboLabel().
export interface DailyMenuItem {
  id: string;
  date: string; // YYYY-MM-DD
  slot: MealSlot;
  variant: MealVariant;
  name: string;
  description?: string;
  price: number;
  imageUrl?: string;
  isCombo: boolean;
  comboComponents: string[];
  sortOrder: number;
}

export interface DailyMenuDay {
  date: string; // YYYY-MM-DD
  isPublished: boolean;
  publishedAt?: string | null;
  items: DailyMenuItem[];
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
  // When set, book this specific dish/combo from the chef's published per-date
  // menu for that date (#406), overriding the weekly cell.
  dailyMenuItemId?: string;
}

/** A chef's published weekly menu (the cells a customer books against).
 *  Short staleTime + refetch-on-mount so a menu the vendor has since cleared or
 *  unpublished doesn't linger in the customer's cache (#433). */
export function useChefWeeklyMenu(chefId: string | undefined) {
  return useQuery<WeeklyMenu>({
    queryKey: ['chef-weekly-menu', chefId],
    queryFn: () =>
      api.get<WeeklyMenu>(`/v1/chefs/${chefId}/weekly-menu`).then((r) => r.data),
    enabled: Boolean(chefId),
    staleTime: 30_000,
    refetchOnMount: 'always',
    refetchOnWindowFocus: true,
  });
}

/**
 * The chef's PUBLISHED per-date menu over [from, to] (#405/#406). Each day lists
 * its dishes + combos; the customer books a specific dish/combo by id.
 */
export function useChefDailyMenu(
  chefId: string | undefined,
  from?: string,
  to?: string,
) {
  const qs = from && to ? `?from=${from}&to=${to}` : '';
  return useQuery<{ days: DailyMenuDay[] }>({
    queryKey: ['chef-daily-menu', chefId, from, to],
    queryFn: () =>
      api
        .get<{ days: DailyMenuDay[] }>(`/v1/chefs/${chefId}/daily-menu${qs}`)
        .then((r) => r.data),
    enabled: Boolean(chefId),
    // Reflect a cleared/unpublished per-date menu promptly (#433).
    staleTime: 30_000,
    refetchOnMount: 'always',
    refetchOnWindowFocus: true,
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

/** The create response: when escrow is on the server returns a Razorpay order to
 *  collect the full advance before the chef is notified; when it's off only the
 *  plan comes back (unpaid handshake). `paymentError` flags an escrow-on plan the
 *  server couldn't attach a payment order to. */
export interface CreateMealPlanResponse {
  mealPlan: MealPlan;
  escrowEnabled?: boolean;
  razorpayOrderId?: string;
  razorpayKeyId?: string;
  paymentError?: string;
}

/** Book a calendar of days from one chef. */
export function useCreateMealPlan() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: { chefId: string; days: CreateMealPlanDay[] }) =>
      api
        .post<CreateMealPlanResponse>('/v1/meal-plans', body)
        .then((r) => r.data),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['meal-plans'] }),
  });
}

/** Confirm the advance payment for a meal plan after Razorpay checkout (escrow). */
export function useVerifyMealPlanPayment() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (vars: {
      id: string;
      razorpayPaymentId: string;
      razorpaySignature: string;
    }) =>
      api
        .post<{ mealPlan: MealPlan }>(
          `/v1/meal-plans/${vars.id}/verify-payment`,
          {
            razorpayPaymentId: vars.razorpayPaymentId,
            razorpaySignature: vars.razorpaySignature,
          },
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

/** Cancel a plan that hasn't started/been served — withdraw a pending request or
 *  cancel a confirmed-but-unstarted plan. Full refund server-side (nothing was
 *  served). Cancelling frees the customer to rebook with the same chef. */
export function useCancelMealPlan() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      api.put<{ mealPlan: MealPlan }>(`/v1/meal-plans/${id}/cancel`).then((r) => r.data),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['meal-plans'] }),
  });
}

/** Whether the customer may cancel this plan for a full refund (before start). */
export function canCancelMealPlan(plan: Pick<MealPlan, 'status' | 'days'>): boolean {
  const live = ['pending_chef', 'chef_accepted_full', 'chef_modified', 'awaiting_customer', 'confirmed'];
  if (!live.includes(plan.status)) return false;
  return !(plan.days ?? []).some((d) => d.status === 'prepared' || d.status === 'delivered');
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
