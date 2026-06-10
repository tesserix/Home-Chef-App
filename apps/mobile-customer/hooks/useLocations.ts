// Hooks wrapping the public /api/v1/locations endpoints (no auth required).
//
// NOTE on paths: the customer api client's baseURL is `…/api`, and every
// customer hook supplies the `/v1` segment itself (e.g. useChefs calls
// `/v1/chefs`). So these calls use `/v1/locations/…` — do NOT drop the
// `/v1` (the vendor app's copy omits it because its baseURL already
// includes `/v1`).

import { useQuery, type UseQueryResult } from '@tanstack/react-query';
import { useEffect, useState } from 'react';
import { api } from '../lib/api';

// AddressSuggestion is the flattened shape returned by
// /v1/locations/autocomplete, which proxies the Photon (OpenStreetMap)
// geocoder for worldwide, India-filtered street-level address search.
export interface AddressSuggestion {
  description: string;
  line1: string;
  city: string;
  region: string;
  postal: string;
  country: string;
}

// PostcodeSearchResult comes from /v1/locations/postcodes/search — the
// seeded PIN registry. Matches `code` prefix or `areaName` substring.
export interface PostcodeSearchResult {
  code: string;
  areaName: string;
  cityId: string;
  cityName: string;
  stateId: string;
  stateName: string;
}

interface Envelope<T> {
  data: T;
}

// Shared debounce — keeps us from firing a request on every keystroke.
// 250ms is a comfortable typing pause without feeling laggy.
function useDebounced(value: string, delay = 250): string {
  const trimmed = value.trim();
  const [debounced, setDebounced] = useState(trimmed);
  useEffect(() => {
    const handle = setTimeout(() => setDebounced(trimmed), delay);
    return () => clearTimeout(handle);
  }, [trimmed, delay]);
  return debounced;
}

// useAddressAutocomplete debounces a free-text query and returns
// street-level address suggestions from the Photon proxy. Photon needs a
// 3-char minimum to return useful results, so we mirror that threshold.
export function useAddressAutocomplete(
  query: string,
): UseQueryResult<AddressSuggestion[]> {
  const debounced = useDebounced(query);
  return useQuery<AddressSuggestion[]>({
    queryKey: ['locations', 'autocomplete', debounced],
    queryFn: async () => {
      if (debounced.length < 3) return [];
      const r = await api.get<Envelope<AddressSuggestion[]>>(
        `/v1/locations/autocomplete?q=${encodeURIComponent(debounced)}`,
      );
      return r.data.data;
    },
    enabled: debounced.length >= 3,
    staleTime: 60_000,
  });
}

// usePostcodeSearch debounces a free-text query and returns matching
// PIN-code results from the seeded registry. Users can type either a code
// ("5600") or an area name ("Koramang") and get the same kind of hit.
export function usePostcodeSearch(
  query: string,
): UseQueryResult<PostcodeSearchResult[]> {
  const debounced = useDebounced(query);
  return useQuery<PostcodeSearchResult[]>({
    queryKey: ['locations', 'postcode-search', debounced],
    queryFn: async () => {
      if (debounced.length < 2) return [];
      const r = await api.get<Envelope<PostcodeSearchResult[]>>(
        `/v1/locations/postcodes/search?q=${encodeURIComponent(debounced)}`,
      );
      return r.data.data;
    },
    enabled: debounced.length >= 2,
    staleTime: 60_000,
  });
}
