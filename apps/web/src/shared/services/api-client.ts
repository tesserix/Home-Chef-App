import type { ApiError } from '@/shared/types';

const API_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080/api/v1';
const BFF_URL = (() => {
  const env = import.meta.env.VITE_BFF_URL;
  if (env) return env;
  if (typeof window !== 'undefined' && window.location.hostname !== 'localhost') {
    return `${window.location.origin}/bff`;
  }
  return '/bff';
})();
const MOCK_MODE = import.meta.env.VITE_MOCK_MODE === 'true';

// In production, use same-origin /bff/ prefix to avoid cross-origin CORS issues.
// The VirtualService rewrites /bff/* → / on the BFF, so /bff/api/v1/... → /api/v1/...
// In development (localhost), use BFF_URL directly since there's no Istio proxy.
const isLocalDev = typeof window !== 'undefined' && window.location.hostname === 'localhost';
const BFF_PROXY_BASE = isLocalDev ? BFF_URL : '/bff';

interface RequestOptions extends RequestInit {
  params?: Record<string, string | number | boolean | undefined>;
}

/**
 * Custom event dispatched when the API returns 401 on a non-auth endpoint.
 * AuthProvider listens for this and runs clearAuth + redirect-to-login,
 * giving the app a single recovery path for expired sessions.
 */
export const AUTH_EXPIRED_EVENT = 'auth:expired';

/** Endpoints that legitimately return 401 during login/session checks — never bubble. */
const AUTH_EXPIRY_BYPASS = [
  '/auth/login',
  '/auth/register',
  '/auth/refresh',
  '/auth/me',
  '/auth/session',
  '/auth/logout',
];

function isAuthBypass(endpoint: string): boolean {
  return AUTH_EXPIRY_BYPASS.some((p) => endpoint.startsWith(p));
}

function dispatchAuthExpired(endpoint: string): void {
  if (typeof window === 'undefined') return;
  if (isAuthBypass(endpoint)) return;
  window.dispatchEvent(new CustomEvent(AUTH_EXPIRED_EVENT, { detail: { endpoint } }));
}

/**
 * Custom event dispatched when the API returns 403 with a body `status` of
 * `account_deactivated` or `account_deleted` — middleware/bff_auth.go's
 * account-lifecycle gate, which rejects every path except /me/reactivate for
 * a paused account. DataPrivacyPage listens for this so a paused user who
 * reloads (or is deep on some other page) lands somewhere that explains why
 * and offers reactivation, instead of a silent, opaque 403.
 */
export const ACCOUNT_BLOCKED_EVENT = 'auth:account-blocked';

export type AccountBlockedStatus = 'account_deactivated' | 'account_deleted';

function accountBlockedStatus(body: unknown): AccountBlockedStatus | null {
  if (typeof body !== 'object' || body === null) return null;
  const status = (body as { status?: unknown }).status;
  return status === 'account_deactivated' || status === 'account_deleted' ? status : null;
}

function dispatchAccountBlocked(status: AccountBlockedStatus): void {
  if (typeof window === 'undefined') return;
  window.dispatchEvent(new CustomEvent(ACCOUNT_BLOCKED_EVENT, { detail: { status } }));
}

class ApiClient {
  private baseUrl: string;
  private bffProxyBase: string;

  constructor(baseUrl: string, bffProxyBase: string) {
    this.baseUrl = baseUrl;
    this.bffProxyBase = bffProxyBase;
  }

  private async getAuthState(): Promise<{ isAuthenticated: boolean; csrfToken: string | null; accessToken: string | null }> {
    const { useAuthStore } = await import('@/app/store/auth-store');
    const state = useAuthStore.getState();
    return { isAuthenticated: state.isAuthenticated, csrfToken: state.csrfToken, accessToken: state.accessToken };
  }

  private buildUrl(base: string, endpoint: string, params?: RequestOptions['params']): string {
    // BFF proxies /api/* to the API, so prefix endpoint with /api/v1
    // Direct API URL already includes /api/v1
    const fullPath = base === this.bffProxyBase ? `/api/v1${endpoint}` : endpoint;
    const url = new URL(`${base}${fullPath}`, window.location.origin);

    if (params) {
      Object.entries(params).forEach(([key, value]) => {
        if (value !== undefined) {
          url.searchParams.append(key, String(value));
        }
      });
    }

    return url.toString();
  }

  private async request<T>(
    method: string,
    endpoint: string,
    options: RequestOptions = {}
  ): Promise<T> {
    // Use mock service in mock mode
    if (MOCK_MODE) {
      const { mockService } = await import('@/mock/mock-service');
      const mockOptions = {
        params: options.params,
        body: typeof options.body === 'string' ? options.body : undefined,
      };
      return mockService.request<T>(method, endpoint, mockOptions);
    }

    const { params, ...fetchOptions } = options;
    const { isAuthenticated, csrfToken, accessToken } = await this.getAuthState();

    // Every authenticated call goes through the BFF proxy, which validates the
    // session cookie and HMAC-signs the upstream request.
    //
    // This used to branch on `accessToken`, sending email/password users
    // straight to /api/v1 with a Bearer token, from an era when the API issued
    // its own JWTs. It no longer does: auth is Zitadel hosted login -> /bff/auth/callback -> an
    // HttpOnly session cookie, `accessToken` is just the Firebase ID token, and
    // the API accepts only HMAC-signed requests from the BFF (see the apiproxy
    // package doc: "there is no Bearer auth path on the API"). So the direct
    // branch could only ever 401 — which it did, on every authenticated request
    // after an email/password login, surfacing as "Your session has expired".
    //
    // Unauthenticated calls still go direct: public endpoints like /currencies
    // need no session and the BFF would only add a hop.
    const base = isAuthenticated ? this.bffProxyBase : this.baseUrl;
    const url = this.buildUrl(base, endpoint, params);

    // Only advertise JSON content when we're actually sending a body.
    // The auth-bff in front is Fastify-based and rejects
    // (FST_ERR_CTP_EMPTY_JSON_BODY / 400) any POST that claims
    // Content-Type: application/json but arrives empty.
    const hasBody = fetchOptions.body !== undefined && fetchOptions.body !== null;
    const headers: HeadersInit = {
      ...(hasBody ? { 'Content-Type': 'application/json' } : {}),
      ...options.headers,
    };

    // Add Bearer token for API-issued JWT auth (email/password login)
    if (accessToken) {
      (headers as Record<string, string>)['X-Auth-Token'] = accessToken;
    }

    // Add CSRF token for state-changing requests (BFF requires it)
    if (method !== 'GET' && csrfToken) {
      (headers as Record<string, string>)['X-CSRF-Token'] = csrfToken;
    }

    const response = await fetch(url, {
      method,
      headers,
      credentials: 'include',
      ...fetchOptions,
    });

    if (!response.ok) {
      const body: ApiError = await response.json().catch(() => ({
        success: false,
        error: {
          code: 'UNKNOWN_ERROR',
          message: response.statusText || 'An error occurred',
        },
      }));
      if (response.status === 401) {
        dispatchAuthExpired(endpoint);
      } else if (response.status === 403) {
        // Read the body's own `status` tag before it gets clobbered below —
        // `status` on the thrown error is documented (see the comment under
        // this block) to mean the HTTP status code, so it can't also carry
        // the account-lifecycle tag.
        const blocked = accountBlockedStatus(body);
        if (blocked) dispatchAccountBlocked(blocked);
      }
      // Attach HTTP status so callers can differentiate 401/403/409 etc.
      throw Object.assign(body, { status: response.status });
    }

    const json = await response.json();
    // Paginated endpoints return { data: [], pagination: {} } — unwrap to data array.
    // Non-paginated endpoints return the object directly.
    if (json && typeof json === 'object' && 'data' in json && 'pagination' in json) {
      return json as T;
    }
    if (json && typeof json === 'object' && 'data' in json) {
      return json.data as T;
    }
    return json as T;
  }

  async get<T>(endpoint: string, params?: RequestOptions['params']): Promise<T> {
    return this.request<T>('GET', endpoint, { params });
  }

  /**
   * GET a binary body (e.g. a PDF invoice) with the same auth as request(), and
   * return the blob plus the server-suggested filename from Content-Disposition.
   * request() always parses JSON, so a file download needs its own path.
   */
  async getBlob(endpoint: string): Promise<{ blob: Blob; filename: string }> {
    const { isAuthenticated, accessToken } = await this.getAuthState();
    // Same rule as request(): authenticated traffic goes through the BFF, which
    // is the only thing the API trusts. See the comment there.
    const base = isAuthenticated ? this.bffProxyBase : this.baseUrl;
    const url = this.buildUrl(base, endpoint);

    const headers: Record<string, string> = {};
    if (accessToken) headers['X-Auth-Token'] = accessToken;

    const response = await fetch(url, { method: 'GET', headers, credentials: 'include' });
    if (!response.ok) {
      if (response.status === 401) dispatchAuthExpired(endpoint);
      const body = await response.json().catch(() => ({}));
      throw Object.assign(body, { status: response.status });
    }

    const disposition = response.headers.get('Content-Disposition') ?? '';
    const match = /filename="?([^";]+)"?/i.exec(disposition);
    const filename = match?.[1] ?? 'invoice.pdf';
    return { blob: await response.blob(), filename };
  }

  async post<T>(endpoint: string, body?: unknown, options?: RequestOptions): Promise<T> {
    return this.request<T>('POST', endpoint, {
      ...options,
      body: body ? JSON.stringify(body) : undefined,
    });
  }

  async put<T>(endpoint: string, body?: unknown, options?: RequestOptions): Promise<T> {
    return this.request<T>('PUT', endpoint, {
      ...options,
      body: body ? JSON.stringify(body) : undefined,
    });
  }

  async patch<T>(endpoint: string, body?: unknown, options?: RequestOptions): Promise<T> {
    return this.request<T>('PATCH', endpoint, {
      ...options,
      body: body ? JSON.stringify(body) : undefined,
    });
  }

  async delete<T>(endpoint: string, options?: RequestOptions): Promise<T> {
    return this.request<T>('DELETE', endpoint, options);
  }

  /** Upload a file via multipart/form-data. Do NOT set Content-Type — the browser handles it. */
  async upload<T>(endpoint: string, formData: FormData): Promise<T> {
    const { isAuthenticated, csrfToken, accessToken } = await this.getAuthState();
    // JWT auth bypasses the BFF (same reasoning as request()); BFF session
    // goes through /bff/ so the cookie is validated.
    const useDirectApi = isAuthenticated && !!accessToken;
    const base = useDirectApi ? this.baseUrl : (isAuthenticated ? this.bffProxyBase : this.baseUrl);
    const url = this.buildUrl(base, endpoint);

    const headers: Record<string, string> = {};
    if (accessToken) {
      headers['X-Auth-Token'] = accessToken;
    }
    if (csrfToken) {
      headers['X-CSRF-Token'] = csrfToken;
    }

    const response = await fetch(url, {
      method: 'POST',
      headers,
      credentials: 'include',
      body: formData,
    });

    if (!response.ok) {
      const body: ApiError = await response.json().catch(() => ({
        success: false,
        error: { code: 'UNKNOWN_ERROR', message: response.statusText || 'An error occurred' },
      }));
      if (response.status === 401) {
        dispatchAuthExpired(endpoint);
      }
      throw Object.assign(body, { status: response.status });
    }

    const json = await response.json();
    if (json && typeof json === 'object' && 'data' in json) {
      return json.data as T;
    }
    return json as T;
  }
}

export const apiClient = new ApiClient(API_URL, BFF_PROXY_BASE);
