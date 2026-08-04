import { useEffect, useRef, useState, useCallback } from 'react';
import { AppState, type AppStateStatus } from 'react-native';
import { socketReconnectDelayWithJitterMs } from '@homechef/mobile-shared/utils';
import { fetchWSTicket, wsEndpointUrl } from '@homechef/mobile-shared/realtime';
import { api } from '../lib/api';
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
 *
 * Authentication is a short-lived ticket minted over the authenticated REST
 * path and spent in the query string of a top-level `/ws/*` URL. This hook
 * previously opened a bare socket against `/api/v1/.../track/ws` with no
 * credential at all, so every attempt 401'd and live tracking has in practice
 * always run on the polling fallback (#982).
 */
export function useOrderTrackingWS(orderId: string, enabled: boolean = true) {
  const wsRef = useRef<WebSocket | null>(null);
  const failureCount = useRef(0);
  const reconnectTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  // Distinguishes the connect attempt that owns wsRef from one superseded
  // while it was awaiting a ticket — without it an unmount during that await
  // leaves an orphan socket that nothing closes.
  const generation = useRef(0);
  const [useFallback, setUseFallback] = useState(false);
  const [driverLocation, setDriverLocation] = useState<DriverLocation | null>(null);

  // Polling fallback — only active when WS has failed MAX_WS_FAILURES times
  const pollingResult = useOrderTracking(orderId, enabled && useFallback);

  const connect = useCallback(() => {
    if (!orderId || !enabled) return;
    const apiBase = process.env.EXPO_PUBLIC_API_URL ?? 'https://fe3dr.com/api';
    const myGeneration = ++generation.current;

    const fail = () => {
      failureCount.current += 1;
      if (failureCount.current >= MAX_WS_FAILURES) setUseFallback(true);
      reconnectTimer.current = setTimeout(
        connect,
        socketReconnectDelayWithJitterMs(failureCount.current),
      );
    };

    void (async () => {
      const ticket = await fetchWSTicket(api);
      if (myGeneration !== generation.current || !enabled) return;
      // A failed mint is a failure like any other: same backoff, same fallback,
      // never a silent dead end.
      if (!ticket) {
        fail();
        return;
      }

      const url = wsEndpointUrl(apiBase, `orders/${orderId}/track`, ticket);
      const ws = new WebSocket(url);
      wsRef.current = ws;

      ws.onopen = () => {
        failureCount.current = 0; // Reset on successful connect
        // Real-time recovered: drop the polling fallback if it was active
        // (#892 — a socket that only fails once must be able to come back).
        setUseFallback(false);
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

      // Always reschedule while enabled — the polling fallback covers the gap,
      // but the socket itself must never permanently give up or it can never
      // recover to real-time (#892). `onerror` is deliberately not counted
      // separately: it is always followed by `onclose`, and counting both
      // double-incremented the failure count and halved the real backoff.
      ws.onclose = () => {
        if (myGeneration !== generation.current || !enabled) return;
        wsRef.current = null;
        fail();
      };
    })();
  }, [orderId, enabled]);

  useEffect(() => {
    connect();
    return () => {
      generation.current += 1;
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
