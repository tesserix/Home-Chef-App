import { useEffect } from 'react';

const BFF_URL = import.meta.env.VITE_BFF_URL || 'https://identity.fe3dr.com';

/**
 * Password reset is handled by Keycloak directly.
 * This page redirects to the Keycloak account management page.
 */
export default function ForgotPasswordPage() {
  useEffect(() => {
    // Redirect to Keycloak's password reset via the BFF login flow
    const params = new URLSearchParams();
    params.set('returnTo', `${window.location.origin}/login`);
    params.set('kc_action', 'UPDATE_PASSWORD');
    window.location.href = `${BFF_URL}/auth/login?${params.toString()}`;
  }, []);

  return (
    <div className="flex min-h-screen items-center justify-center">
      <p className="text-muted-foreground">Redirecting to password reset...</p>
    </div>
  );
}
