// DEV-ONLY escape hatch for the iOS Simulator, which has no Apple-issued signing
// cert and so cannot carry a keychain entitlement — @react-native-firebase/auth
// throws `auth/keychain-error` when it persists credentials. We re-run the same
// GIP sign-in over REST and hold the tokens in memory for the process lifetime.
import { getAuth } from "@react-native-firebase/auth";
import Constants from "expo-constants";

const GIP_HOST = "https://identitytoolkit.googleapis.com/v1";
const REFRESH_HOST = "https://securetoken.googleapis.com/v1";
const EXPIRY_SKEW_MS = 60_000;

interface DevSession {
  idToken: string;
  refreshToken: string;
  expiresAt: number;
  email: string;
}

let session: DevSession | null = null;

export function isKeychainError(err: unknown): boolean {
  const code = (err as { code?: unknown } | null)?.code;
  return typeof code === "string" && code === "auth/keychain-error";
}

export function isDevSimSessionActive(): boolean {
  return session !== null;
}

export function clearDevSimSession(): void {
  session = null;
}

function bundleId(): string {
  return Constants.expoConfig?.ios?.bundleIdentifier ?? "";
}

// RN Firebase reads its key from GoogleService-Info.plist, which is the one the
// GIP REST endpoint accepts for this bundle — not EXPO_PUBLIC_GIP_API_KEY.
function apiKey(): string {
  return getAuth().app.options.apiKey ?? "";
}

async function postJSON<T>(url: string, body: unknown): Promise<T> {
  const r = await fetch(url, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      "X-Ios-Bundle-Identifier": bundleId(),
    },
    body: JSON.stringify(body),
  });
  const parsed = (await r.json()) as T & { error?: { message?: string } };
  if (!r.ok) throw new Error(parsed?.error?.message ?? `gip_http_${r.status}`);
  return parsed;
}

/** Sign in against GIP over REST, bypassing the native Keychain write. */
export async function devSimSignIn(
  tenantId: string,
  email: string,
  password: string
): Promise<string> {
  const res = await postJSON<{
    idToken: string;
    refreshToken: string;
    expiresIn: string;
  }>(`${GIP_HOST}/accounts:signInWithPassword?key=${apiKey()}`, {
    email,
    password,
    tenantId,
    returnSecureToken: true,
  });
  session = {
    idToken: res.idToken,
    refreshToken: res.refreshToken,
    expiresAt: Date.now() + Number(res.expiresIn ?? 3600) * 1000,
    email,
  };
  return res.idToken;
}

/** Current dev id token, refreshed when it is close to expiring. */
export async function getDevSimIdToken(): Promise<string | null> {
  if (!session) return null;
  if (Date.now() < session.expiresAt - EXPIRY_SKEW_MS) return session.idToken;
  const res = await postJSON<{ id_token: string; refresh_token: string; expires_in: string }>(
    `${REFRESH_HOST}/token?key=${apiKey()}`,
    { grant_type: "refresh_token", refresh_token: session.refreshToken }
  );
  session = {
    idToken: res.id_token,
    refreshToken: res.refresh_token,
    expiresAt: Date.now() + Number(res.expires_in ?? 3600) * 1000,
    email: session.email,
  };
  return session.idToken;
}
