import { create } from 'zustand';
import { persist, createJSONStorage } from 'zustand/middleware';
import type { MenuItem, Chef, SelectedModifier } from '@/shared/types';

export interface CartItem {
  id: string;
  menuItemId: string;
  name: string;
  description: string;
  /** UNIT price including any modifier deltas (#232). */
  price: number;
  quantity: number;
  imageUrl?: string;
  notes?: string;
  customizations?: Record<string, string | boolean>;
  /** Selected add-on modifiers for this line (#232). */
  modifiers?: SelectedModifier[];
}

/** Stable key for a menu item + its modifier selection (#232). */
function lineKey(menuItemId: string, modifiers?: SelectedModifier[]): string {
  if (!modifiers || modifiers.length === 0) return menuItemId;
  return `${menuItemId}::${modifiers.map((m) => m.optionId).sort().join(',')}`;
}

interface CartState {
  items: CartItem[];
  chefId: string | null;
  chef: Pick<Chef, 'id' | 'businessName' | 'profileImage' | 'deliveryFee' | 'minimumOrder'> | null;
  // Applied promo (#39). Persisted from the cart so it survives the hop to
  // checkout; the server re-validates + recomputes the real discount at order
  // time, so promoDiscount here is only a preview for display.
  promoCode: string | null;
  promoDiscount: number;
}

interface CartActions {
  addItem: (item: MenuItem, quantity: number, notes?: string, modifiers?: SelectedModifier[]) => void;
  removeItem: (itemId: string) => void;
  updateQuantity: (itemId: string, quantity: number) => void;
  updateNotes: (itemId: string, notes: string) => void;
  setChef: (chef: CartState['chef']) => void;
  setPromo: (code: string, discount: number) => void;
  clearPromo: () => void;
  clearCart: () => void;
  getSubtotal: () => number;
  getItemCount: () => number;
}

type CartStore = CartState & CartActions;

const initialState: CartState = {
  items: [],
  chefId: null,
  chef: null,
  promoCode: null,
  promoDiscount: 0,
};

export const useCartStore = create<CartStore>()(
  persist(
    (set, get) => ({
      ...initialState,

      addItem: (item, quantity, notes, modifiers) => {
        const { items, chefId } = get();

        // If cart has items from different chef, show confirmation
        if (chefId && chefId !== item.chefId) {
          // This will be handled by the UI
          throw new Error('DIFFERENT_CHEF');
        }

        // Merge by line key so the same dish with different add-ons is a
        // distinct line (#232).
        const key = lineKey(item.id, modifiers);
        const existingIndex = items.findIndex((i) => lineKey(i.menuItemId, i.modifiers) === key);

        if (existingIndex > -1) {
          // Update existing line
          const updated = [...items];
          const existing = updated[existingIndex];
          if (existing) {
            existing.quantity += quantity;
            if (notes) existing.notes = notes;
          }
          set({ items: updated });
        } else {
          // Unit price includes any modifier deltas.
          const unitPrice = item.price + (modifiers ?? []).reduce((s, m) => s + m.priceDelta, 0);
          const newItem: CartItem = {
            id: `cart-${Date.now()}-${Math.round(item.price)}`,
            menuItemId: item.id,
            name: item.name,
            description: item.description || '',
            price: unitPrice,
            quantity,
            imageUrl: item.imageUrl,
            notes,
            modifiers,
          };
          set({
            items: [...items, newItem],
            chefId: item.chefId,
          });
        }
      },

      removeItem: (itemId) => {
        const { items } = get();
        const updated = items.filter((i) => i.id !== itemId);
        set({
          items: updated,
          chefId: updated.length === 0 ? null : get().chefId,
          chef: updated.length === 0 ? null : get().chef,
        });
      },

      updateQuantity: (itemId, quantity) => {
        if (quantity < 1) {
          get().removeItem(itemId);
          return;
        }

        const { items } = get();
        const updated = items.map((i) =>
          i.id === itemId ? { ...i, quantity } : i
        );
        set({ items: updated });
      },

      updateNotes: (itemId, notes) => {
        const { items } = get();
        const updated = items.map((i) =>
          i.id === itemId ? { ...i, notes } : i
        );
        set({ items: updated });
      },

      setChef: (chef) => {
        set({ chef, chefId: chef?.id ?? null });
      },

      setPromo: (code, discount) => set({ promoCode: code, promoDiscount: discount }),
      clearPromo: () => set({ promoCode: null, promoDiscount: 0 }),

      clearCart: () => set(initialState),

      getSubtotal: () => {
        const { items } = get();
        return items.reduce((sum, item) => sum + item.price * item.quantity, 0);
      },

      getItemCount: () => {
        const { items } = get();
        return items.reduce((count, item) => count + item.quantity, 0);
      },
    }),
    {
      name: 'homechef-cart',
      storage: createJSONStorage(() => localStorage),
    }
  )
);
