import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { apiClient } from '@/shared/services/api-client';
import type { MealSlot, MealVariant } from '@/shared/types';

// Tiffin meal plans ("plan a week") for the customer web app.
//
// Web shipped the tiffin SUBSCRIPTION (recurring daily tiffin) but never the
// meal PLAN — the pre-book-a-week flow mobile has had all along. The account
// drawer even labelled its entry "Meal Plans" while pointing at /subscriptions,
// so a customer who booked a plan on mobile opened the web app, clicked Meal
// Plans, and was told "No tiffin subscriptions yet" — the plan was real, just
// unreachable. These hooks close that gap against the same endpoints mobile uses.
//
// Backed by /meal-plans (POST create, GET list/detail, PUT approve/reject/cancel,
// PUT days/:dayId/skip, POST verify-payment).

export type MealPlanStatus =
  | 'pending_chef'
  | 'chef_accepted_full'
  | 'chef_modified'
  | 'awaiting_customer'
  | 'confirmed'
  | 'active'
  | 'completed'
  | 'cancelled'
  | 'expired';

export type MealPlanDayStatus =
  | 'requested'
  | 'accepted'
  | 'declined'
  | 'confirmed'
  | 'prepared'
  | 'delivered'
  | 'skip_req'
  | 'skipped'
  | 'cancelled'
  | 'refunded'
  | 'failed';

export interface MealPlanDay {
  id: string;
  date: string;
  slot: MealSlot;
  variant: MealVariant;
  status: MealPlanDayStatus;
  dishName?: string;
  price: number;
}

export interface MealPlan {
  id: string;
  mealPlanNumber: string;
  status: MealPlanStatus;
  startDate: string;
  endDate: string;
  subtotal: number;
  tax: number;
  total: number;
  currency?: string;
  days?: MealPlanDay[];
  /** Minimised chef view — business name only, never the kitchen's address. */
  chef?: { businessName?: string; profileImage?: string };
}

export function useMealPlans() {
  return useQuery<MealPlan[]>({
    queryKey: ['meal-plans'],
    queryFn: () => apiClient.get<{ data: MealPlan[] }>('/meal-plans').then((r) => r?.data ?? []),
    staleTime: 15_000,
  });
}

export function useMealPlan(id: string | undefined) {
  return useQuery<MealPlan>({
    queryKey: ['meal-plans', id],
    // Detail is wrapped in `mealPlan` while the list is wrapped in `data` — the
    // envelopes differ per endpoint, so unwrap explicitly rather than assuming.
    queryFn: () =>
      apiClient.get<{ mealPlan: MealPlan }>(`/meal-plans/${id}`).then((r) => r.mealPlan),
    enabled: !!id,
  });
}

/** One booked (date, slot) selection. `variant` is ignored when a daily-menu dish is picked. */
export interface BookDayInput {
  date: string; // YYYY-MM-DD
  slot: MealSlot;
  variant: MealVariant;
  dailyMenuItemId?: string;
}

export function useCreateMealPlan() {
  const qc = useQueryClient();
  return useMutation<
    { mealPlan: MealPlan; escrowEnabled: boolean },
    unknown,
    { chefId: string; days: BookDayInput[] }
  >({
    mutationFn: (body) =>
      apiClient.post<{ mealPlan: MealPlan; escrowEnabled: boolean }>('/meal-plans', body),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['meal-plans'] });
    },
  });
}

/**
 * Approve response. The server mints the Razorpay advance order and hands back
 * the id + key so the browser can open checkout; `paymentError` means the plan is
 * approved but the charge could not be started and should be retried.
 */
export interface ApproveMealPlanResult {
  razorpayOrderId?: string;
  razorpayKeyId?: string;
  paymentError?: string;
  mealPlan: MealPlan;
}

export function useApproveMealPlan() {
  const qc = useQueryClient();
  return useMutation<ApproveMealPlanResult, unknown, string>({
    mutationFn: (id) => apiClient.put<ApproveMealPlanResult>(`/meal-plans/${id}/approve`, {}),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['meal-plans'] });
    },
  });
}

/** Confirms the advance after Razorpay returns. Flips the plan to `confirmed`. */
export function useVerifyMealPlanPayment() {
  const qc = useQueryClient();
  return useMutation<
    unknown,
    unknown,
    {
      id: string;
      razorpayOrderId: string;
      razorpayPaymentId: string;
      razorpaySignature: string;
    }
  >({
    mutationFn: ({ id, ...body }) => apiClient.post(`/meal-plans/${id}/verify-payment`, body),
    onSuccess: (_r, { id }) => {
      void qc.invalidateQueries({ queryKey: ['meal-plans'] });
      void qc.invalidateQueries({ queryKey: ['meal-plans', id] });
    },
  });
}

type PlanAction = 'reject' | 'cancel';

export function useMealPlanAction() {
  const qc = useQueryClient();
  return useMutation<unknown, unknown, { id: string; action: PlanAction }>({
    mutationFn: ({ id, action }) => apiClient.put(`/meal-plans/${id}/${action}`, {}),
    onSuccess: (_r, { id }) => {
      void qc.invalidateQueries({ queryKey: ['meal-plans'] });
      void qc.invalidateQueries({ queryKey: ['meal-plans', id] });
    },
  });
}

/**
 * Request a skip for one day. Not self-service credit: a skip inside the 12h lock
 * goes to the chef (and then an admin) to decide the refund, so the day moves to
 * `skip_req` rather than straight to refunded.
 */
export function useSkipMealPlanDay() {
  const qc = useQueryClient();
  return useMutation<unknown, unknown, { id: string; dayId: string }>({
    mutationFn: ({ id, dayId }) => apiClient.put(`/meal-plans/${id}/days/${dayId}/skip`, {}),
    onSuccess: (_r, { id }) => {
      void qc.invalidateQueries({ queryKey: ['meal-plans', id] });
    },
  });
}
