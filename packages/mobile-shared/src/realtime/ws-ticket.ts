// Ticket authentication for WebSocket upgrades (#982).
//
// Every mobile socket previously dialled `${API}/v1/…/ws` and authenticated
// with an `Authorization` header. That never worked: the header trick is a
// React-Native extension rather than WHATWG (a browser cannot set it at all),
// and `/api/**` is the BFF's path — the BFF cannot proxy an upgrade. Sockets
// 401'd on every attempt and the apps ran permanently on REST polling. #892,
// #909, #910 and #928 each re-tuned the backoff curve around that failure
// without ever reaching the cause.
//
// The fix: mint a short-lived ticket over the authenticated REST path, then
// spend it in the query string of a top-level `/ws/*` URL, which Istio routes
// straight to the API. One credential, works identically on RN and web.

import type { AxiosInstance } from 'axios';

/** Where the ticket is minted. Version prefix is added by the caller's axios base. */
const WS_TICKET_PATH = '/realtime/ws-ticket';

/**
 * Strips the API path suffix off a base URL to get the origin the `/ws/*`
 * routes hang off.
 *
 * The two apps configure EXPO_PUBLIC_API_URL differently — the customer's ends
 * in `/api`, the vendor's in `/api/v1` — and the WebSocket routes are siblings
 * of `/api`, not children, so both suffixes have to come off.
 */
export function wsOriginFrom(apiBaseUrl: string): string {
  return apiBaseUrl
    .replace(/^http:\/\//, 'ws://')
    .replace(/^https:\/\//, 'wss://')
    .replace(/\/+$/, '')
    .replace(/\/api(\/v\d+)?$/, '');
}

/**
 * Builds a fully-formed socket URL: `wss://host/ws/<path>?ticket=<t>`.
 * `path` is the route below `/ws` with no leading slash (e.g. `notifications`).
 */
export function wsEndpointUrl(
  apiBaseUrl: string,
  path: string,
  ticket: string,
): string {
  const origin = wsOriginFrom(apiBaseUrl);
  const clean = path.replace(/^\/+/, '');
  return `${origin}/ws/${clean}?ticket=${encodeURIComponent(ticket)}`;
}

function ticketPrefix(api: AxiosInstance): string {
  const base = api.defaults?.baseURL;
  return base && /\/v\d+\/?$/.test(base) ? '' : '/v1';
}

/**
 * Mints a ticket for the current session. Goes through the app's axios
 * instance, so it inherits auth, refresh-on-401 and base URL — the socket
 * never has to know how the session is carried.
 *
 * Returns null rather than throwing: a socket that cannot get a ticket should
 * back off and retry like any other failure, not blow up a render.
 */
export async function fetchWSTicket(
  api: AxiosInstance,
): Promise<string | null> {
  try {
    const res = await api.post<{ ticket?: string }>(
      `${ticketPrefix(api)}${WS_TICKET_PATH}`,
    );
    return res.data?.ticket ?? null;
  } catch {
    return null;
  }
}
