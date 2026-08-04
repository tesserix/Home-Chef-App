// useChefAvailabilityWS — live kitchen open/close for the chef the customer is
// looking at (#970).
//
// The server published chef.availability_changed but nothing consumed it, so a
// closed kitchen kept reading "Open" until a manual pull-to-refresh — and the
// customer only found out when Place Order was rejected. This subscribes for as
// long as the screen is mounted and invalidates the chef queries on a change, so
// the badge, the reserve note and the checkout slot rules all re-render together.
import { useCallback, useEffect, useRef, useState } from 'react';
import { useQueryClient } from '@tanstack/react-query';

import { fetchWSTicket, wsEndpointUrl } from '@homechef/mobile-shared/realtime';
import { socketReconnectDelayWithJitterMs } from '@homechef/mobile-shared/utils';
import { api } from '../lib/api';

const MAX_WS_FAILURES = 3;
/** Fallback refetch cadence once the socket has given up. */
const AVAILABILITY_POLL_MS = 60_000;

export function useChefAvailabilityWS(chefId?: string | null): void {
  const queryClient = useQueryClient();
  const wsRef = useRef<WebSocket | null>(null);
  const failureCount = useRef(0);
  const reconnectTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const closedByUs = useRef(false);
  // Guards the async ticket fetch against unmount / supersession.
  const generation = useRef(0);
  // False once the socket has given up, which switches on the polling fallback.
  const [live, setLive] = useState(true);

  const connect = useCallback(() => {
    if (!chefId) return;

    // Ticket auth on a top-level `/ws/*` URL (#982). Browsing is guest-friendly
    // and the payload is public, so the route authenticates optionally: a
    // signed-out visitor has no session to mint from and connects without one.
    const apiBase = process.env.EXPO_PUBLIC_API_URL ?? 'https://fe3dr.com/api';
    const myGeneration = ++generation.current;

    void (async () => {
    const ticket = await fetchWSTicket(api);
    if (myGeneration !== generation.current) return;

    const url = wsEndpointUrl(apiBase, `chefs/${chefId}/availability`, ticket);
    const ws = new WebSocket(url);
    wsRef.current = ws;

    ws.onopen = () => {
      failureCount.current = 0;
      setLive(true);
    };

    ws.onmessage = (event: WebSocketMessageEvent) => {
      try {
        const msg = JSON.parse(String(event.data)) as {
          type?: string;
          chefId?: string;
        };
        if (msg.type !== 'availability') return;
        // Availability feeds the chef card, the chef page and checkout's slot
        // rules, so refresh the chef queries rather than patching one field.
        void queryClient.invalidateQueries({ queryKey: ['chef', chefId] });
        void queryClient.invalidateQueries({ queryKey: ['chefs'] });
      } catch {
        /* Malformed frame — ignore; the next one or a refetch will correct it. */
      }
    };

    ws.onerror = () => {
      /* onclose always follows; reconnect is handled there. */
    };

    // Past the budget, stop dialling and let the refetch fallback below carry
    // the badge. React Native's WebSocket cannot complete the TLS handshake
    // against our edge (close 1006 / OSStatus -9836, #982), so on native the
    // budget is always spent and further attempts never reach the server.
    ws.onclose = () => {
      if (closedByUs.current || myGeneration !== generation.current) return;
      wsRef.current = null;
      failureCount.current += 1;
      if (failureCount.current >= MAX_WS_FAILURES) {
        setLive(false);
        return;
      }
      reconnectTimer.current = setTimeout(
        connect,
        socketReconnectDelayWithJitterMs(failureCount.current),
      );
    };
    })();
  }, [chefId, queryClient]);

  // Fallback: without the socket the badge would silently go stale, which is
  // the exact failure #970 existed to fix. Refetch on a slow cadence instead —
  // far cheaper than the reconnect storm it replaces, and only while the
  // socket is down.
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
      wsRef.current?.close();
      wsRef.current = null;
    };
  }, [connect]);
}
