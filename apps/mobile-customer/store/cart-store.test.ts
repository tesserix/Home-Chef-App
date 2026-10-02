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
  useCartStore.setState({ chefId: null, chefName: null, items: [], baskets: {}, accounts: {}, ownerId: null, eligibleChefIds: null, locationKey: null });
});

describe('cart-store', () => {
  it('limits new vendor carts to ten without replacing saved contents', () => {
    const store = useCartStore.getState();
    for (let i = 0; i < 10; i++) {
      expect(store.addItem(item(), { id: `chef-${i}`, name: `Kitchen ${i}` })).toBe('ok');
    }
    expect(store.addItem(item(), chefA)).toBe('cart-limit');
    expect(Object.keys(useCartStore.getState().baskets)).toHaveLength(10);
    expect(store.addItem(item(), { id: 'chef-0', name: 'Kitchen 0' })).toBe('ok');
    expect(useCartStore.getState().baskets['chef-0']?.items[0]?.quantity).toBe(2);
  });

  it('keeps separate ten-cart allowances for distant locations', () => {
    const store = useCartStore.getState();
    for (const city of ['india', 'auckland']) {
      store.beginLocation(city);
      store.applyAvailability(city, []);
      for (let i = 0; i < 10; i++) {
        expect(store.addItem(item(), { id: `${city}-${i}`, name: city })).toBe('ok');
      }
      expect(store.addItem(item(), { id: `${city}-extra`, name: city })).toBe('cart-limit');
    }
    expect(Object.keys(useCartStore.getState().baskets)).toHaveLength(20);
    store.beginLocation('nearby-auckland');
    store.applyAvailability('nearby-auckland', Array.from({ length: 10 }, (_, i) => `auckland-${i}`));
    expect(store.addItem(item(), chefA)).toBe('cart-limit');
    store.clearCart();
    expect(store.addItem(item(), chefA)).toBe('ok');
  });

  it('waits for location eligibility and excludes expired carts from the limit', () => {
    const store = useCartStore.getState();
    store.beginLocation('auckland');
    expect(store.addItem(item(), chefA)).toBe('location-pending');
    store.applyAvailability('auckland', []);
    for (let i = 0; i < 10; i++) store.addItem(item(), { id: `chef-${i}`, name: 'Kitchen' });
    const baskets = useCartStore.getState().baskets;
    useCartStore.setState({ baskets: { ...baskets, 'chef-0': { ...baskets['chef-0']!, updatedAt: Date.now() - 45 * 86400000 } } });
    expect(store.addItem(item(), chefA)).toBe('ok');
    expect(useCartStore.getState().baskets['chef-0']).toBeUndefined();
  });

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

  it('saves multiple vendors and restores only carts eligible for the address', () => {
    const store = useCartStore.getState();
    store.addItem(item(), chefA);
    store.addItem(item({ menuItemId: 'm2' }), chefB);
    expect(useCartStore.getState().chefId).toBe('chef-b');
    expect(Object.keys(useCartStore.getState().baskets)).toHaveLength(2);
    store.beginLocation('auckland');
    expect(useCartStore.getState().totalCount()).toBe(0);
    store.applyAvailability('auckland', []);
    expect(useCartStore.getState().items).toEqual([]);
    store.beginLocation('india');
    store.applyAvailability('auckland', [chefB.id]);
    expect(useCartStore.getState().items).toEqual([]);
    store.applyAvailability('india', [chefA.id]);
    expect(useCartStore.getState().chefId).toBe(chefA.id);
    expect(useCartStore.getState().items[0]?.menuItemId).toBe('m1');
    expect(Object.keys(useCartStore.getState().baskets)).toHaveLength(2);
  });

  it('expires baskets at 45 days without extending retention on reads', () => {
    useCartStore.getState().addItem(item(), chefA);
    const savedAt = useCartStore.getState().baskets[chefA.id]!.updatedAt;
    useCartStore.getState().pruneExpired(savedAt + 45 * 86400000 - 1);
    expect(useCartStore.getState().totalCount()).toBe(1);
    useCartStore.getState().pruneExpired(savedAt + 45 * 86400000);
    expect(useCartStore.getState().items).toEqual([]);
    expect(useCartStore.getState().baskets).toEqual({});
  });

  it('accepts a different chef after clearCart', () => {
    useCartStore.getState().addItem(item(), chefA);
    useCartStore.getState().clearCart();
    const result = useCartStore.getState().addItem(item({ menuItemId: 'm2' }), chefB);

    expect(result).toBe('ok');
    expect(useCartStore.getState().chefId).toBe('chef-b');
  });

  it('isolates saved carts across accounts and restores the original account', () => {
    const store = useCartStore.getState();
    store.setOwner('customer-a');
    store.addItem(item(), chefA);
    store.setOwner('customer-b');
    expect(useCartStore.getState().baskets).toEqual({});
    store.addItem(item({ menuItemId: 'm2' }), chefB);
    store.setOwner('customer-a');
    store.beginLocation('home');
    store.applyAvailability('home', [chefA.id, chefB.id]);
    expect(useCartStore.getState().items[0]?.menuItemId).toBe('m1');
    expect(useCartStore.getState().baskets[chefB.id]).toBeUndefined();
  });

  it('clears only the selected vendor and retains other saved carts', () => {
    const store = useCartStore.getState();
    store.addItem(item(), chefA);
    store.addItem(item({ menuItemId: 'm2' }), chefB);
    store.clearCart();
    expect(Object.keys(useCartStore.getState().baskets)).toEqual([chefA.id]);
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

// Bakery lines (#1065): two cakes of the same menu item configured differently
// are two different orders to bake, so they must never merge into one line.
describe('bakery lines', () => {
  const bakeryLine = (overrides: Partial<CartItem> = {}): CartItem =>
    item({
      menuItemId: 'cake-1',
      name: 'Chocolate Truffle',
      price: 1700,
      bakery: {
        weightKg: 1.5,
        optionIds: ['shape-heart', 'fl-choc'],
        messageOnCake: 'Happy Birthday Aarav',
      },
      bakerySummary: '1.5 kg · Heart · Belgian chocolate',
      ...overrides,
    });

  it('keeps two differently configured cakes as separate lines', () => {
    const store = useCartStore.getState();
    store.addItem({ ...bakeryLine(), lineId: '' }, chefA);
    store.addItem(
      {
        ...bakeryLine({
          price: 900,
          bakery: { weightKg: 1, optionIds: ['shape-round', 'fl-choc'] },
          bakerySummary: '1 kg · Round · Belgian chocolate',
        }),
        lineId: '',
      },
      chefA,
    );

    expect(useCartStore.getState().items).toHaveLength(2);
    expect(useCartStore.getState().total()).toBe(2600);
  });

  it('merges two identically configured cakes into one line', () => {
    const store = useCartStore.getState();
    store.addItem({ ...bakeryLine(), lineId: '' }, chefA);
    store.addItem({ ...bakeryLine(), lineId: '' }, chefA);

    const items = useCartStore.getState().items;
    expect(items).toHaveLength(1);
    expect(items[0]!.quantity).toBe(2);
  });

  it('carries the configuration through to the line it stores', () => {
    useCartStore.getState().addItem({ ...bakeryLine(), lineId: '' }, chefA);
    const line = useCartStore.getState().items[0]!;
    expect(line.bakery?.weightKg).toBe(1.5);
    expect(line.bakerySummary).toContain('Heart');
  });
});

it('claims a legacy guest cart on first sign-in without exposing another account', () => {
 const store = useCartStore.getState();
 store.addItem(item(),chefA);
 store.setOwner('guest');
 store.setOwner('customer-a');
 expect(useCartStore.getState().baskets[chefA.id]?.items).toHaveLength(1);
 store.setOwner('guest');
 expect(useCartStore.getState().baskets).toEqual({});
 store.setOwner('customer-b');
 expect(useCartStore.getState().baskets).toEqual({});
 store.setOwner('customer-a');
 expect(useCartStore.getState().baskets[chefA.id]?.items).toHaveLength(1);
});

it('clears only the paid vendor basket after switching locations', () => {
 const store = useCartStore.getState();
 store.addItem(item(),chefA);
 store.addItem(item(),chefB);
 store.beginLocation('city-b');
 store.applyAvailability('city-b',[chefB.id]);
 store.clearCart(chefA.id);
 expect(useCartStore.getState().baskets[chefA.id]).toBeUndefined();
 expect(useCartStore.getState().chefId).toBe(chefB.id);
 expect(useCartStore.getState().items).toHaveLength(1);
});
