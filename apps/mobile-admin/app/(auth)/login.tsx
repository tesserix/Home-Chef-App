import { router } from 'expo-router';
import { LoginScreen } from '@homechef/mobile-shared/screens';
import { useAuth, autoLogin, signInWithZitadel } from '@homechef/mobile-shared/auth';
import { authenticateWithBiometrics } from '@homechef/mobile-shared/hooks';
import { useAuthStore } from '../../store/auth-store';
import type { AuthResponse } from '@homechef/mobile-shared/types';

const BFF_URL = process.env.EXPO_PUBLIC_BFF_URL ?? '';
const AUTH_POOL = process.env.EXPO_PUBLIC_AUTH_POOL ?? 'internal';
// Zitadel native client (claim homechef-admin-mobile). Not a secret.
const ZITADEL_CLIENT_ID = process.env.EXPO_PUBLIC_ZITADEL_CLIENT_ID ?? '388815912104362790';

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

  // Hosted Zitadel login. The "internal" pool resolves to role=admin
  // server-side, still gated by the HOMECHEF_ADMIN_ALLOWED_EMAILS allowlist
  // and a verified email — the client asserts nothing.
  const handleHostedSignIn = async () => {
    const { idToken } = await signInWithZitadel({
      clientId: ZITADEL_CLIENT_ID,
      scheme: 'homechef-admin',
    });
    const body = await autoLogin(BFF_URL, idToken, AUTH_POOL);
    await setAuthResponse(bffToAuthResponse(body));
    await completeSignIn();
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
      brand="Fe3dr · Admin"
      title="Admin console"
      subtitle="Sign in to manage the Fe3dr platform"
      onHostedSignIn={handleHostedSignIn}
      onBiometricLogin={biometricsEnabled ? handleBiometricLogin : undefined}
    />
  );
}
