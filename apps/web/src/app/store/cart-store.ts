import { create } from 'zustand';
import { persist, createJSONStorage } from 'zustand/middleware';
import type { MenuItem, Chef } from '@/shared/types';

export interface CartItem {
  id: string;
  menuItemId: string;
  name: string;
  description: string;
  price: number;
  quantity: number;
  imageUrl?: string;
  notes?: string;
  customizations?: Record<string, string | boolean>;
}

interface CartState {
  items: CartItem[];
  chefId: string | null;
  chef: Pick<Chef, 'id' | 'businessName' | 'profileImage' | 'deliveryFee' | 'minimumOrder'> | null;
}

interface CartActions {
  addItem: (item: MenuItem, quantity: number, notes?: string) => void;
  removeItem: (itemId: string) => void;
  updateQuantity: (itemId: string, quantity: number) => void;
  updateNotes: (itemId: string, notes: string) => void;
  setChef: (chef: CartState['chef']) => void;
  clearCart: () => void;
  getSubtotal: () => number;
  getItemCount: () => number;
}

type CartStore = CartState & CartActions;

const initialState: CartState = {
  items: [],
  chefId: null,
  chef: null,
};

export const useCartStore = create<CartStore>()(
  persist(
    (set, get) => ({
      ...initialState,

      addItem: (item, quantity, notes) => {
        const { items, chefId } = get();

        // If cart has items from different chef, show confirmation
        if (chefId && chefId !== item.chefId) {
          // This will be handled by the UI
          throw new Error('DIFFERENT_CHEF');
        }

        const existingIndex = items.findIndex((i) => i.menuItemId === item.id);

        if (existingIndex > -1) {
          // Update existing item
          const updated = [...items];
          const existing = updated[existingIndex];
          if (existing) {
            existing.quantity += quantity;
            if (notes) existing.notes = notes;
          }
          set({ items: updated });
        } else {
          // Add new item
          const newItem: CartItem = {
            id: `cart-${Date.now()}`,
            menuItemId: item.id,
            name: item.name,
            description: item.description || '',
            price: item.price,
            quantity,
            imageUrl: item.imageUrl,
            notes,
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
