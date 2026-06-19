import { describe, it, expect, beforeEach } from '@jest/globals';
import { useCartStore } from './cart-store';
import type { CartItem } from '../types/customer';

const chefA = { id: 'chef-a', name: 'Anita Kitchen' };
const chefB = { id: 'chef-b', name: 'Bhim Bites' };

function item(overrides: Partial<CartItem> = {}): CartItem {
  const menuItemId = overrides.menuItemId ?? 'm1';
  return {
    lineId: overrides.lineId ?? menuItemId,
    menuItemId,
    name: 'Paneer Tikka',
    price: 200,
    quantity: 1,
    ...overrides,
  };
}

// Zustand store is a module singleton — reset to a clean slate before each test.
beforeEach(() => {
  useCartStore.setState({ chefId: null, chefName: null, items: [] });
});

describe('cart-store', () => {
  it('adds an item to an empty cart and records the chef', () => {
    const result = useCartStore.getState().addItem(item(), chefA);
    const state = useCartStore.getState();

    expect(result).toBe('ok');
    expect(state.chefId).toBe('chef-a');
    expect(state.chefName).toBe('Anita Kitchen');
    expect(state.items).toHaveLength(1);
  });

  it('increments quantity when the same item is added again (no duplicate row)', () => {
    const add = useCartStore.getState().addItem;
    add(item({ quantity: 1 }), chefA);
    add(item({ quantity: 2 }), chefA);

    const { items } = useCartStore.getState();
    expect(items).toHaveLength(1);
    expect(items[0]?.quantity).toBe(3);
  });

  it('rejects items from a different chef without mutating the cart', () => {
    useCartStore.getState().addItem(item(), chefA);
    const result = useCartStore.getState().addItem(item({ menuItemId: 'm2' }), chefB);

    expect(result).toBe('cross_chef_conflict');
    const state = useCartStore.getState();
    expect(state.chefId).toBe('chef-a');
    expect(state.items).toHaveLength(1);
  });

  it('accepts a different chef after clearCart', () => {
    useCartStore.getState().addItem(item(), chefA);
    useCartStore.getState().clearCart();
    const result = useCartStore.getState().addItem(item({ menuItemId: 'm2' }), chefB);

    expect(result).toBe('ok');
    expect(useCartStore.getState().chefId).toBe('chef-b');
  });

  it('removes an item by id', () => {
    const add = useCartStore.getState().addItem;
    add(item({ menuItemId: 'm1' }), chefA);
    add(item({ menuItemId: 'm2' }), chefA);

    useCartStore.getState().removeItem('m1');
    const { items } = useCartStore.getState();
    expect(items).toHaveLength(1);
    expect(items[0]?.menuItemId).toBe('m2');
  });

  it('updateQty sets an absolute quantity', () => {
    useCartStore.getState().addItem(item({ quantity: 1 }), chefA);
    useCartStore.getState().updateQty('m1', 5);
    expect(useCartStore.getState().items[0]?.quantity).toBe(5);
  });

  it('updateQty <= 0 removes the item', () => {
    useCartStore.getState().addItem(item(), chefA);
    useCartStore.getState().updateQty('m1', 0);
    expect(useCartStore.getState().items).toHaveLength(0);
  });

  it('setInstructions trims and clears on empty', () => {
    useCartStore.getState().addItem(item(), chefA);

    useCartStore.getState().setInstructions('m1', '  no onions  ');
    expect(useCartStore.getState().items[0]?.instructions).toBe('no onions');

    useCartStore.getState().setInstructions('m1', '   ');
    expect(useCartStore.getState().items[0]?.instructions).toBeUndefined();
  });

  it('total sums price * quantity', () => {
    const add = useCartStore.getState().addItem;
    add(item({ menuItemId: 'm1', price: 200, quantity: 2 }), chefA);
    add(item({ menuItemId: 'm2', price: 50, quantity: 3 }), chefA);
    expect(useCartStore.getState().total()).toBe(550);
  });

  it('totalCount sums quantities', () => {
    const add = useCartStore.getState().addItem;
    add(item({ menuItemId: 'm1', quantity: 2 }), chefA);
    add(item({ menuItemId: 'm2', quantity: 3 }), chefA);
    expect(useCartStore.getState().totalCount()).toBe(5);
  });

  it('clearCart resets chef context and items', () => {
    useCartStore.getState().addItem(item(), chefA);
    useCartStore.getState().clearCart();

    const state = useCartStore.getState();
    expect(state.chefId).toBeNull();
    expect(state.chefName).toBeNull();
    expect(state.items).toHaveLength(0);
  });

  it('does not mutate the previous items array (immutability)', () => {
    useCartStore.getState().addItem(item(), chefA);
    const before = useCartStore.getState().items;
    useCartStore.getState().addItem(item({ menuItemId: 'm2' }), chefA);
    const after = useCartStore.getState().items;
    expect(after).not.toBe(before);
  });
});
