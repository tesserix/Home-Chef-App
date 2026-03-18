import type { SessionResponse, SocialProvider } from '@/shared/types/auth';

const BFF_URL = import.meta.env.VITE_BFF_URL || 'https://identity.fe3dr.com';

// Same-origin BFF proxy to avoid CORS — Istio rewrites /bff/* → / on BFF.
// Login/register URLs need the full BFF_URL because the browser navigates to them.
// Fetch-based calls use same-origin /bff/ prefix to avoid cross-origin preflight issues.
const isLocalDev = typeof window !== 'undefined' && window.location.hostname === 'localhost';
const BFF_FETCH_BASE = isLocalDev ? BFF_URL : '/bff';

/**
 * Auth service that integrates with the Keycloak-backed BFF at identity.fe3dr.com.
 * Uses OIDC redirect flow for login/register and session cookies for auth state.
 */
export const authService = {
  /**
   * Build the OIDC login URL. Redirects the browser to Keycloak via the BFF.
   * Uses full BFF_URL because these are browser navigation redirects, not fetch calls.
   */
  getLoginUrl(options?: { provider?: SocialProvider; returnTo?: string }): string {
    const params = new URLSearchParams();
    params.set('returnTo', options?.returnTo || window.location.origin);
    if (options?.provider) {
      params.set('kc_idp_hint', options.provider);
    }
    return `${BFF_URL}/auth/login?${params.toString()}`;
  },

  /**
   * Build the registration URL. Redirects to Keycloak registration form via BFF.
   */
  getRegisterUrl(returnTo?: string): string {
    const params = new URLSearchParams();
    params.set('returnTo', returnTo || window.location.origin);
    params.set('kc_action', 'register');
    return `${BFF_URL}/auth/login?${params.toString()}`;
  },

  /**
   * Check current session with the BFF.
   * Returns session user data if authenticated, null otherwise.
   */
  async getSession(): Promise<SessionResponse | null> {
    try {
      const res = await fetch(`${BFF_FETCH_BASE}/auth/session`, {
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
      const res = await fetch(`${BFF_FETCH_BASE}/auth/refresh`, {
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
      await fetch(`${BFF_FETCH_BASE}/auth/logout`, {
        method: 'POST',
        credentials: 'include',
      });
    } catch {
      // Ignore errors - we'll clear local state regardless
    }
  },

  /**
   * Get CSRF token for state-changing API requests.
   */
  async getCsrfToken(): Promise<string | null> {
    try {
      const res = await fetch(`${BFF_FETCH_BASE}/auth/csrf`, {
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
