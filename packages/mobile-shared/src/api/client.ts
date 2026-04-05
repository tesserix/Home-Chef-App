// API client factory — creates an axios instance with auth interceptors
// Pattern: adapted from apps/vendor-portal/src/shared/services/api-client.ts

import axios, {
  AxiosInstance,
  AxiosError,
  InternalAxiosRequestConfig,
} from 'axios';
import { getRefreshToken, setTokens, clearTokens } from '../utils/storage';

export interface ApiClientOptions {
  baseURL: string;
  /** Returns current access token synchronously from Zustand store */
  getToken: () => string | null;
  /** Called when refresh fails — app should navigate to login */
  onAuthFailure?: () => void;
}

// Track ongoing refresh to avoid concurrent refresh calls
let refreshPromise: Promise<string> | null = null;

export function createApiClient(options: ApiClientOptions): AxiosInstance {
  const { baseURL, getToken, onAuthFailure } = options;

  const instance = axios.create({
    baseURL,
    headers: { 'Content-Type': 'application/json' },
    timeout: 15000,
  });

  // Request interceptor: inject Bearer token (D-05)
  instance.interceptors.request.use(
    (config: InternalAxiosRequestConfig) => {
      const token = getToken();
      if (token) {
        config.headers.Authorization = `Bearer ${token}`;
      }
      return config;
    },
    (error) => Promise.reject(error)
  );

  // Response interceptor: handle 401 with token refresh (D-06)
  instance.interceptors.response.use(
    (response) => response,
    async (error: AxiosError) => {
      const originalRequest = error.config as InternalAxiosRequestConfig & {
        _retry?: boolean;
      };

      if (error.response?.status === 401 && !originalRequest._retry) {
        originalRequest._retry = true;

        try {
          // Deduplicate concurrent 401s — only one refresh in flight
          if (!refreshPromise) {
            refreshPromise = (async () => {
              const storedRefresh = await getRefreshToken();
              if (!storedRefresh) throw new Error('no_refresh_token');

              const res = await axios.post<{
                accessToken: string;
                refreshToken: string;
              }>(`${baseURL}/auth/refresh`, { refreshToken: storedRefresh });

              await setTokens({
                accessToken: res.data.accessToken,
                refreshToken: res.data.refreshToken,
              });

              return res.data.accessToken;
            })().finally(() => {
              refreshPromise = null;
            });
          }

          const newToken = await refreshPromise;
          originalRequest.headers.Authorization = `Bearer ${newToken}`;
          return instance(originalRequest);
        } catch {
          await clearTokens();
          onAuthFailure?.();
          return Promise.reject(error);
        }
      }

      return Promise.reject(error);
    }
  );

  return instance;
}
