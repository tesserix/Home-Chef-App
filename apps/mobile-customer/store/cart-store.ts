// Zustand cart store for the customer app.
// Cart is pure client state — no API calls until checkout.
// All mutations use immutable spread (never push/splice in place).
// Reference: RESEARCH.md Pattern 3 + CLAUDE.md immutability rule.
//
// Lines are keyed by `lineId` (#232): the same dish with different add-on
// selections is a distinct line. For an item with no modifiers, lineId equals
// menuItemId, so the no-modifier flow is unchanged.

import { create } from 'zustand';
import type { CartItem, SelectedModifier } from '../types/customer';

interface ChefSummary {
  id: string;
  name: string;
}

type AddItemResult = 'ok' | 'cross_chef_conflict';

/** Stable line id for a menu item + its modifier selection. */
export function makeLineId(menuItemId: string, modifiers?: SelectedModifier[]): string {
  if (!modifiers || modifiers.length === 0) return menuItemId;
  const ids = modifiers.map((m) => m.optionId).sort().join(',');
  return `${menuItemId}::${ids}`;
}

interface CartState {
  chefId: string | null;
  chefName: string | null;
  items: CartItem[];

  /**
   * Add a line to the cart. The item's lineId is honored (or derived from its
   * menuItemId + modifiers). An identical line increments its quantity.
   * Returns 'cross_chef_conflict' if the item belongs to a different chef than
   * the current cart — caller must prompt + clearCart() before retrying.
   */
  addItem: (item: CartItem, chef: ChefSummary) => AddItemResult;

  /** Remove a line completely from the cart. */
  removeItem: (lineId: string) => void;

  /** Set absolute quantity for a line. If qty <= 0 the line is removed. */
  updateQty: (lineId: string, quantity: number) => void;

  /** Set per-line special instructions. Empty string clears it. */
  setInstructions: (lineId: string, instructions: string) => void;

  /** Clear all cart items and reset chef context. */
  clearCart: () => void;

  /** Derived: sum of price * quantity for all lines. */
  total: () => number;

  /** Derived: total item count across all lines. */
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

    const lineId = item.lineId || makeLineId(item.menuItemId, item.modifiers);
    const existing = items.find((i) => i.lineId === lineId);

    if (existing) {
      set({
        items: items.map((i) =>
          i.lineId === lineId ? { ...i, quantity: i.quantity + item.quantity } : i
        ),
      });
    } else {
      set({
        chefId: chef.id,
        chefName: chef.name,
        items: [...items, { ...item, lineId }],
      });
    }

    return 'ok';
  },

  removeItem: (lineId: string) => {
    set({ items: get().items.filter((i) => i.lineId !== lineId) });
  },

  updateQty: (lineId: string, quantity: number) => {
    if (quantity <= 0) {
      get().removeItem(lineId);
      return;
    }
    set({
      items: get().items.map((i) => (i.lineId === lineId ? { ...i, quantity } : i)),
    });
  },

  setInstructions: (lineId: string, instructions: string) => {
    const trimmed = instructions.trim();
    set({
      items: get().items.map((i) =>
        i.lineId === lineId ? { ...i, instructions: trimmed || undefined } : i
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
