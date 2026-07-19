import { useQuery } from '@tanstack/react-query';
import { api } from '../lib/api';

// Realistic "preferred delivery/pickup time" suggestions for the home-tiffin
// handshake (#709). The server derives these from the CHEF's meal windows +
// open hours + prep headroom (services/fulfillment_times.go), so they read as
// real meals (lunch / dinner) instead of arbitrary half-hours from "now" — and
// they roll forward across days when today's service is over.

export interface FulfillmentTime {
  /** Exact instant (ISO 8601, +05:30) the customer proposes. */
  at: string;
  /** Clock label already formatted in IST, e.g. "1:00 pm". */
  label: string;
  /** "Today" / "Tomorrow" / short weekday. */
  day: string;
  /** "Breakfast" / "Lunch" / "Snacks" / "Dinner". */
  meal: string;
}

interface FulfillmentTimesResponse {
  times: FulfillmentTime[];
}

/**
 * Chef-aware suggested delivery/pickup times. Same suggestions for both modes —
 * the food is ready when the chef cooks it; delivery vs pickup only changes who
 * carries it. Disabled until a chef id is known.
 */
export function useFulfillmentTimes(chefId?: string) {
  return useQuery<FulfillmentTimesResponse>({
    queryKey: ['fulfillment-times', chefId],
    queryFn: async () =>
      (await api.get(`/v1/chefs/${chefId}/fulfillment-times`)).data as FulfillmentTimesResponse,
    enabled: !!chefId,
    staleTime: 1000 * 60 * 5, // meal windows don't move within a checkout session
  });
}
