import { router, useLocalSearchParams } from 'expo-router';
import { LoginScreen } from '@homechef/mobile-shared/screens';
import { customerColors } from '@homechef/mobile-shared/theme';
import { useAuth, autoLogin, signInWithZitadel } from '@homechef/mobile-shared/auth';
import { getRawFCMToken, registerDeviceToken } from '@homechef/mobile-shared/hooks';
import { useAuthStore } from '../../store/auth-store';
import { api } from '../../lib/api';

const BFF_URL = process.env.EXPO_PUBLIC_BFF_URL ?? '';
const AUTH_POOL = process.env.EXPO_PUBLIC_AUTH_POOL ?? 'customer';
const ZITADEL_CLIENT_ID = process.env.EXPO_PUBLIC_ZITADEL_CLIENT_ID ?? '388815907910058790';

/**
 * Hosted sign-up: Zitadel's registration screen owns names, credentials and
 * email verification, so the in-app multi-step form (and its draft store) is
 * gone. Profile details land in onboarding instead.
 */
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

  const handleHostedSignUp = async () => {
    const { idToken } = await signInWithZitadel({
      clientId: ZITADEL_CLIENT_ID,
      scheme: 'homechef-customer',
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
    try {
      const fcmToken = await getRawFCMToken();
      if (fcmToken) await registerDeviceToken(api, fcmToken);
    } catch { /* non-fatal */ }
    await applyReferral();
    routeAfterAuth();
  };

  return (
    <LoginScreen
      title="Create your account"
      subtitle="Your first home-cooked meal is minutes away"
      accent={customerColors.coral.DEFAULT}
      linkColor={customerColors.coral.pressed}
      onHostedSignIn={handleHostedSignUp}
      hostedCtaLabel="Continue to sign up"
      registerPrompt="Already have an account?"
      registerCta="Sign in"
      onNavigateToRegister={() => router.replace('/(auth)/login')}
    />
  );
}
