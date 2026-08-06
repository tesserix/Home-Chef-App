/**
 * What a cancelled meal plan actually returned (#1041).
 *
 * The agreed percentage applies to each day's refund base (food − commission + GST +
 * delivery), never to the plan total, so quoting the percentage alone cannot be reconciled
 * against what was paid. These are the three figures that can.
 */

const round2 = (n: number) => Math.round(n * 100) / 100;

export interface MealPlanRefundDay {
  refundAmount?: number;
}

export interface MealPlanRefundPlan {
  total: number;
  days?: MealPlanRefundDay[];
}

export interface MealPlanRefundSummary {
  paid: number;
  refunded: number;
  withheld: number;
}

export function mealPlanRefundSummary(plan: MealPlanRefundPlan): MealPlanRefundSummary | null {
  const refunded = round2(
    (plan.days ?? []).reduce((sum, d) => sum + (d.refundAmount ?? 0), 0),
  );
  if (refunded <= 0) return null;

  return {
    paid: round2(plan.total),
    refunded,
    withheld: Math.max(0, round2(plan.total - refunded)),
  };
}
