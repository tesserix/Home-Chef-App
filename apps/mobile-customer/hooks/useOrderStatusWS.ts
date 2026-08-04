import { useEffect, useRef, useCallback } from 'react';
import { AppState, type AppStateStatus } from 'react-native';
import { useQueryClient } from '@tanstack/react-query';
import { socketReconnectDelayWithJitterMs } from '@homechef/mobile-shared/utils';
import { fetchWSTicket, wsEndpointUrl } from '@homechef/mobile-shared/realtime';
import { api } from '../lib/api';
import type { Order } from '../types/customer';

interface NotificationWSMessage {
  type?: string;
  data?: string; // JSON string: { order_id, status }
}

/**
 * Subscribes to the user's real-time stream and flips cached state the moment the
 * chef acts — so the customer sees "Confirmed" / "Preparing" / "Ready", and a plan
 * their chef just answered, instantly instead of waiting for the poll.
 * The polls (`useOrder` on the detail screen, `useActiveOrder` on Home) remain
 * the fallback if the socket can't connect or drops.
 *
 * Pass an `orderId` to track one order (detail screen): its `['order', id]`
 * cache is flipped optimistically, then reconciled.
 *
 * Omit it to track *every* active order (Home): the stream is user-scoped, so
 * one socket covers the whole ActiveOrderStack — any order notification just
 * invalidates `['orders']`, which is what useActiveOrder reads. Refetching is
 * enough here because the card renders from that list, and it avoids
 * hand-patching a paginated cache shape.
 */
export function useOrderStatusWS(
  orderId?: string | null,
  enabled: boolean = true,
): void {
  const queryClient = useQueryClient();
  const wsRef = useRef<WebSocket | null>(null);
  const failureCount = useRef(0);
  const reconnectTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  // Guards the async ticket fetch against unmount / supersession.
  const generation = useRef(0);

  const connect = useCallback(() => {
    if (!enabled) return;
    // Authenticated with a short-lived ticket on a top-level `/ws/*` URL. The
    // previous `Authorization` header on `/api/v1/...` never authenticated the
    // upgrade — it 401'd on every attempt (#982).
    const apiBase = process.env.EXPO_PUBLIC_API_URL ?? 'https://fe3dr.com/api';
    const myGeneration = ++generation.current;

    void (async () => {
    const ticket = await fetchWSTicket(api);
    if (myGeneration !== generation.current || !enabled) return;
    if (!ticket) {
      failureCount.current += 1;
      reconnectTimer.current = setTimeout(
        connect,
        socketReconnectDelayWithJitterMs(failureCount.current),
      );
      return;
    }

    const url = wsEndpointUrl(apiBase, 'notifications', ticket);
    const ws = new WebSocket(url);
    wsRef.current = ws;

    ws.onopen = () => {
      failureCount.current = 0;
      console.info(
        `[order-ws] connected (${orderId ? 'single-order' : 'any-order'} mode)`,
      );
    };

    ws.onmessage = (event: WebSocketMessageEvent) => {
      failureCount.current = 0;
      try {
        const msg = JSON.parse(event.data as string) as NotificationWSMessage;
        if (msg.type !== 'new_notification' || !msg.data) return;
        const payload = JSON.parse(msg.data) as {
          order_id?: string;
          meal_plan_id?: string;
          status?: Order['status'];
        };

        // Meal-plan traffic rides the same user-scoped stream (the chef answering a
        // request, an advance confirming, a plan cancelling). Dropping it for want of
        // an order_id is why a customer had to leave the screen and come back to see
        // that their chef had replied.
        if (payload.meal_plan_id) {
          void queryClient.invalidateQueries({ queryKey: ['meal-plans'] });
          return;
        }
        if (!payload.order_id) return;
        // Single-order mode: ignore other orders. Any-order mode (no orderId)
        // takes every one, which is how the Home stack stays live.
        if (orderId && payload.order_id !== orderId) return;

        // Optimistic flip so the chip/tracker move immediately, then refetch the
        // full order (prices, times) to reconcile. Only meaningful when we know
        // which order's detail cache to patch.
        if (orderId && payload.status) {
          queryClient.setQueryData<{ data: Order }>(['order', orderId], (prev) =>
            prev ? { data: { ...prev.data, status: payload.status as Order['status'] } } : prev,
          );
        }
        // Always refresh the notified order's own cache — on Home this keeps a
        // detail screen the customer opens next from showing a stale status.
        void queryClient.invalidateQueries({ queryKey: ['order', payload.order_id] });
        void queryClient.invalidateQueries({ queryKey: ['orders'] });
      } catch {
        // Ignore malformed / non-order messages (e.g. the initial unread_count).
      }
    };

    // This stream has no fallback (unlike order-tracking's polling or the
    // vendor app's SSE), so giving up permanently here is the worst case of
    // #892 — a customer could go a whole session with no live updates.
    // Retry indefinitely with backoff instead; there is no cap to hit.
    // `onerror` is not counted separately: it is always followed by `onclose`,
    // and counting both double-incremented the count and halved the backoff.
    ws.onclose = () => {
      if (myGeneration !== generation.current || !enabled) return;
      wsRef.current = null;
      failureCount.current += 1;
      reconnectTimer.current = setTimeout(
        connect,
        socketReconnectDelayWithJitterMs(failureCount.current),
      );
    };
    })();
  }, [orderId, enabled, queryClient]);

  useEffect(() => {
    connect();
    return () => {
      wsRef.current?.close();
      wsRef.current = null;
      if (reconnectTimer.current) {
        clearTimeout(reconnectTimer.current);
        reconnectTimer.current = null;
      }
    };
  }, [connect]);

  // Foreground reconnect: don't leave the user waiting out a stale backoff
  // after the app was backgrounded for a while — retry right away (#892).
  useEffect(() => {
    if (!enabled) return;
    const sub = AppState.addEventListener('change', (status: AppStateStatus) => {
      if (status !== 'active') return;
      if (reconnectTimer.current) {
        clearTimeout(reconnectTimer.current);
        reconnectTimer.current = null;
      }
      failureCount.current = 0;
      if (!wsRef.current || wsRef.current.readyState === WebSocket.CLOSED) {
        connect();
      }
    });
    return () => sub.remove();
  }, [enabled, connect]);
}
