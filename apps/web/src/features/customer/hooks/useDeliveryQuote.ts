import { useQuery } from '@tanstack/react-query';
import { apiClient } from '@/shared/services/api-client';

// Checkout pricing preview — the web mirror of
// `apps/mobile-customer/hooks/useDeliveryQuote.ts`, hitting the same
// POST /chefs/:id/delivery-quote endpoint.
//
// The web checkout used to price itself: `deliveryFee = chef.deliveryFee`, a
// hardcoded `serviceFee = subtotal * 0.05`, and a separate /tax-rates/lookup.
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

export interface DeliveryQuote {
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
  /** Platform (service) fee for the sent subtotal, and the tax rule to apply. */
  serviceFee: number;
  taxRatePercent: number;
  taxName: string;
  taxInclusive: boolean;
  /** GST compliance: 'IN' + intra-state → CGST+SGST; inter-state → IGST. */
  taxCountry: string;
  taxIntraState: boolean;
  /** Server-allocated wallet + loyalty credit for this cart. Absent when the
   *  request was unauthenticated. */
  credit?: CreditQuote;
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
