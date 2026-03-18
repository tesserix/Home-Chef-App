import type { SessionResponse, SocialProvider } from '@/shared/types/auth';

const BFF_URL = import.meta.env.VITE_BFF_URL || 'https://identity.fe3dr.com';

/**
 * Auth service that integrates with the Keycloak-backed BFF at identity.fe3dr.com.
 * Uses OIDC redirect flow for login/register and session cookies for auth state.
 */
export const authService = {
  /**
   * Build the OIDC login URL. Redirects the browser to Keycloak via the BFF.
   * @param provider - Optional social provider (google, facebook, instagram) passed as kc_idp_hint
   * @param returnTo - URL to redirect back to after login (defaults to /dashboard)
   */
  getLoginUrl(options?: { provider?: SocialProvider; returnTo?: string }): string {
    const params = new URLSearchParams();
    params.set('returnTo', options?.returnTo || `${window.location.origin}/dashboard`);
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
    params.set('returnTo', returnTo || `${window.location.origin}/dashboard`);
    params.set('kc_action', 'register');
    return `${BFF_URL}/auth/login?${params.toString()}`;
  },

  /**
   * Check current session with the BFF.
   * Returns session user data if authenticated, null otherwise.
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
      // Ignore errors - we'll clear local state regardless
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
};
