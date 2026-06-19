import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api } from '../lib/api';
import type { Chef, MenuItem } from '../types/customer';

export interface FavoriteChefEntry {
  id: string;
  chefId: string;
  chef: Chef;
  createdAt: string;
}

export interface FavoritesResponse {
  data: FavoriteChefEntry[];
  count: number;
  max: number;
}

export function useFavorites() {
  return useQuery<FavoritesResponse>({
    queryKey: ['favorites'],
    queryFn: () =>
      api.get('/v1/favorites/chefs').then((r) => r.data as FavoritesResponse),
    staleTime: 1000 * 60 * 2, // 2 minutes
  });
}

export interface ToggleFavoriteParams {
  chefId: string;
  isFavorited: boolean;
}

export function useToggleFavorite() {
  const queryClient = useQueryClient();

  return useMutation<void, Error, ToggleFavoriteParams>({
    mutationFn: ({ chefId, isFavorited }: ToggleFavoriteParams) => {
      if (isFavorited) {
        return api
          .delete(`/v1/favorites/chefs/${chefId}`)
          .then(() => undefined);
      }
      return api
        .post('/v1/favorites/chefs', { chefId })
        .then(() => undefined);
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['favorites'] });
      void queryClient.invalidateQueries({ queryKey: ['chefs'] });
    },
  });
}

// ─── Favorite dishes (#237) ──────────────────────────────────────────────────

export interface FavoriteDishEntry {
  id: string;
  menuItemId: string;
  menuItem: MenuItem;
  chef: { id: string; businessName: string; profileImage?: string };
  createdAt: string;
}

export interface FavoriteDishesResponse {
  data: FavoriteDishEntry[];
  count: number;
  max: number;
}

/** Full favorite-dish entries (with the dish + its chef) for the Saved tab. */
export function useFavoriteDishes() {
  return useQuery<FavoriteDishesResponse>({
    queryKey: ['favorite-dishes'],
    queryFn: () =>
      api.get('/v1/favorites/dishes').then((r) => r.data as FavoriteDishesResponse),
    staleTime: 1000 * 60 * 2,
  });
}

/** Just the saved menu-item IDs, for cheap heart-state lookup on dish cards. */
export function useFavoriteDishIds() {
  return useQuery<Set<string>>({
    queryKey: ['favorite-dish-ids'],
    queryFn: () =>
      api.get('/v1/favorites/dishes/ids').then(
        (r) => new Set(((r.data as { menuItemIds?: string[] }).menuItemIds) ?? []),
      ),
    staleTime: 1000 * 60 * 2,
  });
}

export interface ToggleFavoriteDishParams {
  menuItemId: string;
  isFavorited: boolean;
}

export function useToggleFavoriteDish() {
  const queryClient = useQueryClient();

  return useMutation<void, Error, ToggleFavoriteDishParams>({
    mutationFn: ({ menuItemId, isFavorited }: ToggleFavoriteDishParams) => {
      if (isFavorited) {
        return api.delete(`/v1/favorites/dishes/${menuItemId}`).then(() => undefined);
      }
      return api.post('/v1/favorites/dishes', { menuItemId }).then(() => undefined);
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['favorite-dishes'] });
      void queryClient.invalidateQueries({ queryKey: ['favorite-dish-ids'] });
    },
  });
}
