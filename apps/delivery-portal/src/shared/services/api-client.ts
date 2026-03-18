import type { ApiError } from '@/shared/types';

// API calls go through whichever BFF the user authenticated with.
// Staff uses /bff (internal realm), drivers use /driver-bff (customer realm).
// Both proxy API requests to the same backend.
function getApiUrl(): string {
  const mode = typeof localStorage !== 'undefined' ? localStorage.getItem('fe3dr-auth-mode') : null;
  const bffPath = mode === 'driver' ? '/driver-bff' : '/bff';
  const env = mode === 'driver' ? import.meta.env.VITE_DRIVER_BFF_URL : import.meta.env.VITE_BFF_URL;
  if (env) return `${env}/api/v1`;
  if (typeof window !== 'undefined' && window.location.hostname !== 'localhost') {
    return `${window.location.origin}${bffPath}/api/v1`;
  }
  return `${bffPath}/api/v1`;
}
// API_URL is resolved dynamically per request via getApiUrl()

interface RequestOptions extends RequestInit {
  params?: Record<string, string | number | boolean | undefined>;
}

class ApiClient {
  private getBaseUrl(): string {
    return getApiUrl();
  }

  private async getCsrfToken(): Promise<string | null> {
    const { useAuthStore } = await import('@/app/store/auth-store');
    return useAuthStore.getState().csrfToken;
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
    const csrfToken = await this.getCsrfToken();

    const headers: HeadersInit = {
      'Content-Type': 'application/json',
      ...options.headers,
    };

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
