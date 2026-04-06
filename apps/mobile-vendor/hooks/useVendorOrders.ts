import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useRef, useEffect, useState } from 'react';
import * as Haptics from 'expo-haptics';
import { api } from '../lib/api';

export interface OrderItem {
  name: string;
  quantity: number;
  price: number;
}

export interface Order {
  id: string;
  customerName: string;
  items: OrderItem[];
  total: number;
  status: 'pending' | 'accepted' | 'rejected' | 'preparing' | 'ready' | 'picked_up' | 'delivered' | 'cancelled';
  createdAt: string;
  deliveryAddress: string;
  specialInstructions?: string;
}

export interface OrdersResponse {
  orders: Order[];
  total: number;
  page: number;
  limit: number;
}

const UNDO_DELAY_MS = 3000;

// Pending orders with 10s polling + new-order haptic detection
export function useVendorPendingOrders() {
  const previousCountRef = useRef(0);

  const query = useQuery<OrdersResponse>({
    queryKey: ['chef', 'orders', 'pending'],
    queryFn: () => api.get<OrdersResponse>('/chef/orders?status=pending&page=1').then((r) => r.data),
    refetchInterval: 10_000,
    refetchIntervalInBackground: false,
  });

  useEffect(() => {
    const count = query.data?.orders?.length ?? 0;
    if (count > previousCountRef.current) {
      Haptics.notificationAsync(Haptics.NotificationFeedbackType.Warning);
    }
    previousCountRef.current = count;
  }, [query.data]);

  return query;
}

// Order history (paginated)
export function useVendorOrderHistory(page = 1) {
  return useQuery<OrdersResponse>({
    queryKey: ['chef', 'orders', 'history', page],
    queryFn: () => api.get<OrdersResponse>(`/chef/orders?page=${page}`).then((r) => r.data),
    staleTime: 30_000,
  });
}

export interface PendingUndo {
  orderId: string;
  action: 'accepted' | 'rejected';
}

// Accept/Reject with optimistic update + undo timer
export function useOrderAction() {
  const queryClient = useQueryClient();
  const [pendingUndo, setPendingUndo] = useState<PendingUndo | null>(null);
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const mutation = useMutation({
    mutationFn: ({ orderId, action, reason }: { orderId: string; action: 'accepted' | 'rejected'; reason?: string }) =>
      api.put(`/chef/orders/${orderId}/status`, { status: action, reason }),
    onMutate: async ({ orderId }) => {
      await queryClient.cancelQueries({ queryKey: ['chef', 'orders', 'pending'] });
      const previous = queryClient.getQueryData<OrdersResponse>(['chef', 'orders', 'pending']);
      queryClient.setQueryData<OrdersResponse>(['chef', 'orders', 'pending'], (old) => {
        if (!old) return old;
        return {
          ...old,
          orders: old.orders.filter((o) => o.id !== orderId),
        };
      });
      return { previous };
    },
    onError: (_err, _vars, context) => {
      if (context?.previous) {
        queryClient.setQueryData(['chef', 'orders', 'pending'], context.previous);
      }
    },
    onSettled: () => queryClient.invalidateQueries({ queryKey: ['chef', 'orders'] }),
  });

  function triggerAction(orderId: string, action: 'accepted' | 'rejected', reason?: string) {
    // Haptic feedback on decisive order action (accept or reject)
    void Haptics.impactAsync(Haptics.ImpactFeedbackStyle.Medium);

    // Optimistically remove card, set undo state, schedule API call after delay
    queryClient.setQueryData<OrdersResponse>(['chef', 'orders', 'pending'], (old) => {
      if (!old) return old;
      return {
        ...old,
        orders: old.orders.filter((o) => o.id !== orderId),
      };
    });
    setPendingUndo({ orderId, action });
    if (timerRef.current) clearTimeout(timerRef.current);
    timerRef.current = setTimeout(() => {
      mutation.mutate({ orderId, action, reason });
      setPendingUndo(null);
    }, UNDO_DELAY_MS);
  }

  function handleUndo() {
    if (timerRef.current) clearTimeout(timerRef.current);
    queryClient.invalidateQueries({ queryKey: ['chef', 'orders', 'pending'] });
    setPendingUndo(null);
  }

  return { triggerAction, handleUndo, pendingUndo, isLoading: mutation.isPending };
}
