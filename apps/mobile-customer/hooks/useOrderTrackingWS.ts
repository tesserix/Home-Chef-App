import { useEffect, useRef, useState, useCallback } from 'react';
import { useOrderTracking } from './useOrderTracking';

const MAX_WS_FAILURES = 3;
const RECONNECT_DELAY_MS = 2000;

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
 * consecutive WebSocket failures (T-04-10: no infinite reconnect loop).
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
    if (!orderId || !enabled || useFallback) return;

    // Build WebSocket URL from API base URL (replace http(s) with ws(s))
    const apiBase =
      process.env.EXPO_PUBLIC_API_URL ?? 'https://api.homechef.app';
    const wsBase = apiBase.replace(/^https?:\/\//, (match: string) =>
      match.startsWith('https') ? 'wss://' : 'ws://',
    );
    const url = `${wsBase}/api/v1/orders/${orderId}/track/ws`;

    const ws = new WebSocket(url);
    wsRef.current = ws;

    ws.onopen = () => {
      failureCount.current = 0; // Reset on successful connect
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
      if (failureCount.current >= MAX_WS_FAILURES) {
        setUseFallback(true); // Give up on WS — switch to polling
      }
    };

    ws.onclose = () => {
      if (failureCount.current < MAX_WS_FAILURES) {
        // Reconnect after delay unless failure cap reached
        reconnectTimer.current = setTimeout(connect, RECONNECT_DELAY_MS);
      }
    };
  }, [orderId, enabled, useFallback]);

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

  return {
    /** Real-time driver location from WebSocket (null until first message) */
    driverLocation,
    /** True when WS failed 3 times and hook fell back to REST polling */
    isPollingFallback: useFallback,
    /** Polling result — populated only when isPollingFallback is true */
    pollingResult,
  };
}
