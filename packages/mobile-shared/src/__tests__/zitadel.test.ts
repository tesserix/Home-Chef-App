import { beforeEach, describe, expect, it, vi } from 'vitest';
import * as WebBrowser from 'expo-web-browser';
import {
  clearZitadelSession,
  getZitadelIdToken,
  hasZitadelSession,
  parseCallbackParams,
  signInWithZitadel,
} from '../auth/zitadel';

type MockBrowser = typeof WebBrowser & {
  __setAuthSessionHandler: (
    h: ((url: string, redirectUri: string) => { type: string; url?: string }) | null
  ) => void;
  __lastAuthRequest: { url: string; redirectUri: string } | null;
};
const browser = WebBrowser as unknown as MockBrowser;

function tokenFetchMock(body: Record<string, unknown>, ok = true) {
  return vi.fn(async () => ({
    ok,
    status: ok ? 200 : 400,
    json: async () => body,
  })) as unknown as typeof fetch;
}

beforeEach(async () => {
  await clearZitadelSession();
  browser.__setAuthSessionHandler(null);
  vi.unstubAllGlobals();
});

describe('parseCallbackParams', () => {
  it('extracts query params and ignores fragments', () => {
    expect(
      parseCallbackParams('homechef-customer:/auth/callback?code=abc%2F1&state=xyz#frag')
    ).toEqual({ code: 'abc/1', state: 'xyz' });
  });

  it('returns empty object when there is no query', () => {
    expect(parseCallbackParams('homechef-customer:/auth/callback')).toEqual({});
  });
});

describe('signInWithZitadel', () => {
  const opts = { clientId: 'client-1', scheme: 'homechef-customer' };

  it('completes the code+PKCE round trip and stores the refresh token', async () => {
    browser.__setAuthSessionHandler((url, redirectUri) => {
      const state = /[?&]state=([^&]+)/.exec(url)?.[1] ?? '';
      expect(redirectUri).toBe('homechef-customer:/auth/callback');
      expect(url).toContain('code_challenge_method=S256');
      expect(url).toContain(
        encodeURIComponent('urn:zitadel:iam:org:project:id:388810586143588367:aud')
      );
      return { type: 'success', url: `${redirectUri}?code=c1&state=${state}` };
    });
    vi.stubGlobal('fetch', tokenFetchMock({ id_token: 'idt', refresh_token: 'rt1' }));

    const { idToken } = await signInWithZitadel(opts);
    expect(idToken).toBe('idt');
    expect(await hasZitadelSession()).toBe(true);
  });

  it('rejects a tampered state', async () => {
    browser.__setAuthSessionHandler((_url, redirectUri) => ({
      type: 'success',
      url: `${redirectUri}?code=c1&state=evil`,
    }));
    await expect(signInWithZitadel(opts)).rejects.toThrow('zitadel_state_mismatch');
    expect(await hasZitadelSession()).toBe(false);
  });

  it('maps a browser cancel to zitadel_cancelled', async () => {
    browser.__setAuthSessionHandler(() => ({ type: 'cancel' }));
    await expect(signInWithZitadel(opts)).rejects.toThrow('zitadel_cancelled');
  });

  it('surfaces an OAuth error from the callback', async () => {
    browser.__setAuthSessionHandler((_url, redirectUri) => ({
      type: 'success',
      url: `${redirectUri}?error=access_denied`,
    }));
    await expect(signInWithZitadel(opts)).rejects.toThrow('zitadel_access_denied');
  });
});

describe('getZitadelIdToken', () => {
  it('returns null without a stored refresh token', async () => {
    expect(await getZitadelIdToken()).toBeNull();
  });

  it('refreshes and rotates the stored refresh token', async () => {
    browser.__setAuthSessionHandler((url, redirectUri) => {
      const state = /[?&]state=([^&]+)/.exec(url)?.[1] ?? '';
      return { type: 'success', url: `${redirectUri}?code=c1&state=${state}` };
    });
    vi.stubGlobal('fetch', tokenFetchMock({ id_token: 'idt', refresh_token: 'rt1' }));
    await signInWithZitadel({ clientId: 'client-1', scheme: 'homechef-customer' });

    const refresh = tokenFetchMock({ id_token: 'idt2', refresh_token: 'rt2' });
    vi.stubGlobal('fetch', refresh);
    expect(await getZitadelIdToken()).toBe('idt2');
    const call = (refresh as unknown as ReturnType<typeof vi.fn>).mock.calls[0] as unknown[];
    expect(String(call[0])).toContain('/oauth/v2/token');

    // invalid_grant clears the session so the app falls back to sign-in
    vi.stubGlobal('fetch', tokenFetchMock({ error: 'invalid_grant' }, false));
    expect(await getZitadelIdToken()).toBeNull();
    expect(await hasZitadelSession()).toBe(false);
  });
});
