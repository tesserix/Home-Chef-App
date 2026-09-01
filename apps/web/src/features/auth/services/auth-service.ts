import type { SessionResponse, SessionUser } from '@/shared/types/auth';

// BFF_URL resolution:
//   1. VITE_BFF_URL env var (escape hatch)
//   2. Same-origin /bff in any non-localhost browser context (Istio
//      VirtualService rewrites /bff/* → / on the homechef-auth-bff service)
//   3. /bff fallback for SSR / build-time evaluation
const BFF_URL = (() => {
  const env = import.meta.env.VITE_BFF_URL;
  if (env) return env;
  if (typeof window !== 'undefined' && window.location.hostname !== 'localhost') {
    return `${window.location.origin}/bff`;
  }
  return '/bff';
})();

const isLocalDev = typeof window !== 'undefined' && window.location.hostname === 'localhost';
const BFF_FETCH_BASE = isLocalDev ? BFF_URL : '/bff';

/**
 * Session shape returned by the BFF. `pool` is the auth pool
 * (customer/business/internal); `expiresAt` is unix seconds.
 */
export interface AuthSession {
  userId: string;
  email: string;
  role: string;
  pool: string;
  expiresAt: number;
}

interface BffSessionResponse {
  user_id: string;
  email: string;
  role: string;
  pool: string;
  expires_at: number;
  csrf_token?: string;
  authenticated?: boolean;
}

function normalizeSession(body: BffSessionResponse): AuthSession {
  return {
    userId: body.user_id,
    email: body.email,
    role: body.role,
    pool: body.pool,
    expiresAt: body.expires_at,
  };
}

/** Convert an `AuthSession` to the legacy `SessionUser` shape. */
export function toSessionUser(session: AuthSession): SessionUser {
  return {
    id: session.userId,
    email: session.email,
    roles: session.role ? [session.role] : [],
    tenantId: session.pool,
  };
}

export interface LoginRedirectOptions {
  /** Same-origin path to land on after the callback (e.g. "/orders/42"). */
  returnTo?: string;
  /** Open the hosted registration screen instead of sign-in. */
  register?: boolean;
  /** DPDP §6 marketing-consent opt-in, only meaningful with register. */
  marketingConsent?: boolean;
}

/**
 * Full-page redirect into the hosted Zitadel login via the BFF's
 * GET /auth/login. The BFF runs the OIDC code+PKCE flow and returns with an
 * encrypted session cookie (HttpOnly, Secure, SameSite=Lax); there is no
 * in-page credential handling any more.
 */
export function redirectToLogin(options: LoginRedirectOptions = {}): void {
  const params = new URLSearchParams();
  if (options.returnTo) params.set('return_to', options.returnTo);
  if (options.register) params.set('screen', 'register');
  if (options.marketingConsent) params.set('marketing_consent', 'true');
  const qs = params.toString();
  window.location.assign(`${BFF_FETCH_BASE}/auth/login${qs ? `?${qs}` : ''}`);
}

/**
 * Ask the Fe3dr API to email a password-reset link for legacy email/password
 * accounts. Zitadel's hosted login has its own reset; this endpoint remains
 * for accounts created before the migration.
 *
 * Resolves on every outcome the server treats as normal, including "no such
 * account": it answers identically either way (anti-enumeration).
 */
export async function sendPasswordReset(email: string): Promise<void> {
  const res = await fetch(`${BFF_FETCH_BASE}/api/v1/auth/password-reset/request`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email, app: 'customer' }),
  });
  if (!res.ok) {
    throw new Error(
      "We couldn't send the reset email just now. Please check your connection and try again.",
    );
  }
}

/**
 * GET /auth/session on the BFF. Returns null when no valid session cookie is
 * present; throws on other errors so the caller can decide retry policy.
 */
export async function fetchSession(): Promise<AuthSession | null> {
  const res = await fetch(`${BFF_FETCH_BASE}/auth/session`, {
    credentials: 'include',
  });
  if (res.status === 401) return null;
  if (!res.ok) {
    throw new Error(`session_${res.status}`);
  }
  const body = (await res.json()) as BffSessionResponse;
  if (body.authenticated === false) return null;
  return normalizeSession(body);
}

/** Clear the BFF session cookie. */
export async function logout(): Promise<void> {
  try {
    const csrfToken = await fetchCsrfToken();
    await fetch(`${BFF_FETCH_BASE}/auth/logout`, {
      method: 'POST',
      credentials: 'include',
      headers: csrfToken ? { 'X-CSRF-Token': csrfToken } : undefined,
    });
  } catch {
    // best-effort
  }
}

/**
 * Best-effort fetch of a CSRF token from the BFF. Returned token is attached
 * to state-changing requests in api-client.
 */
export async function fetchCsrfToken(): Promise<string | null> {
  try {
    const res = await fetch(`${BFF_FETCH_BASE}/auth/csrf`, {
      credentials: 'include',
    });
    if (!res.ok) return null;
    const data = (await res.json()) as { csrfToken?: string };
    return data.csrfToken ?? null;
  } catch {
    return null;
  }
}

/** Legacy `authService` object — preserves the shape existing callers import. */
export const authService = {
  /**
   * Check current BFF session. Returns the legacy `SessionResponse` shape so
   * existing callers in the auth store and api-client keep working.
   */
  async getSession(): Promise<SessionResponse | null> {
    try {
      const session = await fetchSession();
      if (!session) return { authenticated: false };
      const csrfToken = await fetchCsrfToken();
      return {
        authenticated: true,
        user: toSessionUser(session),
        expiresAt: session.expiresAt,
        csrfToken: csrfToken ?? undefined,
      };
    } catch {
      return null;
    }
  },

  /** Cookie sessions refresh transparently on the BFF; kept as a shim. */
  async refreshSession(): Promise<boolean> {
    const session = await fetchSession().catch(() => null);
    return session !== null;
  },

  async logout(): Promise<void> {
    await logout();
  },

  async getCsrfToken(): Promise<string | null> {
    return fetchCsrfToken();
  },
};
