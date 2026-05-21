import { useEffect } from 'react';
import { Platform } from 'react-native';
import { router } from 'expo-router';
import { GoogleSignin } from '@react-native-google-signin/google-signin';
import * as AppleAuthentication from 'expo-apple-authentication';
import { LoginScreen } from '@homechef/mobile-shared/screens';
import {
  signInWithGoogleCredential,
  signInWithAppleCredential,
  signInWithEmail,
  useAuth,
  autoLogin,
  getIdToken,
} from '@homechef/mobile-shared/auth';
import { getRawFCMToken, registerDeviceToken, authenticateWithBiometrics } from '@homechef/mobile-shared/hooks';
import { useAuthStore } from '../../store/auth-store';
import { api } from '../../lib/api';
import type { AuthResponse } from '@homechef/mobile-shared/types';

const BFF_URL = process.env.EXPO_PUBLIC_BFF_URL ?? '';
const GIP_TENANT_ID = process.env.EXPO_PUBLIC_GIP_TENANT_ID ?? '';

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

async function completeBFFLogin(): Promise<AuthResponse> {
  const idToken = await getIdToken();
  if (!idToken) throw new Error('no_id_token_after_sign_in');
  const body = await autoLogin(BFF_URL, idToken, GIP_TENANT_ID);
  return bffToAuthResponse(body);
}

export default function LoginPage() {
  const { setAuthResponse, biometricsEnabled } = useAuthStore();
  const { completeSignIn } = useAuth();

  useEffect(() => {
    const webClientId = process.env.EXPO_PUBLIC_GOOGLE_WEB_CLIENT_ID;
    if (!webClientId) {
      throw new Error('EXPO_PUBLIC_GOOGLE_WEB_CLIENT_ID is not configured');
    }
    GoogleSignin.configure({
      webClientId,
      iosClientId: process.env.EXPO_PUBLIC_GOOGLE_IOS_CLIENT_ID,
    });
  }, []);

  const handleGoogleSignIn = async () => {
    await GoogleSignin.hasPlayServices({ showPlayServicesUpdateDialog: true });
    const result = (await GoogleSignin.signIn()) as { data?: { idToken?: string | null }; idToken?: string | null };
    const googleIdToken = result?.data?.idToken ?? result?.idToken;
    if (!googleIdToken) throw new Error('Google sign-in failed: no ID token');
    await signInWithGoogleCredential(googleIdToken);
    const response = await completeBFFLogin();
    await setAuthResponse(response);
    await completeSignIn();
    try {
      const fcmToken = await getRawFCMToken();
      if (fcmToken) await registerDeviceToken(api, fcmToken);
    } catch { /* non-fatal */ }
    router.replace('/(tabs)');
  };

  const handleAppleSignIn = async () => {
    const cred = await AppleAuthentication.signInAsync({
      requestedScopes: [
        AppleAuthentication.AppleAuthenticationScope.FULL_NAME,
        AppleAuthentication.AppleAuthenticationScope.EMAIL,
      ],
    });
    if (!cred.identityToken) throw new Error('Apple sign-in failed: no identity token');
    await signInWithAppleCredential(cred.identityToken, '');
    const response = await completeBFFLogin();
    await setAuthResponse(response);
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
    if (!accessToken) throw new Error('No saved session found. Please log in with email.');
    router.replace('/(tabs)');
  };

  return (
    <LoginScreen
      title="Welcome back"
      onLogin={async ({ email, password }) => {
        await signInWithEmail(email, password);
        const response = await completeBFFLogin();
        await setAuthResponse(response);
        await completeSignIn();
        // Register FCM token after auth (D-09: raw FCM token)
        try {
          const fcmToken = await getRawFCMToken();
          if (fcmToken) await registerDeviceToken(api, fcmToken);
        } catch {
          // Non-fatal: push registration failure should not block login
        }
        router.replace('/(tabs)');
      }}
      onNavigateToRegister={() => router.push('/(auth)/register')}
      onGoogleSignIn={handleGoogleSignIn}
      onAppleSignIn={Platform.OS === 'ios' ? handleAppleSignIn : undefined}
      onBiometricLogin={biometricsEnabled ? handleBiometricLogin : undefined}
    />
  );
}
