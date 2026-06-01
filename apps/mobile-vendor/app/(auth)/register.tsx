import { useEffect } from 'react';
import { Platform } from 'react-native';
import { router } from 'expo-router';
import { GoogleSignin } from '@react-native-google-signin/google-signin';
import * as AppleAuthentication from 'expo-apple-authentication';
import { RegisterScreen } from '@homechef/mobile-shared/screens';
import {
  registerWithEmail,
  useAuth,
  autoLogin,
  getIdToken,
  signInWithGoogleCredential,
  signInWithAppleCredential,
} from '@homechef/mobile-shared/auth';
import { getRawFCMToken, registerDeviceToken } from '@homechef/mobile-shared/hooks';
import { useAuthStore } from '../../store/auth-store';
import { api } from '../../lib/api';
import type { AuthResponse } from '@homechef/mobile-shared/types';

const BFF_URL = process.env.EXPO_PUBLIC_BFF_URL ?? '';
const GIP_TENANT_ID = process.env.EXPO_PUBLIC_GIP_TENANT_ID ?? '';

function bffToAuthResponse(
  body: { session_token: string; user: { id: string; email: string; role: string } },
  firstName: string,
  lastName: string,
  phone: string,
): AuthResponse {
  return {
    user: {
      id: body.user.id,
      email: body.user.email,
      firstName,
      lastName,
      phone,
      role: body.user.role as AuthResponse['user']['role'],
      avatar: null,
      fcmToken: null,
      createdAt: '',
      updatedAt: '',
    },
    accessToken: body.session_token,
  };
}

export default function RegisterPage() {
  const { setAuthResponse } = useAuthStore();
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

  async function completeOAuthFlow(): Promise<void> {
    const idToken = await getIdToken();
    if (!idToken) throw new Error('no_id_token_after_oauth');
    const body = await autoLogin(BFF_URL, idToken, GIP_TENANT_ID);
    // OAuth flows don't capture first/last name in the form — use empty
    // strings; the user can fill them in from the profile screen later.
    await setAuthResponse(bffToAuthResponse(body, '', '', ''));
    await completeSignIn();
    try {
      const fcmToken = await getRawFCMToken();
      if (fcmToken) await registerDeviceToken(api, fcmToken);
    } catch {
      // Non-fatal
    }
    router.replace('/(tabs)');
  }

  const handleGoogleSignIn = async () => {
    await GoogleSignin.hasPlayServices({ showPlayServicesUpdateDialog: true });
    const result = (await GoogleSignin.signIn()) as {
      data?: { idToken?: string | null };
      idToken?: string | null;
    };
    const googleIdToken = result?.data?.idToken ?? result?.idToken;
    if (!googleIdToken) throw new Error('Google sign-up failed: no ID token');
    await signInWithGoogleCredential(googleIdToken);
    await completeOAuthFlow();
  };

  const handleAppleSignIn = async () => {
    const cred = await AppleAuthentication.signInAsync({
      requestedScopes: [
        AppleAuthentication.AppleAuthenticationScope.FULL_NAME,
        AppleAuthentication.AppleAuthenticationScope.EMAIL,
      ],
    });
    if (!cred.identityToken) throw new Error('Apple sign-up failed: no identity token');
    await signInWithAppleCredential(cred.identityToken, '');
    await completeOAuthFlow();
  };

  return (
    <RegisterScreen
      brand="Home Chef · Vendor"
      title="Open your kitchen"
      subtitle="A few details to get you cooking"
      onRegister={async (data) => {
        await registerWithEmail(data.email, data.password);
        const idToken = await getIdToken();
        if (!idToken) throw new Error('no_id_token_after_register');
        const body = await autoLogin(BFF_URL, idToken, GIP_TENANT_ID);
        await setAuthResponse(
          bffToAuthResponse(body, data.firstName, data.lastName, data.phone ?? ''),
        );
        await completeSignIn();
        // Register FCM token after auth (D-09: raw FCM token)
        try {
          const fcmToken = await getRawFCMToken();
          if (fcmToken) await registerDeviceToken(api, fcmToken);
        } catch {
          // Non-fatal: push registration failure should not block registration
        }
        router.replace('/(tabs)');
      }}
      onGoogleSignIn={handleGoogleSignIn}
      onAppleSignIn={Platform.OS === 'ios' ? handleAppleSignIn : undefined}
      onNavigateToLogin={() => router.back()}
    />
  );
}
