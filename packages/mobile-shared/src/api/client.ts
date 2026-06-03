// API client factory — creates an axios instance with auth interceptors
// for the GIP/BFF session model: the BFF mints a session token at
// auto-login time and owns refresh; the client has no separate refresh
// token. On 401 we clear local tokens and signal onAuthFailure so the
// app can route back to the login screen.

import axios, {
  AxiosInstance,
  AxiosError,
  AxiosRequestConfig,
  InternalAxiosRequestConfig,
} from 'axios';
import { clearTokens } from '../utils/storage';
import { clearStoredSession } from '../auth/bff-session';

export interface ApiClientOptions {
  baseURL: string;
  /** Returns current access token synchronously from Zustand store */
  getToken: () => string | null;
  /** Called on 401 after tokens are cleared — app should navigate to login */
  onAuthFailure?: () => void;
}

export function createApiClient(options: ApiClientOptions): AxiosInstance {
  const { baseURL, getToken, onAuthFailure } = options;

  const instance = axios.create({
    baseURL,
    headers: { 'Content-Type': 'application/json' },
    timeout: 15000,
  });

  // Request interceptor: inject Bearer session token
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

  // Response interceptor: 401 → clear session and bubble up.
  // We do NOT attempt a client-side refresh: the BFF session_token has no
  // matching refresh token on the client. If the Firebase user is still
  // signed in, the app can call AuthProvider.completeSignIn() to mint a
  // fresh session. Otherwise the user must re-authenticate.
  instance.interceptors.response.use(
    (response) => response,
    async (error: AxiosError) => {
      if (error.response?.status === 401) {
        try {
          await clearTokens();
          await clearStoredSession();
        } catch {
          // best-effort
        }
        onAuthFailure?.();
      }
      return Promise.reject(error);
    }
  );

  return instance;
}

// React Native's XHR only appends a multipart boundary to the
// Content-Type header when the header isn't already set. Our axios
// instance defaults to `application/json`, so every multipart upload
// would otherwise ship without a boundary and the Go server's
// ParseMultipartForm would reject the body.
//
// Pass this as the third arg to api.post(...) for any FormData upload:
//   await api.post('/chef/documents', formData, multipartConfig());
export function multipartConfig(extra: AxiosRequestConfig = {}): AxiosRequestConfig {
  return {
    ...extra,
    transformRequest: (data, headers) => {
      if (headers) {
        const h = headers as Record<string, unknown>;
        delete h['Content-Type'];
        delete h['content-type'];
      }
      return data;
    },
  };
}
