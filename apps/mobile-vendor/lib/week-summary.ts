// week-summary.ts — the dashboard's one-line "this week".
//
// The money settles on delivery and the tab counters count placed orders, so the
// two figures come from different populations. Showing them side by side as one
// sentence made the arithmetic look wrong ("₹1,200 · 5 orders" off 3 deliveries);
// this picks the phrasing that matches what the chef is actually looking at.

export interface WeekTotals {
  revenue: number;
  settledOrders: number | undefined;
  orders: number;
}

export type WeekSummaryKey =
  | 'dashboard.weekSummary'
  | 'dashboard.weekSummaryPartial'
  | 'dashboard.weekSummaryPending';

export interface WeekSummary {
  key: WeekSummaryKey;
  params: Record<string, string | number>;
}

function inr(n: number): string {
  return Math.round(n).toLocaleString('en-IN');
}

/** null when the week is empty — the line is hidden rather than showing zeroes. */
export function weekSummary({ revenue, settledOrders, orders }: WeekTotals): WeekSummary | null {
  if (orders <= 0) return null;

  // An API build without weekSettledOrders is read as "all of them": claiming
  // nothing was delivered would be a worse guess than the old behaviour.
  const settled = Math.min(settledOrders ?? orders, orders);

  if (settled <= 0) {
    return { key: 'dashboard.weekSummaryPending', params: { count: orders } };
  }
  if (settled < orders) {
    return {
      key: 'dashboard.weekSummaryPartial',
      params: { amount: inr(revenue), settled, count: orders },
    };
  }
  return { key: 'dashboard.weekSummary', params: { amount: inr(revenue), count: orders } };
}
