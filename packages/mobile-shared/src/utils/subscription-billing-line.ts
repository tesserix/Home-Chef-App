/**
 * subscriptionBillingLine() — when the next charge lands and how much it is,
 * for a tiffin subscription card (#1042).
 *
 * The single most important fact about a recurring charge is its date, and the
 * card printed only cadence and amount. Dates come off the wire as instants and
 * are stated in IST, matching the server's billing calendar.
 */

import { istCalendarDate } from './ist-calendar-date';

export type SubscriptionBillingStatus =
  | 'trialing'
  | 'active'
  | 'paused'
  | 'past_due'
  | 'cancelled';

export interface SubscriptionBillingInput {
  status: SubscriptionBillingStatus;
  currentPeriodEnd?: string;
  cycleAmount: number;
  cadence: string;
}

const MONTHS = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'];

function istDayLabel(iso: string): string | null {
  const instant = new Date(iso);
  if (Number.isNaN(instant.getTime())) return null;
  const [, month, day] = istCalendarDate(instant).split('-');
  return `${Number(day)} ${MONTHS[Number(month) - 1]}`;
}

function cycleAmountLabel(input: SubscriptionBillingInput): string {
  const amount = `₹${Math.round(input.cycleAmount).toLocaleString('en-IN')}`;
  return `${amount}/${input.cadence === 'monthly' ? 'month' : 'week'}`;
}

export function subscriptionBillingLine(input: SubscriptionBillingInput): string | null {
  if (input.status === 'cancelled') return null;
  if (input.status === 'paused') return 'Paused · no charge until you resume';
  if (!input.currentPeriodEnd) return null;

  const day = istDayLabel(input.currentPeriodEnd);
  if (!day) return null;

  if (input.status === 'past_due') return `Payment due since ${day}`;
  if (input.status === 'trialing') return `Trial ends ${day} · then ${cycleAmountLabel(input)}`;
  return `Renews ${day} · ${cycleAmountLabel(input)}`;
}
