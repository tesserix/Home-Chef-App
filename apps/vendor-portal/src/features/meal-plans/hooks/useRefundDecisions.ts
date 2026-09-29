import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { apiClient } from '@/shared/services/api-client';

// Chef refund decisions on late meal-plan skips/cancels (refund policy v3, #834).
//
// When a customer cancels a tiffin day at short notice the chef has already bought
// ingredients, so instead of an automatic full refund the chef sets how much to give
// back — any amount from the day's lead-time FLOOR up to 100%. The floor comes from
// the server (pinned when the request was raised) and the server rejects anything
// below it, so this UI constrains the input for usability, not for safety.
//
// A cancellation with plenty of notice auto-agrees the full amount and never reaches
// this queue.
//
// Backed by GET /chef/meal-plan-days/pending-refund-decisions and
// POST /chef/meal-plan-days/:dayId/refund-decision.

/** Mirrors chefRefundDecisionDay in apps/api/handlers/meal_plan_refund_v2.go. */
export interface RefundDecisionDay {
  dayId: string;
  date: string;
  slot: string;
  dishName: string;
  customerName: string;
  mealPlanNumber: string;
  /** What the food itself cost. NOT the refund base — see fullRefund. */
  foodPrice: number;
  /** This day's lead-time floor: the least the chef may refund. */
  minPercent: number;
  /** That floor in rupees. */
  minRefund: number;
  /** 100% of the day's refundable value — food net of the platform commission, plus that
   *  day's GST and delivery. Server-computed; not derivable from foodPrice. */
  fullRefund: number;
  /** 50%. Retained from the pre-v3 fixed Full/Half pair. */
  halfRefund: number;
}

export type RefundChoice = 'full' | 'half' | 'none';

/** Days awaiting this chef's decision. Empty when the flow is gated off. */
export function useRefundDecisions() {
  return useQuery<RefundDecisionDay[]>({
    queryKey: ['chef', 'refund-decisions'],
    queryFn: () =>
      apiClient
        .get<{ data: RefundDecisionDay[] }>('/chef/meal-plan-days/pending-refund-decisions')
        .then((r) => r?.data ?? []),
    staleTime: 30_000,
  });
}

interface DecisionInput {
  dayId: string;
  /** 0–100, at or above the day's minPercent. Ignored when `decline` is true. */
  percent: number;
  decline?: boolean;
}

export function useSubmitRefundDecision() {
  const qc = useQueryClient();
  return useMutation<unknown, unknown, DecisionInput>({
    mutationFn: ({ dayId, percent, decline }) =>
      apiClient.post(`/chef/meal-plan-days/${dayId}/refund-decision`, { percent, decline }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['chef', 'refund-decisions'] });
    },
  });
}

/** The refund at `percent`, interpolated from the server's 100% figure so the preview
 *  tracks the slider without a round-trip. The server recomputes it authoritatively. */
export function refundAtPercent(day: RefundDecisionDay, percent: number): number {
  return Math.floor(day.fullRefund * Math.min(100, Math.max(0, percent)) + 1e-9) / 100;
}
