import { create } from 'zustand';
import { persist, createJSONStorage } from 'zustand/middleware';
import AsyncStorage from '@react-native-async-storage/async-storage';
import type { CartBakeryConfig, CartItem, SelectedModifier } from '../types/customer';

interface ChefSummary {
  id: string;
  name: string;
}

type AddItemResult = 'ok' | 'cart-limit' | 'location-pending';

export const CART_ADD_ERRORS = {
  'cart-limit': 'You can save up to 10 vendor carts for this location. Remove a saved cart before adding another vendor.',
  'location-pending': 'Checking saved carts for your address. Please try again in a moment.',
};

/**
 * Stable line id for a menu item + its modifier selection + its bakery
 * configuration. Two cakes of the same item at different weights are two
 * different things to bake, so they stay separate lines (#1065).
 */
export function makeLineId(
  menuItemId: string,
  modifiers?: SelectedModifier[],
  bakery?: CartBakeryConfig,
): string {
  const parts: string[] = [];
  if (modifiers && modifiers.length > 0) {
    parts.push(modifiers.map((m) => m.optionId).sort().join(','));
  }
  if (bakery) {
    parts.push(bakeryKey(bakery));
  }
  if (parts.length === 0) return menuItemId;
  return `${menuItemId}::${parts.join('|')}`;
}

function bakeryKey(b: CartBakeryConfig): string {
  return [
    b.weightKg ?? '',
    [...(b.optionIds ?? [])].sort().join(','),
    (b.messageOnCake ?? '').trim(),
    b.occasion ?? '',
    b.referencePhotoUrl ?? '',
  ].join('~');
}

export interface SavedBasket {
  chefId: string;
  chefName: string;
  items: CartItem[];
  updatedAt: number;
  locationKey?: string | null;
}

const RETENTION_MS = 45 * 24 * 60 * 60 * 1000;
const MAX_LOCAL_CARTS = 10;
const emptyCart = { chefId: null, chefName: null, items: [] as CartItem[] };

interface CartState {
  chefId: string | null;
  chefName: string | null;
  items: CartItem[];
  baskets: Record<string, SavedBasket>;
  accounts: Record<string, Record<string, SavedBasket>>;
  ownerId: string | null;
  eligibleChefIds: string[] | null;
  locationKey: string | null;
  setOwner: (id: string) => void;
  beginLocation: (key: string) => void;
  applyAvailability: (key: string, ids: string[] | null) => void;
  selectBasket: (id: string) => void;
  pruneExpired: (now?: number) => void;
  addItem: (item: CartItem, chef: ChefSummary) => AddItemResult;
  removeItem: (lineId: string) => void;
  updateQty: (lineId: string, quantity: number) => void;
  setInstructions: (lineId: string, instructions: string) => void;
  clearCart: (chefId?: string) => void;
  total: () => number;
  totalCount: () => number;
  hasHydrated: boolean;
  setHasHydrated: (value: boolean) => void;
}

function surviving(baskets: Record<string, SavedBasket>, now: number) {
  return Object.fromEntries(Object.entries(baskets).filter(([, b]) =>
    b.items.length > 0 && Number.isFinite(b.updatedAt) && now - b.updatedAt < RETENTION_MS));
}

function project(baskets: Record<string, SavedBasket>, ids: string[], preferred: string | null) {
  const basket = (preferred && ids.includes(preferred) ? baskets[preferred] : undefined)
    ?? Object.values(baskets).sort((a, b) => b.updatedAt - a.updatedAt).find(b => ids.includes(b.chefId));
  return basket ? { chefId: basket.chefId, chefName: basket.chefName, items: basket.items } : emptyCart;
}

export const useCartStore = create<CartState>()(persist((set, get) => {
  const updateItems = (items: CartItem[]) => {
    const state = get();
    if (!state.chefId) return;
    const baskets = { ...state.baskets };
    if (items.length) baskets[state.chefId] = { ...baskets[state.chefId], chefId: state.chefId, chefName: state.chefName ?? '', items, updatedAt: Date.now() };
    else delete baskets[state.chefId];
    set({ baskets, ...(items.length ? { items } : emptyCart) });
  };
  return {
    ...emptyCart, baskets: {}, accounts: {}, ownerId: null, eligibleChefIds: null, locationKey: null,
    setOwner: (ownerId) => {
      const state = get();
      if (state.ownerId === ownerId) return;
      const accounts = { ...state.accounts };
      if (state.ownerId) accounts[state.ownerId] = state.baskets;
      const claimGuest = state.ownerId === 'guest' && ownerId !== 'guest' && !accounts[ownerId];
      const baskets = surviving(state.ownerId === null || claimGuest ? state.baskets : accounts[ownerId] ?? {}, Date.now());
      if (claimGuest) delete accounts.guest;
      set({ ownerId, accounts, baskets, ...emptyCart, eligibleChefIds: null, locationKey: null });
    },
    beginLocation: (locationKey) => {
      if (get().locationKey === locationKey) return;
      get().pruneExpired();
      set({ locationKey, eligibleChefIds: null, ...emptyCart });
    },
    applyAvailability: (key, ids) => {
      if (get().locationKey !== key) return;
      const baskets = surviving(get().baskets, Date.now());
      set({ baskets, eligibleChefIds: ids, ...project(baskets, ids ?? [], get().chefId) });
    },
    selectBasket: (id) => {
      const state = get();
      const baskets = surviving(state.baskets, Date.now());
      if (state.eligibleChefIds?.includes(id) && baskets[id]) {
        set({ baskets, ...project(baskets, [id], id) });
      }
    },
    pruneExpired: (now = Date.now()) => {
      const state = get();
      const baskets = surviving(state.baskets, now);
      const accounts = Object.fromEntries(Object.entries(state.accounts).map(([id, saved]) => [id, surviving(saved, now)]));
      set({ baskets, accounts, ...(state.chefId && !baskets[state.chefId] ? emptyCart : {}) });
    },
    addItem: (item, chef) => {
      const state = get();
      const baskets = surviving(state.baskets, Date.now());
      if (state.locationKey !== null && state.eligibleChefIds === null) return 'location-pending';
      const localCount = Object.values(baskets).filter(b =>
        state.locationKey === null || b.locationKey === state.locationKey || state.eligibleChefIds?.includes(b.chefId)).length;
      if (!baskets[chef.id] && localCount >= MAX_LOCAL_CARTS) return 'cart-limit';
      const items = baskets[chef.id]?.items ?? [];
      const lineId = item.lineId || makeLineId(item.menuItemId, item.modifiers, item.bakery);
      const existing = items.some(i => i.lineId === lineId);
      const next = existing ? items.map(i => i.lineId === lineId ? { ...i, quantity: i.quantity + item.quantity } : i)
        : [...items, { ...item, lineId }];
      baskets[chef.id] = { chefId: chef.id, chefName: chef.name, items: next, updatedAt: Date.now(), locationKey: state.locationKey };
      const visible = state.locationKey === null || state.eligibleChefIds?.includes(chef.id);
      set({ baskets, ...(visible ? { chefId: chef.id, chefName: chef.name, items: next } : {}) });
      return 'ok';
    },
    removeItem: id => updateItems(get().items.filter(i => i.lineId !== id)),
    updateQty: (id, quantity) => quantity <= 0 ? get().removeItem(id) : updateItems(get().items.map(i => i.lineId === id ? { ...i, quantity } : i)),
    setInstructions: (id, value) => updateItems(get().items.map(i => i.lineId === id ? { ...i, instructions: value.trim() || undefined } : i)),
    clearCart: (chefId = get().chefId ?? undefined) => {
      if (!chefId) return;
      const state = get();
      const baskets = { ...state.baskets };
      delete baskets[chefId];
      set({ baskets, ...(state.chefId === chefId ? emptyCart : {}) });
    },
    total: () => get().items.reduce((sum, i) => sum + i.price * i.quantity, 0),
    totalCount: () => get().items.reduce((sum, i) => sum + i.quantity, 0),
    hasHydrated: false,
    setHasHydrated: hasHydrated => set({ hasHydrated }),
  };
}, {
  name: 'customer-cart',
  version: 1,
  storage: createJSONStorage(() => AsyncStorage),
  partialize: state => ({ baskets: state.baskets, accounts: state.accounts, ownerId: state.ownerId }),
  migrate: (persisted) => {
    const old = persisted as Partial<CartState>;
    const baskets = old.baskets ?? (old.chefId && old.items?.length ? {
      [old.chefId]: { chefId: old.chefId, chefName: old.chefName ?? '', items: old.items, updatedAt: Date.now() },
    } : {});
    return { baskets: surviving(baskets, Date.now()), accounts: old.accounts ?? {}, ownerId: old.ownerId ?? null };
  },
  onRehydrateStorage: () => state => {
    state?.pruneExpired();
    state?.setHasHydrated(true);
  },
}));
