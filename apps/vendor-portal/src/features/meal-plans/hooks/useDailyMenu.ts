import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { apiClient } from '@/shared/services/api-client';

// Per-date tiffin menu (#405/#406) — the web twin of
// apps/mobile-vendor/app/meal-plans/daily-menu.tsx.
//
// Distinct from the WEEKLY menu: the weekly one is a repeating template, this is
// what is actually being served on a specific date, including priced combos and
// thalis. Web had no way to set it, so a chef who published per-date menus from
// the phone could not see or change them on the portal.
//
// Backed by GET /chef/daily-menu?from&to and PUT /chef/daily-menu/:date.

export type MealSlot = 'lunch' | 'dinner';
export type MealVariant = 'veg' | 'nonveg';

export interface DailyMenuItem {
  id?: string;
  slot: MealSlot;
  variant: MealVariant;
  name: string;
  description?: string;
  price: number;
  imageUrl?: string;
  dietaryTags?: string[];
  allergens?: string[];
  menuItemId?: string | null;
  /** A priced bundle (thali) rather than a single dish. */
  isCombo?: boolean;
  comboComponents?: string[];
}

export interface DailyMenuDay {
  /** YYYY-MM-DD, IST. */
  date: string;
  isPublished: boolean;
  publishedAt?: string;
  items: DailyMenuItem[];
}

/**
 * The chef's own per-date menus, drafts included.
 *
 * Defaults to today..+14d when the range is omitted, matching the customer
 * booking horizon — asking for more would show the chef dates nobody can book.
 */
export function useDailyMenu(from?: string, to?: string) {
  return useQuery<DailyMenuDay[]>({
    queryKey: ['chef', 'daily-menu', from, to],
    queryFn: () =>
      apiClient
        .get<{ days: DailyMenuDay[] }>('/chef/daily-menu', {
          ...(from ? { from } : {}),
          ...(to ? { to } : {}),
        })
        .then((r) => r?.days ?? []),
    staleTime: 30_000,
  });
}

interface SaveDayInput {
  date: string;
  isPublished: boolean;
  items: DailyMenuItem[];
}

/**
 * Replace one date's dishes and its publish state.
 *
 * Replace-all, matching PutWeeklyMenu: the payload is the complete set for that
 * date, so removing a dish means sending the list without it. A partial update
 * would leave orphans the editor never showed.
 */
export function useSaveDailyMenu() {
  const qc = useQueryClient();
  return useMutation<unknown, unknown, SaveDayInput>({
    mutationFn: ({ date, isPublished, items }) =>
      apiClient.put(`/chef/daily-menu/${date}`, { isPublished, items }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['chef', 'daily-menu'] });
    },
  });
}
