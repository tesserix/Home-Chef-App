import { useEffect, useRef, useState, useCallback } from 'react';
import { AppState, type AppStateStatus } from 'react-native';
import { socketReconnectDelayMs } from '@homechef/mobile-shared/utils';
import { useOrderTracking } from './useOrderTracking';

const MAX_WS_FAILURES = 3;

interface DriverLocation {
  latitude: number;
  longitude: number;
  timestamp: string;
}

interface WSLocationMessage {
  latitude: number;
  longitude: number;
  timestamp: string;
}

/**
 * WebSocket-based order tracking hook that subscribes to real-time driver
 * location updates. Falls back to polling via useOrderTracking after 3
 * consecutive WebSocket failures (T-04-10: no unbounded-frequency retry —
 * the polling fallback keeps location fresh while the socket is down).
 *
 * The socket keeps retrying with capped backoff even while the polling
 * fallback is active, so a connectivity blip can recover to real-time
 * updates instead of being stuck on REST polling for the rest of the
 * session (#892).
 */
export function useOrderTrackingWS(orderId: string, enabled: boolean = true) {
  const wsRef = useRef<WebSocket | null>(null);
  const failureCount = useRef(0);
  const reconnectTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const [useFallback, setUseFallback] = useState(false);
  const [driverLocation, setDriverLocation] = useState<DriverLocation | null>(null);

  // Polling fallback — only active when WS has failed MAX_WS_FAILURES times
  const pollingResult = useOrderTracking(orderId, enabled && useFallback);

  const connect = useCallback(() => {
    if (!orderId || !enabled) return;

    // Build WebSocket URL from API base URL (replace http(s) with ws(s)).
    // EXPO_PUBLIC_API_URL already ends in `/api` (e.g. https://fe3dr.com/api),
    // so the path is `/v1/...` — NOT `/api/v1/...`, which doubled to
    // `.../api/api/v1/...` and made every WS connect fail (silent fall back to
    // REST polling). The REST tracking hook uses `/v1/...` on the same base.
    const apiBase = process.env.EXPO_PUBLIC_API_URL ?? 'https://fe3dr.com/api';
    const wsBase = apiBase.replace(/^https?:\/\//, (match: string) =>
      match.startsWith('https') ? 'wss://' : 'ws://',
    );
    const url = `${wsBase}/v1/orders/${orderId}/track/ws`;

    const ws = new WebSocket(url);
    wsRef.current = ws;

    ws.onopen = () => {
      failureCount.current = 0; // Reset on successful connect
      // Real-time recovered: drop the polling fallback if it was active
      // (#892 — a socket that only fails once must be able to come back).
      setUseFallback((wasFallback) => {
        if (wasFallback) {
          console.info(`[tracking-ws] reconnected on order ${orderId} — leaving polling fallback`);
        } else {
          console.info(`[tracking-ws] connected (order ${orderId})`);
        }
        return false;
      });
    };

    ws.onmessage = (event: WebSocketMessageEvent) => {
      failureCount.current = 0;
      try {
        const data = JSON.parse(event.data as string) as WSLocationMessage;
        setDriverLocation({
          latitude: data.latitude,
          longitude: data.longitude,
          timestamp: data.timestamp,
        });
      } catch {
        // Ignore malformed messages
      }
    };

    ws.onerror = () => {
      failureCount.current += 1;
      console.warn(
        `[tracking-ws] error on order ${orderId} (${failureCount.current}/${MAX_WS_FAILURES} failures)`,
      );
      if (failureCount.current >= MAX_WS_FAILURES) {
        console.error(
          `[tracking-ws] falling back to polling on order ${orderId} after ${failureCount.current} consecutive failures — still retrying the socket in the background`,
        );
        setUseFallback(true); // Fall back to polling, but keep retrying WS below.
      }
    };

    // Always reschedule while enabled — the polling fallback (activated above
    // once MAX_WS_FAILURES is hit) covers the gap, but the socket itself must
    // never permanently give up or it can never recover to real-time (#892).
    ws.onclose = () => {
      if (!enabled) return;
      const delay = socketReconnectDelayMs(failureCount.current);
      console.warn(`[tracking-ws] closed on order ${orderId}, reconnecting in ${delay}ms`);
      reconnectTimer.current = setTimeout(connect, delay);
    };
  }, [orderId, enabled]);

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

  return {
    /** Real-time driver location from WebSocket (null until first message) */
    driverLocation,
    /** True when WS failed 3 times and hook fell back to REST polling */
    isPollingFallback: useFallback,
    /** Polling result — populated only when isPollingFallback is true */
    pollingResult,
  };
}
