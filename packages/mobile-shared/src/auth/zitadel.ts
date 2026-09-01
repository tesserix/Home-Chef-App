// Zitadel hosted login for the mobile apps: OIDC authorization-code + PKCE
// against the native clients (ZitadelApplication claims homechef-*-mobile).
// The system browser owns credentials; the app only ever sees the id_token,
// which it exchanges at the BFF's /auth/auto-login for a session token.
import * as Crypto from "expo-crypto";
import * as SecureStore from "expo-secure-store";
import * as WebBrowser from "expo-web-browser";

const DEFAULT_ISSUER =
  process.env.EXPO_PUBLIC_ZITADEL_ISSUER ?? "https://auth.tesserix.app";
// HomeChef project in org TESSERIX; the project-aud scope below makes the
// id_token acceptable to the BFF verifier without per-client config.
const DEFAULT_PROJECT_ID =
  process.env.EXPO_PUBLIC_ZITADEL_PROJECT_ID ?? "388810586143588367";

const STORE_KEY = "hc_zitadel_auth";
// AFTER_FIRST_UNLOCK for the same reason as the BFF session token (#428):
// a background/push launch before first unlock must not read null.
const KEYCHAIN_OPTIONS: SecureStore.SecureStoreOptions = {
  keychainAccessible: SecureStore.AFTER_FIRST_UNLOCK,
};

interface StoredZitadelAuth {
  refreshToken: string;
  clientId: string;
  issuer: string;
}

// In-memory fallback mirrors bff-session.ts: unsigned simulator builds have no
// keychain entitlement, so SecureStore throws — hold the tokens in process
// memory for the app's lifetime instead of failing sign-in.
let memoryAuth: StoredZitadelAuth | null = null;

async function writeAuth(auth: StoredZitadelAuth): Promise<void> {
  try {
    await SecureStore.setItemAsync(STORE_KEY, JSON.stringify(auth), KEYCHAIN_OPTIONS);
    memoryAuth = null;
  } catch {
    memoryAuth = auth;
  }
}

async function readAuth(): Promise<StoredZitadelAuth | null> {
  try {
    const raw = await SecureStore.getItemAsync(STORE_KEY);
    if (raw != null) return JSON.parse(raw) as StoredZitadelAuth;
  } catch {
    /* Keychain unavailable or corrupt entry — fall through. */
  }
  return memoryAuth;
}

export async function clearZitadelSession(): Promise<void> {
  memoryAuth = null;
  try {
    await SecureStore.deleteItemAsync(STORE_KEY);
  } catch {
    /* Keychain unavailable — the in-memory copy is already cleared. */
  }
}

/** True when a Zitadel refresh token is stored (used to keep the session alive). */
export async function hasZitadelSession(): Promise<boolean> {
  return (await readAuth()) !== null;
}

const B64 = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";

function bytesToBase64(bytes: Uint8Array): string {
  let out = "";
  for (let i = 0; i < bytes.length; i += 3) {
    const b0 = bytes[i] ?? 0;
    const b1 = bytes[i + 1];
    const b2 = bytes[i + 2];
    out += B64.charAt(b0 >> 2);
    out += B64.charAt(((b0 & 3) << 4) | ((b1 ?? 0) >> 4));
    out += b1 === undefined ? "=" : B64.charAt(((b1 & 15) << 2) | ((b2 ?? 0) >> 6));
    out += b2 === undefined ? "=" : B64.charAt(b2 & 63);
  }
  return out;
}

function toUrlSafe(base64: string): string {
  return base64.replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
}

/** Query params from a callback URL. Hand-rolled: Hermes' URL is partial. */
export function parseCallbackParams(url: string): Record<string, string> {
  const query = (url.split("#")[0] ?? "").split("?")[1] ?? "";
  const out: Record<string, string> = {};
  for (const pair of query.split("&")) {
    if (!pair) continue;
    const eq = pair.indexOf("=");
    const key = decodeURIComponent(eq === -1 ? pair : pair.slice(0, eq));
    out[key] = eq === -1 ? "" : decodeURIComponent(pair.slice(eq + 1));
  }
  return out;
}

interface TokenResponse {
  id_token?: string;
  refresh_token?: string;
  error?: string;
}

async function tokenRequest(
  issuer: string,
  form: Record<string, string>
): Promise<TokenResponse> {
  const body = Object.entries(form)
    .map(([k, v]) => `${encodeURIComponent(k)}=${encodeURIComponent(v)}`)
    .join("&");
  const r = await fetch(`${issuer}/oauth/v2/token`, {
    method: "POST",
    headers: { "Content-Type": "application/x-www-form-urlencoded" },
    body,
  });
  let parsed: TokenResponse = {};
  try {
    parsed = (await r.json()) as TokenResponse;
  } catch {
    /* non-JSON error body */
  }
  if (!r.ok) throw new Error(`zitadel_token_${parsed.error ?? r.status}`);
  return parsed;
}

export interface ZitadelSignInOptions {
  /** Native OIDC clientId from the app's ZitadelApplication claim. */
  clientId: string;
  /** App scheme, e.g. "homechef-customer" — redirect is `${scheme}:/auth/callback`. */
  scheme: string;
  /** Open the hosted registration screen instead of sign-in. */
  register?: boolean;
  issuer?: string;
  projectId?: string;
}

/**
 * Run the hosted-login round trip in the system browser and return the
 * id_token for the BFF's /auth/auto-login. Stores the refresh token so
 * getZitadelIdToken() can silently re-mint sessions later.
 */
export async function signInWithZitadel(
  opts: ZitadelSignInOptions
): Promise<{ idToken: string }> {
  const issuer = (opts.issuer ?? DEFAULT_ISSUER).replace(/\/$/, "");
  const projectId = opts.projectId ?? DEFAULT_PROJECT_ID;
  // Single slash after the scheme — must byte-match the claim's redirectUris.
  const redirectUri = `${opts.scheme}:/auth/callback`;
  const verifier = toUrlSafe(bytesToBase64(Crypto.getRandomBytes(32)));
  const state = toUrlSafe(bytesToBase64(Crypto.getRandomBytes(16)));
  const challenge = toUrlSafe(
    await Crypto.digestStringAsync(Crypto.CryptoDigestAlgorithm.SHA256, verifier, {
      encoding: Crypto.CryptoEncoding.BASE64,
    })
  );
  const params: Record<string, string> = {
    client_id: opts.clientId,
    redirect_uri: redirectUri,
    response_type: "code",
    scope: `openid profile email offline_access urn:zitadel:iam:org:project:id:${projectId}:aud`,
    state,
    code_challenge: challenge,
    code_challenge_method: "S256",
  };
  if (opts.register) params.prompt = "create";
  const qs = Object.entries(params)
    .map(([k, v]) => `${k}=${encodeURIComponent(v)}`)
    .join("&");

  const result = await WebBrowser.openAuthSessionAsync(
    `${issuer}/oauth/v2/authorize?${qs}`,
    redirectUri
  );
  if (result.type !== "success" || !result.url) {
    throw new Error(
      result.type === "cancel" || result.type === "dismiss"
        ? "zitadel_cancelled"
        : "zitadel_browser_failed"
    );
  }
  const cb = parseCallbackParams(result.url);
  if (cb.error) throw new Error(`zitadel_${cb.error}`);
  if (cb.state !== state) throw new Error("zitadel_state_mismatch");
  if (!cb.code) throw new Error("zitadel_no_code");

  const tokens = await tokenRequest(issuer, {
    grant_type: "authorization_code",
    code: cb.code,
    redirect_uri: redirectUri,
    client_id: opts.clientId,
    code_verifier: verifier,
  });
  if (!tokens.id_token) throw new Error("zitadel_no_id_token");
  if (tokens.refresh_token) {
    await writeAuth({ refreshToken: tokens.refresh_token, clientId: opts.clientId, issuer });
  }
  return { idToken: tokens.id_token };
}

/**
 * Mint a fresh id_token from the stored refresh token, or null when there is
 * none (legacy-GIP install or signed out). An invalid_grant clears the stored
 * token so the app falls back to interactive sign-in instead of looping.
 */
export async function getZitadelIdToken(): Promise<string | null> {
  const auth = await readAuth();
  if (!auth) return null;
  try {
    const tokens = await tokenRequest(auth.issuer, {
      grant_type: "refresh_token",
      refresh_token: auth.refreshToken,
      client_id: auth.clientId,
    });
    if (tokens.refresh_token && tokens.refresh_token !== auth.refreshToken) {
      await writeAuth({ ...auth, refreshToken: tokens.refresh_token });
    }
    return tokens.id_token ?? null;
  } catch (err) {
    if (err instanceof Error && err.message.includes("invalid_grant")) {
      await clearZitadelSession();
    }
    return null;
  }
}
