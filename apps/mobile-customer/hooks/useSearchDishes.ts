import { useQuery } from '@tanstack/react-query';
import { api } from '../lib/api';

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

export function useSearchDishes(q: string) {
  return useQuery<DishResult[]>({
    queryKey: ['dish-search', q],
    queryFn: async () => {
      const r = await api.get('/v1/search/dishes', { params: { q, page: 1, limit: 30 } });
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
