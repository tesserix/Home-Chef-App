// useChefAvailabilityWS — live kitchen open/close for the chef the customer is
// looking at (#970).
//
// The server published chef.availability_changed but nothing consumed it, so a
// closed kitchen kept reading "Open" until a manual pull-to-refresh — and the
// customer only found out when Place Order was rejected. This subscribes for as
// long as the screen is mounted and invalidates the chef queries on a change, so
// the badge, the reserve note and the checkout slot rules all re-render together.
//
// SSE, not WebSocket: React Native does TLS through SocketRocket/CFStream and
// that handshake fails outright against our edge (close 1006 / OSStatus -9836,
// #982), so on native the socket's budget was always spent on attempts that
// never reached the server — leaving the badge on a 60-second poll, which is
// most of the staleness #970 set out to remove.
//
// The stream authenticates optionally: browsing is guest-friendly and a
// kitchen's open/closed state is public, so a signed-out visitor connects with
// no token at all.
import { useCallback, useEffect, useRef, useState } from 'react';
import { useQueryClient } from '@tanstack/react-query';

import {
  openEventStream,
  streamRetryPlan,
  type EventStreamHandle,
} from '@homechef/mobile-shared/realtime';
import { useAuthStore } from '../store/auth-store';

const MAX_STREAM_FAILURES = 3;
/** Fallback refetch cadence once the stream has given up. */
const AVAILABILITY_POLL_MS = 60_000;

export function useChefAvailabilityWS(chefId?: string | null): void {
  const queryClient = useQueryClient();
  const streamRef = useRef<EventStreamHandle | null>(null);
  const failureCount = useRef(0);
  const reconnectTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const closedByUs = useRef(false);
  // Guards an attempt against unmount / supersession.
  const generation = useRef(0);
  // False once the stream has given up, which switches on the polling fallback.
  const [live, setLive] = useState(true);

  const connect = useCallback(() => {
    if (!chefId) return;
    const apiBase = process.env.EXPO_PUBLIC_API_URL ?? 'https://fe3dr.com/api';
    const myGeneration = ++generation.current;
    // Guests have no session; the route allows that.
    const token = useAuthStore.getState().accessToken ?? '';

    const onFrame = (raw: string) => {
      failureCount.current = 0;
      setLive(true);
      try {
        const msg = JSON.parse(raw) as { type?: string; chefId?: string };
        if (msg.type !== 'availability') return;
        // Availability feeds the chef card, the chef page and checkout's slot
        // rules, so refresh the chef queries rather than patching one field.
        void queryClient.invalidateQueries({ queryKey: ['chef', chefId] });
        void queryClient.invalidateQueries({ queryKey: ['chefs'] });
      } catch {
        /* Malformed frame — ignore; the next one or a refetch will correct it. */
      }
    };

    streamRef.current = openEventStream(
      `${apiBase}/v1/chefs/${chefId}/availability/sse`,
      token,
      onFrame,
      (status) => {
        if (closedByUs.current || myGeneration !== generation.current) return;
        streamRef.current = null;
        failureCount.current += 1;
        const plan = streamRetryPlan(status, failureCount.current);
        // Past the budget the refetch fallback below carries the badge, but keep
        // dialling so a stream that recovers takes over again.
        if (plan.degrade || failureCount.current >= MAX_STREAM_FAILURES) {
          setLive(false);
        }
        reconnectTimer.current = setTimeout(connect, plan.delayMs);
      },
    );
  }, [chefId, queryClient]);

  // Fallback: without the stream the badge would silently go stale, which is
  // the exact failure #970 existed to fix. Refetch on a slow cadence instead —
  // far cheaper than the reconnect storm it replaces, and only while the
  // stream is down.
  useEffect(() => {
    if (live || !chefId) return;
    const id = setInterval(() => {
      void queryClient.invalidateQueries({ queryKey: ['chef', chefId] });
    }, AVAILABILITY_POLL_MS);
    return () => clearInterval(id);
  }, [live, chefId, queryClient]);

  useEffect(() => {
    closedByUs.current = false;
    connect();
    return () => {
      closedByUs.current = true;
      generation.current += 1;
      if (reconnectTimer.current) clearTimeout(reconnectTimer.current);
      streamRef.current?.close();
      streamRef.current = null;
    };
  }, [connect]);
}
