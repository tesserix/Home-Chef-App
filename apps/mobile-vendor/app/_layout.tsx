import '../global.css';

import { useEffect, useRef } from 'react';
import { Platform, View } from 'react-native';
import { Stack, router } from 'expo-router';
import { OfflineBanner } from '@homechef/mobile-shared';
import * as Notifications from 'expo-notifications';
import * as SecureStore from 'expo-secure-store';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { useQuery } from '@tanstack/react-query';
import { GestureHandlerRootView } from 'react-native-gesture-handler';
import { BottomSheetModalProvider } from '@gorhom/bottom-sheet';
import { AuthProvider } from '@homechef/mobile-shared/auth';
import { useAuthStore } from '../store/auth-store';
import { useBiometricLock } from '@homechef/mobile-shared/hooks';
import { getRawFCMToken, registerDeviceToken } from '@homechef/mobile-shared/hooks';
import { api } from '../lib/api';

interface OnboardingStatusResponse {
  status: 'not_started' | 'in_progress' | 'pending_review' | 'submitted' | 'verified' | 'rejected';
  completed: boolean;
  step: number;
  chefId: string | null;
  profile: object | null;
}

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

// API base URL for background fetch (no React context available in background).
const API_BASE_URL = process.env.EXPO_PUBLIC_API_URL ?? '';

function AppNavigator() {
  const { isAuthenticated, isLoading, hydrateFromStorage } = useAuthStore();

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

  // iOS notification categories MUST be registered at app launch, BEFORE auth,
  // so Accept/Reject buttons appear even on the very first notification.
  useEffect(() => {
    Notifications.setNotificationCategoryAsync('new_order', [
      {
        identifier: 'ACCEPT_ORDER',
        buttonTitle: 'Accept',
        options: {
          opensAppToForeground: false,
        },
      },
      {
        identifier: 'REJECT_ORDER',
        buttonTitle: 'Reject',
        options: {
          isDestructive: true,
          opensAppToForeground: false,
        },
      },
    ]).catch((err: unknown) => {
      console.warn('[push] iOS category registration failed', err);
    });
  }, []);

  const { data: onboardingStatus, isLoading: onboardingLoading } = useQuery({
    queryKey: ['chef', 'onboarding', 'status'],
    queryFn: () => api.get<OnboardingStatusResponse>('/chef/onboarding/status'),
    enabled: isAuthenticated && !isLoading,
    staleTime: 60_000,
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
        await Notifications.setNotificationChannelAsync('new-orders', {
          name: 'New Orders',
          importance: Notifications.AndroidImportance.MAX,
          sound: 'default',
          vibrationPattern: [0, 250, 250, 250],
        });
        await Notifications.setNotificationChannelAsync('order-updates', {
          name: 'Order Updates',
          importance: Notifications.AndroidImportance.HIGH,
          sound: 'default',
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

      // Handle notification tap and lock-screen actions.
      // CRITICAL: Accept/Reject MUST use SecureStore + fetch — NOT the React-context-bound
      // axios instance. The listener fires when the app is backgrounded or killed;
      // React context is unavailable and axios auth headers are not set.
      const responseSub = Notifications.addNotificationResponseReceivedListener(
        async (response) => {
          const data = response.notification.request.content.data as Record<
            string,
            string
          >;
          const actionId = response.actionIdentifier;

          if (
            (actionId === 'ACCEPT_ORDER' || actionId === 'REJECT_ORDER') &&
            data?.orderId
          ) {
            // Lock-screen action — fetch token from SecureStore (no React context).
            const authToken = await SecureStore.getItemAsync('access_token');
            if (!authToken) return;
            const orderStatus =
              actionId === 'ACCEPT_ORDER' ? 'accepted' : 'rejected';
            await fetch(
              `${API_BASE_URL}/chef/orders/${data.orderId}/status`,
              {
                method: 'PUT',
                headers: {
                  'X-Auth-Token': authToken,
                  'Content-Type': 'application/json',
                },
                body: JSON.stringify({ status: orderStatus }),
              }
            );
            return; // do not navigate
          }

          // Default tap: open orders tab.
          if (data?.type === 'new_order') {
            router.push('/orders');
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
    if (isLoading || onboardingLoading) return;
    if (!isAuthenticated) {
      router.replace('/(auth)/login');
    } else if (!onboardingStatus) {
      return; // still fetching
    } else if (onboardingStatus.data.status === 'verified') {
      router.replace('/(tabs)');
    } else if (onboardingStatus.data.status === 'not_started') {
      router.replace('/(onboarding)/personal-info');
    } else {
      // pending_review, submitted, rejected, in_progress
      router.replace('/(onboarding)/pending');
    }
  }, [isAuthenticated, isLoading, onboardingStatus, onboardingLoading]);

  return (
    <View style={{ flex: 1 }}>
      <OfflineBanner />
      <Stack screenOptions={{ headerShown: false }} />
    </View>
  );
}

export default function RootLayout() {
  return (
    <AuthProvider
      bffUrl={process.env.EXPO_PUBLIC_BFF_URL ?? ''}
      tenantId={process.env.EXPO_PUBLIC_GIP_TENANT_ID ?? ''}
    >
      <GestureHandlerRootView style={{ flex: 1 }}>
        <QueryClientProvider client={queryClient}>
          <BottomSheetModalProvider>
            <AppNavigator />
          </BottomSheetModalProvider>
        </QueryClientProvider>
      </GestureHandlerRootView>
    </AuthProvider>
  );
}
