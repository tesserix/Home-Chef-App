import type { ApiError } from '@/shared/types';

const API_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080/api/v1';
const BFF_URL = import.meta.env.VITE_BFF_URL || 'https://identity.fe3dr.com';
const MOCK_MODE = import.meta.env.VITE_MOCK_MODE === 'true';

// In production, use same-origin /bff/ prefix to avoid cross-origin CORS issues.
// The VirtualService rewrites /bff/* → / on the BFF, so /bff/api/v1/... → /api/v1/...
// In development (localhost), use BFF_URL directly since there's no Istio proxy.
const isLocalDev = typeof window !== 'undefined' && window.location.hostname === 'localhost';
const BFF_PROXY_BASE = isLocalDev ? BFF_URL : '/bff';

interface RequestOptions extends RequestInit {
  params?: Record<string, string | number | boolean | undefined>;
}

class ApiClient {
  private baseUrl: string;
  private bffProxyBase: string;

  constructor(baseUrl: string, bffProxyBase: string) {
    this.baseUrl = baseUrl;
    this.bffProxyBase = bffProxyBase;
  }

  private async getAuthState(): Promise<{ isAuthenticated: boolean; csrfToken: string | null }> {
    const { useAuthStore } = await import('@/app/store/auth-store');
    const state = useAuthStore.getState();
    return { isAuthenticated: state.isAuthenticated, csrfToken: state.csrfToken };
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
    const { isAuthenticated, csrfToken } = await this.getAuthState();

    // Route through BFF when authenticated so session cookie is validated
    // and x-jwt-claim-sub header is injected for the API
    const base = isAuthenticated ? this.bffProxyBase : this.baseUrl;
    const url = this.buildUrl(base, endpoint, params);

    const headers: HeadersInit = {
      'Content-Type': 'application/json',
      ...options.headers,
    };

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
    const { isAuthenticated, csrfToken } = await this.getAuthState();
    const base = isAuthenticated ? this.bffProxyBase : this.baseUrl;
    const url = this.buildUrl(base, endpoint);

    const headers: Record<string, string> = {};
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
