// Browser-side helper that exchanges a Google credential JWT for a GIP
// id_token in the per-tenant pool, using the Identity Toolkit REST API.
// Ported from mark8ly/apps/onboarding/lib/gip/customer-signin.ts.
//
// The GSI credential is signed by Google; signInWithIdp verifies the
// signature, scopes the user to the configured GIP tenant, and returns
// a GIP-issued id_token suitable for /bff/auth/exchange.

const GIP_API_KEY = import.meta.env.VITE_GIP_API_KEY as string | undefined;
const GIP_TENANT_ID = import.meta.env.VITE_GIP_TENANT_ID as string | undefined;

export class GIPError extends Error {
  constructor(
    public code: string,
    message: string,
  ) {
    super(message);
  }
}

export type GIPSignInResult =
  | { kind: 'ok'; uid: string; idToken: string }
  | {
      kind: 'needConfirmation';
      email: string;
      pendingIdpCredential: string;
      verifiedProvider: string[];
    };

interface IdpResponseBody {
  localId?: string;
  idToken?: string;
  needConfirmation?: boolean;
  email?: string;
  oauthIdToken?: string;
  verifiedProvider?: string[];
}

/**
 * Exchange a Google credential JWT for a GIP id_token in the tenant pool
 * configured via VITE_GIP_TENANT_ID. Returns a discriminated union: when
 * the email is already linked to a different provider AND the GIP project
 * has account linking enabled, GIP returns 200 with needConfirmation=true
 * plus a pending Google credential the UI can use to link providers.
 */
export async function signInWithGoogleCredential(
  googleIdToken: string,
): Promise<GIPSignInResult> {
  if (!GIP_API_KEY) {
    throw new GIPError(
      'config_missing',
      'GIP Web API key is not configured (VITE_GIP_API_KEY)',
    );
  }
  if (!GIP_TENANT_ID) {
    throw new GIPError(
      'config_missing',
      'GIP tenant id is not configured (VITE_GIP_TENANT_ID)',
    );
  }

  const url = `https://identitytoolkit.googleapis.com/v1/accounts:signInWithIdp?key=${encodeURIComponent(GIP_API_KEY)}`;
  const requestUri =
    typeof window !== 'undefined'
      ? window.location.origin
      : 'https://fe3dr.com';

  const res = await fetch(url, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      tenantId: GIP_TENANT_ID,
      requestUri,
      postBody: `id_token=${encodeURIComponent(googleIdToken)}&providerId=google.com`,
      returnSecureToken: true,
      returnIdpCredential: true,
    }),
  });

  if (!res.ok) {
    let body: { error?: { message?: string } } = {};
    try {
      body = (await res.json()) as { error?: { message?: string } };
    } catch {
      // ignore parse errors
    }
    throw new GIPError(
      'google_signin_failed',
      body.error?.message ?? `HTTP ${res.status}`,
    );
  }

  const data = (await res.json()) as IdpResponseBody;

  if (data.needConfirmation && data.email && data.oauthIdToken) {
    return {
      kind: 'needConfirmation',
      email: data.email,
      pendingIdpCredential: data.oauthIdToken,
      verifiedProvider: data.verifiedProvider ?? [],
    };
  }

  if (!data.localId || !data.idToken) {
    throw new GIPError(
      'malformed_response',
      'GIP response missing required fields',
    );
  }

  return { kind: 'ok', uid: data.localId, idToken: data.idToken };
}
