// Two-factor support for the vendor portal.
//
// The web half of the same feature the mobile apps use: present a remembered
// device so the challenge is skipped, and route to it when the API says one is
// owed.
//
// The device token lives in localStorage rather than a cookie. It is deliberately
// NOT a cookie: the API is on another origin (api.fe3dr.com), so a cookie set
// here would never be sent with it, and one set there would be third-party and
// dropped by default in most browsers. A header the client sets explicitly is
// the only thing that reliably arrives.

const DEVICE_TOKEN_KEY = 'homechef.mfa.deviceToken';

export type MFAChannel = 'email' | 'phone';

export interface MFAChallengePrompt {
  channels: MFAChannel[];
  masked: Partial<Record<MFAChannel, string>>;
}

export function getDeviceToken(): string | null {
  try {
    return localStorage.getItem(DEVICE_TOKEN_KEY);
  } catch {
    // Private mode / storage disabled. Treat as no token — the user is
    // challenged each session rather than the portal failing to load.
    return null;
  }
}

export function setDeviceToken(token: string): void {
  try {
    localStorage.setItem(DEVICE_TOKEN_KEY, token);
  } catch {
    // ignore — the session still works, the challenge just repeats next time
  }
}

export function clearDeviceToken(): void {
  try {
    localStorage.removeItem(DEVICE_TOKEN_KEY);
  } catch {
    // ignore
  }
}

// Bridge from the api-client (module scope) to whatever is rendering the
// challenge. Mirrors emitMFARequired in mobile-shared.
type Handler = (challenge: MFAChallengePrompt) => void;
let handler: Handler | null = null;

export function setMFAChallengeHandler(fn: Handler | null): void {
  handler = fn;
}

/**
 * Raise a challenge. Called from the api-client's 403 branch.
 *
 * A no-op before anything registers, which is correct: a 403 that arrives
 * before the app has mounted has nowhere to go, and the next request will
 * raise it again.
 */
export function emitMFARequired(challenge: MFAChallengePrompt): void {
  handler?.(challenge);
}

/** Shape of the 403 body the API sends when a second factor is owed. */
export function parseMFARequired(body: unknown): MFAChallengePrompt | null {
  const data = body as
    | { error?: string; channels?: MFAChannel[]; masked?: Record<string, string> }
    | undefined;
  if (data?.error !== 'mfa_required') return null;
  return {
    channels: data.channels ?? [],
    masked: (data.masked ?? {}) as MFAChallengePrompt['masked'],
  };
}
