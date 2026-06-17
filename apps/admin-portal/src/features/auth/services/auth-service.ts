import {
  signInWithEmailAndPassword,
  signOut as firebaseSignOut,
  onAuthStateChanged,
  sendPasswordResetEmail,
  type User as FirebaseUser,
} from 'firebase/auth';
import { firebaseAuth } from '@/lib/firebase';
import type { SessionResponse, SessionUser, SocialProvider } from '@/shared/types/auth';

// BFF_URL resolution:
//   1. VITE_BFF_URL env var (escape hatch)
//   2. Same-origin /bff in any non-localhost browser context (Istio
//      VirtualService rewrites /bff/* → / on the homechef-auth-bff service
//      with x-auth-context: admin, pinning the internal GIP tenant pool)
//   3. /bff fallback for SSR / build-time evaluation
const BFF_URL = (() => {
  const env = import.meta.env.VITE_BFF_URL;
  if (env) return env;
  if (typeof window !== 'undefined' && window.location.hostname !== 'localhost') {
    return `${window.location.origin}/bff`;
  }
  return '/bff';
})();

// For fetch calls we always prefer same-origin /bff in non-local environments
// to avoid CORS preflight; for localhost dev we hit BFF_URL directly.
const isLocalDev = typeof window !== 'undefined' && window.location.hostname === 'localhost';
const BFF_FETCH_BASE = isLocalDev ? BFF_URL : '/bff';

/**
 * Session shape returned by the GIP-backed BFF.
 * `pool` is the GIP tenant ID; `expiresAt` is unix seconds.
 */
export interface AuthSession {
  userId: string;
  email: string;
  role: string;
  pool: string;
  expiresAt: number;
}

interface ExchangeResponse {
  user_id: string;
  email: string;
  role: string;
  pool: string;
  expires_at: number;
  csrf_token?: string;
}

interface BffSessionResponse extends ExchangeResponse {
  authenticated?: boolean;
}

/**
 * POST a Firebase ID token to the BFF's /auth/exchange endpoint.
 * The BFF verifies the token with GIP, upserts the user (admin allowlist
 * enforced server-side), and sets an encrypted session cookie. Returns the
 * normalized session for the local store.
 */
export async function postExchange(idToken: string): Promise<AuthSession> {
  const res = await fetch(`${BFF_FETCH_BASE}/auth/exchange`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    credentials: 'include',
    body: JSON.stringify({ id_token: idToken }),
  });
  if (!res.ok) {
    throw new Error(`exchange_failed_${res.status}`);
  }
  const body = (await res.json()) as ExchangeResponse;
  return normalizeSession(body);
}

function normalizeSession(body: ExchangeResponse): AuthSession {
  return {
    userId: body.user_id,
    email: body.email,
    role: body.role,
    pool: body.pool,
    expiresAt: body.expires_at,
  };
}

/**
 * Convert an `AuthSession` to the legacy `SessionUser` shape so callers
 * that read `user.roles` keep working.
 */
export function toSessionUser(session: AuthSession): SessionUser {
  return {
    id: session.userId,
    email: session.email,
    roles: session.role ? [session.role] : [],
    tenantId: session.pool,
  };
}

// =============================================================================
// Firebase-backed sign-in flows — admin portal supports Google only.
// (Apple, Facebook, and phone are intentionally omitted; internal staff
// must sign in with their corporate Google account; the BFF enforces the
// admin allowlist on /auth/exchange.)
// =============================================================================

export async function signInWithGoogle(): Promise<AuthSession> {
  // Google sign-in runs through GSI (<GoogleSignInButton/>) — avoids
  // signInWithPopup's COOP failure + hides the firebaseapp.com URL.
  throw new Error(
    'signInWithGoogle() is no longer supported — mount <GoogleSignInButton/> directly.',
  );
}

export async function signInWithEmail(
  email: string,
  password: string,
): Promise<AuthSession> {
  const cred = await signInWithEmailAndPassword(firebaseAuth, email, password);
  const idToken = await cred.user.getIdToken();
  return postExchange(idToken);
}

export async function sendPasswordReset(email: string): Promise<void> {
  await sendPasswordResetEmail(firebaseAuth, email);
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

/**
 * Sign out of Firebase and clear the BFF session cookie. Best-effort: even if
 * Firebase sign-out fails, we still call the BFF logout endpoint.
 */
export async function logout(): Promise<void> {
  try {
    await firebaseSignOut(firebaseAuth);
  } catch {
    // best-effort
  }
  try {
    await fetch(`${BFF_FETCH_BASE}/auth/logout`, {
      method: 'POST',
      credentials: 'include',
    });
  } catch {
    // ignore
  }
}

/**
 * Subscribe to Firebase auth-state changes. On sign-in, the BFF session is
 * fetched and forwarded to the callback. On sign-out, the callback receives
 * null. Returns the Firebase unsubscribe handle.
 */
export function subscribeAuth(
  cb: (session: AuthSession | null) => void,
): () => void {
  return onAuthStateChanged(firebaseAuth, async (firebaseUser: FirebaseUser | null) => {
    if (!firebaseUser) {
      cb(null);
      return;
    }
    try {
      const session = await fetchSession();
      cb(session);
    } catch {
      cb(null);
    }
  });
}

/**
 * Best-effort fetch of a CSRF token from the BFF.
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

// =============================================================================
// Legacy `authService` object — preserves the shape callers import. Firebase
// handles social sign-in directly. MFA is deferred (per migration spec); the
// admin allowlist is enforced server-side at the BFF on /auth/exchange.
// =============================================================================

export const authService = {
  /**
   * Deprecated. Callers should call `signInWithGoogle()` directly. Returns
   * null so existing callsites can detect the new auth flow and adapt.
   */
  getLoginUrl(_options?: { provider?: SocialProvider; returnTo?: string }): null {
    return null;
  },

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

  // ---------------------------------------------------------------------------
  // Email/password — backed by Firebase + BFF exchange. MFA is deferred per
  // the migration spec; the admin allowlist is enforced server-side on
  // /auth/exchange, so a successful exchange always means the user is
  // authorized.
  // ---------------------------------------------------------------------------

  async loginWithEmail(email: string, password: string): Promise<AuthSession> {
    const cred = await signInWithEmailAndPassword(firebaseAuth, email, password);
    return postExchange(await cred.user.getIdToken());
  },

  /**
   * Refresh the Firebase ID token. Returns a fresh access token; refresh
   * token is rotated by Firebase internally.
   */
  async refreshApiToken(_refreshToken: string) {
    if (!firebaseAuth.currentUser) {
      throw new Error('not_authenticated');
    }
    const accessToken = await firebaseAuth.currentUser.getIdToken(true);
    return {
      accessToken,
      refreshToken: firebaseAuth.currentUser.refreshToken,
    };
  },

  async logoutApi(_refreshToken: string) {
    await logout();
  },
};
