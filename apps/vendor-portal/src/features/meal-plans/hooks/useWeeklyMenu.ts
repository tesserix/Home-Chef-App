import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { apiClient } from '@/shared/services/api-client';

// Weekly-menu editor hooks (#1) — the fixed dishes a chef offers per
// day × slot × veg/nonveg. Mirrors the mobile-vendor editor; replace-all save.

export type MealSlot = 'lunch' | 'dinner';
export type MealVariant = 'veg' | 'nonveg';

export interface WeeklyMenuItem {
  id?: string;
  dayOfWeek: number; // 0=Sun .. 6=Sat
  slot: MealSlot;
  variant: MealVariant;
  name: string;
  description?: string;
  price: number;
  imageUrl?: string;
}

export interface WeeklyMenu {
  isPublished: boolean;
  publishedAt?: string | null;
  items: WeeklyMenuItem[];
}

const DAY_SHORT = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'];

/**
 * Mirrors the API's validatePublishableGrid (#1): a publishable week must have
 * every offered (day × slot) filled — no holes. Returns a human-readable error
 * for the first missing cell, or null when the grid is complete. Used to block
 * publish client-side before the round-trip.
 */
export function weeklyMenuHole(items: WeeklyMenuItem[]): string | null {
  if (items.length === 0) return 'Add at least one dish before publishing.';
  const days = new Set<number>();
  const slots = new Set<MealSlot>();
  const present = new Set<string>();
  for (const it of items) {
    days.add(it.dayOfWeek);
    slots.add(it.slot);
    present.add(`${it.dayOfWeek}|${it.slot}`);
  }
  for (let day = 0; day < 7; day++) {
    if (!days.has(day)) continue;
    for (const slot of ['lunch', 'dinner'] as MealSlot[]) {
      if (!slots.has(slot)) continue;
      if (!present.has(`${day}|${slot}`)) {
        return `${DAY_SHORT[day]} is missing its ${slot} dish — every offered day needs a ${slot} dish.`;
      }
    }
  }
  return null;
}

export function useWeeklyMenu() {
  return useQuery<WeeklyMenu>({
    queryKey: ['chef', 'weekly-menu'],
    queryFn: () => apiClient.get<WeeklyMenu>('/chef/weekly-menu'),
  });
}

export function useSaveWeeklyMenu() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (body: { isPublished: boolean; items: WeeklyMenuItem[] }) =>
      apiClient.put<WeeklyMenu>('/chef/weekly-menu', body),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['chef', 'weekly-menu'] });
    },
  });
}
