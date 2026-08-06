// cancellation-refund-estimate.ts — what the customer is told they will get
// back BEFORE they ask to cancel (#1032).
//
// WHY THIS IS AN ESTIMATE AND NOT A FIGURE
//
// #1032 asks for "the exact amount that will be refunded". There is no exact
// amount to show yet, and the API has no preview endpoint to ask for one
// (routes.go carries only POST/GET `/orders/:id/cancel-request` and its
// `/dispute` — no dry-run). The reason is structural, not a missing endpoint:
// on an accepted or preparing order the refund is selected by the CHEF, who
// picks a reason when they confirm the cancellation, and each reason maps to a
// food-refund tier (not_started 90% / materials_purchased 40% / in_preparation
// 0% / ready 0%, services/cancellation_refund.go). Nobody — server included —
// knows which tier applies until the chef answers. Printing a single precise
// number here would be a promise the platform cannot keep, which is the exact
// failure mode #1032 exists to prevent.
//
// So this returns BOUNDS, and both bounds are deliberately conservative:
//
//  1. The GST is excluded from both. Tax refunds in proportion to the refunded
//     pre-tax base, and since #1033 the server splits it per supply
//     (order.taxFood / taxDelivery / taxService) — but OrderResponse does NOT
//     expose that split (models/order.go: the fields exist on the row, not on
//     the response), so the client cannot compute the refundable share without
//     re-deriving it from an assumed rate. Leaving it out makes both bounds
//     UNDERSTATE. The customer is then told the GST on whatever comes back is
//     added on top — an under-promise that resolves upward, never a shortfall.
//  2. `maxRefundPaise` uses the 90% top tier, not 100%: on an accepted order
//     the chef can never return more than the not-started tier.
//
// The one case that IS determinate: a `pending` order. handlers/cancellation.go
// routes it through services.ClassifyCancellation → CancelPathFullRefund →
// snapshotFor(order, 100), so the food comes back in full with no chef involved
// at all. `exact` reports that, and min === max.
//
// Vocabulary and arithmetic match the POST-cancellation breakdown on the order
// detail screen (a883a9c7, #1048/#1033): platform fee non-refundable, delivery
// kept only when a driver is dispatched, GST on the retained amount retained.
// The estimate and the outcome must read as the same story.

/** The top food-refund tier — services.DefaultCancellationTiers().NotStartedPct.
 *  Overridable per call because the tiers are admin-configurable at runtime
 *  (services.ResolveCancellationTiers); 90 is the shipped default. */
export const MAX_FOOD_REFUND_PCT = 90;

/** Statuses at which the API considers a driver to be carrying the order, so the
 *  delivery fee stops being refundable (handlers/cancellation.go
 *  orderDispatched). None of them is cancellable via this flow — every
 *  cancellable status (pending / accepted / preparing) refunds delivery in full
 *  — but the check is kept rather than assumed, so a future status change
 *  degrades to a smaller estimate rather than an overstated one. */
const DISPATCHED_STATUSES = ['picked_up', 'delivering', 'delivered'];

/** All amounts in PAISE (integers), matching the server's money model. Callers
 *  hold rupee floats and convert at the boundary — see toPaise. */
export interface CancellationRefundEstimateInput {
  /** The order's status right now. Decides both the tier certainty and whether
   *  delivery is still refundable. */
  status: string;
  subtotalPaise: number;
  /** The promo the customer never paid. Subtracted from the food base exactly as
   *  services.CancellationOrder.EffectiveFoodPaise does — #962 was the bug where
   *  the tier was applied to the list price instead. */
  discountPaise: number;
  /** The EFFECTIVE delivery fee (deliveryFeeFinal ?? deliveryFee, #703) — the
   *  same figure handlers/cancellation.go feeds the model. */
  deliveryFeePaise: number;
  platformFeePaise: number;
  taxPaise: number;
  totalPaise: number;
  /** Already refunded on this order through any other channel. Caps the estimate
   *  the way services.CancellationRefund.CappedAt caps the real refund (#642) —
   *  without it, an order with a prior partial refund would be quoted more than
   *  is still owed. */
  alreadyRefundedPaise: number;
  maxFoodRefundPct?: number;
}

export interface CancellationRefundEstimate {
  /** The food the customer actually paid for (subtotal − discount). */
  foodPaise: number;
  /** Delivery given back — the full fee unless a driver is already carrying it. */
  deliveryRefundPaise: number;
  /** Never refundable. Named, not derived, so this line equals the Platform fee
   *  line in the order's own price breakdown. */
  platformFeeKeptPaise: number;
  /** Total GST on the order. Split, not the refundable share — see the header. */
  taxPaise: number;
  /** The floor: what comes back even if the chef has already cooked. GST-exclusive. */
  minRefundPaise: number;
  /** The ceiling: the top tier applied to the food. GST-exclusive. */
  maxRefundPaise: number;
  /** True when no chef decision is pending, so min === max and the estimate can
   *  be stated as a single figure. */
  exact: boolean;
}

/** Rupee float → paise integer. Rounds, because a float subtotal of 404.07
 *  arrives as 404.07000000000005 often enough to matter. */
export function toPaise(rupees: number | null | undefined): number {
  return typeof rupees === 'number' && Number.isFinite(rupees) ? Math.round(rupees * 100) : 0;
}

function clampPct(pct: number): number {
  if (!Number.isFinite(pct) || pct < 0) return 0;
  return pct > 100 ? 100 : Math.trunc(pct);
}

function atLeastZero(n: number): number {
  return Number.isFinite(n) && n > 0 ? Math.trunc(n) : 0;
}

/**
 * Bound the refund a customer would receive if they asked to cancel right now.
 *
 * Pure: no network, no clock, no cache — the same inputs always give the same
 * bounds, so the numbers a customer is shown can be reproduced exactly from a
 * support ticket. Mirrors services.ComputeCancellationRefund's integer
 * arithmetic (truncating division, never rounding up) so the client never quotes
 * a rupee the server would not pay.
 */
export function estimateCancellationRefund(
  input: CancellationRefundEstimateInput,
): CancellationRefundEstimate {
  const food = atLeastZero(input.subtotalPaise - input.discountPaise);
  const dispatched = DISPATCHED_STATUSES.includes(input.status);
  const deliveryRefund = dispatched ? 0 : atLeastZero(input.deliveryFeePaise);

  // `pending` means the chef has not accepted, so there is no chef to consult and
  // no tier to pick: the server refunds the food in full on the spot.
  const settledNow = input.status === 'pending';
  const minPct = settledNow ? 100 : 0;
  const maxPct = settledNow ? 100 : clampPct(input.maxFoodRefundPct ?? MAX_FOOD_REFUND_PCT);

  // Integer division, truncating — the server's `foodPaise * pct / 100` in Go.
  const minRefundRaw = Math.trunc((food * minPct) / 100) + deliveryRefund;
  const maxRefundRaw = Math.trunc((food * maxPct) / 100) + deliveryRefund;

  // Nothing may be quoted above what is still owed on the order (#642).
  const remaining = atLeastZero(input.totalPaise - input.alreadyRefundedPaise);
  const minRefund = Math.min(minRefundRaw, remaining);
  const maxRefund = Math.min(maxRefundRaw, remaining);

  return {
    foodPaise: food,
    deliveryRefundPaise: deliveryRefund,
    platformFeeKeptPaise: atLeastZero(input.platformFeePaise),
    taxPaise: atLeastZero(input.taxPaise),
    minRefundPaise: minRefund,
    maxRefundPaise: maxRefund,
    exact: minRefund === maxRefund,
  };
}
