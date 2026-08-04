import { useEffect, useRef, useState, useCallback } from 'react';
import { AppState, type AppStateStatus } from 'react-native';
import { socketReconnectDelayWithJitterMs } from '@homechef/mobile-shared/utils';
import { openEventStream, type EventStreamHandle } from '@homechef/mobile-shared/realtime';
import { useAuthStore } from '../store/auth-store';
import { useOrderTracking } from './useOrderTracking';

const MAX_STREAM_FAILURES = 3;

interface DriverLocation {
  latitude: number;
  longitude: number;
  timestamp: string;
}

interface LocationFrame {
  latitude: number;
  longitude: number;
  timestamp: string;
}

/**
 * Live driver location for one order, falling back to REST polling after 3
 * consecutive stream failures (T-04-10: no unbounded-frequency retry — the
 * polling fallback keeps location fresh while the stream is down).
 *
 * SSE, not WebSocket. React Native does TLS through SocketRocket/CFStream and
 * that handshake fails outright against our edge (close 1006 / OSStatus -9836),
 * so no HTTP request is ever issued — which is why those attempts appear nowhere
 * in the API logs. Notifications and order status moved to SSE when that was
 * found (#982/#983); this hook did not, so live tracking spent its budget on
 * calls that could never land and every order ran on polling. The 4 Aug test run
 * measured 33 consecutive failures on a single order.
 *
 * The stream keeps retrying with capped backoff even while the polling fallback
 * is active, so a connectivity blip recovers to real-time instead of leaving the
 * session stuck on REST (#892).
 */
export function useOrderTrackingWS(orderId: string, enabled: boolean = true) {
  const streamRef = useRef<EventStreamHandle | null>(null);
  const failureCount = useRef(0);
  const reconnectTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  // Distinguishes the connect attempt that owns streamRef from one superseded
  // while it was starting — without it an unmount leaves an orphan stream.
  const generation = useRef(0);
  const [useFallback, setUseFallback] = useState(false);
  const [driverLocation, setDriverLocation] = useState<DriverLocation | null>(null);

  // Polling fallback — only active once the stream has failed MAX_STREAM_FAILURES times.
  const pollingResult = useOrderTracking(orderId, enabled && useFallback);

  const connect = useCallback(() => {
    if (!orderId || !enabled) return;
    const token = useAuthStore.getState().accessToken;
    if (!token) return;
    const apiBase = process.env.EXPO_PUBLIC_API_URL ?? 'https://fe3dr.com/api';
    const myGeneration = ++generation.current;

    const fail = () => {
      if (myGeneration !== generation.current || !enabled) return;
      streamRef.current = null;
      failureCount.current += 1;
      // Past the budget the screen runs on polling — but keep dialling, so a
      // stream that recovers takes over again (#892).
      if (failureCount.current >= MAX_STREAM_FAILURES) setUseFallback(true);
      reconnectTimer.current = setTimeout(
        connect,
        socketReconnectDelayWithJitterMs(failureCount.current),
      );
    };

    const onFrame = (raw: string) => {
      failureCount.current = 0;
      // A frame arriving proves the stream works: drop the polling fallback if
      // it was active, rather than paying for both for the rest of the trip.
      setUseFallback(false);
      try {
        const data = JSON.parse(raw) as LocationFrame;
        if (typeof data.latitude !== 'number' || typeof data.longitude !== 'number') return;
        setDriverLocation({
          latitude: data.latitude,
          longitude: data.longitude,
          timestamp: data.timestamp,
        });
      } catch {
        // Ignore malformed frames.
      }
    };

    streamRef.current = openEventStream(
      `${apiBase}/v1/orders/${orderId}/track/sse`,
      token,
      onFrame,
      fail,
    );
  }, [orderId, enabled]);

  useEffect(() => {
    connect();
    return () => {
      generation.current += 1;
      streamRef.current?.close();
      streamRef.current = null;
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
      if (!streamRef.current) connect();
    });
    return () => sub.remove();
  }, [enabled, connect]);

  return {
    /** Real-time driver location from the stream (null until the first frame) */
    driverLocation,
    /** True once the stream failed MAX_STREAM_FAILURES times and REST polling took over */
    isPollingFallback: useFallback,
    /** Polling result — populated only when isPollingFallback is true */
    pollingResult,
  };
}
