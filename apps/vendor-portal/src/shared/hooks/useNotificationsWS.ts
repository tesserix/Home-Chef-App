import { useState, useEffect, useRef, useCallback } from 'react';

interface NotificationMessage {
  type: 'unread_count' | 'new_notification';
  unreadCount: number;
  id?: string;
  title?: string;
  message?: string;
}

const BFF_URL = (() => {
  const env = import.meta.env.VITE_BFF_URL;
  if (env) return env;
  if (typeof window !== 'undefined' && window.location.hostname !== 'localhost') {
    return `${window.location.origin}/bff`;
  }
  return '/bff';
})();

// WebSockets go to same-origin /ws/notifications, which the Istio
// VirtualService forwards directly to homechef-api with a 3600s upgrade
// timeout. /bff/ can't proxy WS upgrades (Node/Express limitation).
function getWSUrl(ticket: string): string {
  const origin =
    typeof window !== 'undefined' && window.location.hostname !== 'localhost'
      ? window.location.origin.replace(/^http/, 'ws')
      : 'ws://localhost:8080';
  return `${origin}/ws/notifications?ticket=${encodeURIComponent(ticket)}`;
}

/**
 * Mints a short-lived WebSocket ticket over the authenticated BFF path.
 *
 * This used to pass the raw access token as `?token=`, which the server never
 * read — the upgrade was rejected on every attempt and the bell has always run
 * on its polling fallback (#982). A browser cannot set a handshake header, so
 * the credential must ride in the query string; a 60s ticket is the narrow,
 * expiring stand-in for a session token that should never appear in a URL.
 */
async function mintWSTicket(accessToken: string | null): Promise<string | null> {
  try {
    const headers: Record<string, string> = {};
    if (accessToken) headers['X-Auth-Token'] = accessToken;
    const res = await fetch(`${BFF_URL}/api/v1/realtime/ws-ticket`, {
      method: 'POST',
      credentials: 'include',
      headers,
    });
    if (!res.ok) return null;
    const data = (await res.json()) as { ticket?: string };
    return data.ticket ?? null;
  } catch {
    return null;
  }
}

async function readAccessToken(): Promise<string | null> {
  const { useAuthStore } = await import('@/app/store/auth-store');
  return useAuthStore.getState().accessToken;
}

/**
 * WebSocket hook for real-time notification bell updates.
 * Falls back to polling if WebSocket connection fails.
 */
export function useNotificationsWS(enabled = true) {
  const [unreadCount, setUnreadCount] = useState(0);
  const [lastNotification, setLastNotification] = useState<NotificationMessage | null>(null);
  const [connected, setConnected] = useState(false);
  const wsRef = useRef<WebSocket | null>(null);
  const reconnectTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const pollTimer = useRef<ReturnType<typeof setInterval> | null>(null);
  const failures = useRef(0);

  const pollUnreadCount = useCallback(async () => {
    try {
      const accessToken = await readAccessToken();
      const headers: Record<string, string> = {};
      if (accessToken) headers['X-Auth-Token'] = accessToken;
      const res = await fetch(`${BFF_URL}/api/v1/notifications/unread-count`, {
        credentials: 'include',
        headers,
      });
      if (res.ok) {
        const data = await res.json();
        setUnreadCount(data.unreadCount ?? 0);
      }
    } catch {
      // Silently fail — next poll will retry
    }
  }, []);

  const startPolling = useCallback(() => {
    if (pollTimer.current) return;
    pollTimer.current = setInterval(pollUnreadCount, 30000);
    void pollUnreadCount();
  }, [pollUnreadCount]);

  const connect = useCallback(async () => {
    if (!enabled || wsRef.current?.readyState === WebSocket.OPEN) return;

    // Capped exponential backoff with jitter, rather than a fixed 5s: every
    // client dropped by one server-side event would otherwise retry in the
    // same second, forever.
    const retryAfterFailure = () => {
      failures.current += 1;
      const base = Math.min(30_000, 1_000 * 2 ** Math.max(0, failures.current - 1));
      reconnectTimer.current = setTimeout(connect, Math.round(base * (1 - 0.3 * Math.random())));
      startPolling();
    };

    try {
      const accessToken = await readAccessToken();
      const ticket = await mintWSTicket(accessToken);
      // A failed mint never opens a socket, so `onclose` — which is what
      // schedules the next attempt — can never fire. Without rescheduling here
      // the bell went permanently dead on one bad mint, with no polling either.
      if (!ticket) {
        retryAfterFailure();
        return;
      }
      const ws = new WebSocket(getWSUrl(ticket));
      wsRef.current = ws;

      ws.onopen = () => {
        setConnected(true);
        failures.current = 0;
        // Stop polling — WS is live
        if (pollTimer.current) {
          clearInterval(pollTimer.current);
          pollTimer.current = null;
        }
      };

      ws.onmessage = (event) => {
        try {
          const msg: NotificationMessage = JSON.parse(event.data);
          if (msg.unreadCount !== undefined) {
            setUnreadCount(msg.unreadCount);
          }
          if (msg.type === 'new_notification') {
            setLastNotification(msg);
          }
        } catch {
          // Ignore malformed messages
        }
      };

      ws.onclose = () => {
        setConnected(false);
        wsRef.current = null;
        retryAfterFailure();
      };

      ws.onerror = () => {
        ws.close(); // onclose does the counting and rescheduling
      };
    } catch {
      // The constructor itself threw — no socket, so nothing will call onclose.
      retryAfterFailure();
    }
  }, [enabled, startPolling]);

  useEffect(() => {
    if (!enabled) return;

    // Initial poll for immediate unread count, then connect WS
    pollUnreadCount();
    connect();

    return () => {
      if (wsRef.current) {
        wsRef.current.close();
        wsRef.current = null;
      }
      if (reconnectTimer.current) {
        clearTimeout(reconnectTimer.current);
      }
      if (pollTimer.current) {
        clearInterval(pollTimer.current);
      }
    };
  }, [enabled, connect, pollUnreadCount]);

  return { unreadCount, lastNotification, connected };
}
