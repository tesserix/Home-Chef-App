import { router } from 'expo-router';
import { LoginScreen } from '@homechef/mobile-shared/screens';
import { customerColors } from '@homechef/mobile-shared/theme';
import { useAuth, autoLogin, signInWithZitadel } from '@homechef/mobile-shared/auth';
import { getRawFCMToken, registerDeviceToken, authenticateWithBiometrics } from '@homechef/mobile-shared/hooks';
import { useAuthStore } from '../../store/auth-store';
import { api } from '../../lib/api';
import type { AuthResponse } from '@homechef/mobile-shared/types';

const BFF_URL = process.env.EXPO_PUBLIC_BFF_URL ?? '';
const AUTH_POOL = process.env.EXPO_PUBLIC_AUTH_POOL ?? 'customer';
// Zitadel native client (claim homechef-customer-mobile). Not a secret.
const ZITADEL_CLIENT_ID = process.env.EXPO_PUBLIC_ZITADEL_CLIENT_ID ?? '388815907910058790';

/**
 * Convert a BFF auto-login response into the legacy AuthResponse shape so the
 * existing Zustand auth-store + axios api client keep working unchanged.
 * The session_token from the BFF replaces the previous JWT access token.
 * No refresh token: the BFF owns refresh; the client re-auto-logins via the
 * stored Zitadel refresh token (see provider's session refresher).
 */
function bffToAuthResponse(body: {
  session_token: string;
  user: { id: string; email: string; role: string };
}): AuthResponse {
  return {
    user: {
      id: body.user.id,
      email: body.user.email,
      firstName: '',
      lastName: '',
      phone: '',
      role: body.user.role as AuthResponse['user']['role'],
      avatar: null,
      fcmToken: null,
      createdAt: '',
      updatedAt: '',
    },
    accessToken: body.session_token,
  };
}

export default function LoginPage() {
  const { setAuthResponse, biometricsEnabled } = useAuthStore();
  // useAuth is wired via <AuthProvider> in _layout.tsx; ensures BFF session
  // is mirrored into AuthContext state.
  const { completeSignIn } = useAuth();

  // Hosted Zitadel login: credentials + social providers live on the hosted
  // page, so this screen only opens the browser and finishes the exchange.
  const handleHostedSignIn = async () => {
    const { idToken } = await signInWithZitadel({
      clientId: ZITADEL_CLIENT_ID,
      scheme: 'homechef-customer',
    });
    const body = await autoLogin(BFF_URL, idToken, AUTH_POOL);
    await setAuthResponse(bffToAuthResponse(body));
    await completeSignIn();
    try {
      const fcmToken = await getRawFCMToken();
      if (fcmToken) await registerDeviceToken(api, fcmToken);
    } catch { /* non-fatal */ }
    router.replace('/(tabs)');
  };

  const handleBiometricLogin = async () => {
    const success = await authenticateWithBiometrics();
    if (!success) throw new Error('Biometric authentication failed');
    // Token is already in secure store from previous login — just confirm it's still valid
    const { accessToken: token } = useAuthStore.getState();
    if (!token) throw new Error('No saved session found. Please sign in again.');
    // Auth guard in _layout.tsx will detect isAuthenticated=true and redirect
    router.replace('/(tabs)');
  };

  return (
    <LoginScreen
      title="Welcome back"
      subtitle="For those who love to eat — and those who love to cook."
      accent={customerColors.coral.DEFAULT}
      // THE SPEC §2 AA micro-adjustment: link text uses coral-pressed
      // (#E00B41), not the coral fill (#FF385C), which fails AA at link/body
      // text size. Fills (CTA, focus rings) stay coral via `accent` above.
      linkColor={customerColors.coral.pressed}
      onHostedSignIn={handleHostedSignIn}
      onNavigateToRegister={() => router.push('/(auth)/register')}
      onBiometricLogin={biometricsEnabled ? handleBiometricLogin : undefined}
      // App Review 5.1.1(iv): browsing chefs and menus needs no account, so the
      // wall moves to the point of ordering (hooks/useRequireAccount.ts).
      onContinueAsGuest={async () => {
        await useAuthStore.getState().setGuest(true);
        router.replace('/(tabs)');
      }}
    />
  );
}
