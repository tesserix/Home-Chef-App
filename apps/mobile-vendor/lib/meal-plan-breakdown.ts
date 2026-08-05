// meal-plan-breakdown.ts — turns the server's plan settlement into the two line
// lists a chef reads: what the customer was charged, and what the kitchen earns.
//
// Every amount comes from the API. Nothing is recomputed here, so a plan and an
// à la carte order can never disagree by a paise — the drift that makes a chef
// distrust their payout. This file only decides order, labels and signs.

import type { MealPlanChefEarnings } from '../hooks/useMealPlans';

export interface BreakdownLine {
  label: string;
  amount: number;
}

/** Trims 6.00% → 6%, 7.50% → 7.5%. */
function ratePercent(rate: number): string {
  return `${parseFloat((rate * 100).toFixed(2))}%`;
}

function nonZero(lines: BreakdownLine[]): BreakdownLine[] {
  return lines.filter((l) => Math.abs(l.amount) >= 0.005);
}

/** The customer's side of the plan, in receipt order. Sums to customerTotal. */
export function customerChargeLines(e: MealPlanChefEarnings): BreakdownLine[] {
  const lines = nonZero([
    { label: 'Food', amount: e.customerSubtotal },
    { label: 'Platform fee', amount: e.customerPlatformFee },
    { label: 'Delivery', amount: e.customerDelivery },
    { label: 'GST on food', amount: e.customerTaxFood },
    { label: 'GST on platform fee', amount: e.customerTaxService },
    { label: 'GST on delivery', amount: e.customerTaxDelivery },
  ]);
  if (e.refundedToCustomer >= 0.005) {
    lines.push({ label: 'Refunded', amount: -e.refundedToCustomer });
  }
  return lines;
}

/** The kitchen's side. Sums to netPayout — delivery and the platform fee are absent
 *  because neither is the chef's money. */
export function chefEarningLines(e: MealPlanChefEarnings): BreakdownLine[] {
  if (e.gross < 0.005) return [];
  return nonZero([
    { label: 'Food', amount: e.foodSubtotal },
    { label: 'GST on food', amount: e.foodGst },
    {
      label: `Platform commission (${ratePercent(e.commissionRate)})`,
      amount: -e.platformCommission,
    },
    { label: `TDS (${ratePercent(e.tdsRate)})`, amount: -e.tds },
  ]);
}

/** True only while the plan is still waiting on this chef's accept/trim. */
export function planRespondable(status?: string): boolean {
  return status === 'pending_chef';
}

const STATUS_LABELS: Record<string, string> = {
  pending_chef: 'Awaiting your response',
  chef_accepted_full: 'Accepted — awaiting payment',
  chef_modified: 'Trimmed — awaiting customer',
  awaiting_customer: 'Awaiting customer approval',
  confirmed: 'Confirmed',
  active: 'In progress',
  completed: 'Completed',
  cancelled: 'Cancelled',
  expired: 'Expired',
};

export function planStatusLabel(status: string): string {
  return STATUS_LABELS[status] ?? status;
}
