import { router } from 'expo-router';
import { LoginScreen } from '@homechef/mobile-shared/screens';
import { useAuth, autoLogin, signInWithZitadel } from '@homechef/mobile-shared/auth';
import { getRawFCMToken, registerDeviceToken, authenticateWithBiometrics } from '@homechef/mobile-shared/hooks';
import { useAuthStore } from '../../store/auth-store';
import { api } from '../../lib/api';
import type { AuthResponse } from '@homechef/mobile-shared/types';

const BFF_URL = process.env.EXPO_PUBLIC_BFF_URL ?? '';
const AUTH_POOL = process.env.EXPO_PUBLIC_AUTH_POOL ?? 'business';
// Zitadel native client (claim homechef-vendor-mobile). Not a secret.
const ZITADEL_CLIENT_ID = process.env.EXPO_PUBLIC_ZITADEL_CLIENT_ID ?? '388815909956879142';

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
  const { completeSignIn } = useAuth();

  // Hosted Zitadel login: credentials + social providers live on the hosted
  // page; errors bubble to LoginScreen's inline banner.
  const handleHostedSignIn = async () => {
    const { idToken } = await signInWithZitadel({
      clientId: ZITADEL_CLIENT_ID,
      scheme: 'homechef-vendor',
    });
    const body = await autoLogin(BFF_URL, idToken, AUTH_POOL);
    await setAuthResponse(bffToAuthResponse(body));
    await completeSignIn();
    // Register FCM token after auth (D-09: raw FCM token)
    try {
      const fcmToken = await getRawFCMToken();
      if (fcmToken) await registerDeviceToken(api, fcmToken);
    } catch { /* non-fatal */ }
    router.replace('/(tabs)');
  };

  const handleBiometricLogin = async () => {
    const success = await authenticateWithBiometrics();
    if (!success) throw new Error('Biometric authentication failed');
    const { accessToken } = useAuthStore.getState();
    if (!accessToken) throw new Error('No saved session found. Please sign in again.');
    router.replace('/(tabs)');
  };

  return (
    <LoginScreen
      presentation="kitchen"
      heroImage={require('../../assets/auth-food.png')}
      brand="Fe3dr · Vendor"
      title="Welcome back"
      subtitle={'Your kitchen. Your community.\nLet’s get cooking.'}
      onHostedSignIn={handleHostedSignIn}
      onNavigateToRegister={() => router.push('/(auth)/register')}
      onBiometricLogin={biometricsEnabled ? handleBiometricLogin : undefined}
    />
  );
}
