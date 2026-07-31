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

/** ws(s):// base for the API. EXPO_PUBLIC_API_URL already ends in `/api`, so callers add
 *  `/v1/...` — a `/api/v1/...` here doubles the prefix and every connect fails silently. */
function streamBase(): { ws: string; http: string } {
  const http = process.env.EXPO_PUBLIC_API_URL ?? 'https://fe3dr.com/api';
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
      sseRef.current = openEventStream(
        `${base.http}/v1/notifications/sse`,
        token,
        applyFrame,
      );
      return;
    }

    const ws = new (WebSocket as unknown as WSCtor)(`${base.ws}/v1/notifications/ws`, undefined, {
      headers: { Authorization: `Bearer ${token}` },
    });
    wsRef.current = ws;

    ws.onopen = () => {
      failureCount.current = 0;
    };
    ws.onmessage = (event: WebSocketMessageEvent) => {
      failureCount.current = 0;
      applyFrame(event.data as string);
    };
    ws.onerror = () => {
      failureCount.current += 1;
    };
    ws.onclose = () => {
      // Always reschedule: at the budget the next attempt is SSE, so the chef ends up on
      // a working transport rather than simply giving up.
      if (enabled) {
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
 * Minimal SSE reader over fetch's streaming body. React Native has no EventSource, and the
 * frames we care about are single-line `data:` payloads, so a full parser would be dead
 * weight — this handles the one shape the server sends and ignores comments/heartbeats.
 */
function openEventStream(
  url: string,
  token: string,
  onFrame: (raw: string) => void,
): { close: () => void } {
  const controller = new AbortController();

  void (async () => {
    try {
      const res = await fetch(url, {
        headers: { Authorization: `Bearer ${token}`, Accept: 'text/event-stream' },
        signal: controller.signal,
      });
      const body = res.body as ReadableStream<Uint8Array> | null;
      if (!body) return;
      const reader = body.getReader();
      const decoder = new TextDecoder();
      let buffer = '';
      for (;;) {
        const { done, value } = await reader.read();
        if (done) break;
        buffer += decoder.decode(value, { stream: true });
        // Frames are separated by a blank line; keep the trailing partial in the buffer.
        const frames = buffer.split('\n\n');
        buffer = frames.pop() ?? '';
        for (const frame of frames) {
          for (const line of frame.split('\n')) {
            if (line.startsWith('data:')) onFrame(line.slice(5).trim());
          }
        }
      }
    } catch {
      // Aborted on unmount, or the stream dropped — the caller's remount reconnects.
    }
  })();

  return { close: () => controller.abort() };
}
