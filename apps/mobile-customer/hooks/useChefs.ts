import { useQuery } from '@tanstack/react-query';
import { api } from '../lib/api';
import type { Chef, MenuItem } from '../types/customer';

export interface ChefFilters {
  search?: string;
  cuisine?: string;
  dietary?: string;
  rating?: number;
  isOpen?: boolean;
  // #36 discovery: price-range + near-me geo + distance sort (the API params
  // shipped in #128). Passed straight through as query params by useChefs.
  minPrice?: number;
  maxPrice?: number;
  lat?: number;
  lng?: number;
  sort?: 'rating' | 'orders' | 'newest' | 'price' | 'distance';
  page?: number;
  limit?: number;
}

// ─── API response shapes (Go ChefProfileResponse / MenuItemResponse) ─────────
// The backend speaks businessName / isOnline / acceptingOrders / cuisines[] /
// totalReviews; the customer UI speaks name / isOpen / cuisine / reviewCount.
// These mappers are the single translation point — never let the raw API
// shape reach a screen, or names go blank and open-status reads wrong.

interface ApiChefProfile {
  id: string;
  businessName?: string;
  description?: string;
  profileImage?: string;
  bannerImage?: string;
  kitchenPhotos?: string[];
  cuisines?: string[];
  prepTime?: string;
  minimumOrder?: number;
  deliveryFee?: number;
  rating?: number;
  totalReviews?: number;
  isOnline?: boolean;
  acceptingOrders?: boolean;
  latitude?: number;
  longitude?: number;
  foodSafetyBadge?: boolean;
  offersPickup?: boolean;
}

interface ApiMenuItem {
  id: string;
  chefId: string;
  categoryId?: string;
  name: string;
  description?: string;
  price: number;
  imageUrl?: string;
  images?: { url?: string }[];
  dietaryTags?: string[];
  allergens?: string[];
  isVeg?: boolean | null;
  isAvailable?: boolean;
  rating?: number;
  totalReviews?: number;
  dailyCapacity?: number;
  remainingToday?: number;
  soldOut?: boolean;
  // Add-ons / combos (#52)
  isCombo?: boolean;
  modifierGroups?: import('../types/customer').ModifierGroup[];
  comboItems?: import('../types/customer').ComboItemRef[];
}

function firstNonEmpty(...vals: (string | undefined)[]): string | undefined {
  return vals.find((v) => typeof v === 'string' && v.trim().length > 0);
}

export function mapChef(c: ApiChefProfile): Chef {
  return {
    id: c.id,
    name: c.businessName ?? '',
    cuisine: (c.cuisines ?? []).filter(Boolean).join(' · '),
    rating: c.rating ?? 0,
    reviewCount: c.totalReviews ?? 0,
    // Open to customers only when the kitchen is online AND accepting orders.
    isOpen: Boolean(c.isOnline && c.acceptingOrders),
    // Photo-led card: prefer the wide cover (bannerImage), then a kitchen
    // photo, and only fall back to the avatar (a face crop makes a weak 4:3
    // hero). The avatar still surfaces separately where an identity glyph is
    // wanted.
    imageUrl: firstNonEmpty(c.bannerImage, c.kitchenPhotos?.[0], c.profileImage),
    latitude: c.latitude,
    longitude: c.longitude,
    deliveryTime: c.prepTime,
    minimumOrder: c.minimumOrder,
    deliveryFee: c.deliveryFee,
    foodSafetyBadge: Boolean(c.foodSafetyBadge),
    offersPickup: c.offersPickup,
  };
}

export function mapMenuItem(
  m: ApiMenuItem,
  categoryNames: Map<string, string>,
): MenuItem {
  return {
    id: m.id,
    chefId: m.chefId,
    name: m.name,
    description: m.description,
    price: m.price,
    imageUrl: firstNonEmpty(m.imageUrl, m.images?.[0]?.url),
    category: m.categoryId ? categoryNames.get(m.categoryId) : undefined,
    isAvailable: m.isAvailable ?? true,
    dietaryTags: m.dietaryTags,
    allergens: m.allergens,
    isVeg: m.isVeg,
    isCombo: m.isCombo,
    modifierGroups: m.modifierGroups,
    comboItems: m.comboItems,
    rating: m.rating,
    reviewCount: m.totalReviews,
    dailyCapacity: m.dailyCapacity,
    remainingToday: m.remainingToday,
    soldOut: m.soldOut,
  };
}

// ─── Hooks ────────────────────────────────────────────────────────────────

export function useChefs(filters: ChefFilters = {}) {
  return useQuery<{ data: Chef[] }>({
    queryKey: ['chefs', filters],
    queryFn: async () => {
      const r = await api.get('/v1/chefs', { params: filters });
      const list = (r.data?.data ?? []) as ApiChefProfile[];
      return { data: list.map(mapChef) };
    },
    staleTime: 1000 * 60 * 2, // 2 minutes — chef list changes infrequently
  });
}

export function useChef(id: string) {
  return useQuery<{ data: Chef }>({
    queryKey: ['chef', id],
    queryFn: async () => {
      // GetChef returns the chef object directly (not wrapped in `data`).
      const r = await api.get(`/v1/chefs/${id}`);
      return { data: mapChef(r.data as ApiChefProfile) };
    },
    enabled: !!id,
  });
}

export function useChefMenu(chefId: string) {
  return useQuery<{ data: MenuItem[] }>({
    queryKey: ['chef-menu', chefId],
    queryFn: async () => {
      // Menu endpoint returns { categories: [...], items: [...] }.
      const r = await api.get(`/v1/chefs/${chefId}/menu`);
      const body = r.data as {
        categories?: { id: string; name: string }[];
        items?: ApiMenuItem[];
      };
      const categoryNames = new Map(
        (body.categories ?? []).map((c) => [c.id, c.name] as const),
      );
      return {
        data: (body.items ?? []).map((it) => mapMenuItem(it, categoryNames)),
      };
    },
    enabled: !!chefId,
    staleTime: 1000 * 60 * 5, // 5 minutes — menu changes rarely
  });
}
