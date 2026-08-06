import { useEffect, useRef, useCallback } from 'react';
import { AppState, type AppStateStatus } from 'react-native';
import { useQueryClient } from '@tanstack/react-query';
import { socketReconnectDelayWithJitterMs } from '@homechef/mobile-shared/utils';
import { openEventStream } from '@homechef/mobile-shared/realtime';

import { useAuthStore } from '../store/auth-store';
import { invalidationsFor, parseLiveFrame } from '../lib/live-updates';


/**
 * The `/v1` base for the stream.
 *
 * The apps disagree about where `/v1` lives: this one's EXPO_PUBLIC_API_URL ends in
 * `/api/v1`, the customer's in `/api`. Appending `/v1` blindly produced `/api/v1/v1/...`,
 * which 404s — and because the stream fails silently, the only symptom was stale data.
 * Normalise instead of assuming either convention.
 */
export function streamBase(base?: string): { http: string } {
  const raw = (base ?? process.env.EXPO_PUBLIC_API_URL ?? 'https://vendors.fe3dr.com/api/v1')
    .replace(/\/+$/, '');
  return { http: /\/v1$/.test(raw) ? raw : `${raw}/v1` };
}

/**
 * Keeps the chef's lists live: a new order, a customer approving a plan, a refund landing in
 * their queue all arrive pushed instead of on the next poll.
 *
 * Retries with capped backoff, so a connectivity blip recovers to real-time instead of
 * leaving the session on polling for the rest of the shift (#892).
 *
 * Mount ONCE, high in the tree. The stream is user-scoped, so a second mount is a second
 * connection for the same events.
 */
export function useLiveUpdates(enabled: boolean = true): void {
  const queryClient = useQueryClient();
  const sseRef = useRef<{ close: () => void } | null>(null);
  const failureCount = useRef(0);
  const reconnectTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  // Guards the async ticket fetch against unmount / supersession.
  const generation = useRef(0);

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
    const myGeneration = ++generation.current;

    // SSE is the primary transport, not a fallback. React Native's WebSocket
    // cannot complete the TLS handshake against our edge (close 1006 /
    // OSStatus -9836, #982) — it never issued a request, so this app has in
    // practice always run on this stream. Dialling the socket first only
    // bought three guaranteed failures and a ticket mint per attempt.
    sseRef.current = openEventStream(
      `${base.http}/notifications/sse`,
      token,
      applyFrame,
      () => {
        if (myGeneration !== generation.current || !enabled) return;
        sseRef.current = null;
        failureCount.current += 1;
        reconnectTimer.current = setTimeout(
          connect,
          socketReconnectDelayWithJitterMs(failureCount.current),
        );
      },
    );
  }, [enabled, applyFrame]);

  useEffect(() => {
    connect();
    return () => {
      sseRef.current?.close();
      sseRef.current = null;
      if (reconnectTimer.current) {
        clearTimeout(reconnectTimer.current);
        reconnectTimer.current = null;
      }
    };
  }, [connect]);

  // Foreground reconnect: don't leave the chef waiting out a stale backoff
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
      if (!sseRef.current) {
        connect();
      }
    });
    return () => sub.remove();
  }, [enabled, connect]);
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
