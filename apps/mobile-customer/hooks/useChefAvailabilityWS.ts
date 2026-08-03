// useChefAvailabilityWS — live kitchen open/close for the chef the customer is
// looking at (#970).
//
// The server published chef.availability_changed but nothing consumed it, so a
// closed kitchen kept reading "Open" until a manual pull-to-refresh — and the
// customer only found out when Place Order was rejected. This subscribes for as
// long as the screen is mounted and invalidates the chef queries on a change, so
// the badge, the reserve note and the checkout slot rules all re-render together.
import { useCallback, useEffect, useRef } from 'react';
import { useQueryClient } from '@tanstack/react-query';

import { useAuthStore } from '@homechef/mobile-shared/hooks';

// React Native's WebSocket accepts a headers option that the DOM lib doesn't.
type WSCtor = new (
  url: string,
  protocols?: string | string[],
  options?: { headers?: Record<string, string> },
) => WebSocket;

const MAX_BACKOFF_MS = 30_000;

export function useChefAvailabilityWS(chefId?: string | null): void {
  const queryClient = useQueryClient();
  const wsRef = useRef<WebSocket | null>(null);
  const failureCount = useRef(0);
  const reconnectTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const closedByUs = useRef(false);

  const connect = useCallback(() => {
    if (!chefId) return;

    // EXPO_PUBLIC_API_URL ends in `/api`, so the WS path is `/v1/...` — a
    // `/api/v1/...` here would double the prefix.
    const apiBase = process.env.EXPO_PUBLIC_API_URL ?? 'https://fe3dr.com/api';
    const wsBase = apiBase.replace(/^https?:\/\//, (m: string) =>
      m.startsWith('https') ? 'wss://' : 'ws://',
    );
    const url = `${wsBase}/v1/chefs/${chefId}/availability/ws`;

    // Browsing is guest-friendly, so the token is optional here.
    const token = useAuthStore.getState().accessToken;
    const ws = new (WebSocket as unknown as WSCtor)(
      url,
      undefined,
      token ? { headers: { Authorization: `Bearer ${token}` } } : undefined,
    );
    wsRef.current = ws;

    ws.onopen = () => {
      failureCount.current = 0;
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

    ws.onclose = () => {
      if (closedByUs.current) return;
      failureCount.current += 1;
      const delay = Math.min(
        1000 * 2 ** (failureCount.current - 1),
        MAX_BACKOFF_MS,
      );
      reconnectTimer.current = setTimeout(connect, delay);
    };
  }, [chefId, queryClient]);

  useEffect(() => {
    closedByUs.current = false;
    connect();
    return () => {
      closedByUs.current = true;
      if (reconnectTimer.current) clearTimeout(reconnectTimer.current);
      wsRef.current?.close();
      wsRef.current = null;
    };
  }, [connect]);
}
