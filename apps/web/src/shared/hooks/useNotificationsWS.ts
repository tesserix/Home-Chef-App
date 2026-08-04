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

/**
 * The socket origin. WebSocket routes live at top-level `/ws/*`, which Istio
 * forwards straight to the API — NOT under `/bff`, which cannot proxy an
 * upgrade at all. This hook previously dialled `${BFF_URL}/api/v1/...` and so
 * never connected once (#982).
 */
function wsOrigin(): string {
  if (typeof window !== 'undefined' && window.location.hostname !== 'localhost') {
    return window.location.origin.replace(/^http/, 'ws');
  }
  return 'ws://localhost:8080';
}

/**
 * Mints a short-lived ticket over the authenticated BFF path. A browser cannot
 * set a header on a WebSocket handshake, so the credential has to ride in the
 * query string; the ticket is the narrow, expiring stand-in for the session.
 *
 * Returns null on any failure so the caller can back off like any other
 * connection failure rather than throwing into a render.
 */
async function mintWSTicket(): Promise<string | null> {
  try {
    const res = await fetch(`${BFF_URL}/api/v1/realtime/ws-ticket`, {
      method: 'POST',
      credentials: 'include',
    });
    if (!res.ok) return null;
    const data = (await res.json()) as { ticket?: string };
    return data.ticket ?? null;
  } catch {
    return null;
  }
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
  // Guards the async ticket mint against unmount / supersession.
  const generation = useRef(0);

  const pollUnreadCount = useCallback(async () => {
    try {
      const res = await fetch(`${BFF_URL}/api/v1/notifications/unread-count`, {
        credentials: 'include',
      });
      if (res.ok) {
        const data = await res.json();
        setUnreadCount(data.unreadCount ?? 0);
      }
    } catch {
      // Silently fail — next poll will retry
    }
  }, []);

  const connect = useCallback(() => {
    if (!enabled || wsRef.current?.readyState === WebSocket.OPEN) return;
    const myGeneration = ++generation.current;

    void (async () => {
    const ticket = await mintWSTicket();
    // Signed out or the mint failed: stay on the polling fallback and let the
    // existing reconnect schedule try again.
    if (!ticket || myGeneration !== generation.current || !enabled) return;

    try {
      const ws = new WebSocket(`${wsOrigin()}/ws/notifications?ticket=${encodeURIComponent(ticket)}`);
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
        if (myGeneration !== generation.current) return;
        setConnected(false);
        wsRef.current = null;
        failures.current += 1;
        // Capped exponential backoff with jitter, rather than a fixed 5s: every
        // client dropped by one server-side event would otherwise retry in the
        // same second, forever.
        const base = Math.min(30_000, 1_000 * 2 ** Math.max(0, failures.current - 1));
        reconnectTimer.current = setTimeout(connect, Math.round(base * (1 - 0.3 * Math.random())));
        // Start polling as fallback
        if (!pollTimer.current) {
          pollTimer.current = setInterval(pollUnreadCount, 30000);
          pollUnreadCount();
        }
      };

      ws.onerror = () => {
        ws.close(); // onclose does the counting and rescheduling
      };
    } catch {
      // WS not available — fall back to polling
      if (!pollTimer.current) {
        pollTimer.current = setInterval(pollUnreadCount, 30000);
        pollUnreadCount();
      }
    }
    })();
  }, [enabled, pollUnreadCount]);

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
