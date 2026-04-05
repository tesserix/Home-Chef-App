import { router } from 'expo-router';
import { LoginScreen } from '@homechef/mobile-shared/screens';
import { loginWithEmail } from '@homechef/mobile-shared/api';
import { getRawFCMToken, registerDeviceToken } from '@homechef/mobile-shared/hooks';
import { useAuthStore } from '../../store/auth-store';
import { api } from '../../lib/api';

export default function LoginPage() {
  const { setAuthResponse } = useAuthStore();

  return (
    <LoginScreen
      title="Welcome back"
      onLogin={async ({ email, password }) => {
        const response = await loginWithEmail(api, { email, password });
        await setAuthResponse(response);
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
      onNavigateToForgotPassword={() => router.push('/(auth)/forgot-password')}
    />
  );
}
