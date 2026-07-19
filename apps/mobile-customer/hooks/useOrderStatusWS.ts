import { useEffect, useRef, useCallback } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { useAuthStore } from '../store/auth-store';
import type { Order } from '../types/customer';

const MAX_WS_FAILURES = 4;
const RECONNECT_DELAY_MS = 3000;

// React Native's WebSocket accepts a headers option (a 3rd constructor arg) that
// the DOM type omits — the notification stream is user-scoped and authenticates
// with the mobile Bearer token via bffAuth's verifyBearer fallback.
type WSCtor = {
  new (
    url: string,
    protocols?: string | string[],
    options?: { headers?: Record<string, string> },
  ): WebSocket;
};

interface NotificationWSMessage {
  type?: string;
  data?: string; // JSON string: { order_id, status }
}

/**
 * Subscribes to the user's real-time notification stream and flips the given
 * order's cached status the moment the chef accepts / advances a stage — so the
 * customer sees "Confirmed" / "Preparing" / "Ready" instantly instead of waiting
 * for the poll. The `useOrder` poll (5s while active) remains the fallback if the
 * socket can't connect or drops.
 */
export function useOrderStatusWS(orderId: string, enabled: boolean = true): void {
  const queryClient = useQueryClient();
  const wsRef = useRef<WebSocket | null>(null);
  const failureCount = useRef(0);
  const reconnectTimer = useRef<ReturnType<typeof setTimeout> | null>(null);

  const connect = useCallback(() => {
    if (!orderId || !enabled) return;
    const token = useAuthStore.getState().accessToken;
    if (!token) return;

    // EXPO_PUBLIC_API_URL ends in `/api`, so the WS path is `/v1/...` (mirrors
    // useOrderTrackingWS — a `/api/v1/...` here would double the prefix).
    const apiBase = process.env.EXPO_PUBLIC_API_URL ?? 'https://fe3dr.com/api';
    const wsBase = apiBase.replace(/^https?:\/\//, (match: string) =>
      match.startsWith('https') ? 'wss://' : 'ws://',
    );
    const url = `${wsBase}/v1/notifications/ws`;

    const ws = new (WebSocket as unknown as WSCtor)(url, undefined, {
      headers: { Authorization: `Bearer ${token}` },
    });
    wsRef.current = ws;

    ws.onopen = () => {
      failureCount.current = 0;
    };

    ws.onmessage = (event: WebSocketMessageEvent) => {
      failureCount.current = 0;
      try {
        const msg = JSON.parse(event.data as string) as NotificationWSMessage;
        if (msg.type !== 'new_notification' || !msg.data) return;
        const payload = JSON.parse(msg.data) as { order_id?: string; status?: Order['status'] };
        if (payload.order_id !== orderId) return;

        // Optimistic flip so the chip/tracker move immediately, then refetch the
        // full order (prices, times) to reconcile.
        if (payload.status) {
          queryClient.setQueryData<{ data: Order }>(['order', orderId], (prev) =>
            prev ? { data: { ...prev.data, status: payload.status as Order['status'] } } : prev,
          );
        }
        void queryClient.invalidateQueries({ queryKey: ['order', orderId] });
        void queryClient.invalidateQueries({ queryKey: ['orders'] });
      } catch {
        // Ignore malformed / non-order messages (e.g. the initial unread_count).
      }
    };

    ws.onerror = () => {
      failureCount.current += 1;
    };

    ws.onclose = () => {
      if (enabled && failureCount.current < MAX_WS_FAILURES) {
        reconnectTimer.current = setTimeout(connect, RECONNECT_DELAY_MS);
      }
    };
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
}
