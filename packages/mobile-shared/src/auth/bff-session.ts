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
    throw new Error(`auto_login_${r.status}`);
  }
  const body: BFFAutoLoginResponse = await r.json();
  await SecureStore.setItemAsync(SESSION_KEY, body.session_token);
  return body;
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
