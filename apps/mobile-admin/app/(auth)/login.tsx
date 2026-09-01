import { useEffect } from 'react';
import { Platform } from 'react-native';
import { router } from 'expo-router';
import { GoogleSignin } from '@react-native-google-signin/google-signin';
import { LoginScreen } from '@homechef/mobile-shared/screens';
import {
  signInWithGoogleCredential,
  signInWithApple,
  signInWithEmail,
  useAuth,
  autoLogin,
  getIdToken,
  resolveAuthErrorMessage,
} from '@homechef/mobile-shared/auth';
import { authenticateWithBiometrics } from '@homechef/mobile-shared/hooks';
import { useAuthStore } from '../../store/auth-store';
import type { AuthResponse } from '@homechef/mobile-shared/types';
import { useAlert } from '@homechef/mobile-shared/ui';

const BFF_URL = process.env.EXPO_PUBLIC_BFF_URL ?? '';
const AUTH_POOL = process.env.EXPO_PUBLIC_AUTH_POOL ?? 'internal';

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

// Exchange the Firebase ID token for a BFF admin session. The "internal" pool
// (HomeChef-Internal-gyofe) resolves to role=admin server-side.
async function completeBFFLogin(): Promise<AuthResponse> {
  const idToken = await getIdToken();
  if (!idToken) throw new Error('no_id_token_after_sign_in');
  const body = await autoLogin(BFF_URL, idToken, AUTH_POOL);
  return bffToAuthResponse(body);
}

export default function LoginPage() {
  const { showAlert } = useAlert();
  const { setAuthResponse, biometricsEnabled } = useAuthStore();
  const { completeSignIn } = useAuth();

  useEffect(() => {
    const webClientId = process.env.EXPO_PUBLIC_GOOGLE_WEB_CLIENT_ID;
    if (!webClientId) return; // Google sign-in optional; email/password still works.
    GoogleSignin.configure({
      webClientId,
      iosClientId: process.env.EXPO_PUBLIC_GOOGLE_IOS_CLIENT_ID,
    });
  }, []);

  const handleGoogleSignIn = async () => {
    try {
      await GoogleSignin.hasPlayServices({ showPlayServicesUpdateDialog: true });
      const result = (await GoogleSignin.signIn()) as {
        data?: { idToken?: string | null };
        idToken?: string | null;
      };
      const googleIdToken = result?.data?.idToken ?? result?.idToken;
      if (!googleIdToken) throw new Error('Google sign-in failed: no ID token');
      await signInWithGoogleCredential(googleIdToken);
      const response = await completeBFFLogin();
      await setAuthResponse(response);
      await completeSignIn();
      router.replace('/(tabs)');
    } catch (err: unknown) {
      showAlert('Sign-in failed', resolveAuthErrorMessage(err));
    }
  };

  const handleAppleSignIn = async () => {
    try {
      await signInWithApple();
      const response = await completeBFFLogin();
      await setAuthResponse(response);
      await completeSignIn();
      router.replace('/(tabs)');
    } catch (err: unknown) {
      showAlert('Sign-in failed', resolveAuthErrorMessage(err));
    }
  };

  const handleBiometricLogin = async () => {
    try {
      const success = await authenticateWithBiometrics();
      if (!success) throw new Error('Biometric authentication failed');
      const { accessToken } = useAuthStore.getState();
      if (!accessToken) throw new Error('No saved session found. Please log in with email.');
      router.replace('/(tabs)');
    } catch (err: unknown) {
      showAlert('Sign-in failed', resolveAuthErrorMessage(err));
    }
  };

  return (
    <LoginScreen
      brand="Fe3dr · Admin"
      title="Admin console"
      subtitle="Sign in to manage the Fe3dr platform"
      onLogin={async ({ email, password }) => {
        try {
          await signInWithEmail(email, password);
          const response = await completeBFFLogin();
          await setAuthResponse(response);
          await completeSignIn();
          router.replace('/(tabs)');
        } catch (err: unknown) {
          showAlert('Sign-in failed', resolveAuthErrorMessage(err));
        }
      }}
      onGoogleSignIn={handleGoogleSignIn}
      onAppleSignIn={Platform.OS === 'ios' ? handleAppleSignIn : undefined}
      onBiometricLogin={biometricsEnabled ? handleBiometricLogin : undefined}
    />
  );
}
