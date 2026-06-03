import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useRef, useEffect, useState } from 'react';
import * as Haptics from 'expo-haptics';
import { api } from '../lib/api';

export interface OrderItem {
  name: string;
  quantity: number;
  price: number;
  subtotal?: number;
  notes?: string;
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
  // Optional pricing breakdown — only present on the detail/list ToResponse
  // payload. Hero card + history row don't need these, but the detail screen
  // surfaces them in the totals block.
  subtotal?: number;
  deliveryFee?: number;
  serviceFee?: number;
  tax?: number;
  taxName?: string;
  tip?: number;
  discount?: number;
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

// Generic status mutation used by the order detail screen for the in-flight
// transitions (accepted → preparing → ready). Unlike useOrderAction these
// transitions don't get an undo timer — the chef wouldn't expect to "undo
// marking ready" after the fact. Optimistic update on the detail cache so
// the footer flips immediately.
export function useUpdateOrderStatus() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      orderId,
      status,
    }: {
      orderId: string;
      status: Order['status'];
    }) => api.put(`/chef/orders/${orderId}/status`, { status }),
    onMutate: async ({ orderId, status }) => {
      await queryClient.cancelQueries({ queryKey: ['chef', 'orders', 'detail', orderId] });
      const previous = queryClient.getQueryData<Order>(['chef', 'orders', 'detail', orderId]);
      if (previous) {
        queryClient.setQueryData<Order>(['chef', 'orders', 'detail', orderId], {
          ...previous,
          status,
        });
      }
      return { previous };
    },
    onError: (_err, { orderId }, context) => {
      if (context?.previous) {
        queryClient.setQueryData(['chef', 'orders', 'detail', orderId], context.previous);
      }
    },
    onSettled: () => {
      // Refresh dashboard + queue + history so card moves between tabs
      queryClient.invalidateQueries({ queryKey: ['chef', 'orders'] });
      queryClient.invalidateQueries({ queryKey: ['chef', 'dashboard'] });
    },
  });
}

// Lookup an order by id. Cache-first across the pending + history page
// caches, falling back to a targeted fetch when the detail is opened from
// a deep link (push notification). The detail screen consumes this hook.
//
// NOTE: backend has no GET /chef/orders/:orderId endpoint yet, so we can't
// refetch the single order directly — we paginate the lists. Acceptable for
// Sprint 1 because the only deep-link entry today is push notifications,
// and those always represent a pending order (cache hit guaranteed within
// the 10s polling window).
export function useVendorOrderDetail(orderId: string | null | undefined) {
  const queryClient = useQueryClient();

  return useQuery<Order>({
    queryKey: ['chef', 'orders', 'detail', orderId],
    enabled: !!orderId,
    queryFn: async () => {
      if (!orderId) throw new Error('orderId required');

      // 1. Peek pending cache.
      const pending = queryClient.getQueryData<OrdersResponse>(['chef', 'orders', 'pending']);
      const hitPending = pending?.orders.find((o) => o.id === orderId);
      if (hitPending) return hitPending;

      // 2. Peek any history page cache.
      const historyEntries = queryClient.getQueriesData<OrdersResponse>({
        queryKey: ['chef', 'orders', 'history'],
      });
      for (const [, data] of historyEntries) {
        const hit = data?.orders.find((o) => o.id === orderId);
        if (hit) return hit;
      }

      // 3. Refetch pending. Covers the push-notification deep-link case
      // where the cache was cold (app launched from a kill state).
      const freshPending = await api
        .get<OrdersResponse>('/chef/orders?status=pending&page=1')
        .then((r) => r.data);
      const hitFreshPending = freshPending.orders.find((o) => o.id === orderId);
      queryClient.setQueryData(['chef', 'orders', 'pending'], freshPending);
      if (hitFreshPending) return hitFreshPending;

      // 4. Refetch the first page of history. Covers a chef opening their
      // most recent completed order from history.
      const freshHistory = await api
        .get<OrdersResponse>('/chef/orders?page=1')
        .then((r) => r.data);
      const hitFreshHistory = freshHistory.orders.find((o) => o.id === orderId);
      if (hitFreshHistory) return hitFreshHistory;

      throw new Error('Order not found in cache or recent history');
    },
    staleTime: 5_000,
  });
}
