import { create } from 'zustand';
import { persist, createJSONStorage } from 'zustand/middleware';
import type { MenuItem, Chef, SelectedModifier, BakeryLineInput } from '@/shared/types';

export interface CartItem {
  id: string;
  menuItemId: string;
  name: string;
  description: string;
  /** UNIT price including any modifier deltas (#232) or bakery configuration (#1065). */
  price: number;
  quantity: number;
  imageUrl?: string;
  notes?: string;
  customizations?: Record<string, string | boolean>;
  /** Selected add-on modifiers for this line (#232). */
  modifiers?: SelectedModifier[];
  /** Bakery configuration, sent verbatim at checkout for the server to re-price (#1065). */
  bakery?: BakeryLineInput;
  /** The configuration as one line of display text. */
  bakerySummary?: string;
  /** Advance notice this line needs, in hours — checkout forces a later slot. */
  bakeryLeadTimeHours?: number;
}

/** What the configurator hands the cart for a configured bake (#1065). */
export interface BakeryLineConfig {
  bakery: BakeryLineInput;
  unitPrice: number;
  summary: string;
  leadTimeHours: number;
}

/** Stable key for a menu item + its modifier selection (#232) + its bake (#1065). */
function lineKey(
  menuItemId: string,
  modifiers?: SelectedModifier[],
  bakery?: BakeryLineInput
): string {
  const parts: string[] = [];
  if (modifiers && modifiers.length > 0) {
    parts.push(modifiers.map((m) => m.optionId).sort().join(','));
  }
  if (bakery) parts.push(bakeryKey(bakery));
  if (parts.length === 0) return menuItemId;
  return `${menuItemId}::${parts.join('|')}`;
}

function bakeryKey(b: BakeryLineInput): string {
  return [
    b.weightKg ?? '',
    [...(b.optionIds ?? [])].sort().join(','),
    (b.messageOnCake ?? '').trim(),
    b.occasion ?? '',
    b.referencePhotoUrl ?? '',
  ].join('~');
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
  addItem: (
    item: MenuItem,
    quantity: number,
    notes?: string,
    modifiers?: SelectedModifier[],
    bakery?: BakeryLineConfig
  ) => void;
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

      addItem: (item, quantity, notes, modifiers, bakery) => {
        const { items, chefId } = get();

        // If cart has items from different chef, show confirmation
        if (chefId && chefId !== item.chefId) {
          // This will be handled by the UI
          throw new Error('DIFFERENT_CHEF');
        }

        // Merge by line key so the same dish with different add-ons is a
        // distinct line (#232).
        const key = lineKey(item.id, modifiers, bakery?.bakery);
        const existingIndex = items.findIndex(
          (i) => lineKey(i.menuItemId, i.modifiers, i.bakery) === key
        );

        if (existingIndex > -1) {
          // Update existing line
          const updated = [...items];
          const existing = updated[existingIndex];
          if (existing) {
            existing.quantity += quantity;
            if (notes) existing.notes = notes;
          }
          // Changing the cart invalidates any applied promo preview (#39) — drop
          // it so the customer re-validates against the new subtotal.
          set({ items: updated, promoCode: null, promoDiscount: 0 });
        } else {
          // A configured bake is priced by the configurator (per-kg ladder plus
          // option deltas); everything else is item price plus modifier deltas.
          const unitPrice =
            bakery?.unitPrice ??
            item.price + (modifiers ?? []).reduce((s, m) => s + m.priceDelta, 0);
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
            bakery: bakery?.bakery,
            bakerySummary: bakery?.summary,
            bakeryLeadTimeHours: bakery?.leadTimeHours,
          };
          set({
            items: [...items, newItem],
            chefId: item.chefId,
            promoCode: null,
            promoDiscount: 0,
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
          // Cart changed → drop the applied promo preview (#39).
          promoCode: null,
          promoDiscount: 0,
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
        set({ items: updated, promoCode: null, promoDiscount: 0 });
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
