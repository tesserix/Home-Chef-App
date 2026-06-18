// useDietaryConflicts (#41) — returns the conflicts between the signed-in
// customer's dietary profile and a dish, using the shared matcher. Reactive +
// cached via useProfile (one fetch shared across every menu card). Returns []
// for anonymous users or an empty profile.

import { useQuery } from '@tanstack/react-query';
import { useProfile } from './useProfile';
import { api } from '../lib/api';
import { findItemConflicts, type DietaryItem, type DietConflict } from '@homechef/mobile-shared/dietary';

export function useDietaryConflicts(item: DietaryItem): DietConflict[] {
  const { data: profile } = useProfile();
  if (!profile) return [];
  return findItemConflicts(
    { dietaryPreferences: profile.dietaryPreferences, foodAllergies: profile.foodAllergies },
    item,
  );
}

export interface DietaryWarning {
  menuItemId: string;
  name: string;
  conflicts: DietConflict[];
}

export interface DietaryCheckResult {
  hasConflicts: boolean;
  warnings: DietaryWarning[];
}

// Server-side conflict check for a set of cart items vs the caller's saved
// profile (#41) — the authoritative source for the at-checkout warning.
export function useDietaryCheck(menuItemIds: string[], enabled = true) {
  return useQuery<DietaryCheckResult>({
    queryKey: ['dietary-check', [...menuItemIds].sort()],
    queryFn: () =>
      api
        .post('/v1/dietary/check', { menuItemIds })
        .then((r) => r.data as DietaryCheckResult),
    enabled: enabled && menuItemIds.length > 0,
    staleTime: 1000 * 60,
  });
}
