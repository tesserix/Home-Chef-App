import { useQuery } from '@tanstack/react-query';
import { apiClient } from '@/shared/services/api-client';

// Realistic "preferred delivery/pickup time" suggestions for the home-tiffin
// handshake (#709) — the web mirror of
// `apps/mobile-customer/hooks/useFulfillmentTimes.ts`, hitting the same
// GET /chefs/:id/fulfillment-times endpoint.
//
// The server derives these from the CHEF's meal windows + open hours + prep
// headroom (services/fulfillment_times.go), so they read as real meals (lunch /
// dinner) instead of arbitrary half-hours from "now" — and they roll forward
// across days when today's service is over. That matters: a naive "now + 1h"
// proposes 9am for a 6am order, which no home kitchen can serve.

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
export function useFulfillmentTimes(chefId: string | undefined) {
  return useQuery<FulfillmentTimesResponse>({
    queryKey: ['fulfillment-times', chefId],
    queryFn: () => apiClient.get<FulfillmentTimesResponse>(`/chefs/${chefId}/fulfillment-times`),
    enabled: Boolean(chefId),
    staleTime: 5 * 60_000, // meal windows don't move within a checkout session
  });
}

/**
 * Group suggested times by "day · meal" so the picker reads as a few labelled
 * clusters (Today · Lunch, Today · Dinner …) instead of a wall of chips.
 */
export function groupFulfillmentTimes(
  times: FulfillmentTime[]
): { key: string; times: FulfillmentTime[] }[] {
  const groups: { key: string; times: FulfillmentTime[] }[] = [];
  for (const t of times) {
    const key = `${t.day} · ${t.meal}`;
    const existing = groups.find((g) => g.key === key);
    if (existing) existing.times.push(t);
    else groups.push({ key, times: [t] });
  }
  return groups;
}
