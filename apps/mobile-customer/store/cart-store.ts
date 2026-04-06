// Zustand cart store for the customer app.
// Cart is pure client state — no API calls until checkout.
// All mutations use immutable spread (never push/splice in place).
// Reference: RESEARCH.md Pattern 3 + CLAUDE.md immutability rule.

import { create } from 'zustand';
import type { CartItem } from '../types/customer';

interface ChefSummary {
  id: string;
  name: string;
}

type AddItemResult = 'ok' | 'cross_chef_conflict';

interface CartState {
  chefId: string | null;
  chefName: string | null;
  items: CartItem[];

  /**
   * Add an item to the cart.
   * Returns 'cross_chef_conflict' if the item belongs to a different chef than
   * the current cart — caller must prompt user and call clearCart() before retrying.
   * Returns 'ok' on success.
   */
  addItem: (item: CartItem, chef: ChefSummary) => AddItemResult;

  /** Remove an item completely from the cart. */
  removeItem: (menuItemId: string) => void;

  /** Set absolute quantity for an item. If qty <= 0 the item is removed. */
  updateQty: (menuItemId: string, quantity: number) => void;

  /** Clear all cart items and reset chef context. */
  clearCart: () => void;

  /** Derived: sum of price * quantity for all items. */
  total: () => number;

  /** Derived: total item count across all entries. */
  totalCount: () => number;
}

export const useCartStore = create<CartState>((set, get) => ({
  chefId: null,
  chefName: null,
  items: [],

  addItem: (item: CartItem, chef: ChefSummary): AddItemResult => {
    const { chefId, items } = get();

    // Cross-chef conflict: caller must confirm clear before re-adding
    if (chefId !== null && chefId !== chef.id) {
      return 'cross_chef_conflict';
    }

    const existing = items.find((i) => i.menuItemId === item.menuItemId);

    if (existing) {
      // Increment quantity immutably
      set({
        items: items.map((i) =>
          i.menuItemId === item.menuItemId
            ? { ...i, quantity: i.quantity + item.quantity }
            : i
        ),
      });
    } else {
      // Append immutably
      set({
        chefId: chef.id,
        chefName: chef.name,
        items: [...items, { ...item }],
      });
    }

    return 'ok';
  },

  removeItem: (menuItemId: string) => {
    set({ items: get().items.filter((i) => i.menuItemId !== menuItemId) });
  },

  updateQty: (menuItemId: string, quantity: number) => {
    if (quantity <= 0) {
      get().removeItem(menuItemId);
      return;
    }
    set({
      items: get().items.map((i) =>
        i.menuItemId === menuItemId ? { ...i, quantity } : i
      ),
    });
  },

  clearCart: () => {
    set({ chefId: null, chefName: null, items: [] });
  },

  total: () => {
    return get().items.reduce((sum, i) => sum + i.price * i.quantity, 0);
  },

  totalCount: () => {
    return get().items.reduce((sum, i) => sum + i.quantity, 0);
  },
}));
