import type { SessionResponse, SessionUser } from '@/shared/types/auth';

export type LoginResult =
  | { kind: 'success'; user: SessionUser; accessToken: string; refreshToken: string }
  | { kind: '2fa_required'; challengeToken: string }
  | { kind: '2fa_enrollment_required'; enrollmentToken: string };

/**
 * Admin auth service — uses same-origin /bff/ proxy to the auth BFF.
 * In production, Istio VirtualService on admin.fe3dr.com routes /bff/* to the
 * auth-bff service with x-auth-context: admin, which makes the BFF use the
 * internal Keycloak realm (tesserix-internal) for authentication.
 *
 * For the login redirect, we use the /bff/ prefix so the BFF constructs the
 * Keycloak authorization URL with the correct internal realm.
 */
const BFF_URL = (() => {
  const env = import.meta.env.VITE_BFF_URL;
  if (env) return env;
  // In production, use same-origin /bff/ proxy
  if (typeof window !== 'undefined' && window.location.hostname !== 'localhost') {
    return `${window.location.origin}/bff`;
  }
  // In development, proxy through vite dev server
  return '/bff';
})();

export const authService = {
  /**
   * Build the OIDC login URL via the BFF.
   * Supports Google and Facebook social login via kc_idp_hint.
   */
  getLoginUrl(options?: { provider?: 'google' | 'facebook'; returnTo?: string }): string {
    const params = new URLSearchParams();
    params.set('returnTo', options?.returnTo || `${window.location.origin}/dashboard`);
    if (options?.provider) {
      params.set('kc_idp_hint', options.provider);
    }
    return `${BFF_URL}/auth/login?${params.toString()}`;
  },

  /**
   * Check current session with the BFF.
   */
  async getSession(): Promise<SessionResponse | null> {
    try {
      const res = await fetch(`${BFF_URL}/auth/session`, {
        credentials: 'include',
      });
      if (!res.ok) return null;
      return await res.json();
    } catch {
      return null;
    }
  },

  /**
   * Refresh the session access token via BFF.
   */
  async refreshSession(): Promise<boolean> {
    try {
      const res = await fetch(`${BFF_URL}/auth/refresh`, {
        method: 'POST',
        credentials: 'include',
      });
      return res.ok;
    } catch {
      return false;
    }
  },

  /**
   * Logout: revoke session at BFF and clear cookies.
   */
  async logout(): Promise<void> {
    try {
      await fetch(`${BFF_URL}/auth/logout`, {
        method: 'POST',
        credentials: 'include',
      });
    } catch {
      // Ignore errors - clear local state regardless
    }
  },

  /**
   * Get CSRF token for state-changing API requests.
   */
  async getCsrfToken(): Promise<string | null> {
    try {
      const res = await fetch(`${BFF_URL}/auth/csrf`, {
        credentials: 'include',
      });
      if (!res.ok) return null;
      const data = await res.json();
      return data.csrfToken || null;
    } catch {
      return null;
    }
  },

  // =========================================================================
  // Direct API auth (email/password — no Keycloak redirect)
  // =========================================================================

  async loginWithEmail(email: string, password: string): Promise<LoginResult> {
    const res = await fetch('/api/v1/auth/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email, password }),
    });
    const data = await res.json();
    if (!res.ok) throw new Error(data.error || 'Invalid email or password');
    if (data.twoFactorRequired) {
      return { kind: '2fa_required', challengeToken: data.challengeToken as string };
    }
    if (data.twoFactorEnrollmentRequired) {
      return {
        kind: '2fa_enrollment_required',
        enrollmentToken: data.enrollmentToken as string,
      };
    }
    return {
      kind: 'success',
      user: data.user,
      accessToken: data.accessToken,
      refreshToken: data.refreshToken,
    };
  },

  /** Complete 2FA login by submitting the 6-digit TOTP code. */
  async verifyTotp(challengeToken: string, code: string) {
    const res = await fetch('/api/v1/auth/totp/verify', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ challengeToken, code }),
    });
    const data = await res.json();
    if (!res.ok) throw new Error(data.error || 'Invalid code');
    return data as {
      user: import('@/shared/types/auth').SessionUser;
      accessToken: string;
      refreshToken: string;
    };
  },

  /** Begin TOTP enrollment during a forced-enrollment login flow. */
  async enrollTotpDuringLogin(enrollmentToken: string) {
    const res = await fetch('/api/v1/auth/totp/enroll-start', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ enrollmentToken }),
    });
    const data = await res.json();
    if (!res.ok) throw new Error(data.error || 'Failed to start enrollment');
    return data as { secret: string; otpAuthUrl: string; qrCodeBase64: string };
  },

  /** Complete forced enrollment; returns real session tokens. */
  async confirmTotpDuringLogin(enrollmentToken: string, code: string) {
    const res = await fetch('/api/v1/auth/totp/enroll-verify', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ enrollmentToken, code }),
    });
    const data = await res.json();
    if (!res.ok) throw new Error(data.error || 'Invalid code');
    return data as {
      user: import('@/shared/types/auth').SessionUser;
      accessToken: string;
      refreshToken: string;
    };
  },

  async refreshApiToken(refreshToken: string) {
    const res = await fetch('/api/v1/auth/refresh', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ refreshToken }),
    });
    const data = await res.json();
    if (!res.ok) throw new Error('Token refresh failed');
    return data as { accessToken: string; refreshToken: string };
  },

  async logoutApi(refreshToken: string) {
    await fetch('/api/v1/auth/logout', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ refreshToken }),
    }).catch(() => {});
  },
};
