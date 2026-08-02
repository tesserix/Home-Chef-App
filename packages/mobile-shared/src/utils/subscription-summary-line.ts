/**
 * subscriptionRowSummary() — the single source of the Plans-tab subscription
 * row's display text, across all five states (#907).
 *
 * The `!picked` branch is not an edge case: new subscriptions land `trialing`,
 * auto-activate is off in production, and the daily-order generator only runs
 * for `active` subscriptions, so `pickSubscriptionMealDay` returns `null` for
 * every current subscriber today. Every branch must therefore identify itself
 * as the tiffin subscription, not just the two that already scheduled a meal.
 */

import { istCalendarDate } from './ist-calendar-date';
import type { PickedSubscriptionMealDay, SubscriptionMealDayLike } from './subscription-meal-day';

export function subscriptionRowSummary<T extends SubscriptionMealDayLike>(
  isLoading: boolean,
  isError: boolean,
  picked: PickedSubscriptionMealDay<T> | null,
): string {
  if (isLoading) return 'Tiffin · loading…';
  if (isError) return "Tiffin · couldn't load — tap to view";
  if (!picked) return 'Tiffin · no meals scheduled yet';
  if (picked.isToday) return `Tiffin · today: ${slotLabel(picked.day.slot)}`;
  return `Tiffin · next: ${shortWeekday(picked.day.date)} ${slotLabel(picked.day.slot).toLowerCase()}`;
}

function slotLabel(slot: string): string {
  if (slot === 'lunch') return 'Lunch';
  if (slot === 'dinner') return 'Dinner';
  return slot.charAt(0).toUpperCase() + slot.slice(1);
}

function shortWeekday(dateISO: string): string {
  const [y, m, d] = istCalendarDate(new Date(dateISO)).split('-').map(Number);
  return new Date(Date.UTC(y!, m! - 1, d!)).toLocaleDateString('en-US', {
    weekday: 'short',
    timeZone: 'UTC',
  });
}
