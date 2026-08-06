// The home feed's chef query, with one guest-only concession.
//
// Kitchens are hyperlocal, so the feed filters by where the customer is. A guest
// has no delivery address — only device GPS — and if that lands somewhere no
// kitchen serves yet, the geo filter empties the feed and guest browsing shows
// nothing to browse. Rather than an address wall, drop the geography and let
// them see real kitchens; the screen says the results are outside their area.
//
// A signed-in customer with an address keeps the honest empty state: for someone
// who can actually order, kitchens they cannot reach are noise.

import { useChefs, type ChefFilters } from './useChefs';
import { useIsGuest } from './useRequireAccount';
import type { Chef } from '../types/customer';

export interface DiscoveryChefs {
  chefs: Chef[];
  isLoading: boolean;
  isFetching: boolean;
  refetch: () => void;
  /** True when the results shown are from outside the customer's area. */
  showingBeyondArea: boolean;
}

function withoutArea(filters: ChefFilters): ChefFilters {
  const { lat: _lat, lng: _lng, radius: _radius, state: _state, ...rest } = filters;
  return rest;
}

function hasArea(filters: ChefFilters): boolean {
  return filters.lat !== undefined || filters.state !== undefined;
}

export function useDiscoveryChefs(filters: ChefFilters): DiscoveryChefs {
  const isGuest = useIsGuest();

  const located = useChefs(filters);
  const locatedChefs = located.data?.data ?? [];

  const wantFallback =
    isGuest && hasArea(filters) && !located.isLoading && locatedChefs.length === 0;

  const beyond = useChefs(withoutArea(filters), { enabled: wantFallback });
  const beyondChefs = beyond.data?.data ?? [];
  const showingBeyondArea = wantFallback && beyondChefs.length > 0;

  return {
    chefs: showingBeyondArea ? beyondChefs : locatedChefs,
    isLoading: located.isLoading || (wantFallback && beyond.isLoading),
    isFetching: located.isFetching || (wantFallback && beyond.isFetching),
    refetch: () => {
      located.refetch();
      if (wantFallback) beyond.refetch();
    },
    showingBeyondArea,
  };
}
