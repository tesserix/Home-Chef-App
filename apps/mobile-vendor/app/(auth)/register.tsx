import { useEffect } from 'react';
import { Platform } from 'react-native';
import { router } from 'expo-router';
import { GoogleSignin } from '@react-native-google-signin/google-signin';
import { RegisterScreen } from '@homechef/mobile-shared/screens';
import {
  registerWithEmail,
  useAuth,
  autoLogin,
  getIdToken,
  signInWithGoogleCredential,
  signInWithApple,
  linkPendingAppleGrant,
} from '@homechef/mobile-shared/auth';
import { getRawFCMToken, registerDeviceToken } from '@homechef/mobile-shared/hooks';
import { useMemo } from 'react';
import { useAuthStore } from '../../store/auth-store';
import { useRegisterDraftStore } from '../../store/register-draft-store';
import { api } from '../../lib/api';
import type { AuthResponse } from '@homechef/mobile-shared/types';

const BFF_URL = process.env.EXPO_PUBLIC_BFF_URL ?? '';
const AUTH_POOL = process.env.EXPO_PUBLIC_AUTH_POOL ?? 'business';

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
  // Resume a half-finished sign-up after an app kill (name/email/phone only —
  // passwords are never persisted). Cleared on success or explicit cancel.
  const draft = useRegisterDraftStore();
  const draftValues = useMemo(
    () => ({
      firstName: draft.firstName,
      lastName: draft.lastName,
      email: draft.email,
      phone: draft.phone,
    }),
    [draft.firstName, draft.lastName, draft.email, draft.phone],
  );

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
    const body = await autoLogin(BFF_URL, idToken, AUTH_POOL);
    // OAuth flows don't capture first/last name in the form — use empty
    // strings; the user can fill them in from the profile screen later.
    await setAuthResponse(bffToAuthResponse(body, '', '', ''));
    await completeSignIn();
    // No-ops unless the credential above came from Apple. Records the grant so
    // account deletion can revoke it (App Review 5.1.1(v)).
    await linkPendingAppleGrant(api);
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
    await signInWithApple();
    await completeOAuthFlow();
  };

  return (
    <RegisterScreen
      brand="Fe3dr · Vendor"
      title="Open your kitchen"
      subtitle="A few details to get you cooking"
      step2Title="Almost there"
      step2Subtitle="Set a password and your kitchen is open."
      backdrop="kitchen"
      incentive={{
        title: 'Turn your cooking into income',
        body: 'Your menu, your prices — payouts go straight to your bank.',
      }}
      initialValues={draftValues}
      onDraftSave={(d) => draft.update({ ...d, phone: d.phone ?? '' })}
      onCancel={() => {
        useRegisterDraftStore.getState().reset();
        router.replace('/(auth)/login');
      }}
      onRegister={async (data) => {
        await registerWithEmail(data.email, data.password);
        const idToken = await getIdToken();
        if (!idToken) throw new Error('no_id_token_after_register');
        const body = await autoLogin(BFF_URL, idToken, AUTH_POOL);
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
        useRegisterDraftStore.getState().reset();
        router.replace('/(tabs)');
      }}
      onGoogleSignIn={handleGoogleSignIn}
      onAppleSignIn={Platform.OS === 'ios' ? handleAppleSignIn : undefined}
      onNavigateToLogin={() => router.back()}
    />
  );
}
