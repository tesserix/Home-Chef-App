import '../global.css';

import '../lib/background-location'; // registers background task at module load
import { useEffect, useRef } from 'react';
import { AppState, type AppStateStatus, Platform, View } from 'react-native';
import { OfflineBanner } from '@homechef/mobile-shared';
import { Stack, router } from 'expo-router';
import * as Notifications from 'expo-notifications';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { AxiosResponse } from 'axios';
import { GestureHandlerRootView } from 'react-native-gesture-handler';
import { BottomSheetModalProvider } from '@gorhom/bottom-sheet';
import { AuthProvider, ensureFreshInstallReset } from '@homechef/mobile-shared/auth';
import { MFAGateProvider } from '@homechef/mobile-shared/mfa';
import { api as apiClient } from '../lib/api';
import { DialogProvider } from '@homechef/mobile-shared/ui';
import { theme } from '@homechef/mobile-shared/theme';
import { useAuthStore } from '../store/auth-store';
import { useBiometricLock } from '@homechef/mobile-shared/hooks';
import { getRawFCMToken, registerDeviceTokenSafe } from '@homechef/mobile-shared/hooks';
import { api } from '../lib/api';
import { useQuery } from '@tanstack/react-query';

interface DriverOnboardingStatus {
  step: number;
  status: 'not_started' | 'in_progress' | 'pending_review' | 'approved' | 'rejected';
  onboardingComplete: boolean;
  verificationStatus: string;
  profile: unknown | null;
}

const queryClient = new QueryClient({
  defaultOptions: {
    queries: { staleTime: 1000 * 60 * 5, retry: 2 },
  },
});

// Map the server-reported onboarding step to the wizard screen so an
// interrupted application resumes where the driver left off. Return type is
// left to inference (a union of route literals) so it stays assignable to the
// typed-routes Href the router expects.
function wizardPathForStep(step: number) {
  switch (step) {
    case 2:
      return '/(onboarding)/vehicle' as const;
    case 3:
      return '/(onboarding)/documents' as const;
    case 4:
      return '/(onboarding)/payout' as const;
    case 5:
      return '/(onboarding)/subscription' as const;
    case 6:
      return '/(onboarding)/review' as const;
    default:
      return '/(onboarding)/personal' as const;
  }
}

// Set global notification handler at module level — before any notification arrives.
Notifications.setNotificationHandler({
  handleNotification: async () => ({
    shouldShowAlert: true,
    shouldShowBanner: true,
    shouldShowList: true,
    shouldPlaySound: true,
    shouldSetBadge: true,
  }),
});

function AppNavigator() {
  const { isAuthenticated, isLoading, hydrateFromStorage } = useAuthStore();

  // Cleanup ref for push subscription teardown.
  const pushCleanupRef = useRef<(() => void) | null>(null);
  // Whether a token is already registered for this session — guards the
  // AppState reattempt below so it doesn't re-register on every foreground.
  const tokenRegisteredRef = useRef(false);

  useEffect(() => {
    // Fresh-install guard MUST finish before hydrateFromStorage reads the
    // Keychain: a reinstall leaves stale access/session tokens behind that
    // hydrate would otherwise treat as "signed in", skipping login. On a normal
    // app restart the guard is a no-op, preserving stay-logged-in (#428/#436).
    void (async () => {
      await ensureFreshInstallReset();
      await hydrateFromStorage();
    })();
  }, []);

  // Wipe the React Query cache whenever the signed-in identity changes (logout,
  // or login as a different user). Without this, the previous user's cached
  // data leaks into the next session until staleTime expires.
  const userId = useAuthStore((s) => s.user?.id ?? null);
  const prevUserIdRef = useRef<string | null>(userId);
  useEffect(() => {
    if (prevUserIdRef.current !== userId) {
      queryClient.clear();
      prevUserIdRef.current = userId;
    }
  }, [userId]);

  // useBiometricLock internally waits for isLoading === false before registering
  // the AppState listener — safe to call unconditionally here.
  useBiometricLock({
    onLockout: () => router.replace('/(auth)/login'),
  });

  const { data: onboardingStatus, isLoading: onboardingLoading } =
    useQuery<AxiosResponse<DriverOnboardingStatus>>({
      queryKey: ['driver', 'onboarding', 'status'],
      queryFn: () => api.get<DriverOnboardingStatus>('/driver/onboarding/status'),
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
        await Notifications.setNotificationChannelAsync('new-deliveries', {
          name: 'New Deliveries',
          importance: Notifications.AndroidImportance.MAX,
          sound: 'default',
          vibrationPattern: [0, 250, 250, 250],
        });
        await Notifications.setNotificationChannelAsync('delivery-updates', {
          name: 'Delivery Updates',
          importance: Notifications.AndroidImportance.HIGH,
          sound: 'default',
        });
      }

      // Register FCM token with the API.
      const token = await getRawFCMToken();
      if (token) {
        tokenRegisteredRef.current = await registerDeviceTokenSafe(api, token);
      }

      // Handle FCM token rotation.
      const tokenSub = Notifications.addPushTokenListener((event) => {
        void registerDeviceTokenSafe(api, event.data).then((ok) => {
          tokenRegisteredRef.current = ok;
        });
      });

      // Increment badge on foreground notification receipt.
      const notifSub = Notifications.addNotificationReceivedListener(async () => {
        const current = await Notifications.getBadgeCountAsync();
        await Notifications.setBadgeCountAsync(current + 1);
      });

      // Deep link routing on notification tap — navigate to available deliveries tab.
      const responseSub = Notifications.addNotificationResponseReceivedListener(
        (response) => {
          const data = response.notification.request.content.data as Record<
            string,
            string
          >;
          if (data?.type === 'new_delivery') {
            router.push('/available');
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
      tokenRegisteredRef.current = false;
    };
  }, [isAuthenticated, isLoading]);

  // Push reattempt on foreground. Unlike customer/vendor, this app has no
  // existing AppState/focusManager effect to extend — this listener exists
  // SOLELY for the push reattempt below; it does not wire React Query's
  // focus manager (that would be a separate, larger change not asked for
  // here). A driver who denied the permission prompt and later granted it
  // in Settings gets registered on the next foreground, without a second
  // prompt (getRawFCMToken only prompts when the status isn't already
  // decided) and without re-registering once already registered this
  // session.
  useEffect(() => {
    const sub = AppState.addEventListener('change', (status: AppStateStatus) => {
      if (
        status === 'active' &&
        !tokenRegisteredRef.current &&
        useAuthStore.getState().isAuthenticated
      ) {
        void getRawFCMToken().then((token) => {
          if (!token) return;
          void registerDeviceTokenSafe(api, token).then((ok) => {
            tokenRegisteredRef.current = ok;
          });
        });
      }
    });
    return () => sub.remove();
  }, []);

  useEffect(() => {
    if (isLoading || onboardingLoading) return;
    if (!isAuthenticated) {
      router.replace('/(auth)/login');
    } else if (!onboardingStatus) {
      return;
    } else if (onboardingStatus.data.onboardingComplete) {
      router.replace('/(tabs)');
    } else if (onboardingStatus.data.status === 'not_started') {
      router.replace('/(onboarding)/personal');
    } else if (onboardingStatus.data.status === 'in_progress') {
      // Resume the half-finished application at the step the driver reached,
      // instead of bouncing them to the "under review" screen — that used to
      // strand them mid-wizard with no way back in (and, with the draft now
      // persisted, would have hidden their saved progress).
      router.replace(wizardPathForStep(onboardingStatus.data.step));
    } else {
      // pending_review / rejected — application already submitted; show status.
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
      pool={process.env.EXPO_PUBLIC_AUTH_POOL ?? 'business'}
        tenantId={process.env.EXPO_PUBLIC_GIP_TENANT_ID ?? ''}
    >
      <GestureHandlerRootView style={{ flex: 1 }}>
        <QueryClientProvider client={queryClient}>
          <BottomSheetModalProvider>
            <DialogProvider accentColor={theme.colors.brand[500]}>
              <MFAGateProvider
                api={apiClient}
                accentColor={theme.colors.brand[500]}
                onSignOut={() => { void useAuthStore.getState().logout(); }}
              >
                <AppNavigator />
              </MFAGateProvider>
            </DialogProvider>
          </BottomSheetModalProvider>
        </QueryClientProvider>
      </GestureHandlerRootView>
    </AuthProvider>
  );
}
