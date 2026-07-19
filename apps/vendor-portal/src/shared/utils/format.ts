// Shared formatting helpers. formatCurrency was previously duplicated verbatim
// across DashboardPage / PayoutsPage / EarningsPage — one source of truth now.

const inrFormatter = new Intl.NumberFormat('en-IN', {
  style: 'currency',
  currency: 'INR',
  minimumFractionDigits: 0,
  maximumFractionDigits: 0,
});

/** Formats an amount as Indian Rupees with no decimals, e.g. ₹1,240. */
export function formatCurrency(amount: number): string {
  return inrFormatter.format(amount);
}
