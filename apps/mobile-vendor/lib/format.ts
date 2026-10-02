// Money formatting for the vendor app.
//
// Every figure here is a real amount somebody was charged or will be paid, so it
// is rendered to the paise. The app previously used `toFixed(0)` ad hoc, which
// showed a ₹351.97 order as "₹352" on the chef's card while the customer's
// receipt said ₹351.97 — and, worse, pre-filled the delivery-fee field with "39"
// for a fee of ₹39.16, so a chef accepting the value they were shown refunded
// ₹0.16 they never meant to.
//
// Mirrors mobile-customer/lib/format.ts exactly so the two apps can never render
// the same amount differently.

/** Exact paise when the amount has them, whole rupees when it doesn't. */
function parts(amount: number | null | undefined): {
  n: number;
  hasPaise: boolean;
} {
  const n = typeof amount === "number" && Number.isFinite(amount) ? amount : 0;
  return { n, hasPaise: Math.round(n) !== n };
}

/** "₹351.97" / "₹352" — for display next to a label. */
export function formatMoney(
  amount: number | null | undefined,
  currency = "INR",
): string {
  const { n, hasPaise } = parts(amount);
  const code = currency.trim().toUpperCase() || "INR";
  const locale = code === "INR" ? "en-IN" : code === "NZD" ? "en-NZ" : "en-AU";
  const prefix = code === "INR" ? "₹" : `${code} `;
  return `${prefix}${n.toLocaleString(locale, {
    minimumFractionDigits: hasPaise ? 2 : 0,
    maximumFractionDigits: 2,
  })}`;
}

/** "351.97" / "352" — where the ₹ is already rendered, e.g. an input prefix. */
export function moneyValue(amount: number | null | undefined): string {
  const { n, hasPaise } = parts(amount);
  return n.toFixed(hasPaise ? 2 : 0);
}
