import { create } from 'zustand';
import { apiClient } from '@/shared/services/api-client';

export type ToggleResult = 'ok' | 'max_limit' | 'unauthorized' | 'error';

interface FavoritesState {
  /** Set of favorited chef IDs for O(1) lookup */
  chefIds: Set<string>;
  isLoaded: boolean;
  /** Fetch the user's favorite chef IDs from the API */
  load: () => Promise<void>;
  /** Toggle a chef's favorite status. */
  toggle: (chefId: string) => Promise<ToggleResult>;
  isFavorite: (chefId: string) => boolean;
  clear: () => void;
}

const MAX_FAVORITES = 7;

export const useFavoritesStore = create<FavoritesState>((set, get) => ({
  chefIds: new Set(),
  isLoaded: false,

  load: async () => {
    try {
      const res = await apiClient.get<{ chefIds: string[] }>('/favorites/chefs/ids');
      set({ chefIds: new Set(res.chefIds), isLoaded: true });
    } catch {
      // Not authenticated or network error — silently ignore
      set({ isLoaded: true });
    }
  },

  toggle: async (chefId: string): Promise<ToggleResult> => {
    const { chefIds } = get();
    const wasFavorite = chefIds.has(chefId);

    if (wasFavorite) {
      // Optimistic remove
      const next = new Set(chefIds);
      next.delete(chefId);
      set({ chefIds: next });

      try {
        await apiClient.delete(`/favorites/chefs/${chefId}`);
      } catch (err: unknown) {
        // Rollback
        const rollback = new Set(get().chefIds);
        rollback.add(chefId);
        set({ chefIds: rollback });
        const status = (err as { status?: number })?.status;
        if (status === 401 || status === 403) return 'unauthorized';
        return 'error';
      }
      return 'ok';
    } else {
      // Check limit client-side
      if (chefIds.size >= MAX_FAVORITES) {
        return 'max_limit';
      }

      // Optimistic add
      const next = new Set(chefIds);
      next.add(chefId);
      set({ chefIds: next });

      try {
        await apiClient.post('/favorites/chefs', { chefId });
      } catch (err: unknown) {
        // Rollback
        const rollback = new Set(get().chefIds);
        rollback.delete(chefId);
        set({ chefIds: rollback });
        const status = (err as { status?: number })?.status;
        if (status === 401 || status === 403) return 'unauthorized';
        if (status === 409) return 'max_limit';
        return 'error';
      }
      return 'ok';
    }
  },

  isFavorite: (chefId: string) => get().chefIds.has(chefId),

  clear: () => set({ chefIds: new Set(), isLoaded: false }),
}));
