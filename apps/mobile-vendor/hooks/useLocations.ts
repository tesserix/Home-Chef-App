// Hooks that wrap the public /api/v1/locations endpoints (no auth required).
//
// HomeChef ships India-only today so the country defaults to "IN" and the
// hooks don't need a `useCountries`-style picker. Add one here when a
// second country is seeded.
//
// React Query handles caching: states + cities are tagged with
// `staleTime: Infinity` because reference data only changes when a new
// seeder ships, not at runtime.

import { useQuery, type UseQueryResult } from '@tanstack/react-query';
import { useEffect, useState } from 'react';
import { api } from '../lib/api';

const COUNTRY_CODE = 'IN';

export interface Country {
  code: string;
  name: string;
  nativeName: string;
  callingCode: string;
  currencyCode: string;
  flagEmoji: string;
  region: string;
}

export interface State {
  id: string; // "IN-MH"
  countryCode: string; // "IN"
  code: string; // "MH"
  name: string;
  type: 'state' | 'territory' | string;
}

export interface City {
  id: string; // "IN-MH-mumbai"
  stateId: string; // "IN-MH"
  name: string;
  isMajor: boolean;
  latitude?: number;
  longitude?: number;
}

export interface Postcode {
  code: string;
  cityId: string;
  areaName: string;
}

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

// useStates lists every state in the configured country (India today).
// Cached for the session — reference data doesn't change at runtime.
export function useStates(): UseQueryResult<State[]> {
  return useQuery<State[]>({
    queryKey: ['locations', 'states', COUNTRY_CODE],
    queryFn: async () => {
      const r = await api.get<Envelope<State[]>>(
        `/locations/countries/${COUNTRY_CODE}/states`,
      );
      return r.data.data;
    },
    staleTime: Infinity,
    gcTime: Infinity,
  });
}

// useCities lists cities for the given state code (e.g. "MH"). Returns
// an empty list while no state is selected so the consumer can render
// the picker unconditionally.
export function useCities(stateCode: string | null): UseQueryResult<City[]> {
  return useQuery<City[]>({
    queryKey: ['locations', 'cities', COUNTRY_CODE, stateCode],
    queryFn: async () => {
      if (!stateCode) return [];
      const r = await api.get<Envelope<City[]>>(
        `/locations/states/${stateCode}/cities?country=${COUNTRY_CODE}`,
      );
      return r.data.data;
    },
    enabled: Boolean(stateCode),
    staleTime: Infinity,
    gcTime: Infinity,
  });
}

// usePostcodeSearch debounces a free-text query and returns matching
// PIN-code results. Backend matches both `code` prefix and `areaName`
// substring, so users can type either "5600" or "Koramang" and get
// the same kind of hit.
//
// The debounce keeps us from firing a request on every keystroke; 250ms
// is a comfortable typing pause without feeling laggy on a fast network.
export function usePostcodeSearch(query: string): UseQueryResult<PostcodeSearchResult[]> {
  const trimmed = query.trim();
  const [debounced, setDebounced] = useState(trimmed);

  useEffect(() => {
    const handle = setTimeout(() => setDebounced(trimmed), 250);
    return () => clearTimeout(handle);
  }, [trimmed]);

  return useQuery<PostcodeSearchResult[]>({
    queryKey: ['locations', 'postcode-search', debounced],
    queryFn: async () => {
      if (debounced.length < 2) return [];
      const r = await api.get<Envelope<PostcodeSearchResult[]>>(
        `/locations/postcodes/search?q=${encodeURIComponent(debounced)}`,
      );
      return r.data.data;
    },
    enabled: debounced.length >= 2,
    staleTime: 60_000, // PIN data is stable; brief cache is fine
  });
}
