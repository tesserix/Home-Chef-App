/**
 * Meal-plan charge display (#1039).
 *
 * The server folds delivery into `total` and reports food as `subtotal` and GST
 * as `tax`, so delivery is the residual. Screens printed only `total`, leaving
 * the difference from the listed meal prices unexplained.
 */

const round2 = (n: number) => Math.round(n * 100) / 100;

export interface MealPlanCharge {
  subtotal: number;
  tax?: number;
  total: number;
}

export interface MealPlanChargeLine {
  label: string;
  amount: number;
}

export interface MealPlanApprovalInput {
  /** Food value of the days the chef accepted. */
  acceptedFood: number;
  acceptedDayCount: number;
  totalDayCount: number;
}

export interface MealPlanApprovalEstimate {
  food: number;
  delivery: number;
  gst: number;
  total: number;
}

function deliveryOf(plan: MealPlanCharge): number {
  return Math.max(0, round2(plan.total - (plan.subtotal ?? 0) - (plan.tax ?? 0)));
}

/** The charge, itemised. Lines the plan was not charged are left out. */
export function mealPlanChargeLines(plan: MealPlanCharge): MealPlanChargeLine[] {
  const lines: MealPlanChargeLine[] = [{ label: 'Food subtotal', amount: round2(plan.subtotal ?? 0) }];
  const delivery = deliveryOf(plan);
  if (delivery > 0) lines.push({ label: 'Delivery', amount: delivery });
  const gst = round2(plan.tax ?? 0);
  if (gst > 0) lines.push({ label: 'GST', amount: gst });
  return lines;
}

/**
 * What approving a partly-accepted plan will cost. Delivery is per day and GST
 * follows the food, so each is scaled by its own basis rather than by the food
 * share alone — the approval figure was food only, which understated the charge
 * by exactly the amount #402 was raised about.
 */
export function mealPlanApprovalEstimate(
  plan: MealPlanCharge,
  input: MealPlanApprovalInput,
): MealPlanApprovalEstimate {
  if (input.acceptedDayCount <= 0 || input.acceptedFood <= 0) {
    return { food: 0, delivery: 0, gst: 0, total: 0 };
  }

  const food = round2(input.acceptedFood);
  const foodShare = plan.subtotal > 0 ? input.acceptedFood / plan.subtotal : 0;
  const dayShare = input.totalDayCount > 0 ? input.acceptedDayCount / input.totalDayCount : 0;

  const delivery = round2(deliveryOf(plan) * dayShare);
  const gst = round2((plan.tax ?? 0) * foodShare);
  return { food, delivery, gst, total: round2(food + delivery + gst) };
}
