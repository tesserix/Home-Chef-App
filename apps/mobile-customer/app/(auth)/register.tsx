import { useEffect } from 'react';
import { Platform } from 'react-native';
import { router, useLocalSearchParams } from 'expo-router';
import { GoogleSignin } from '@react-native-google-signin/google-signin';
import * as AppleAuthentication from 'expo-apple-authentication';
import { RegisterScreen } from '@homechef/mobile-shared/screens';
import { customerColors } from '@homechef/mobile-shared/theme';
import {
  registerWithEmail,
  signInWithGoogleCredential,
  signInWithAppleCredential,
  useAuth,
  autoLogin,
  getIdToken,
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
  // A referral code may arrive via a deep link (fe3dr.com/refer/CODE → ?ref=CODE)
  // (#38). Applied once, right after the account is authenticated; best-effort.
  const { ref } = useLocalSearchParams<{ ref?: string }>();
  const applyReferral = async () => {
    const code = typeof ref === 'string' ? ref.trim() : '';
    if (!code) return;
    try {
      await api.post('/v1/customer/referral/accept', { code });
    } catch {
      // Invalid/ineligible code shouldn't block sign-up — silently ignore.
    }
  };

  // After auth, send the user to onboarding unless they've already completed it
  // (matches the root layout's gate). A brand-new sign-up has onboardingComplete
  // = false, so it lands on the onboarding wizard — not the dashboard.
  const routeAfterAuth = () => {
    const done = useAuthStore.getState().onboardingComplete;
    router.replace(done ? '/(tabs)' : '/(onboarding)/user-info');
  };

  useEffect(() => {
    const webClientId = process.env.EXPO_PUBLIC_GOOGLE_WEB_CLIENT_ID;
    if (!webClientId) {
      throw new Error('EXPO_PUBLIC_GOOGLE_WEB_CLIENT_ID is not configured');
    }
    GoogleSignin.configure({
      webClientId,
      // iosClientId is required for native sign-in on iOS. Missing on Android.
      iosClientId: process.env.EXPO_PUBLIC_GOOGLE_IOS_CLIENT_ID,
    });
  }, []);

  // Shared post-credential flow for social sign-up: exchange the federated
  // session for a BFF session and complete sign-in, mirroring the email path.
  const completeSocialSignUp = async () => {
    const idToken = await getIdToken();
    if (!idToken) throw new Error('no_id_token_after_social_sign_up');
    const body = await autoLogin(BFF_URL, idToken, GIP_TENANT_ID);
    await setAuthResponse(bffToAuthResponse(body, '', '', ''));
    await completeSignIn();
    try {
      const fcmToken = await getRawFCMToken();
      if (fcmToken) await registerDeviceToken(api, fcmToken);
    } catch {
      // Non-fatal: push registration failure should not block sign-up
    }
    await applyReferral();
    routeAfterAuth();
  };

  const handleGoogleSignIn = async () => {
    await GoogleSignin.hasPlayServices({ showPlayServicesUpdateDialog: true });
    const result = (await GoogleSignin.signIn()) as { data?: { idToken?: string | null }; idToken?: string | null };
    const googleIdToken = result?.data?.idToken ?? result?.idToken;
    if (!googleIdToken) throw new Error('Google sign-in failed: no ID token');
    await signInWithGoogleCredential(googleIdToken);
    await completeSocialSignUp();
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
    await completeSocialSignUp();
  };

  return (
    <RegisterScreen
      accent={customerColors.coral.DEFAULT}
      // THE SPEC §2 AA micro-adjustment: link text uses coral-pressed
      // (#E00B41), not the coral fill (#FF385C), which fails AA at link/body
      // text size. Fills (CTA, focus rings) stay coral via `accent` above.
      linkColor={customerColors.coral.pressed}
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
        await applyReferral();
        routeAfterAuth();
      }}
      onGoogleSignIn={handleGoogleSignIn}
      onAppleSignIn={Platform.OS === 'ios' ? handleAppleSignIn : undefined}
      onNavigateToLogin={() => router.back()}
    />
  );
}
