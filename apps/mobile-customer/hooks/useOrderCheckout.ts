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
  deliveryAddressId: string;
  // Order-level note to the chef. Backend reads `specialInstructions`
  // (CreateOrderRequest) — a `note` field is silently dropped.
  specialInstructions?: string;
  // Scheduled delivery slot (#51) — optional. "lunch"|"dinner" + "YYYY-MM-DD"
  // (IST). When set, the server resolves the slot window → ScheduledFor and
  // reserves the chef's per-slot daily capacity. Absent = ASAP.
  deliverySlot?: string;
  deliveryDate?: string;
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
