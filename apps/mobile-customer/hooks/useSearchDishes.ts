import { useQuery } from '@tanstack/react-query';
import { api } from '../lib/api';
import type { Coords } from './useCustomerCoords';

// Mirrors GET /v1/search/dishes (#36/#143): menu items across active,
// non-FSSAI-locked chefs, matched by name/description.
export interface DishResult {
  id: string;
  chefId: string;
  name: string;
  description?: string;
  price: number;
  rating?: number;
  imageUrl?: string;
}

interface ApiDish {
  id: string;
  chefId?: string;
  name?: string;
  description?: string;
  price?: number;
  rating?: number;
  imageUrl?: string;
  images?: { url?: string }[];
}

export function dishSearchParams(q: string, coords: Coords | null) {
  return { q, page: 1, limit: 30, ...(coords ? { lat: coords.lat, lng: coords.lng } : {}) };
}

export function useSearchDishes(q: string, coords: Coords | null) {
  return useQuery<DishResult[]>({
    queryKey: ['dish-search', q, coords?.lat, coords?.lng],
    queryFn: async () => {
      const r = await api.get('/v1/search/dishes', { params: dishSearchParams(q, coords) });
      const list = (r.data?.data ?? []) as ApiDish[];
      return list.map((d) => ({
        id: d.id,
        chefId: d.chefId ?? '',
        name: d.name ?? '',
        description: d.description,
        price: d.price ?? 0,
        rating: d.rating,
        imageUrl: d.imageUrl ?? d.images?.[0]?.url,
      }));
    },
    // The API requires at least 2 chars; don't fire below that.
    enabled: q.trim().length >= 2,
  });
}
