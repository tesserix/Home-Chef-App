// Bakery browse (#1065) — the customer's cake/bread section. The vocabulary
// (product types, occasions, diets) comes from the server so the filter chips
// can never drift from what bakers can actually set.

import { useQuery } from '@tanstack/react-query';
import { api } from '../lib/api';
import type { MenuItem } from '../types/customer';

export interface BakeryVocabItem {
  value: string;
  label: string;
}

export interface BakeryOptionsResponse {
  productTypes: BakeryVocabItem[];
  optionKinds: BakeryVocabItem[];
  occasions: BakeryVocabItem[];
  diets: BakeryVocabItem[];
  allergens: BakeryVocabItem[];
}

export interface BakeryProduct extends MenuItem {
  chefName?: string;
  chefCity?: string;
  chefRating?: number;
}

export interface BakeryProductsResponse {
  products: BakeryProduct[];
  total: number;
  limit: number;
  offset: number;
}

export interface BakeryFilters {
  productType?: string;
  occasion?: string;
  dietary?: string;
  maxPrice?: number;
  city?: string;
  chefId?: string;
}

export function useBakeryOptions() {
  return useQuery<BakeryOptionsResponse>({
    queryKey: ['bakery', 'options'],
    queryFn: async () => (await api.get('/v1/bakery/options')).data,
    staleTime: 60 * 60_000,
  });
}

export function useBakeryProducts(filters: BakeryFilters = {}) {
  return useQuery<BakeryProductsResponse>({
    queryKey: ['bakery', 'products', filters],
    queryFn: async () => {
      const params: Record<string, string> = {};
      if (filters.productType) params.productType = filters.productType;
      if (filters.occasion) params.occasion = filters.occasion;
      if (filters.dietary) params.dietary = filters.dietary;
      if (filters.maxPrice) params.maxPrice = String(filters.maxPrice);
      if (filters.city) params.city = filters.city;
      if (filters.chefId) params.chefId = filters.chefId;
      const r = await api.get('/v1/bakery/products', { params });
      return r.data as BakeryProductsResponse;
    },
    staleTime: 60_000,
  });
}
