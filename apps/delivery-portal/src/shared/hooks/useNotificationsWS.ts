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

function getWSUrl(accessToken: string | null): string {
  const base = BFF_URL.replace(/^http/, 'ws');
  // Browsers can't set Authorization headers on WebSocket handshakes, so for
  // JWT (email/password) sessions we pass the token via query param. The
  // backend only honors ?token= on Upgrade: websocket requests.
  const qs = accessToken ? `?token=${encodeURIComponent(accessToken)}` : '';
  return `${base}/api/v1/notifications/ws${qs}`;
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

  const pollUnreadCount = useCallback(async () => {
    try {
      const accessToken = await readAccessToken();
      const headers: Record<string, string> = {};
      if (accessToken) headers['Authorization'] = `Bearer ${accessToken}`;
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

  const connect = useCallback(async () => {
    if (!enabled || wsRef.current?.readyState === WebSocket.OPEN) return;

    try {
      const accessToken = await readAccessToken();
      const ws = new WebSocket(getWSUrl(accessToken));
      wsRef.current = ws;

      ws.onopen = () => {
        setConnected(true);
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
        // Reconnect after 5 seconds
        reconnectTimer.current = setTimeout(connect, 5000);
        // Start polling as fallback
        if (!pollTimer.current) {
          pollTimer.current = setInterval(pollUnreadCount, 30000);
          pollUnreadCount();
        }
      };

      ws.onerror = () => {
        ws.close();
      };
    } catch {
      // WS not available — fall back to polling
      if (!pollTimer.current) {
        pollTimer.current = setInterval(pollUnreadCount, 30000);
        pollUnreadCount();
      }
    }
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
