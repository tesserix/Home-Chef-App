import type { AxiosInstance } from 'axios';

// The apps mount their API client at different depths: the customer's
// EXPO_PUBLIC_API_URL ends at `/api`, the vendor's and driver's at `/api/v1`.
// Shared code must therefore derive the version prefix rather than hardcode
// `/v1` — that mistake has silently 404'd the vendor three times now (device
// token, notification feed, feedback form), because each call site kept its own
// private copy of this rule.

/** `/v1` when a base URL still needs a version, `''` when it already carries one. */
export function versionPrefixFor(baseUrl: string | undefined | null): string {
  const base = baseUrl?.replace(/\/+$/, '');
  return base && /\/v\d+$/.test(base) ? '' : '/v1';
}

/** The same rule, read off an axios client. */
export function apiVersionPrefix(api: AxiosInstance): string {
  return versionPrefixFor(api.defaults?.baseURL);
}
