import { useQuery } from '@tanstack/react-query';
import { apiClient } from '@/shared/services/api-client';

// The chef's à-la-carte menu, as options for the plan editors.
//
// Weekly and daily menus made the chef re-type a dish they had already created
// on the Menu screen — same name, same price, typed again, per day, per slot,
// per variant. Seven days of lunch and dinner is fourteen chances to typo a
// price or spell "Bhindi Masala" two different ways, and nothing linked the
// result back to the item it came from.
//
// Picking instead of typing fixes all of that AND fills in the fields the chef
// would otherwise have to remember: price, portion, serves, dietary tags,
// allergens, image. The API already carries menuItemId on both weekly cells and
// daily entries — it was simply never sent.

export interface MenuItemOption {
  id: string;
  name: string;
  description?: string;
  price: number;
  imageUrl?: string;
  portionSize?: string;
  serves: number;
  dietaryTags?: string[];
  allergens?: string[];
  isVeg?: boolean;
  isCombo?: boolean;
  isAvailable?: boolean;
}

/**
 * Every menu item this chef has. Cached for the session — a chef does not add
 * dishes while filling in a week, and re-fetching per cell would be absurd.
 */
export function useMenuItemOptions() {
  return useQuery<MenuItemOption[]>({
    queryKey: ['chef', 'menu-item-options'],
    queryFn: () =>
      apiClient
        .get<MenuItemOption[] | { data: MenuItemOption[] }>('/chef/menu')
        .then((r) => (Array.isArray(r) ? r : (r?.data ?? []))),
    staleTime: 5 * 60_000,
  });
}

/**
 * Narrow the options to what belongs in a veg or non-veg cell.
 *
 * `isVeg` is optional on the API, so an item that never declared it stays in
 * BOTH lists rather than vanishing — hiding a dish the chef can see on their
 * Menu screen would read as a bug, and the cell's own variant is what the
 * customer is shown either way.
 */
export function optionsForVariant(
  items: MenuItemOption[],
  variant: 'veg' | 'nonveg',
): MenuItemOption[] {
  return items.filter((i) => {
    if (i.isVeg === undefined || i.isVeg === null) return true;
    return variant === 'veg' ? i.isVeg : !i.isVeg;
  });
}

/** The fields a picked item pre-fills on a plan cell. */
export interface PrefillFromMenuItem {
  menuItemId: string;
  name: string;
  description?: string;
  price: number;
  imageUrl?: string;
  portionSize?: string;
  serves: number;
  dietaryTags?: string[];
  allergens?: string[];
}

export function prefillFrom(item: MenuItemOption): PrefillFromMenuItem {
  return {
    menuItemId: item.id,
    name: item.name,
    description: item.description,
    price: item.price,
    imageUrl: item.imageUrl,
    portionSize: item.portionSize,
    // Serves defaults to 1 on the API, so this is always a usable number.
    serves: item.serves || 1,
    dietaryTags: item.dietaryTags,
    allergens: item.allergens,
  };
}
