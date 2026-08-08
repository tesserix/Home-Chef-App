// Lets a customer dismiss the floating active-order tracker on Home.
//
// Dismissal is scoped to the order AND its current stage: hiding is a "not now"
// for a card the customer has already read, not an unsubscribe. When the chef
// advances the order there is something new to say, so the card returns.
// Deliberately in-memory — a hidden card should come back on a fresh launch.

import { useCallback, useState } from 'react';
import type { Order } from '../types/customer';

export function hiddenKeyFor(order: Pick<Order, 'id' | 'status'>): string {
  return `${order.id}:${order.status}`;
}

export function visibleAfterHiding(orders: Order[], hidden: readonly string[]): Order[] {
  if (hidden.length === 0) return orders;
  return orders.filter((o) => !hidden.includes(hiddenKeyFor(o)));
}

export function useHiddenActiveOrders(orders: Order[]): {
  visible: Order[];
  hide: (order: Order) => void;
} {
  const [hidden, setHidden] = useState<string[]>([]);

  const hide = useCallback((order: Order) => {
    setHidden((prev) => {
      const key = hiddenKeyFor(order);
      return prev.includes(key) ? prev : [...prev, key];
    });
  }, []);

  return { visible: visibleAfterHiding(orders, hidden), hide };
}
