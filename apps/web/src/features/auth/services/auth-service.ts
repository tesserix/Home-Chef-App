import {
  signInWithPopup,
  GoogleAuthProvider,
  OAuthProvider,
  FacebookAuthProvider,
  signInWithEmailAndPassword,
  createUserWithEmailAndPassword,
  PhoneAuthProvider,
  RecaptchaVerifier,
  signInWithCredential,
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
 * The BFF verifies the token with GIP, upserts the user, and sets an
 * encrypted session cookie (HttpOnly, Secure, SameSite=Lax). Returns the
 * normalized session for the local store.
 */
async function postExchange(idToken: string): Promise<AuthSession> {
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
 * that read `user.firstName` / `user.roles` keep working.
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
// Firebase-backed sign-in flows
// =============================================================================

export async function signInWithGoogle(): Promise<AuthSession> {
  const cred = await signInWithPopup(firebaseAuth, new GoogleAuthProvider());
  const idToken = await cred.user.getIdToken();
  return postExchange(idToken);
}

export async function signInWithApple(): Promise<AuthSession> {
  const provider = new OAuthProvider('apple.com');
  provider.addScope('email');
  provider.addScope('name');
  const cred = await signInWithPopup(firebaseAuth, provider);
  const idToken = await cred.user.getIdToken();
  return postExchange(idToken);
}

export async function signInWithFacebook(): Promise<AuthSession> {
  const cred = await signInWithPopup(firebaseAuth, new FacebookAuthProvider());
  const idToken = await cred.user.getIdToken();
  return postExchange(idToken);
}

export async function signInWithEmail(
  email: string,
  password: string,
): Promise<AuthSession> {
  const cred = await signInWithEmailAndPassword(firebaseAuth, email, password);
  const idToken = await cred.user.getIdToken();
  return postExchange(idToken);
}

export async function registerWithEmail(
  email: string,
  password: string,
): Promise<AuthSession> {
  const cred = await createUserWithEmailAndPassword(firebaseAuth, email, password);
  const idToken = await cred.user.getIdToken();
  return postExchange(idToken);
}

export async function startPhoneSignIn(
  phone: string,
  recaptchaContainerId: string,
): Promise<string> {
  const verifier = new RecaptchaVerifier(firebaseAuth, recaptchaContainerId, {
    size: 'invisible',
  });
  const provider = new PhoneAuthProvider(firebaseAuth);
  return provider.verifyPhoneNumber(phone, verifier);
}

export async function completePhoneSignIn(
  verificationId: string,
  code: string,
): Promise<AuthSession> {
  const phoneCred = PhoneAuthProvider.credential(verificationId, code);
  const cred = await signInWithCredential(firebaseAuth, phoneCred);
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

// =============================================================================
// Legacy `authService` object — preserves the shape that existing callers
// import. Firebase handles social sign-in directly on the page, so
// getLoginUrl/getRegisterUrl return null and callers use the typed helpers
// above (signInWithGoogle, etc).
// =============================================================================

export const authService = {
  /**
   * Deprecated. Callers should call `signInWithGoogle()` / `signInWithFacebook()`
   * directly. Returns null so existing callsites can detect the new auth flow
   * and adapt.
   */
  getLoginUrl(_options?: { provider?: SocialProvider; returnTo?: string }): null {
    return null;
  },

  /**
   * Deprecated. Registration is handled inline via `registerWithEmail()` or
   * social sign-in helpers.
   */
  getRegisterUrl(_returnTo?: string): null {
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

  /**
   * Cookie sessions on the BFF are refreshed transparently on each
   * `/auth/exchange` round-trip; there's no separate refresh endpoint to
   * call from the client. Kept as a shim for any legacy callers.
   */
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
  // Email/password — now backed by Firebase instead of the direct API JWT path.
  // The return shape (`{ user, accessToken, refreshToken }`) is kept for
  // backward compatibility with callers; accessToken/refreshToken come from
  // Firebase so the api-client can keep sending them as bearer headers if
  // needed, but the canonical auth surface is the BFF session cookie.
  // ---------------------------------------------------------------------------

  async loginWithEmail(email: string, password: string) {
    const cred = await signInWithEmailAndPassword(firebaseAuth, email, password);
    const accessToken = await cred.user.getIdToken();
    const session = await postExchange(accessToken);
    return {
      user: toSessionUser(session),
      accessToken,
      refreshToken: cred.user.refreshToken,
    };
  },

  async registerWithEmail(input: {
    email: string;
    password: string;
    firstName: string;
    lastName: string;
  }) {
    const cred = await createUserWithEmailAndPassword(
      firebaseAuth,
      input.email,
      input.password,
    );
    const accessToken = await cred.user.getIdToken();
    const session = await postExchange(accessToken);
    return {
      user: {
        ...toSessionUser(session),
        firstName: input.firstName,
        lastName: input.lastName,
      },
      accessToken,
      refreshToken: cred.user.refreshToken,
    };
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
