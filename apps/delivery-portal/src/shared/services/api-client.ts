import type { ApiError } from '@/shared/types';

// Both driver and staff use the same customer realm BFF (/bff/).
// Staff are 3rd party fleet managers, not platform admins.
function getApiUrl(): string {
  const env = import.meta.env.VITE_BFF_URL;
  if (env) return `${env}/api/v1`;
  if (typeof window !== 'undefined' && window.location.hostname !== 'localhost') {
    return `${window.location.origin}/bff/api/v1`;
  }
  return '/bff/api/v1';
}
// API_URL is resolved dynamically per request via getApiUrl()

interface RequestOptions extends RequestInit {
  params?: Record<string, string | number | boolean | undefined>;
}

class ApiClient {
  private getBaseUrl(): string {
    return getApiUrl();
  }

  private async getAuthState(): Promise<{ csrfToken: string | null; accessToken: string | null }> {
    const { useAuthStore } = await import('@/app/store/auth-store');
    const state = useAuthStore.getState();
    return { csrfToken: state.csrfToken, accessToken: state.accessToken };
  }

  private buildUrl(endpoint: string, params?: RequestOptions['params']): string {
    const url = new URL(`${this.getBaseUrl()}${endpoint}`);
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
    const { csrfToken, accessToken } = await this.getAuthState();

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
      (headers as Record<string, string>)['Authorization'] = `Bearer ${accessToken}`;
    }

    if (csrfToken && method !== 'GET') {
      (headers as Record<string, string>)['X-CSRF-Token'] = csrfToken;
    }

    const response = await fetch(url, {
      method,
      headers,
      credentials: 'include',
      ...fetchOptions,
    });

    if (response.status === 401) {
      const { useAuthStore } = await import('@/app/store/auth-store');
      useAuthStore.getState().clearAuth();
      throw { success: false, error: { code: 'SESSION_EXPIRED', message: 'Session expired' } };
    }

    if (!response.ok) {
      const error: ApiError = await response.json().catch(() => ({
        success: false,
        error: { code: 'UNKNOWN_ERROR', message: response.statusText || 'An error occurred' },
      }));
      throw error;
    }

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

export const apiClient = new ApiClient();
