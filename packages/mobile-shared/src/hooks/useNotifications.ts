// Shared notification-center data + real-time socket, used by both the customer
// and vendor apps. The backend feed is fully built (GET /v1/notifications,
// /unread-count, PUT /:id/read, /read-all, and a per-user WebSocket at
// /v1/notifications/ws) — these hooks are the mobile client for it.
//
// Design-system UI (the bell + list rows) lives per-app because the two apps
// theme and deep-link differently; only the data logic is shared here. Each app
// passes its own axios `api` instance + a token getter (the WS authenticates
// with the mobile Bearer token, like useOrderStatusWS).

import { useCallback, useEffect, useRef } from 'react';
import {
  useQuery,
  useMutation,
  useQueryClient,
} from '@tanstack/react-query';
import type { AxiosInstance } from 'axios';

export interface AppNotification {
  id: string;
  type: string;
  title: string;
  message: string;
  /** Deep-link payload (order_id, chefId, dayId, …). Backend stores it as a
   * JSON string; some rows send an object. Normalised by `parseNotificationData`. */
  data?: string | Record<string, string> | null;
  isRead: boolean;
  readAt?: string | null;
  createdAt: string;
}

export const NOTIFICATION_LIST_KEY = ['notifications', 'list'] as const;
export const NOTIFICATION_UNREAD_KEY = ['notifications', 'unread'] as const;

// The two apps configure EXPO_PUBLIC_API_URL differently — the customer's ends
// in `/api` (so paths carry the `/v1` version), the vendor's already ends in
// `/api/v1` (so paths must NOT repeat it). Derive the version prefix from the
// base URL so the same shared paths work in both, instead of hardcoding `/v1`
// (which double-prefixed the vendor → 404 → an empty, silent feed).
function versionPrefix(baseUrl: string | undefined | null): string {
  return baseUrl && /\/v\d+\/?$/.test(baseUrl) ? '' : '/v1';
}

function restPrefix(api: AxiosInstance): string {
  return versionPrefix(api.defaults?.baseURL);
}

/** Normalise a notification's `data` (string or object) into a flat record. */
export function parseNotificationData(
  n: AppNotification,
): Record<string, string> {
  if (!n.data) return {};
  if (typeof n.data === 'string') {
    try {
      return JSON.parse(n.data) as Record<string, string>;
    } catch {
      return {};
    }
  }
  return n.data;
}

/** The user's notification feed, newest first. */
export function useNotificationList(api: AxiosInstance) {
  return useQuery({
    queryKey: NOTIFICATION_LIST_KEY,
    queryFn: () =>
      api
        .get<{ data: AppNotification[] }>(`${restPrefix(api)}/notifications`)
        .then((r) => r.data.data ?? []),
    staleTime: 30_000,
  });
}

/** Unread count for the bell badge. */
export function useUnreadCount(api: AxiosInstance) {
  return useQuery({
    queryKey: NOTIFICATION_UNREAD_KEY,
    queryFn: () =>
      api
        .get<{ unreadCount: number }>(`${restPrefix(api)}/notifications/unread-count`)
        .then((r) => r.data.unreadCount ?? 0),
    staleTime: 15_000,
  });
}

/** Mark a single notification read (on tap). Optimistic so the row + badge
 * update instantly. */
export function useMarkNotificationRead(api: AxiosInstance) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.put(`${restPrefix(api)}/notifications/${id}/read`),
    onMutate: async (id) => {
      await qc.cancelQueries({ queryKey: NOTIFICATION_LIST_KEY });
      const prevList = qc.getQueryData<AppNotification[]>(NOTIFICATION_LIST_KEY);
      const prevCount = qc.getQueryData<number>(NOTIFICATION_UNREAD_KEY);
      qc.setQueryData<AppNotification[]>(NOTIFICATION_LIST_KEY, (l) =>
        (l ?? []).map((n) => (n.id === id && !n.isRead ? { ...n, isRead: true } : n)),
      );
      const wasUnread = (prevList ?? []).some((n) => n.id === id && !n.isRead);
      if (wasUnread && typeof prevCount === 'number') {
        qc.setQueryData<number>(NOTIFICATION_UNREAD_KEY, Math.max(0, prevCount - 1));
      }
      return { prevList, prevCount };
    },
    onError: (_e, _id, ctx) => {
      if (ctx?.prevList) qc.setQueryData(NOTIFICATION_LIST_KEY, ctx.prevList);
      if (typeof ctx?.prevCount === 'number')
        qc.setQueryData(NOTIFICATION_UNREAD_KEY, ctx.prevCount);
    },
    onSettled: () => {
      qc.invalidateQueries({ queryKey: NOTIFICATION_UNREAD_KEY });
    },
  });
}

/** Mark every notification read (the "Mark all read" action). */
export function useMarkAllNotificationsRead(api: AxiosInstance) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: () => api.put(`${restPrefix(api)}/notifications/read-all`),
    onSuccess: () => {
      qc.setQueryData<AppNotification[]>(NOTIFICATION_LIST_KEY, (l) =>
        (l ?? []).map((n) => ({ ...n, isRead: true })),
      );
      qc.setQueryData<number>(NOTIFICATION_UNREAD_KEY, 0);
    },
  });
}

type WSCtor = {
  new (
    url: string,
    protocols?: string | string[],
    options?: { headers?: Record<string, string> },
  ): WebSocket;
};

const MAX_WS_FAILURES = 4;
const RECONNECT_DELAY_MS = 3000;

/**
 * Holds the user's real-time notification socket. On any server message
 * (`unread_count` on connect, `new_notification` thereafter) it refreshes the
 * bell's list + count queries, so a new notification lights the bell instantly.
 * The REST queries above remain the fallback if the socket can't connect.
 *
 * `apiBaseUrl` is the app's EXPO_PUBLIC_API_URL (ends in `/api`); the WS path is
 * `/v1/notifications/ws`. `getToken` returns the current Bearer token.
 */
export function useNotificationSocket(opts: {
  apiBaseUrl: string | undefined;
  getToken: () => string | null | undefined;
  enabled?: boolean;
}): void {
  const { apiBaseUrl, getToken, enabled = true } = opts;
  const qc = useQueryClient();
  const wsRef = useRef<WebSocket | null>(null);
  const failures = useRef(0);
  const reconnectTimer = useRef<ReturnType<typeof setTimeout> | null>(null);

  const connect = useCallback(() => {
    if (!enabled) return;
    const token = getToken();
    if (!token || !apiBaseUrl) return;

    const wsBase = apiBaseUrl.replace(/^https?:\/\//, (m: string) =>
      m.startsWith('https') ? 'wss://' : 'ws://',
    );
    const url = `${wsBase}${versionPrefix(apiBaseUrl)}/notifications/ws`;

    const ws = new (WebSocket as unknown as WSCtor)(url, undefined, {
      headers: { Authorization: `Bearer ${token}` },
    });
    wsRef.current = ws;

    ws.onopen = () => {
      failures.current = 0;
    };
    ws.onmessage = () => {
      failures.current = 0;
      // The socket signals "something changed" (a new notification, or the
      // initial count). Refetch the two feed queries — cheap, and avoids
      // hand-patching the list/count from a message whose shape varies by event.
      qc.invalidateQueries({ queryKey: NOTIFICATION_LIST_KEY });
      qc.invalidateQueries({ queryKey: NOTIFICATION_UNREAD_KEY });
    };
    ws.onerror = () => {
      failures.current += 1;
    };
    ws.onclose = () => {
      wsRef.current = null;
      if (!enabled) return;
      if (failures.current >= MAX_WS_FAILURES) return; // give up; REST polling covers it
      reconnectTimer.current = setTimeout(connect, RECONNECT_DELAY_MS);
    };
  }, [apiBaseUrl, getToken, enabled, qc]);

  useEffect(() => {
    connect();
    return () => {
      if (reconnectTimer.current) clearTimeout(reconnectTimer.current);
      wsRef.current?.close();
      wsRef.current = null;
    };
  }, [connect]);
}
