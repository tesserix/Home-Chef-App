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

import { socketReconnectDelayMs } from '../utils/socket-backoff';

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

interface ErrorWithStatus {
  response?: { status?: number };
}

function hasStatus(error: unknown): error is ErrorWithStatus {
  return typeof error === 'object' && error !== null && 'response' in error;
}

/**
 * React Query retry predicate for the notification REST queries. A 401 means
 * the axios interceptor's own single refresh+retry already ran (or the user
 * is genuinely signed out) — retrying it again here would just re-storm the
 * endpoint, so 401s are never retried. Every other error (network, 5xx, 403)
 * keeps the app's existing transient-retry cap of 2.
 */
export function shouldRetryNotificationQuery(
  failureCount: number,
  error: unknown,
): boolean {
  if (hasStatus(error) && error.response?.status === 401) return false;
  return failureCount < 2;
}

/** Options shared by the notification REST queries. */
export interface NotificationQueryOptions {
  /** Gates the query on auth state — callers must pass their auth store's
   * `isAuthenticated`. The query was previously unconditionally mounted,
   * causing a 401 storm for guests (unauthenticated users on Home). */
  enabled?: boolean;
}

/** The user's notification feed, newest first. */
export function useNotificationList(
  api: AxiosInstance,
  options: NotificationQueryOptions = {},
) {
  return useQuery({
    queryKey: NOTIFICATION_LIST_KEY,
    queryFn: () =>
      api
        .get<{ data: AppNotification[] }>(`${restPrefix(api)}/notifications`)
        .then((r) => r.data.data ?? []),
    staleTime: 30_000,
    enabled: options.enabled,
    retry: shouldRetryNotificationQuery,
  });
}

/** Unread count for the bell badge. */
export function useUnreadCount(
  api: AxiosInstance,
  options: NotificationQueryOptions = {},
) {
  return useQuery({
    queryKey: NOTIFICATION_UNREAD_KEY,
    queryFn: () =>
      api
        .get<{ unreadCount: number }>(`${restPrefix(api)}/notifications/unread-count`)
        .then((r) => r.data.unreadCount ?? 0),
    staleTime: 15_000,
    enabled: options.enabled,
    retry: shouldRetryNotificationQuery,
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

/**
 * Fixes #909; mirrors #892's fix already applied to the other three sockets
 * (`useOrderStatusWS`, `useOrderTrackingWS`, vendor `useLiveUpdates`).
 * `onclose` delegates to these two pure functions so the hook and its tests
 * exercise the exact same decision logic instead of parallel copies.
 *
 * Accepts consecutiveFailures so the signature itself documents that #909
 * removed the cap — it never gates on failure count, only on enabled.
 */
export function shouldReconnectNotificationSocket(
  enabled: boolean,
  _consecutiveFailures: number,
): boolean {
  return enabled;
}

/** Delay before the next reconnect attempt, per the shared capped-exponential
 * backoff curve (1s, 2s, 4s, 8s, 16s, capped at 30s) already proven for the
 * other three sockets in #892. */
export function notificationSocketReconnectDelayMs(
  consecutiveFailures: number,
): number {
  return socketReconnectDelayMs(consecutiveFailures);
}

/** How long a connection must survive to count as healthy rather than a flap. */
export const NOTIFICATION_SOCKET_STABLE_MS = 30_000;

/**
 * Failure count to carry into the next reconnect, given how long the socket
 * that just closed stayed open (null = it never opened).
 *
 * Reaching `onopen` is not proof the session works: against production the
 * socket opened and dropped roughly once a second, and because `onopen` reset
 * the counter the curve in #910 re-armed at 1s every time and never escalated
 * (#928). Only surviving `NOTIFICATION_SOCKET_STABLE_MS` clears the count.
 */
export function nextConsecutiveFailures(
  previousFailures: number,
  openForMs: number | null,
): number {
  if (openForMs !== null && openForMs >= NOTIFICATION_SOCKET_STABLE_MS) return 0;
  return previousFailures + 1;
}

/**
 * Holds the user's real-time notification socket. On any server message
 * (`unread_count` on connect, `new_notification` thereafter) it refreshes the
 * bell's list + count queries, so a new notification lights the bell instantly.
 *
 * Unlike `useOrderStatusWS` (no fallback), this socket already degrades
 * gracefully today via REST polling (`useUnreadCount` / `useNotificationList`)
 * — those remain exactly as-is. The socket itself now retries indefinitely
 * with capped exponential backoff instead of giving up permanently after a
 * handful of failures, so REST polling only has to cover the gap while the
 * socket is reconnecting, not stand in forever after a one-time give-up.
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
  const openedAt = useRef<number | null>(null);
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
      // Deliberately does NOT clear `failures` — see nextConsecutiveFailures.
      openedAt.current = Date.now();
      console.info(`[notif-ws] connected ${url}`);
    };
    ws.onmessage = () => {
      // The socket signals "something changed" (a new notification, or the
      // initial count). Refetch the two feed queries — cheap, and avoids
      // hand-patching the list/count from a message whose shape varies by event.
      qc.invalidateQueries({ queryKey: NOTIFICATION_LIST_KEY });
      qc.invalidateQueries({ queryKey: NOTIFICATION_UNREAD_KEY });
    };
    ws.onerror = () => {
      console.warn('[notif-ws] error');
    };
    ws.onclose = () => {
      wsRef.current = null;
      const openForMs =
        openedAt.current === null ? null : Date.now() - openedAt.current;
      openedAt.current = null;
      failures.current = nextConsecutiveFailures(failures.current, openForMs);
      if (!shouldReconnectNotificationSocket(enabled, failures.current)) return;
      const delay = notificationSocketReconnectDelayMs(failures.current);
      console.warn(
        `[notif-ws] closed after ${openForMs ?? 0}ms, reconnecting in ${delay}ms (${failures.current} consecutive failures)`,
      );
      reconnectTimer.current = setTimeout(connect, delay);
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
