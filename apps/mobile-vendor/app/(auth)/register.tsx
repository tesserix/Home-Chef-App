import { router } from 'expo-router';
import { LoginScreen } from '@homechef/mobile-shared/screens';
import { useAuth, autoLogin, signInWithZitadel } from '@homechef/mobile-shared/auth';
import { getRawFCMToken, registerDeviceToken } from '@homechef/mobile-shared/hooks';
import { useAuthStore } from '../../store/auth-store';
import { api } from '../../lib/api';

const BFF_URL = process.env.EXPO_PUBLIC_BFF_URL ?? '';
const AUTH_POOL = process.env.EXPO_PUBLIC_AUTH_POOL ?? 'business';
const ZITADEL_CLIENT_ID = process.env.EXPO_PUBLIC_ZITADEL_CLIENT_ID ?? '388815909956879142';

/** Hosted sign-up: Zitadel's registration screen owns names and credentials. */
export default function RegisterPage() {
  const { setAuthResponse } = useAuthStore();
  const { completeSignIn } = useAuth();

  const handleHostedSignUp = async () => {
    const { idToken } = await signInWithZitadel({
      clientId: ZITADEL_CLIENT_ID,
      scheme: 'homechef-vendor',
      register: true,
    });
    const body = await autoLogin(BFF_URL, idToken, AUTH_POOL);
    await setAuthResponse({
      user: {
        id: body.user.id,
        email: body.user.email,
        firstName: '',
        lastName: '',
        phone: '',
        role: body.user.role as never,
        avatar: null,
        fcmToken: null,
        createdAt: '',
        updatedAt: '',
      },
      accessToken: body.session_token,
    });
    await completeSignIn();
    // Register FCM token after auth (D-09: raw FCM token)
    try {
      const fcmToken = await getRawFCMToken();
      if (fcmToken) await registerDeviceToken(api, fcmToken);
    } catch { /* non-fatal */ }
    router.replace('/(tabs)');
  };

  return (
    <LoginScreen
      presentation="kitchen"
      heroImage={require('../../assets/auth-food.png')}
      brand="Fe3dr · Vendor"
      title="Open your kitchen"
      subtitle="Turn your cooking into income — your menu, your prices."
      onHostedSignIn={handleHostedSignUp}
      hostedCtaLabel="Continue to sign up"
      registerPrompt="Already have an account?"
      registerCta="Sign in"
      onNavigateToRegister={() => router.replace('/(auth)/login')}
    />
  );
}
