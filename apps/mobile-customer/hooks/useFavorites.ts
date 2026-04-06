import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api } from '../lib/api';
import type { Chef } from '../types/customer';

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
