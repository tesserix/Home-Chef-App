import * as SecureStore from "expo-secure-store";

const SESSION_KEY = "hc_session_token";

export interface BFFAutoLoginResponse {
  session_token: string;
  expires_at: number;
  user: {
    id: string;
    email: string;
    role: string;
    pool: string;
  };
}

export interface BFFSessionUser {
  user_id: string;
  email: string;
  role: string;
  pool: string;
  expires_at: number;
}

export async function autoLogin(
  bffUrl: string,
  idToken: string,
  expectedTenantId: string
): Promise<BFFAutoLoginResponse> {
  // Catch missing-env-var silent failures before they become a mysterious
  // network error against `//auth/auto-login`. The most common cause is a
  // forgotten EXPO_PUBLIC_BFF_URL / EXPO_PUBLIC_GIP_TENANT_ID in the EAS
  // build profile.
  if (!bffUrl) {
    throw new Error(
      "autoLogin: bffUrl is empty. Set EXPO_PUBLIC_BFF_URL in your .env.local or EAS build profile."
    );
  }
  if (!expectedTenantId) {
    throw new Error(
      "autoLogin: expectedTenantId is empty. Set EXPO_PUBLIC_GIP_TENANT_ID in your .env.local or EAS build profile."
    );
  }
  if (!idToken) {
    throw new Error("autoLogin: idToken is empty");
  }
  const r = await fetch(`${bffUrl}/auth/auto-login`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ id_token: idToken, expected_tenant_id: expectedTenantId }),
  });
  if (!r.ok) {
    // Try to extract the BFF's structured error body (e.g.
    // {"error":"tenant_not_allowed"}); fall back to a status-coded throw.
    // The caller is expected to map these to user-facing messages —
    // see resolveAuthErrorMessage() in this file.
    let bffError = '';
    try {
      const body = await r.json();
      bffError = typeof body?.error === 'string' ? body.error : '';
    } catch { /* not json */ }
    const e = new Error(bffError ? `auto_login_${bffError}` : `auto_login_${r.status}`);
    (e as Error & { status?: number; bffError?: string }).status = r.status;
    (e as Error & { status?: number; bffError?: string }).bffError = bffError;
    throw e;
  }
  const body: BFFAutoLoginResponse = await r.json();
  await SecureStore.setItemAsync(SESSION_KEY, body.session_token);
  return body;
}

// Maps the cryptic error codes thrown by autoLogin() (and by Firebase
// during sign-in) to short, user-facing sentences. Screens should pass
// the caught error here rather than rendering `err.message` directly —
// it's almost always more cryptic than is friendly.
export function resolveAuthErrorMessage(err: unknown): string {
  const raw = err instanceof Error ? err.message : String(err);
  const codeMap: Record<string, string> = {
    auto_login_invalid_token: "We couldn't verify your sign-in. Please try again.",
    auto_login_tenant_not_allowed:
      "This account isn't allowed for the vendor app. Use a customer or driver app instead.",
    auto_login_upstream_error:
      "We couldn't reach the kitchen service. Check your connection and try again.",
    auto_login_400: 'Something was off with the sign-in details. Please try again.',
    auto_login_401: "We couldn't verify your sign-in. Please try again.",
    auto_login_403: "This account isn't allowed in this app.",
    auto_login_500: "We're having a hiccup on our end. Try again in a moment.",
    auto_login_502: "We're having a hiccup on our end. Try again in a moment.",
    auto_login_503: 'Service is briefly unavailable. Try again in a moment.',
    auto_login_504: 'Service is briefly unavailable. Try again in a moment.',
  };
  if (codeMap[raw]) return codeMap[raw];
  // Common Firebase Auth errors
  if (raw.includes('auth/user-not-found')) return 'No account found for that email.';
  if (raw.includes('auth/wrong-password')) return 'Wrong password. Please try again.';
  if (raw.includes('auth/invalid-email')) return 'That email address looks off.';
  if (raw.includes('auth/email-already-in-use')) return 'An account already exists for that email.';
  if (raw.includes('auth/network-request-failed')) return 'Network connection lost. Try again.';
  if (raw.includes('auth/too-many-requests'))
    return 'Too many attempts. Wait a minute and try again.';
  if (raw.includes('cancel')) return 'Sign-in was cancelled.';
  // Anything else: best-effort plain-English fallback. We strip technical
  // prefixes so the user doesn't see `auto_login_502` literally.
  return "We couldn't sign you in. Please try again.";
}

export async function getStoredSession(): Promise<string | null> {
  return SecureStore.getItemAsync(SESSION_KEY);
}

export async function clearStoredSession(): Promise<void> {
  await SecureStore.deleteItemAsync(SESSION_KEY);
}

export async function fetchSessionUser(bffUrl: string): Promise<BFFSessionUser | null> {
  const tok = await getStoredSession();
  if (!tok) return null;
  const r = await fetch(`${bffUrl}/auth/session`, {
    headers: { Authorization: `Bearer ${tok}` },
  });
  if (r.status === 401) {
    await clearStoredSession();
    return null;
  }
  if (!r.ok) return null;
  return r.json();
}

export async function logoutBFF(bffUrl: string): Promise<void> {
  const tok = await getStoredSession();
  if (tok) {
    try {
      await fetch(`${bffUrl}/auth/logout`, {
        method: "POST",
        headers: { Authorization: `Bearer ${tok}` },
      });
    } catch {
      // best-effort
    }
  }
  await clearStoredSession();
}
