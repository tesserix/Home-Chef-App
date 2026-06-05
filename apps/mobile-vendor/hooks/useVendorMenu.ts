import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { multipartConfig } from '@homechef/mobile-shared/api';
import { api } from '../lib/api';

export interface MenuItemImage {
  id: string;
  url: string;
}

export interface MenuItem {
  id: string;
  name: string;
  description: string;
  price: number;
  categoryId: string | null;
  isAvailable: boolean;
  // Backend has no `isVeg` field — it stores `dietaryTags: string[]`.
  // We derive `isVeg` from the array on the way in (see `useVendorMenu`
  // below) and write back as a tag on mutations. Keeping the boolean on
  // the public type so UI code stays simple.
  isVeg: boolean;
  dietaryTags: string[];
  images: MenuItemImage[];
  preparationTime: number;
}

// Treat any of these tag strings as "vegetarian". Lowercased + trimmed
// match — defensive against backend casing inconsistencies.
const VEG_TAG_VALUES = new Set(['veg', 'vegetarian', 'pure-veg', 'pure veg']);

function deriveIsVeg(tags: string[] | null | undefined): boolean {
  if (!tags || tags.length === 0) return false;
  return tags.some((t) => VEG_TAG_VALUES.has(t.trim().toLowerCase()));
}

export interface Category {
  id: string;
  name: string;
}

export interface MenuResponse {
  items: MenuItem[];
  categories: Category[];
}

export interface CreateMenuItemPayload {
  name: string;
  description: string;
  price: number;
  categoryId: string;
  // Frontend convenience flag. The mutation translates this into a
  // `dietaryTags: ['vegetarian']` array before hitting the backend.
  isVeg: boolean;
  preparationTime: number;
}

// Translate the frontend `isVeg` boolean to the backend's tag array.
// Returns the canonical lowercase tag string.
function tagsForIsVeg(isVeg: boolean): string[] {
  return isVeg ? ['vegetarian'] : ['non-vegetarian'];
}

export interface UpdateMenuItemPayload extends Partial<CreateMenuItemPayload> {
  isAvailable?: boolean;
}

const MENU_KEY = ['chef', 'menu'] as const;

// Normalize an item from the API: backend returns `dietaryTags` + `prepTime`,
// frontend consumes `dietaryTags` + derived `isVeg` + `preparationTime`.
// The prep-time field-name mismatch was silently swallowing every form
// edit — backend ignored `preparationTime`, sent back `prepTime` which the
// frontend then ignored. Map both directions here.
function normalizeItem(
  item: MenuItem & {
    dietaryTags?: string[] | null;
    prepTime?: number;
    preparationTime?: number;
  },
): MenuItem {
  const tags = item.dietaryTags ?? [];
  const prep = item.preparationTime ?? item.prepTime ?? 15;
  return {
    ...item,
    dietaryTags: tags,
    isVeg: deriveIsVeg(tags),
    preparationTime: prep,
  };
}

export function useVendorMenu() {
  return useQuery<MenuResponse>({
    queryKey: MENU_KEY,
    queryFn: () =>
      api.get<MenuResponse>('/chef/menu').then((r) => ({
        ...r.data,
        items: (r.data.items ?? []).map(normalizeItem),
      })),
    staleTime: 30_000,
  });
}

export function useCreateCategory() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (name: string) =>
      api.post<Category>('/chef/menu/categories', { name }).then((r) => r.data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: MENU_KEY });
    },
  });
}

export function useCreateMenuItem() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (payload: CreateMenuItemPayload) => {
      const { isVeg, preparationTime, ...rest } = payload;
      const body = {
        ...rest,
        dietaryTags: tagsForIsVeg(isVeg),
        prepTime: preparationTime,
      };
      return api
        .post<{ item: MenuItem }>('/chef/menu/items', body)
        .then((r) => r.data);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: MENU_KEY });
    },
  });
}

export function useUpdateMenuItem() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      itemId,
      payload,
    }: {
      itemId: string;
      payload: UpdateMenuItemPayload;
    }) => {
      // Translate the frontend convenience fields to the backend shape:
      //  - `isVeg` → `dietaryTags`
      //  - `preparationTime` → `prepTime` (backend never read the former,
      //    so prep time changes were silently dropped before this fix).
      const { isVeg, preparationTime, ...rest } = payload;
      const body: Record<string, unknown> = { ...rest };
      if (typeof isVeg === 'boolean') {
        body.dietaryTags = tagsForIsVeg(isVeg);
      }
      if (typeof preparationTime === 'number') {
        body.prepTime = preparationTime;
      }
      return api
        .put<{ item: MenuItem }>(`/chef/menu/items/${itemId}`, body)
        .then((r) => r.data);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: MENU_KEY });
    },
  });
}

export function useDeleteMenuItem() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (itemId: string) => api.delete(`/chef/menu/items/${itemId}`),
    onMutate: async (itemId) => {
      await queryClient.cancelQueries({ queryKey: MENU_KEY });
      const previous = queryClient.getQueryData<MenuResponse>(MENU_KEY);
      queryClient.setQueryData<MenuResponse>(MENU_KEY, (old) => {
        if (!old) return old;
        return { ...old, items: old.items.filter((item) => item.id !== itemId) };
      });
      return { previous };
    },
    onError: (_err, _itemId, context) => {
      if (context?.previous) {
        queryClient.setQueryData(MENU_KEY, context.previous);
      }
    },
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: MENU_KEY });
    },
  });
}

export function useUploadMenuPhoto(itemId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (uri: string) => {
      const formData = new FormData();
      const filename = uri.split('/').pop() ?? 'menu-photo.jpg';
      const type = filename.endsWith('.png') ? 'image/png' : 'image/jpeg';
      formData.append('file', { uri, name: filename, type } as unknown as Blob);
      return api.post(`/chef/menu/items/${itemId}/images`, formData, multipartConfig());
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: MENU_KEY });
    },
  });
}

export function useToggleAvailability() {
  const queryClient = useQueryClient();
  return useMutation({
    // Use the dedicated availability endpoint so the backend can apply
    // any stock-management side effects independently of a full item update.
    mutationFn: ({ itemId, isAvailable }: { itemId: string; isAvailable: boolean }) =>
      api.put(`/chef/menu/items/${itemId}/availability`, { isAvailable }),
    onMutate: async ({ itemId, isAvailable }) => {
      await queryClient.cancelQueries({ queryKey: MENU_KEY });
      const previous = queryClient.getQueryData<MenuResponse>(MENU_KEY);
      queryClient.setQueryData<MenuResponse>(MENU_KEY, (old) => {
        if (!old) return old;
        return {
          ...old,
          items: old.items.map((item) =>
            item.id === itemId ? { ...item, isAvailable } : item,
          ),
        };
      });
      return { previous };
    },
    onError: (_err, _vars, context) => {
      // Rollback optimistic update on failure
      if (context?.previous) {
        queryClient.setQueryData(MENU_KEY, context.previous);
      }
    },
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: MENU_KEY });
    },
  });
}
