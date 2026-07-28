import { useEffect, useState } from 'react';
import { useQuery, type UseQueryResult } from '@tanstack/react-query';
import { apiClient } from '@/shared/services/api-client';

// Web mirror of `apps/mobile-customer/hooks/useLocations.ts`, hitting the same
// GET /locations/autocomplete (Mappls primary, Photon fallback).
//
// The web address form had no geocoder at all: it saved label/line1/city/state/
// postcode and nothing else, so every address created on web carried
// latitude = longitude = 0. CreateOrder hard-rejects a delivery order it cannot
// range-check (422 `delivery_location_required`), which meant NO web delivery
// order could ever be placed — the customer only saw "Failed to initiate
// payment". Coordinates are not a nicety here; they are the difference between
// a checkout that works and one that cannot.

/** The flattened suggestion the API returns. Lat/Lon come from the geocoder's
 *  geometry and are what the address must persist for range checks + fee quotes. */
export interface AddressSuggestion {
  description: string;
  line1: string;
  city: string;
  region: string;
  postal: string;
  country: string;
  lat?: number;
  lon?: number;
}

/**
 * What `apiClient` hands back for this endpoint.
 *
 * It is NOT simply the response body. apiClient unwraps envelopes, and which
 * one you get depends on the shape the server sent:
 *
 *   { data, pagination }  → returned whole  (paginated endpoints)
 *   { data }              → returned as `json.data`  ← this endpoint
 *
 * /locations/autocomplete returns `{ "data": [...] }` with no pagination, so
 * apiClient already gives us the ARRAY. Reading `.data` off it — which is what
 * the mobile version does, because axios has no unwrapping and hands back the
 * raw body — yields undefined, and the picker silently reported "No matches"
 * for every address on every lookup.
 *
 * Typed as both shapes and narrowed at runtime so this cannot rot again if the
 * server ever starts (or stops) paginating.
 */
type AutocompleteResult = AddressSuggestion[] | { data?: AddressSuggestion[] };

function toSuggestions(r: AutocompleteResult | null | undefined): AddressSuggestion[] {
  if (Array.isArray(r)) return r;
  return r?.data ?? [];
}

/** Keeps us from firing a request on every keystroke. 250ms is a comfortable
 *  typing pause without feeling laggy — same value the mobile picker uses. */
function useDebounced(value: string, delay = 250): string {
  const trimmed = value.trim();
  const [debounced, setDebounced] = useState(trimmed);
  useEffect(() => {
    const handle = setTimeout(() => setDebounced(trimmed), delay);
    return () => clearTimeout(handle);
  }, [trimmed, delay]);
  return debounced;
}

/**
 * Debounce a free-text query and return street-level address suggestions.
 *
 * The geocoders need a 3-character minimum to return anything useful, so we
 * mirror that threshold rather than sending doomed one-letter queries.
 */
export function useAddressAutocomplete(query: string): UseQueryResult<AddressSuggestion[]> {
  const debounced = useDebounced(query);
  return useQuery<AddressSuggestion[]>({
    queryKey: ['locations', 'autocomplete', debounced],
    queryFn: async () => {
      const r = await apiClient.get<AutocompleteResult>(
        `/locations/autocomplete?q=${encodeURIComponent(debounced)}`
      );
      return toSuggestions(r);
    },
    enabled: debounced.length >= 3,
    staleTime: 60_000,
  });
}

/** Coordinates worth persisting. The geocoder omits them on a miss, and a
 *  literal 0,0 is the null island, never an Indian delivery address. */
export function suggestionCoords(s: AddressSuggestion): { lat: number; lon: number } | null {
  if (typeof s.lat !== 'number' || typeof s.lon !== 'number') return null;
  if (s.lat === 0 && s.lon === 0) return null;
  return { lat: s.lat, lon: s.lon };
}
