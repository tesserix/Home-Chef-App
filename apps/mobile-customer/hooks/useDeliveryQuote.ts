import { useQuery } from '@tanstack/react-query';
import { api } from '../lib/api';
import type { TaxLine } from '../types/customer';

// Checkout delivery-fee preview (#pickup-incentive). The screen used to show
// "Delivery fee — Free" for every mode, which both hid the real fee and made
// pickup's saving invisible. This asks the server for the fee it WILL charge, so
// checkout shows the real number and the pickup saving — and never displays a
// total different from what CreateOrder bills.

/**
 * Itemised, capped self-delivery estimate (#702). Present only when the chef
 * self-delivers. `fee` is the approx MAX the chef can charge — at accept the chef
 * can only bring it down (#703), never above it.
 */
export interface SelfDeliveryBreakdown {
  baseFee: number;
  distanceKnown: boolean;
  distanceKm: number;
  freeRadiusKm: number;
  billableKm: number;
  perKm: number;
  distanceComponent: number;
  /** Drop is inside the chef's free radius — no distance charge. */
  withinFreeZone: boolean;
  maxFee: number;
  capped: boolean;
  /** Fuel surge folded into the distance component (≥1). 1 = no surge. */
  fuelSurge: number;
  /** Weather surge at the drop location (≥1). 1 = clear / no provider. */
  weatherSurge: number;
  /** Traffic surge at the drop location (≥1). 1 = free-flowing / no provider. */
  trafficSurge: number;
  /** Combined surge multiplier applied to the estimate (≥1). */
  surgeMultiplier: number;
  fee: number;
}

export interface DeliveryQuote {
  deliveryFee: number;
  pickupFee: number;
  /** What the customer keeps by collecting. 0 when delivery is itself free. */
  pickupSaving: number;
  currency: string;
  offersPickup: boolean;
  offersSelfDelivery: boolean;
  /** Approx-max self-delivery fee (₹). Present only when the chef self-delivers. */
  selfDeliveryFee?: number;
  selfDeliveryBreakdown?: SelfDeliveryBreakdown;
  /** Serviceability for the drop coords (#709): whether this address is within the
   *  kitchen's delivery range, the straight-line distance, and the range cap. */
  deliverable: boolean;
  distanceKm: number;
  maxRadiusKm: number;
  /** false when coords were missing so the distance couldn't be measured. */
  rangeKnown: boolean;
  /** The priced breakdown for the cart and fulfilment mode sent, from the SAME
   *  function CreateOrder charges with (models/pricing.go). Render these; do not
   *  recompute. Checkout doing its own arithmetic is how it came to show ₹264.58
   *  for an order whose receipt said ₹264.57. */
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
   *  request was unauthenticated. Every figure the credits card renders comes
   *  from here — the screen does no money arithmetic of its own, which is what
   *  keeps what the customer sees and what they are charged in agreement. */
  credit?: CreditQuote;
  /** The gateway that will actually take this payment, as RESOLVED by the server
   *  (`SelectCheckoutGateway`) — not the chef's stored column, which the selection
   *  may override. The RBI Payment Aggregator disclosure on checkout must name the
   *  aggregator that processes the charge, so it has to come from the same
   *  resolution the charge itself uses. The screen hardcoded "Razorpay" while
   *  Cashfree took the money seconds later in the same flow (#933). */
  paymentProvider?: string;
}

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
  /** Upper bound for the wallet slider — order-sized, not the raw balance. */
  walletMax: number;
  pointsBalance: number;
  pointsApplied: number;
  /** Rupee value of pointsApplied. */
  pointsValue: number;
  /** Upper bound for the points slider. */
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

/** The customer's credit intent. Amounts are omitted while a rail is on "auto",
 *  which asks the server to apply as much as its ceilings allow. */
export interface CreditIntent {
  useWallet: boolean;
  walletAmount?: number;
  useLoyalty: boolean;
  loyaltyPoints?: number;
}

/**
 * Quote the per-mode delivery fee for a chef + drop coordinates.
 *
 * Coords are optional — without them the server returns the flat policy fee,
 * exactly as CreateOrder falls back, so the preview never blocks on a location.
 * Disabled until a chef id is known.
 */
export function useDeliveryQuote(
  chefId: string | undefined,
  drop: {
    latitude?: number;
    longitude?: number;
    city?: string;
    country?: string;
    state?: string;
    subtotal?: number;
    /** Applied promo, so the credit ceiling is computed on the discounted food. */
    discount?: number;
    /** 'pickup' zeroes the delivery fee in the credit ceiling, as the order will. */
    fulfillment?: string;
    /** Tip to the chef. Rides in the payable but never in the redeemable base —
     *  sending it keeps the previewed "to pay" equal to what the gateway charges. */
    tip?: number;
    /** Which credit rails to apply, and optionally how much of each. */
    credit?: CreditIntent;
  },
) {
  const { latitude, longitude, city, country, state, subtotal, discount, fulfillment, tip, credit } = drop;
  return useQuery<DeliveryQuote>({
    // Keyed on everything that moves the fee, the tax OR the credit allocation —
    // a stale credit block would put the screen back in the business of guessing.
    queryKey: [
      'delivery-quote', chefId, latitude, longitude, city, state, subtotal,
      discount, fulfillment, tip, credit?.useWallet, credit?.walletAmount,
      credit?.useLoyalty, credit?.loyaltyPoints,
    ],
    queryFn: async () =>
      (
        await api.post(`/v1/chefs/${chefId}/delivery-quote`, {
          latitude,
          longitude,
          city,
          country,
          state,
          subtotal,
          discount,
          fulfillment,
          tip,
          ...credit,
        })
      ).data as DeliveryQuote,
    enabled: !!chefId,
    // No staleTime: the credit block must track the sliders, and a cached quote
    // would show credit the server is no longer applying.
    staleTime: 0,
    placeholderData: (prev) => prev, // hold the last good quote while re-fetching
  });
}
