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
import { clearStoredSession, refreshSession } from '../auth/bff-session';

export interface UpgradeRequiredPayload {
  minVersion?: string;
  storeUrl?: string;
}

export interface ApiClientOptions {
  baseURL: string;
  /** Returns current access token synchronously from Zustand store */
  getToken: () => string | null;
  /** Called on 401 after tokens are cleared — app should navigate to login */
  onAuthFailure?: () => void;
  /** App version string sent as X-App-Version on every request (e.g. "1.0.3+12") */
  appVersion?: string;
  /** Platform sent as X-Platform on every request — needed by the server-side min-version check */
  platform?: 'ios' | 'android';
  /** Called on 426 Upgrade Required — app should route to the upgrade wall */
  onUpgradeRequired?: (payload: UpgradeRequiredPayload) => void;
  /**
   * Called on 403 when the server reports the account is paused or deleted.
   *
   * A deactivated account 403s on every protected request, so without this the
   * user signs in successfully and then hits a dead app with no way back —
   * pausing would be a one-way door only support could open. The app routes to
   * a screen offering reactivation.
   */
  onAccountBlocked?: (status: AccountBlockedStatus) => void;
  /**
   * Which app this client belongs to. Sent as X-Client-App so device trust
   * stays scoped per app — a remembered vendor device must not skip the
   * two-factor challenge in the admin app.
   */
  clientApp?: MFAClientApp;
  /**
   * Returns the stored device/elevation token, sent as X-Device-Token. Synchronous
   * because it runs on every request; the app keeps it in memory after reading
   * it from SecureStore at start-up.
   */
  getDeviceToken?: () => string | null;
  /**
   * Called on 403 mfa_required — the app routes to the challenge screen.
   *
   * Every protected request 403s until the challenge is passed, so without this
   * the user signs in and lands in an app where nothing loads and nothing
   * explains why.
   */
  onMFARequired?: (challenge: MFAChallengePrompt) => void;
}

/** Server-side reason a signed-in account is being refused. */
export type AccountBlockedStatus = 'account_deactivated' | 'account_deleted';

/** App scopes the API recognises for device trust. */
export type MFAClientApp = 'customer' | 'vendor' | 'delivery' | 'admin';

/** What the server tells the client when a second factor is owed. */
export interface MFAChallengePrompt {
  /** Channels this user can be challenged on. */
  channels: Array<'email' | 'phone'>;
  /** Masked hints ("s•••k@gmail.com") — never the full address. */
  masked: Partial<Record<'email' | 'phone', string>>;
}

export function createApiClient(options: ApiClientOptions): AxiosInstance {
  const {
    baseURL,
    getToken,
    onAuthFailure,
    appVersion,
    platform,
    onUpgradeRequired,
    onAccountBlocked,
    clientApp,
    getDeviceToken,
    onMFARequired,
  } = options;

  const instance = axios.create({
    baseURL,
    headers: { 'Content-Type': 'application/json' },
    timeout: 15000,
  });

  // Request interceptor: inject Bearer session token + version headers.
  // X-App-Version / X-Platform are a defense-in-depth backstop to the
  // explicit min-version poll — backend middleware can return 426
  // Upgrade Required on any authenticated request when a too-old
  // client tries to talk to a newer API.
  instance.interceptors.request.use(
    (config: InternalAxiosRequestConfig) => {
      const token = getToken();
      if (token) {
        config.headers.Authorization = `Bearer ${token}`;
      }
      if (appVersion) {
        config.headers['X-App-Version'] = appVersion;
      }
      if (platform) {
        config.headers['X-Platform'] = platform;
      }
      // Two-factor: name the app so device trust stays scoped, and present the
      // remembered-device token so a trusted device skips the challenge.
      if (clientApp) {
        config.headers['X-Client-App'] = clientApp;
      }
      const deviceToken = getDeviceToken?.();
      if (deviceToken) {
        config.headers['X-Device-Token'] = deviceToken;
      }
      return config;
    },
    (error) => Promise.reject(error)
  );

  // Response interceptor: 401 → silent refresh + retry once, else clear session;
  // 426 → upgrade wall.
  //
  // On a 401 we ask refreshSession() for a fresh session token (the AuthProvider
  // registers a strategy that re-mints from the durable Firebase identity — the
  // BFF /auth/refresh only sets a cookie, so mobile can't use it). If we get a
  // token, we retry the ORIGINAL request once with it; only when refresh yields
  // nothing do we tear down the session. This keeps a logged-in user signed in
  // across an expired access token instead of bouncing them to login. (#428)
  instance.interceptors.response.use(
    (response) => response,
    async (error: AxiosError) => {
      if (error.response?.status === 401) {
        // Some best-effort calls (e.g. device-token registration, which can
        // race a session refresh on cold start) must NOT tear down the session
        // on a 401 — that would bounce a logged-in user to the login screen.
        const cfg = error.config as
          | (InternalAxiosRequestConfig & {
              skipAuthFailure?: boolean;
              _retried?: boolean;
            })
          | undefined;
        if (cfg?.skipAuthFailure) {
          return Promise.reject(error);
        }
        // Try a single silent refresh + retry before giving up on the session.
        if (cfg && !cfg._retried) {
          const newToken = await refreshSession();
          if (newToken) {
            cfg._retried = true;
            cfg.headers = cfg.headers ?? {};
            (cfg.headers as Record<string, string>).Authorization = `Bearer ${newToken}`;
            return instance(cfg);
          }
        }
        try {
          await clearTokens();
          await clearStoredSession();
        } catch {
          // best-effort
        }
        onAuthFailure?.();
      } else if (error.response?.status === 426) {
        const data = (error.response.data ?? {}) as UpgradeRequiredPayload;
        onUpgradeRequired?.({
          minVersion: data.minVersion,
          storeUrl: data.storeUrl,
        });
      } else if (error.response?.status === 403) {
        // Only the account-state 403s route to the paused screen. Ordinary
        // permission 403s (wrong role, not your order) carry no status field
        // and must keep surfacing as normal errors.
        const data = error.response.data as
          | { status?: string; error?: string; channels?: Array<'email' | 'phone'>; masked?: Record<string, string> }
          | undefined;
        if (data?.status === 'account_deactivated' || data?.status === 'account_deleted') {
          onAccountBlocked?.(data.status);
        } else if (data?.error === 'mfa_required') {
          // Every protected request 403s until the challenge is passed, so this
          // has to route somewhere rather than surfacing as a generic error.
          onMFARequired?.({
            channels: data.channels ?? [],
            masked: (data.masked ?? {}) as MFAChallengePrompt['masked'],
          });
        }
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
