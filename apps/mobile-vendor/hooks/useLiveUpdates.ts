import { useEffect, useRef, useCallback } from 'react';
import { useQueryClient } from '@tanstack/react-query';

import { useAuthStore } from '../store/auth-store';
import { invalidationsFor, parseLiveFrame } from '../lib/live-updates';

const MAX_WS_FAILURES = 4;
const RECONNECT_DELAY_MS = 3000;

// React Native's WebSocket takes a headers option as a 3rd argument that the DOM type
// omits. The stream is user-scoped and authenticates with the Bearer token.
type WSCtor = {
  new (
    url: string,
    protocols?: string | string[],
    options?: { headers?: Record<string, string> },
  ): WebSocket;
};

/**
 * The `/v1` base, over http and ws.
 *
 * The apps disagree about where `/v1` lives: this one's EXPO_PUBLIC_API_URL ends in
 * `/api/v1`, the customer's in `/api`. Appending `/v1` blindly produced `/api/v1/v1/...`,
 * which 404s — and because the socket fails silently and falls back, the only symptom was
 * stale data. Normalise instead of assuming either convention.
 */
export function streamBase(base?: string): { ws: string; http: string } {
  const raw = (base ?? process.env.EXPO_PUBLIC_API_URL ?? 'https://vendors.fe3dr.com/api/v1')
    .replace(/\/+$/, '');
  const http = /\/v1$/.test(raw) ? raw : `${raw}/v1`;
  return {
    http,
    ws: http.replace(/^https?:\/\//, (m: string) => (m.startsWith('https') ? 'wss://' : 'ws://')),
  };
}

/**
 * Keeps the chef's lists live: a new order, a customer approving a plan, a refund landing in
 * their queue all arrive pushed instead of on the next poll.
 *
 * WebSocket first, SSE after repeated failures. Both carry the same per-user stream, so the
 * fallback is a transport swap and nothing else — the invalidation routing (lib/live-updates)
 * is shared, and cannot drift between them.
 *
 * Mount ONCE, high in the tree. The stream is user-scoped, so a second mount is a second
 * connection for the same events.
 */
export function useLiveUpdates(enabled: boolean = true): void {
  const queryClient = useQueryClient();
  const wsRef = useRef<WebSocket | null>(null);
  const sseRef = useRef<{ close: () => void } | null>(null);
  const failureCount = useRef(0);
  const reconnectTimer = useRef<ReturnType<typeof setTimeout> | null>(null);

  const applyFrame = useCallback(
    (raw: string) => {
      const payload = parseLiveFrame(raw);
      if (!payload) return;
      for (const key of invalidationsFor(payload)) {
        void queryClient.invalidateQueries({ queryKey: key });
      }
    },
    [queryClient],
  );

  const connect = useCallback(() => {
    if (!enabled) return;
    const token = useAuthStore.getState().accessToken;
    if (!token) return;

    const base = streamBase();

    // Past the WS failure budget, hold the door open with SSE instead. A blocked upgrade
    // is a property of the network, not the request, so retrying WS forever just leaves
    // the chef on stale data.
    if (failureCount.current >= MAX_WS_FAILURES) {
      console.warn(
        `[live-ws] falling back to SSE after ${failureCount.current} WS failures`,
      );
      sseRef.current = openEventStream(
        `${base.http}/notifications/sse`,
        token,
        applyFrame,
      );
      return;
    }

    const ws = new (WebSocket as unknown as WSCtor)(`${base.ws}/notifications/ws`, undefined, {
      headers: { Authorization: `Bearer ${token}` },
    });
    wsRef.current = ws;

    ws.onopen = () => {
      failureCount.current = 0;
      console.info('[live-ws] connected');
    };
    ws.onmessage = (event: WebSocketMessageEvent) => {
      failureCount.current = 0;
      applyFrame(event.data as string);
    };
    ws.onerror = () => {
      failureCount.current += 1;
      console.warn(
        `[live-ws] error (${failureCount.current}/${MAX_WS_FAILURES} failures)`,
      );
    };
    ws.onclose = () => {
      // Always reschedule: at the budget the next attempt is SSE, so the chef ends up on
      // a working transport rather than simply giving up.
      if (enabled) {
        console.warn(`[live-ws] closed, reconnecting in ${RECONNECT_DELAY_MS}ms`);
        reconnectTimer.current = setTimeout(connect, RECONNECT_DELAY_MS);
      }
    };
  }, [enabled, applyFrame]);

  useEffect(() => {
    connect();
    return () => {
      wsRef.current?.close();
      wsRef.current = null;
      sseRef.current?.close();
      sseRef.current = null;
      if (reconnectTimer.current) {
        clearTimeout(reconnectTimer.current);
        reconnectTimer.current = null;
      }
    };
  }, [connect]);
}

/**
 * Minimal SSE reader over XMLHttpRequest.
 *
 * NOT fetch: React Native's fetch is the whatwg-fetch XHR polyfill, so `response.body` is
 * undefined and a `getReader()` stream reader silently reads nothing — the fallback would
 * look wired up and deliver no events. XHR exposes the partial `responseText` as it
 * arrives, which is all an SSE reader needs.
 *
 * There is no EventSource in RN either, and the server sends one shape (single-line
 * `data:` frames plus `:` heartbeats), so a full spec parser would be dead weight.
 */
function openEventStream(
  url: string,
  token: string,
  onFrame: (raw: string) => void,
): { close: () => void } {
  const xhr = new XMLHttpRequest();
  // How much of responseText has already been turned into frames. XHR keeps the whole
  // response in memory and grows it, so we parse only the tail each time.
  let consumed = 0;

  const drain = () => {
    const text = xhr.responseText ?? '';
    // Only complete frames (terminated by a blank line) are safe to parse — the tail may
    // be half a frame still in flight.
    const lastBreak = text.lastIndexOf('\n\n');
    if (lastBreak < consumed) return;
    const chunk = text.slice(consumed, lastBreak);
    consumed = lastBreak + 2;
    for (const frame of chunk.split('\n\n')) {
      for (const line of frame.split('\n')) {
        // `:` lines are comments (our heartbeat) — ignore them.
        if (line.startsWith('data:')) onFrame(line.slice(5).trim());
      }
    }
  };

  xhr.open('GET', url);
  xhr.setRequestHeader('Authorization', `Bearer ${token}`);
  xhr.setRequestHeader('Accept', 'text/event-stream');
  xhr.onreadystatechange = () => {
    // 3 = LOADING: body is arriving. Parsing here rather than on completion is the whole
    // point — this response never completes.
    if (xhr.readyState >= 3) drain();
  };
  xhr.send();

  return { close: () => xhr.abort() };
}
