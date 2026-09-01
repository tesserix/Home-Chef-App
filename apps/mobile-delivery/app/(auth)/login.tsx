import { router } from 'expo-router';
import { LoginScreen } from '@homechef/mobile-shared/screens';
import { useAuth, autoLogin, signInWithZitadel } from '@homechef/mobile-shared/auth';
import { getRawFCMToken, registerDeviceToken, authenticateWithBiometrics } from '@homechef/mobile-shared/hooks';
import { useAuthStore } from '../../store/auth-store';
import { api } from '../../lib/api';
import type { AuthResponse } from '@homechef/mobile-shared/types';

const BFF_URL = process.env.EXPO_PUBLIC_BFF_URL ?? '';
const AUTH_POOL = process.env.EXPO_PUBLIC_AUTH_POOL ?? 'business';
// Zitadel native client (claim homechef-delivery-mobile). Not a secret.
const ZITADEL_CLIENT_ID = process.env.EXPO_PUBLIC_ZITADEL_CLIENT_ID ?? '388815912691565350';

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

  const handleHostedSignIn = async () => {
    const { idToken } = await signInWithZitadel({
      clientId: ZITADEL_CLIENT_ID,
      scheme: 'homechef-delivery',
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
    const { accessToken } = useAuthStore.getState();
    if (!accessToken) throw new Error('No saved session found. Please sign in again.');
    router.replace('/(tabs)');
  };

  return (
    <LoginScreen
      brand="Fe3dr · Delivery"
      title="Welcome back"
      subtitle="Sign in to start delivering"
      onHostedSignIn={handleHostedSignIn}
      onNavigateToRegister={() => router.push('/(auth)/register')}
      onBiometricLogin={biometricsEnabled ? handleBiometricLogin : undefined}
    />
  );
}
