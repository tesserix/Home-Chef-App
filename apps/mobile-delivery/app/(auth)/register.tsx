import { router } from 'expo-router';
import { RegisterScreen } from '@homechef/mobile-shared/screens';
import { registerUser } from '@homechef/mobile-shared/api';
import { getRawFCMToken, registerDeviceToken } from '@homechef/mobile-shared/hooks';
import { useAuthStore } from '../../store/auth-store';
import { api } from '../../lib/api';

export default function RegisterPage() {
  const { setAuthResponse } = useAuthStore();

  return (
    <RegisterScreen
      onRegister={async (data) => {
        const response = await registerUser(api, data);
        await setAuthResponse(response);
        // Register FCM token after auth (D-09: raw FCM token)
        try {
          const fcmToken = await getRawFCMToken();
          if (fcmToken) await registerDeviceToken(api, fcmToken);
        } catch {
          // Non-fatal: push registration failure should not block registration
        }
        router.replace('/(tabs)');
      }}
      onNavigateToLogin={() => router.back()}
    />
  );
}
