import { beforeEach, describe, expect, it } from 'vitest';
import { useCartStore } from './cart-store';
import type { MenuItem } from '@/shared/types';

// Two cakes of the same item at different weights are two different things to
// bake, so they must stay separate cart lines and carry their configuration
// through to checkout (#1065).

const CAKE = {
  id: 'cake-1',
  chefId: 'chef-1',
  name: 'Truffle Cake',
  price: 0,
  dietaryTags: [],
  allergens: [],
  prepTime: 30,
  isAvailable: true,
  isFeatured: false,
  serves: 4,
} as MenuItem;

beforeEach(() => {
  useCartStore.getState().clearCart();
});

describe('cart lines for a configured bake', () => {
  it('keeps two weights of the same cake as separate lines', () => {
    const { addItem } = useCartStore.getState();
    addItem(CAKE, 1, undefined, undefined, {
      bakery: { weightKg: 1 },
      unitPrice: 800,
      summary: '1 kg',
      leadTimeHours: 24,
    });
    addItem(CAKE, 1, undefined, undefined, {
      bakery: { weightKg: 2 },
      unitPrice: 1600,
      summary: '2 kg',
      leadTimeHours: 24,
    });

    const { items } = useCartStore.getState();
    expect(items).toHaveLength(2);
    expect(items.map((i) => i.price)).toEqual([800, 1600]);
  });

  it('merges an identical configuration into one line', () => {
    const { addItem } = useCartStore.getState();
    const config = {
      bakery: { weightKg: 1, optionIds: ['o1'], messageOnCake: 'Hi' },
      unitPrice: 900,
      summary: '1 kg · Hi',
      leadTimeHours: 24,
    };
    addItem(CAKE, 1, undefined, undefined, config);
    addItem(CAKE, 1, undefined, undefined, config);

    const { items } = useCartStore.getState();
    expect(items).toHaveLength(1);
    expect(items[0]?.quantity).toBe(2);
  });

  it('prices the line at the configured price, not the item price', () => {
    useCartStore.getState().addItem(CAKE, 2, undefined, undefined, {
      bakery: { weightKg: 1.5 },
      unitPrice: 1200,
      summary: '1.5 kg',
      leadTimeHours: 48,
    });

    expect(useCartStore.getState().getSubtotal()).toBe(2400);
  });

  it('carries the configuration, summary and lead time on the line', () => {
    useCartStore.getState().addItem(CAKE, 1, undefined, undefined, {
      bakery: { weightKg: 1, occasion: 'birthday' },
      unitPrice: 800,
      summary: '1 kg · Birthday',
      leadTimeHours: 48,
    });

    const line = useCartStore.getState().items[0];
    expect(line?.bakery).toEqual({ weightKg: 1, occasion: 'birthday' });
    expect(line?.bakerySummary).toBe('1 kg · Birthday');
    expect(line?.bakeryLeadTimeHours).toBe(48);
  });

  it('leaves a plain dish untouched', () => {
    useCartStore.getState().addItem({ ...CAKE, id: 'dal', price: 180 } as MenuItem, 1);
    const line = useCartStore.getState().items[0];
    expect(line?.price).toBe(180);
    expect(line?.bakery).toBeUndefined();
  });
});
