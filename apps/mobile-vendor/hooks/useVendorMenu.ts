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
  // Declared allergens (#41) — surfaced as customer badges + checkout warnings.
  allergens: string[];
  // Weekly-menu schedule: weekdays (0=Sun..6=Sat) the dish is offered. Empty =
  // every day. Drives which dishes auto-show to customers each day.
  availableDays: number[];
  images: MenuItemImage[];
  preparationTime: number;
  // HSN/SAC code for GST classification. Surfaces the backend's value
  // (defaults to 996331 — services provided by restaurants); chefs
  // override per item when their tax advisor wants a more specific
  // code. Printed on customer invoices.
  hsn?: string;
  // Capacity & cutoff controls (#48). dailyCapacity null/absent = unlimited;
  // remainingToday/soldOut reflect today's IST cap usage (derived server-side).
  dailyCapacity?: number | null;
  remainingToday?: number | null;
  soldOut?: boolean;
  // Add-ons / combos (#52).
  isCombo?: boolean;
  modifierGroups?: MenuItemModifierGroup[];
  comboItems?: MenuItemComboItem[];
}

// Read shapes for an item's modifier groups + combo components (#52).
export interface MenuItemModifierOption {
  id: string;
  name: string;
  priceDelta: number;
  isAvailable: boolean;
}
export interface MenuItemModifierGroup {
  id: string;
  name: string;
  required: boolean;
  minSelect: number;
  maxSelect: number;
  options: MenuItemModifierOption[];
}
export interface MenuItemComboItem {
  menuItemId: string;
  name: string;
  quantity: number;
}

// Write (replace-all) input shapes sent with a save (#52).
export interface ModifierGroupInput {
  name: string;
  required: boolean;
  minSelect: number;
  maxSelect: number;
  options: { name: string; priceDelta: number; isAvailable?: boolean }[];
}
export interface ComboItemInput {
  menuItemId: string;
  quantity: number;
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
  // Extra diet tags beyond veg/non-veg (e.g. jain, gluten-free) — merged with
  // the veg tag on save (#41).
  dietaryTags?: string[];
  // Declared allergens (#41).
  allergens?: string[];
  preparationTime: number;
  // Optional HSN — empty string lets the DB default (996331) apply.
  hsn?: string;
  // Weekly-menu schedule (0=Sun..6=Sat). Empty/omitted = every day.
  availableDays?: number[];
  // Add-ons / combos (#52) — replace-all on save, passed straight to the API.
  isCombo?: boolean;
  modifierGroups?: ModifierGroupInput[];
  comboItems?: ComboItemInput[];
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
// frontend consumes `dietaryTags` + `isVeg` + `preparationTime`.
//
// The prep-time field-name mismatch was silently swallowing every form
// edit — backend ignored `preparationTime`, sent back `prepTime` which the
// frontend then ignored. Map both directions here.
//
// The `is_veg` column was added to `menu_items` in the chef-app v2 work.
// Prefer the authoritative backend boolean when present; fall back to
// deriving from dietary tags only for legacy items that predate the
// column (those rows have NULL on the wire, omitted by `omitempty`).
function normalizeItem(
  item: MenuItem & {
    dietaryTags?: string[] | null;
    allergens?: string[] | null;
    availableDays?: number[] | null;
    prepTime?: number;
    preparationTime?: number;
    isVeg?: boolean | null;
  },
): MenuItem {
  const tags = item.dietaryTags ?? [];
  const prep = item.preparationTime ?? item.prepTime ?? 15;
  const isVeg =
    item.isVeg === true || item.isVeg === false
      ? item.isVeg
      : deriveIsVeg(tags);
  return {
    ...item,
    dietaryTags: tags,
    allergens: item.allergens ?? [],
    availableDays: item.availableDays ?? [],
    isVeg,
    preparationTime: prep,
    hsn: (item as { hsn?: string }).hsn ?? '',
  };
}

// VEG_FLAG_TAGS are the tokens tagsForIsVeg owns; extra diet tags exclude them
// so the veg flag and the extra tags don't fight on save (#41).
const VEG_FLAG_TAGS = new Set(['vegetarian', 'non-vegetarian', 'veg', 'non-veg', 'nonveg']);

/** The diet tags a chef edits directly (everything except the veg-flag tokens). */
export function extraDietTags(tags: string[] | null | undefined): string[] {
  return (tags ?? []).filter((t) => !VEG_FLAG_TAGS.has(t.trim().toLowerCase()));
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
      const { isVeg, preparationTime, dietaryTags: extraDiet, allergens, ...rest } = payload;
      const body = {
        ...rest,
        // Send BOTH: dietaryTags drives the vendor-app diet icon, while the
        // backend's nullable `isVeg` column is what the customer storefront
        // and order detail read. The veg-flag tag is merged with any extra
        // diet tags the chef picked (#41).
        dietaryTags: [...tagsForIsVeg(isVeg), ...(extraDiet ?? [])],
        allergens: allergens ?? [],
        isVeg,
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
      //  - `isVeg` → `dietaryTags` (vendor-app icon) AND `isVeg` (storefront
      //    reads the backend's nullable column — must be sent explicitly or
      //    edits leave it stale).
      //  - `preparationTime` → `prepTime` (backend never read the former,
      //    so prep time changes were silently dropped before this fix).
      const { isVeg, preparationTime, hsn, dietaryTags: extraDiet, allergens, ...rest } = payload;
      const body: Record<string, unknown> = { ...rest };
      if (typeof isVeg === 'boolean') {
        body.dietaryTags = [...tagsForIsVeg(isVeg), ...(extraDiet ?? [])];
        body.isVeg = isVeg;
      } else if (extraDiet) {
        body.dietaryTags = extraDiet;
      }
      if (allergens) {
        body.allergens = allergens;
      }
      if (typeof preparationTime === 'number') {
        body.prepTime = preparationTime;
      }
      if (typeof hsn === 'string') {
        body.hsn = hsn;
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

/**
 * Builds the photo-upload path, refusing a missing id.
 *
 * An empty id collapses the URL to `/chef/menu/items//images`, which is a
 * different route: it 404s, and because the caller swallows upload errors the
 * chef sees a saved item with no photos and no explanation. Failing loudly
 * here turns a silent data-loss bug into an obvious one.
 */
export function menuItemImagesPath(itemId: string): string {
  if (!itemId || !itemId.trim()) {
    throw new Error('menuItemImagesPath: item id is required to upload a photo');
  }
  return `/chef/menu/items/${itemId}/images`;
}

/**
 * Uploads one menu photo. The item id is supplied per call rather than
 * captured at render: a newly created item's id only exists after the create
 * mutation resolves, and setState does not update an already-constructed
 * mutation within the same handler.
 */
export function useUploadMenuPhoto() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ itemId, uri }: { itemId: string; uri: string }) => {
      const formData = new FormData();
      const filename = uri.split('/').pop() ?? 'menu-photo.jpg';
      const type = filename.endsWith('.png') ? 'image/png' : 'image/jpeg';
      formData.append('file', { uri, name: filename, type } as unknown as Blob);
      return api.post(menuItemImagesPath(itemId), formData, multipartConfig());
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
