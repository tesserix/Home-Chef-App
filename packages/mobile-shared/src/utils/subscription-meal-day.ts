/**
 * "Which day to show" selector for a recurring meal-subscription summary.
 *
 * The Plans tab (#900) needs to show one line per active subscription --
 * today's meal if one is scheduled, otherwise the next upcoming one -- and
 * reuses the IST day-boundary rule already established for Skip eligibility
 * (#696) so "next" never points at a day whose IST calendar date has already
 * started from the server's point of view.
 */

import { isPastMealDay, istCalendarDate } from './ist-calendar-date';

export interface SubscriptionMealDayLike {
  date: string;
  slot: string;
  status: string;
}

export interface PickedSubscriptionMealDay<T extends SubscriptionMealDayLike> {
  day: T;
  isToday: boolean;
}

const SLOT_ORDER: Record<string, number> = {
  lunch: 0,
  dinner: 1,
};

function slotSortRank(slot: string): number {
  return slot in SLOT_ORDER ? SLOT_ORDER[slot]! : Number.MAX_SAFE_INTEGER;
}

export function pickSubscriptionMealDay<T extends SubscriptionMealDayLike>(
  days: readonly T[],
  now: Date = new Date(),
): PickedSubscriptionMealDay<T> | null {
  const upcoming = days
    .filter((d) => d.status === 'scheduled' && !isPastMealDay(d.date, now))
    .slice()
    .sort((a, b) => {
      const byDate = a.date.localeCompare(b.date);
      if (byDate !== 0) return byDate;
      return slotSortRank(a.slot) - slotSortRank(b.slot);
    });

  const day = upcoming[0];
  if (!day) return null;

  return {
    day,
    isToday: istCalendarDate(new Date(day.date)) === istCalendarDate(now),
  };
}
