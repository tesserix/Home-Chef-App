import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useRef, useEffect, useState } from 'react';
import * as Haptics from 'expo-haptics';
import { multipartConfig } from '@homechef/mobile-shared/api';
import { api } from '../lib/api';
import type { DashboardData, RecentOrder } from './useVendorDashboard';

/** Lifecycle photo kinds the chef attaches to an order. */
export type OrderPhotoKind = 'ready' | 'handover';

// Uploads a required lifecycle photo (food-ready or proof-of-handover) for an
// order. Returns the public URL. The order-detail screen calls this BEFORE the
// matching status mutation, so a failed upload leaves the order in its current
// state (the chef just retries) rather than advancing without a photo.
export function useUploadOrderPhoto() {
  return useMutation({
    mutationFn: async (vars: {
      orderId: string;
      kind: OrderPhotoKind;
      uri: string;
    }) => {
      const formData = new FormData();
      const filename = vars.uri.split('/').pop() ?? `${vars.kind}.jpg`;
      const type = filename.toLowerCase().endsWith('.png')
        ? 'image/png'
        : 'image/jpeg';
      formData.append('file', {
        uri: vars.uri,
        name: filename,
        type,
      } as unknown as Blob);
      formData.append('kind', vars.kind);
      const res = await api.post<{ kind: OrderPhotoKind; url: string }>(
        `/chef/orders/${vars.orderId}/photos`,
        formData,
        multipartConfig(),
      );
      return res.data;
    },
  });
}

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
  // How the order reaches the customer. 'pickup' → customer collects; the chef
  // confirms handover (ready → delivered). Legacy orders default to delivery.
  fulfillmentType?: 'delivery' | 'chef_delivery' | 'pickup';
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

// Order IDs the chef has optimistically accepted/rejected but whose server
// mutation may still be in its undo window. The pending query filters these out
// so a 10s background poll can't resurrect a card the chef already swiped away
// (the order is still `pending` on the server until the deferred PUT lands).
const actionedOrderIds = new Set<string>();

// Pending orders with 10s polling + new-order haptic detection
export function useVendorPendingOrders() {
  const previousCountRef = useRef(0);

  const query = useQuery<OrdersResponse>({
    queryKey: ['chef', 'orders', 'pending'],
    queryFn: () => api.get<OrdersResponse>('/chef/orders?status=pending&page=1').then((r) => r.data),
    refetchInterval: 10_000,
    refetchIntervalInBackground: false,
    // Hide orders that are mid-action so a background refetch can't re-add them.
    select: (data) => ({
      ...data,
      orders: data.orders.filter((o) => !actionedOrderIds.has(o.id)),
    }),
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
    onError: (_err, vars, context) => {
      // Action failed — let the order resurface so the chef can retry, and drop
      // the optimistic "In Progress" entry we may have added on accept.
      actionedOrderIds.delete(vars.orderId);
      if (context?.previous) {
        queryClient.setQueryData(['chef', 'orders', 'pending'], context.previous);
      }
      queryClient.setQueryData<DashboardData>(['chef', 'dashboard'], (old) =>
        old
          ? { ...old, recentOrders: old.recentOrders.filter((o) => o.id !== vars.orderId) }
          : old,
      );
    },
    onSettled: (_data, _err, vars) => {
      actionedOrderIds.delete(vars.orderId);
      // Refresh the order lists AND the dashboard stats (pending count, today's
      // totals) — the dashboard was previously left stale after accept/reject.
      queryClient.invalidateQueries({ queryKey: ['chef', 'orders'] });
      queryClient.invalidateQueries({ queryKey: ['chef', 'dashboard'] });
    },
  });

  function triggerAction(orderId: string, action: 'accepted' | 'rejected', reason?: string) {
    // Haptic feedback on decisive order action (accept or reject)
    void Haptics.impactAsync(Haptics.ImpactFeedbackStyle.Medium);

    // Mark actioned so background polls can't resurrect the card during the
    // undo window, then optimistically remove it and schedule the API call.
    actionedOrderIds.add(orderId);
    // Grab the order before removing it — for an accept we move it straight
    // into the dashboard "In Progress" list so it doesn't vanish during the
    // undo window and then pop back a few seconds later (the flicker).
    const pending = queryClient.getQueryData<OrdersResponse>(['chef', 'orders', 'pending']);
    const accepted = pending?.orders.find((o) => o.id === orderId);
    queryClient.setQueryData<OrdersResponse>(['chef', 'orders', 'pending'], (old) => {
      if (!old) return old;
      return {
        ...old,
        orders: old.orders.filter((o) => o.id !== orderId),
      };
    });
    if (action === 'accepted' && accepted) {
      queryClient.setQueryData<DashboardData>(['chef', 'dashboard'], (old) => {
        if (!old || old.recentOrders.some((o) => o.id === orderId)) return old;
        const optimistic: RecentOrder = {
          id: accepted.id,
          customerName: accepted.customerName,
          total: accepted.total,
          status: 'accepted',
          createdAt: accepted.createdAt,
          fulfillmentType: accepted.fulfillmentType,
        };
        return { ...old, recentOrders: [optimistic, ...old.recentOrders] };
      });
    }
    setPendingUndo({ orderId, action });
    if (timerRef.current) clearTimeout(timerRef.current);
    timerRef.current = setTimeout(() => {
      mutation.mutate({ orderId, action, reason });
      setPendingUndo(null);
    }, UNDO_DELAY_MS);
  }

  function handleUndo() {
    if (timerRef.current) clearTimeout(timerRef.current);
    if (pendingUndo) {
      const undoId = pendingUndo.orderId;
      actionedOrderIds.delete(undoId);
      // Pull the order back out of "In Progress" — the accept was undone.
      queryClient.setQueryData<DashboardData>(['chef', 'dashboard'], (old) =>
        old
          ? { ...old, recentOrders: old.recentOrders.filter((o) => o.id !== undoId) }
          : old,
      );
    }
    queryClient.invalidateQueries({ queryKey: ['chef', 'orders', 'pending'] });
    queryClient.invalidateQueries({ queryKey: ['chef', 'dashboard'] });
    setPendingUndo(null);
  }

  return { triggerAction, handleUndo, pendingUndo, isLoading: mutation.isPending };
}

// Generic status mutation used by the order detail screen and Dashboard "In
// Progress" section for the in-flight
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
      carrier,
    }: {
      orderId: string;
      status: Order['status'];
      // Optional carrier choice the chef makes at Mark Ready (self-delivery
      // chefs only): 'chef_delivery' = I'll deliver, 'delivery' = hand to a rider.
      carrier?: 'chef_delivery' | 'delivery';
    }) => api.put(`/chef/orders/${orderId}/status`, { status, carrier }),
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
