import { useQuery } from '@tanstack/react-query';
import { api } from '../lib/api';

// Checkout delivery-fee preview (#pickup-incentive). The screen used to show
// "Delivery fee — Free" for every mode, which both hid the real fee and made
// pickup's saving invisible. This asks the server for the fee it WILL charge, so
// checkout shows the real number and the pickup saving — and never displays a
// total different from what CreateOrder bills.

export interface DeliveryQuote {
  deliveryFee: number;
  pickupFee: number;
  /** What the customer keeps by collecting. 0 when delivery is itself free. */
  pickupSaving: number;
  currency: string;
  offersPickup: boolean;
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
  drop: { latitude?: number; longitude?: number; city?: string; country?: string },
) {
  const { latitude, longitude, city, country } = drop;
  return useQuery<DeliveryQuote>({
    // Keyed on the coords so moving the drop address re-quotes the distance fee.
    queryKey: ['delivery-quote', chefId, latitude, longitude, city],
    queryFn: async () =>
      (
        await api.post(`/v1/chefs/${chefId}/delivery-quote`, {
          latitude,
          longitude,
          city,
          country,
        })
      ).data as DeliveryQuote,
    enabled: !!chefId,
    staleTime: 1000 * 60, // a minute — the fee doesn't move within a checkout
  });
}
