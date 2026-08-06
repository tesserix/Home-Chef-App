// Display formatting for tiffin subscriptions (#283). Kept out of the screen so
// the money-facing strings — the ones a customer reads to decide whether to keep
// paying — are unit-testable without mounting React.

import type { MealSubscription } from '../hooks/useMealSubscription';

/** Whole rupees, Indian grouping. Cycle amounts are always whole rupees. */
export function money(n: number): string {
  return `₹${Math.round(n).toLocaleString('en-IN')}`;
}

/** "12 Aug" — the charge date, short enough to sit beside the cadence. */
export function fmtChargeDate(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return '';
  return d.toLocaleDateString(undefined, { day: 'numeric', month: 'short' });
}

/**
 * What the customer is next charged, and when (#1042).
 *
 * The card used to print cadence and amount only — `currentPeriodEnd` was on the
 * model and never referenced — so someone trialing a ₹900/week plan could not
 * tell when the trial ended or when they were first billed. That is the single
 * most important fact about a recurring payment.
 *
 * Returns '' when there is genuinely nothing to promise: no period end from the
 * server, an unparseable one, or a cancelled subscription that has no next
 * charge. Saying nothing beats naming a date that will not happen.
 */
export function renewalLine(sub: MealSubscription): string {
  if (!sub.currentPeriodEnd || sub.status === 'cancelled') return '';
  const when = fmtChargeDate(sub.currentPeriodEnd);
  if (!when) return '';
  const per = sub.cadence === 'monthly' ? 'month' : 'week';
  switch (sub.status) {
    case 'trialing':
      return `Trial ends ${when} · then ${money(sub.cycleAmount)}/${per}`;
    case 'paused':
      return `Paused · next charge ${when} if resumed`;
    case 'past_due':
      return `Payment due · retried ${when}`;
    default:
      return `Renews ${when} · ${money(sub.cycleAmount)}/${per}`;
  }
}
