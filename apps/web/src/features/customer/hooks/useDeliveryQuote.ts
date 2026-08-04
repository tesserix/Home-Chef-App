import { useQuery } from '@tanstack/react-query';
import { apiClient } from '@/shared/services/api-client';
import type { TaxLine } from '@/shared/types';

// Checkout pricing preview — the web mirror of
// `apps/mobile-customer/hooks/useDeliveryQuote.ts`, hitting the same
// POST /chefs/:id/delivery-quote endpoint.
//
// The web checkout used to price itself: `deliveryFee = chef.deliveryFee`, a
// hardcoded `platformFee = subtotal * 0.05`, and a separate /tax-rates/lookup.
// None of those are what CreateOrder charges — the delivery fee is quoted from
// the drop distance and the service fee comes from the platform policy — so the
// customer agreed to one total and was billed another. Worse, the wallet and
// loyalty balances they hold were invisible here, because the credit allocation
// only ever arrives on this response.
//
// Everything money-shaped on the checkout screen now comes from this one call.

/** Why the loyalty row is capped, so the UI can explain the limit rather than
 *  reimplementing the cap logic to guess at it. */
export type LoyaltyLimit =
  | 'balance'
  | 'per_order_cap'
  | 'monthly_cap'
  | 'order_covered'
  | 'disabled';

export interface CreditQuote {
  /** Ceiling credit may fund: food + delivery. Fees and tax are never included. */
  redeemableCap: number;
  /** Platform fee + tax — always paid in real money. */
  nonRedeemable: number;
  walletBalance: number;
  walletApplied: number;
  /** Upper bound for the wallet control — order-sized, not the raw balance. */
  walletMax: number;
  pointsBalance: number;
  pointsApplied: number;
  /** Rupee value of pointsApplied. */
  pointsValue: number;
  /** Upper bound for the points control. */
  pointsMax: number;
  /** Rupee values of the balance and the per-order maximum, computed server-side
   *  so the app never multiplies points by the redeem rate itself. */
  pointsBalanceValue: number;
  pointsMaxValue: number;
  payable: number;
  loyaltyLimit: LoyaltyLimit;
  walletEnabled: boolean;
  loyaltyEnabled: boolean;
}

/** Live delivery conditions behind a quote. Each factor is ≥1 (1 = neutral). */
export interface SurgeFactors {
  /** Pump-price index vs the baseline the chef's per-km rate assumes. */
  fuel: number;
  /** Congestion on the kitchen→drop route (live vs free-flow drive time). */
  traffic: number;
  /** Weather at the drop. */
  weather: number;
  /** Product of the three, clamped. */
  combined: number;
}

/**
 * Itemised self-delivery fee, present only when the chef delivers themselves.
 * `fee` is the approx MAX: at accept the chef can only bring it down, never up.
 */
export interface SelfDeliveryBreakdown {
  baseFee: number;
  distanceKnown: boolean;
  /** ROAD distance chef→drop, not straight line. */
  distanceKm: number;
  freeRadiusKm: number;
  billableKm: number;
  perKm: number;
  distanceComponent: number;
  /** Drop is inside the chef's free radius — the whole delivery is free. */
  withinFreeZone: boolean;
  maxFee: number;
  capped: boolean;
  fuelSurge: number;
  weatherSurge: number;
  trafficSurge: number;
  /** Combined multiplier actually applied to the distance component (≥1). */
  surgeMultiplier: number;
  fee: number;
}

export interface DeliveryQuote {
  /**
   * The gateway that will actually process this order, resolved server-side.
   *
   * NOT the chef's stored provider: the server may prefer a different gateway,
   * so this is the only value safe to render. The checkout's RBI Payment
   * Aggregator disclosure names an aggregator, and naming the wrong one is a
   * compliance problem rather than a cosmetic bug — so it reads this and nothing
   * else. Absent on an older API, which the UI handles by falling back to
   * neutral wording.
   */
  paymentProvider?: string;
  deliveryFee: number;
  pickupFee: number;
  pickupSaving: number;
  currency: string;
  offersPickup: boolean;
  offersSelfDelivery: boolean;
  /**
   * Whether a DELIVERY order is fulfillable at all — the chef self-delivers, or a
   * 3PL provider is live. This is the exact condition CreateOrder gates on, so
   * when it is false the checkout must not offer delivery: the server would 422,
   * and an order that slipped through would reach `ready` with no carrier and
   * strand there.
   *
   * Defaults to true when an older API omits it, matching the mobile app.
   */
  offersDelivery?: boolean;
  /** Serviceability for the drop coords: within the kitchen's range, the
   *  straight-line distance, and the range cap. */
  deliverable: boolean;
  distanceKm: number;
  maxRadiusKm: number;
  /** false when coords were missing so the distance couldn't be measured. */
  rangeKnown: boolean;
  /** The priced breakdown for the cart and fulfilment mode sent, from the SAME
   *  function CreateOrder charges with (models/pricing.go). Render these; do not
   *  recompute, or checkout and the receipt disagree by a paise. */
  platformFee: number;
  /** The delivery line that went INTO total — 0 for pickup, unlike deliveryFee. */
  effectiveDeliveryFee: number;
  tax: number;
  taxLines: TaxLine[];
  total: number;
  taxRatePercent: number;
  taxName: string;
  taxInclusive: boolean;
  taxCountry: string;
  taxIntraState: boolean;
  /** Server-allocated wallet + loyalty credit for this cart. Absent when the
   *  request was unauthenticated. */
  credit?: CreditQuote;
  /** Approx-max self-delivery fee (₹), present only when the chef self-delivers. */
  selfDeliveryFee?: number;
  selfDeliveryBreakdown?: SelfDeliveryBreakdown;
  /** The live conditions behind the fee, so checkout can say WHY delivery costs
   *  more today rather than showing an unexplained number. */
  surge?: SurgeFactors;
  /** Signed multiplier this quote was priced at. Send it back on order creation so
   *  the order is charged the conditions the customer saw — traffic and weather
   *  move between the checkout screen and payment. Empty when there is no surge. */
  surgePin?: string;
}

/** The customer's credit intent. Amounts are omitted while a rail is on "auto",
 *  which asks the server to apply as much as its ceilings allow. */
export interface CreditIntent {
  useWallet: boolean;
  walletAmount?: number;
  useLoyalty: boolean;
  loyaltyPoints?: number;
}

export interface DeliveryQuoteInput {
  latitude?: number;
  longitude?: number;
  city?: string;
  country?: string;
  state?: string;
  subtotal?: number;
  /** Applied promo, so the credit ceiling is computed on the discounted food. */
  discount?: number;
  /** Tip to the chef — in the payable, never in the redeemable base. */
  tip?: number;
  /**
   * Chosen fulfilment mode. The server prices the credit ceiling against the
   * EFFECTIVE carry fee, which is 0 for pickup — omitting it made the preview
   * allocate credit against a delivery fee the pickup order would never be
   * charged.
   */
  fulfillment?: 'delivery' | 'pickup';
  /** Which credit rails to apply, and optionally how much of each. */
  credit?: CreditIntent;
}

/**
 * Quote fees, tax and credit for a chef + drop address.
 *
 * Coords are optional — without them the server returns the flat policy fee,
 * exactly as CreateOrder falls back, so the preview never blocks on a location.
 * Disabled until a chef id is known.
 */
export function useDeliveryQuote(chefId: string | undefined, input: DeliveryQuoteInput) {
  const { latitude, longitude, city, country, state, subtotal, discount, tip, fulfillment, credit } =
    input;
  return useQuery<DeliveryQuote>({
    // Keyed on everything that moves the fee, the tax OR the credit allocation —
    // a stale credit block would put the screen back in the business of guessing.
    queryKey: [
      'delivery-quote',
      chefId,
      latitude,
      longitude,
      city,
      state,
      subtotal,
      discount,
      tip,
      fulfillment,
      credit?.useWallet,
      credit?.walletAmount,
      credit?.useLoyalty,
      credit?.loyaltyPoints,
    ],
    queryFn: () =>
      apiClient.post<DeliveryQuote>(`/chefs/${chefId}/delivery-quote`, {
        latitude,
        longitude,
        city,
        country,
        state,
        subtotal,
        discount,
        tip,
        fulfillment,
        ...credit,
      }),
    enabled: Boolean(chefId),
    // No staleTime: the credit block must track the controls, and a cached quote
    // would show credit the server is no longer applying.
    staleTime: 0,
    placeholderData: (prev) => prev, // hold the last good quote while re-fetching
  });
}
