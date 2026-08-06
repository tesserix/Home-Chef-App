// Which meal a plan-holder should see on the Home screen (#1037).
//
// Home was pure chef-browsing: a customer mid-plan got no signal that dinner was
// already booked and being cooked. This picks the ONE meal worth surfacing —
// across every live plan, not just the most recent one — so the card can stay a
// single glanceable row rather than a list.
//
// Pure and clock-injected (`todayKey` is passed in) so the today/next/hidden
// boundaries are testable without mocking Date.

import type { MealPlan, MealPlanDay } from '../hooks/useMealPlans';
import { isDeclinedDayStatus, isLiveMealPlanStatus, toLocalDateKey } from './meal-plan';

export interface ActiveMealPlanMeal {
  plan: MealPlan;
  day: MealPlanDay;
  /** 1-based position of `day` among the plan's still-standing meals. */
  mealNumber: number;
  /** Meals the plan still counts — declined/skipped/cancelled/refunded excluded. */
  totalMeals: number;
  /** `day` falls on the caller's local calendar today. */
  isToday: boolean;
}

// Two meals can share a date, so the slot breaks the tie. Unknown slots sort
// last rather than colliding at 0 with breakfast.
const SLOT_ORDER: Record<string, number> = { breakfast: 0, lunch: 1, dinner: 2 };
const slotRank = (slot: string): number => SLOT_ORDER[slot] ?? 9;

function byDateThenSlot(a: MealPlanDay, b: MealPlanDay): number {
  const d = toLocalDateKey(a.date).localeCompare(toLocalDateKey(b.date));
  return d !== 0 ? d : slotRank(a.slot) - slotRank(b.slot);
}

/**
 * The meal to show on Home, or null when there is nothing worth showing.
 *
 * Rules, in order:
 *  - Only plans in a live status count (a cancelled or completed plan is not news).
 *  - Only STANDING meals count — a declined/skipped/cancelled/refunded day is
 *    struck through everywhere else and must not be presented as upcoming.
 *  - Never a past-dated meal. #1034 leaves finished days sitting at 'Scheduled'
 *    indefinitely, so trusting status alone would put a phantom "Scheduled" for
 *    last Tuesday on the Home screen. The date is the authority here, not the
 *    status.
 *  - Prefer the earliest still-to-come meal (today's before tomorrow's).
 *  - If everything from today on is already delivered, fall back to today's last
 *    meal so the card reads "Delivered" for the rest of the day instead of
 *    vanishing the moment lunch arrives.
 *  - Across several live plans, whichever meal lands soonest wins.
 */
export function selectActiveMealPlanMeal(
  plans: MealPlan[] | undefined,
  todayKey: string,
): ActiveMealPlanMeal | null {
  const candidates: ActiveMealPlanMeal[] = [];

  for (const plan of plans ?? []) {
    if (!isLiveMealPlanStatus(plan.status)) continue;

    const standing = (plan.days ?? [])
      .filter((d) => !isDeclinedDayStatus(d.status))
      .sort(byDateThenSlot);
    if (standing.length === 0) continue;

    const fromTodayOn = standing.filter((d) => toLocalDateKey(d.date) >= todayKey);
    const day =
      fromTodayOn.find((d) => d.status !== 'delivered') ??
      // Nothing left to come: keep today's last meal on screen, but only today's.
      [...fromTodayOn].reverse().find((d) => toLocalDateKey(d.date) === todayKey);
    if (!day) continue;

    candidates.push({
      plan,
      day,
      mealNumber: standing.findIndex((d) => d.id === day.id) + 1,
      totalMeals: standing.length,
      isToday: toLocalDateKey(day.date) === todayKey,
    });
  }

  if (candidates.length === 0) return null;
  return candidates.sort((a, b) => byDateThenSlot(a.day, b.day))[0] ?? null;
}
