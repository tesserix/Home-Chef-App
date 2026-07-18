// Checkout-specific hooks for order creation and payment status polling.
// NOTE: useOrders / useOrder (order history) live in hooks/useOrderHistory.ts — Plan 05.
// Do NOT add general order list/detail hooks here.

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '../lib/api';
import type { Order } from '../types/customer';

interface CreateOrderPayload {
  chefId: string;
  // `notes` is the per-item wire field (apps/api CreateOrderItem.json:"notes").
  // modifierOptionIds are the selected add-ons for the line (#232).
  items: { menuItemId: string; quantity: number; notes?: string; modifierOptionIds?: string[] }[];
  // Omit when fulfillmentType is 'pickup' — the server does not require an
  // address for pickup orders (backend Task 4).
  deliveryAddressId?: string;
  // Order-level note to the chef. Backend reads `specialInstructions`
  // (CreateOrderRequest) — a `note` field is silently dropped.
  specialInstructions?: string;
  // Scheduled delivery slot (#51) — optional. "lunch"|"dinner" + "YYYY-MM-DD"
  // (IST). When set, the server resolves the slot window → ScheduledFor and
  // reserves the chef's per-slot daily capacity. Absent = ASAP.
  deliverySlot?: string;
  deliveryDate?: string;
  // Home-tiffin suggested time (#709) — ISO 8601. The customer's PREFERRED
  // delivery-arrival (delivery) or pickup-collection (pickup) time; the chef
  // confirms or proposes a different one at accept. Absent = "as soon as ready".
  requestedFulfillmentAt?: string;
  // Applied promo code (#39). The server re-validates and computes the discount;
  // an invalid/exhausted code is rejected so the client can't fake a discount.
  promoCode?: string;
  // 'delivery' (default) | 'chef_delivery' | 'pickup'. Omit → server defaults
  // to delivery.
  fulfillmentType?: 'delivery' | 'chef_delivery' | 'pickup';
}

export function useCreateOrder() {
  const queryClient = useQueryClient();
  // CreateOrder returns the order BARE (c.JSON(StatusCreated, order.ToResponse())),
  // with `id` at the top level — not wrapped in {data}. Wrap it here so callers
  // can keep reading `result.data.id`.
  return useMutation<{ data: Order }, Error, CreateOrderPayload>({
    mutationFn: (payload) =>
      api.post<Order>('/v1/orders', payload).then((r) => ({ data: r.data })),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['orders'] });
    },
  });
}

// Used for polling after payment — stops when order status shows payment confirmed or beyond.
// Interval: 3 seconds while status === 'pending'; stops on any other status.
export function useOrderStatus(id: string, enabled: boolean) {
  // GET /v1/orders/:id also returns the order bare — wrap to {data}.
  return useQuery<{ data: Order }>({
    queryKey: ['order-status', id],
    queryFn: () =>
      api.get<Order>(`/v1/orders/${id}`).then((r) => ({ data: r.data })),
    enabled: enabled && !!id,
    refetchInterval: (query) => {
      const status = query.state.data?.data?.status;
      // Stop polling when order has moved past pending (payment confirmed, or any terminal state)
      if (status && status !== 'pending') return false;
      return 3000; // poll every 3 seconds
    },
    refetchIntervalInBackground: false,
  });
}
