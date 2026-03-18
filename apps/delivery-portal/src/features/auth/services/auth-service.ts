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
};
