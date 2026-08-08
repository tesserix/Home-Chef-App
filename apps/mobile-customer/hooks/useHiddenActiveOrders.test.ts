import { describe, it, expect } from '@jest/globals';

import { hiddenKeyFor, visibleAfterHiding } from './useHiddenActiveOrders';
import type { Order } from '../types/customer';

const order = (id: string, status: string) => ({ id, status }) as Order;

describe('visibleAfterHiding', () => {
  it('passes every order through while nothing is hidden', () => {
    const orders = [order('a', 'confirmed'), order('b', 'preparing')];

    expect(visibleAfterHiding(orders, []).map((o) => o.id)).toEqual(['a', 'b']);
  });

  it('drops only the order that was hidden', () => {
    const orders = [order('a', 'confirmed'), order('b', 'preparing')];
    const hidden = [hiddenKeyFor(orders[0]!)];

    expect(visibleAfterHiding(orders, hidden).map((o) => o.id)).toEqual(['b']);
  });

  // The card is how a customer learns their food moved. Hiding it is a "not
  // now", not an unsubscribe — the next stage is news, so it comes back.
  it('brings a hidden order back when its stage advances', () => {
    const hidden = [hiddenKeyFor(order('a', 'confirmed'))];

    expect(visibleAfterHiding([order('a', 'on_the_way')], hidden)).toHaveLength(1);
  });

  it('keeps an order hidden while its stage is unchanged', () => {
    const hidden = [hiddenKeyFor(order('a', 'confirmed'))];

    expect(visibleAfterHiding([order('a', 'confirmed')], hidden)).toHaveLength(0);
  });

  it('shows a newly placed order even while another is hidden', () => {
    const hidden = [hiddenKeyFor(order('a', 'confirmed'))];
    const orders = [order('b', 'pending'), order('a', 'confirmed')];

    expect(visibleAfterHiding(orders, hidden).map((o) => o.id)).toEqual(['b']);
  });

  it('keys on the order, so two orders at the same stage hide independently', () => {
    expect(hiddenKeyFor(order('a', 'confirmed'))).not.toEqual(
      hiddenKeyFor(order('b', 'confirmed')),
    );
  });
});
