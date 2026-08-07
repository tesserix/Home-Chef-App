// The payment aggregator's display name for the RBI disclosure on checkout.
//
// Checkout hardcoded "Razorpay" while Cashfree took the money seconds later in
// the same flow (#933). The API already resolves and sends the real gateway on
// the delivery quote (`paymentProvider`, from `SelectCheckoutGateway`) precisely
// so this line can name it — the client simply discarded the field.
//
// The disclosure has to name the aggregator that actually processes the charge,
// so an unknown or missing value must never fall back to guessing a brand: it
// drops to wording that is true regardless.

// Razorpay is deliberately absent (#1086): it can no longer take a payment, so a
// stale or replayed `razorpay` drops to the neutral wording rather than naming a
// processor that will not touch the money — which is the defect above, inverted.
const DISPLAY_NAMES: Record<string, string> = {
  cashfree: 'Cashfree',
  stripe: 'Stripe',
};

/**
 * The sentence fragment naming who secures the payment.
 * Returns e.g. "Payments secured by Cashfree (RBI-licensed)." — or, when the
 * provider is unknown, "Payments are processed by an RBI-licensed payment
 * aggregator." which is accurate for every gateway we use.
 */
export function paymentSecuredByLine(provider?: string | null): string {
  const name = DISPLAY_NAMES[(provider ?? '').trim().toLowerCase()];
  if (!name) return 'Payments are processed by an RBI-licensed payment aggregator.';
  // Stripe is not RBI-licensed — it serves non-INR Connect chefs, where the
  // Indian aggregator framing does not apply.
  if (name === 'Stripe') return 'Payments secured by Stripe.';
  return `Payments secured by ${name} (RBI-licensed).`;
}
