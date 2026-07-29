import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { apiClient } from '@/shared/services/api-client';

// Chef refund decisions on late meal-plan skips/cancels (meal-plan refund v2).
//
// When a customer skips or cancels a tiffin day with ≤12h notice, the chef has
// already bought ingredients — so instead of an automatic full refund, the chef
// decides: Full, Half, None, or Decline the request outright. Over 12h it
// auto-agrees a full refund and never reaches this queue.
//
// The portal had no way to make that decision, so a web-only chef silently left
// customers waiting on a refund they had asked for.
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
  /** What the food itself cost — the base the refund options are computed from. */
  foodPrice: number;
  /** Server-computed amounts. The UI never derives these: the fee, GST and
   *  delivery slices are excluded server-side and halving foodPrice here would
   *  quietly disagree with what is actually paid out. */
  fullRefund: number;
  halfRefund: number;
}

export type RefundChoice = 'full' | 'half' | 'none';

/** Days awaiting this chef's decision. Empty when the v2 flow is gated off. */
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
  /** Ignored by the server when `decline` is true. */
  choice: RefundChoice;
  decline?: boolean;
}

export function useSubmitRefundDecision() {
  const qc = useQueryClient();
  return useMutation<unknown, unknown, DecisionInput>({
    mutationFn: ({ dayId, choice, decline }) =>
      apiClient.post(`/chef/meal-plan-days/${dayId}/refund-decision`, { choice, decline }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['chef', 'refund-decisions'] });
    },
  });
}
