import { useEffect, useRef } from 'react';
import { Platform, View } from 'react-native';
import { Stack, router } from 'expo-router';
import { OfflineBanner } from '@homechef/mobile-shared';
import * as Notifications from 'expo-notifications';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { useAuthStore } from '../store/auth-store';
import { useBiometricLock } from '@homechef/mobile-shared/hooks';
import { getRawFCMToken, registerDeviceToken } from '@homechef/mobile-shared/hooks';
import { api } from '../lib/api';

const queryClient = new QueryClient({
  defaultOptions: {
    queries: { staleTime: 1000 * 60 * 5, retry: 2 },
  },
});

// Set global notification handler at module level — before any notification arrives.
Notifications.setNotificationHandler({
  handleNotification: async () => ({
    shouldShowAlert: true,
    shouldPlaySound: true,
    shouldSetBadge: true,
  }),
});

export default function RootLayout() {
  const { isAuthenticated, isLoading, onboardingComplete, hydrateFromStorage } =
    useAuthStore();

  // Cleanup ref for push subscription teardown.
  const pushCleanupRef = useRef<(() => void) | null>(null);

  useEffect(() => {
    hydrateFromStorage();
  }, []);

  // useBiometricLock internally waits for isLoading === false before registering
  // the AppState listener — safe to call unconditionally here.
  useBiometricLock({
    onLockout: () => router.replace('/(auth)/login'),
  });

  // Register Android channels and FCM token after auth is confirmed.
  useEffect(() => {
    if (!isAuthenticated || isLoading) return;

    async function setupPushNotifications(): Promise<void> {
      // Request push notification permissions.
      const { status } = await Notifications.requestPermissionsAsync();
      if (status !== 'granted') return;

      // Android channels MUST be created before first notification arrives.
      if (Platform.OS === 'android') {
        await Notifications.setNotificationChannelAsync('order-updates', {
          name: 'Order Updates',
          importance: Notifications.AndroidImportance.HIGH,
          sound: 'default',
          vibrationPattern: [0, 250, 250, 250],
        });
        await Notifications.setNotificationChannelAsync('promotions', {
          name: 'Promotions',
          importance: Notifications.AndroidImportance.DEFAULT,
        });
      }

      // Register FCM token with the API.
      const token = await getRawFCMToken();
      if (token) {
        await registerDeviceToken(api, token);
      }

      // Handle FCM token rotation.
      const tokenSub = Notifications.addPushTokenListener((event) => {
        registerDeviceToken(api, event.data).catch((err: unknown) => {
          console.warn('[push] Token rotation registration failed', err);
        });
      });

      // Increment badge on foreground notification receipt.
      const notifSub = Notifications.addNotificationReceivedListener(async () => {
        const current = await Notifications.getBadgeCountAsync();
        await Notifications.setBadgeCountAsync(current + 1);
      });

      // Deep link routing on notification tap.
      const responseSub = Notifications.addNotificationResponseReceivedListener(
        (response) => {
          const data = response.notification.request.content.data as Record<
            string,
            string
          >;
          if (data?.type === 'order_update' && data?.orderId) {
            router.push(`/order/${data.orderId}`);
          }
        }
      );

      pushCleanupRef.current = () => {
        tokenSub.remove();
        notifSub.remove();
        responseSub.remove();
      };
    }

    setupPushNotifications().catch((err: unknown) => {
      console.warn('[push] Setup failed', err);
    });

    return () => {
      if (pushCleanupRef.current) {
        pushCleanupRef.current();
        pushCleanupRef.current = null;
      }
    };
  }, [isAuthenticated, isLoading]);

  useEffect(() => {
    if (!isLoading) {
      if (!isAuthenticated) {
        router.replace('/(auth)/login');
      } else if (!onboardingComplete) {
        // Gate: authenticated but hasn't completed onboarding wizard
        router.replace('/(onboarding)/user-info');
      } else {
        router.replace('/(tabs)');
      }
    }
  }, [isAuthenticated, isLoading, onboardingComplete]);

  return (
    <QueryClientProvider client={queryClient}>
      <View style={{ flex: 1 }}>
        <OfflineBanner />
        <Stack screenOptions={{ headerShown: false }} />
      </View>
    </QueryClientProvider>
  );
}
