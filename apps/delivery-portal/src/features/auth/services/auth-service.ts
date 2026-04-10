import type { SessionResponse } from '@/shared/types/auth';

/**
 * Delivery portal auth service — supports dual auth:
 * - Staff: /bff/* → auth-bff with x-auth-context: admin → internal Keycloak realm
 * - Driver: /driver-bff/* → auth-bff without x-auth-context → customer Keycloak realm
 */
function getBffUrl(mode: 'staff' | 'driver' = 'staff'): string {
  if (mode === 'driver') {
    const env = import.meta.env.VITE_DRIVER_BFF_URL;
    if (env) return env;
    if (typeof window !== 'undefined' && window.location.hostname !== 'localhost') {
      return `${window.location.origin}/driver-bff`;
    }
    return '/driver-bff';
  }

  const env = import.meta.env.VITE_BFF_URL;
  if (env) return env;
  if (typeof window !== 'undefined' && window.location.hostname !== 'localhost') {
    return `${window.location.origin}/bff`;
  }
  return '/bff';
}

export const authService = {
  getLoginUrl(mode: 'staff' | 'driver' = 'staff', options?: { provider?: 'google' | 'facebook'; returnTo?: string }): string {
    const bffUrl = getBffUrl(mode);
    const params = new URLSearchParams();
    params.set('returnTo', options?.returnTo || `${window.location.origin}/dashboard`);
    if (options?.provider) {
      params.set('kc_idp_hint', options.provider);
    }
    return `${bffUrl}/auth/login?${params.toString()}`;
  },

  async getSession(mode: 'staff' | 'driver' = 'staff'): Promise<SessionResponse | null> {
    try {
      const bffUrl = getBffUrl(mode);
      const res = await fetch(`${bffUrl}/auth/session`, {
        credentials: 'include',
      });
      if (!res.ok) return null;
      return await res.json();
    } catch {
      return null;
    }
  },

  async refreshSession(mode: 'staff' | 'driver' = 'staff'): Promise<boolean> {
    try {
      const bffUrl = getBffUrl(mode);
      const res = await fetch(`${bffUrl}/auth/refresh`, {
        method: 'POST',
        credentials: 'include',
      });
      return res.ok;
    } catch {
      return false;
    }
  },

  async logout(mode: 'staff' | 'driver' = 'staff'): Promise<void> {
    try {
      const bffUrl = getBffUrl(mode);
      await fetch(`${bffUrl}/auth/logout`, {
        method: 'POST',
        credentials: 'include',
      });
    } catch {
      // Ignore errors - clear local state regardless
    }
  },

  async getCsrfToken(mode: 'staff' | 'driver' = 'staff'): Promise<string | null> {
    try {
      const bffUrl = getBffUrl(mode);
      const res = await fetch(`${bffUrl}/auth/csrf`, {
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
  // The Go API at /api/v1/auth/* handles registration and login with bcrypt
  // locally, returning API-issued JWTs. Social login still uses the BFF/Keycloak
  // redirect flow above.
  // =========================================================================

  async loginWithEmail(email: string, password: string) {
    const res = await fetch('/api/v1/auth/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email, password }),
    });
    const data = await res.json();
    if (!res.ok) throw new Error(data.error || 'Invalid email or password');
    return data as { user: import('@/shared/types/auth').SessionUser; accessToken: string; refreshToken: string };
  },

  async registerWithEmail(input: {
    email: string;
    password: string;
    firstName: string;
    lastName: string;
  }) {
    const res = await fetch('/api/v1/auth/register', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(input),
    });
    const data = await res.json();
    if (!res.ok) throw new Error(data.error || 'Registration failed');
    return data as { user: import('@/shared/types/auth').SessionUser; accessToken: string; refreshToken: string };
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
