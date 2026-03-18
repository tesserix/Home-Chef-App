import { useState, useEffect } from 'react';
import { apiClient } from '@/shared/services/api-client';
import {
  FALLBACK_OPTIONS,
  type PreferenceOption,
} from '@/shared/constants/preferences';

interface APIPreferenceOption {
  value: string;
  label: string;
  description?: string;
}

type PreferenceMap = Record<string, PreferenceOption[]>;

let cachedOptions: PreferenceMap | null = null;
let fetchPromise: Promise<PreferenceMap> | null = null;

function fetchPreferences(): Promise<PreferenceMap> {
  if (cachedOptions) return Promise.resolve(cachedOptions);
  if (fetchPromise) return fetchPromise;

  fetchPromise = apiClient
    .get<Record<string, APIPreferenceOption[]>>('/preferences')
    .then((grouped) => {
      const result: PreferenceMap = {};
      for (const [category, items] of Object.entries(grouped)) {
        result[category] = items.map((item) => ({
          value: item.value,
          label: item.label,
          description: item.description,
        }));
      }
      cachedOptions = result;
      return result;
    })
    .catch(() => {
      // Fall back to hardcoded options
      cachedOptions = FALLBACK_OPTIONS;
      return FALLBACK_OPTIONS;
    })
    .finally(() => {
      fetchPromise = null;
    });

  return fetchPromise;
}

export function usePreferences() {
  const [options, setOptions] = useState<PreferenceMap>(cachedOptions ?? FALLBACK_OPTIONS);
  const [isLoading, setIsLoading] = useState(!cachedOptions);

  useEffect(() => {
    if (cachedOptions) {
      setOptions(cachedOptions);
      setIsLoading(false);
      return;
    }
    fetchPreferences().then((result) => {
      setOptions(result);
      setIsLoading(false);
    });
  }, []);

  return {
    dietary: options.dietary ?? FALLBACK_OPTIONS.dietary ?? [],
    allergy: options.allergy ?? FALLBACK_OPTIONS.allergy ?? [],
    cuisine: options.cuisine ?? FALLBACK_OPTIONS.cuisine ?? [],
    spiceLevel: options.spice_level ?? FALLBACK_OPTIONS.spice_level ?? [],
    householdSize: options.household_size ?? FALLBACK_OPTIONS.household_size ?? [],
    isLoading,
  } as {
    dietary: PreferenceOption[];
    allergy: PreferenceOption[];
    cuisine: PreferenceOption[];
    spiceLevel: PreferenceOption[];
    householdSize: PreferenceOption[];
    isLoading: boolean;
  };
}
