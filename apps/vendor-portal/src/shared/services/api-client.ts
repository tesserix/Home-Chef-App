import type { ApiError } from '@/shared/types';

// API calls go through the BFF proxy which handles session auth (cookies → JWT)
const BFF_URL = import.meta.env.VITE_BFF_URL || 'https://identity.fe3dr.com';
const API_URL = `${BFF_URL}/api/v1`;

interface RequestOptions extends RequestInit {
  params?: Record<string, string | number | boolean | undefined>;
}

class ApiClient {
  private baseUrl: string;

  constructor(baseUrl: string) {
    this.baseUrl = baseUrl;
  }

  private async getCsrfToken(): Promise<string | null> {
    const { useAuthStore } = await import('@/app/store/auth-store');
    return useAuthStore.getState().csrfToken;
  }

  private buildUrl(endpoint: string, params?: RequestOptions['params']): string {
    const url = new URL(`${this.baseUrl}${endpoint}`);

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
    const { params, ...fetchOptions } = options;
    const url = this.buildUrl(endpoint, params);

    const csrfToken = await this.getCsrfToken();

    const headers: HeadersInit = {
      'Content-Type': 'application/json',
      ...options.headers,
    };

    // Include CSRF token for state-changing requests
    if (csrfToken && method !== 'GET') {
      (headers as Record<string, string>)['X-CSRF-Token'] = csrfToken;
    }

    const response = await fetch(url, {
      method,
      headers,
      credentials: 'include', // Send session cookies with all requests
      ...fetchOptions,
    });

    // Handle 401 - session expired
    if (response.status === 401) {
      const { useAuthStore } = await import('@/app/store/auth-store');
      useAuthStore.getState().clearAuth();
      // Don't hard-redirect here — the route guards (ProtectedRoute) will
      // detect isAuthenticated=false and redirect to /login, avoiding loops.
      throw { success: false, error: { code: 'SESSION_EXPIRED', message: 'Session expired' } };
    }

    if (!response.ok) {
      const error: ApiError = await response.json().catch(() => ({
        success: false,
        error: {
          code: 'UNKNOWN_ERROR',
          message: response.statusText || 'An error occurred',
        },
      }));
      throw error;
    }

    // The Go API is mixed: some endpoints return raw JSON, others wrap
    // in { data: T, pagination: {...} }. Auto-detect and unwrap when needed.
    const json = await response.json();
    if (json && typeof json === 'object' && 'data' in json && 'pagination' in json) {
      return json.data;
    }
    return json;
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
}

export const apiClient = new ApiClient(API_URL);
