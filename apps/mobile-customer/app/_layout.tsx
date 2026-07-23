import '../global.css';

import { useEffect, useRef, useState } from 'react';
import { ActivityIndicator, AppState, type AppStateStatus, Platform, View } from 'react-native';
import { GestureHandlerRootView } from 'react-native-gesture-handler';
import { BottomSheetModalProvider } from '@gorhom/bottom-sheet';
import { Stack, router } from 'expo-router';
import { OfflineBanner } from '@homechef/mobile-shared';
import * as Notifications from 'expo-notifications';
import {
  QueryClient,
  QueryClientProvider,
  focusManager,
} from '@tanstack/react-query';
import { AuthProvider } from '@homechef/mobile-shared/auth';
import { useAuthStore } from '../store/auth-store';
import { useBiometricLock } from '@homechef/mobile-shared/hooks';
import { getRawFCMToken, registerDeviceToken } from '@homechef/mobile-shared/hooks';
import { api } from '../lib/api';
import { useFonts } from 'expo-font';
import { Geist_600SemiBold } from '@expo-google-fonts/geist/600SemiBold';
import { Geist_700Bold } from '@expo-google-fonts/geist/700Bold';
import { Inter_400Regular } from '@expo-google-fonts/inter/400Regular';
import { Inter_500Medium } from '@expo-google-fonts/inter/500Medium';
import { Inter_600SemiBold } from '@expo-google-fonts/inter/600SemiBold';
import { customerColors } from '@homechef/mobile-shared/theme';

const queryClient = new QueryClient({
  defaultOptions: {
    queries: { staleTime: 1000 * 60 * 5, retry: 2 },
  },
});

// Set global notification handler at module level — before any notification arrives.
Notifications.setNotificationHandler({
  handleNotification: async () => ({
    // SDK 53 split shouldShowAlert into banner + list; keep the legacy field
    // for older runtimes and add the new required ones.
    shouldShowAlert: true,
    shouldShowBanner: true,
    shouldShowList: true,
    shouldPlaySound: true,
    shouldSetBadge: true,
  }),
});

export default function RootLayout() {
  // Load Geist + Inter at boot. Until they resolve, every Text falls back to the
  // OS system font (San Francisco / Roboto) and the brand's geometric-sans
  // identity is absent — so we hold an opaque splash over the first frames to
  // avoid a visible System→Geist/Inter snap. Weight is encoded in the family key
  // ('Inter-Medium') because RN doesn't reliably honor fontWeight on custom fonts.
  const [fontsLoaded] = useFonts({
    Geist: Geist_600SemiBold, // default Geist = 600 (display)
    'Geist-Bold': Geist_700Bold,
    Inter: Inter_400Regular, // default Inter = 400 (body)
    'Inter-Medium': Inter_500Medium,
    'Inter-SemiBold': Inter_600SemiBold,
  });

  const { isAuthenticated, isLoading, onboardingComplete, hydrateFromStorage } =
    useAuthStore();
  const setOnboardingComplete = useAuthStore((s) => s.setOnboardingComplete);
  // Whether we've reconciled onboarding state with the SERVER for this session.
  // The local `onboardingComplete` flag is device-only and is cleared on logout,
  // so on re-login we must ask the server before deciding — otherwise a returning
  // user with a completed profile gets wrongly sent back through setup (#profile).
  const [onboardingChecked, setOnboardingChecked] = useState(false);

  // Cleanup ref for push subscription teardown.
  const pushCleanupRef = useRef<(() => void) | null>(null);

  useEffect(() => {
    hydrateFromStorage();
  }, []);

  // Wire React Query's focusManager to React Native AppState. By default RQ
  // listens for the web `visibilitychange` event, which never fires in RN — so
  // refetchOnWindowFocus is dead and queries stay on stale data when the app
  // returns from the background (e.g. after the Razorpay payment sheet, or when
  // the customer switches away while the chef advances their order). Marking the
  // app focused on foreground triggers a refetch of stale active queries, so
  // order status refreshes live instead of only after a full app restart.
  useEffect(() => {
    const sub = AppState.addEventListener('change', (status: AppStateStatus) => {
      focusManager.setFocused(status === 'active');
    });
    return () => sub.remove();
  }, []);

  // Wipe the React Query cache whenever the signed-in identity changes (logout,
  // or login as a different user). Without this, the previous user's cached
  // profile/orders/addresses leak into the next session until staleTime expires
  // — e.g. log out as A, sign up as B, and B briefly sees A's account.
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
          // No sound key → default notification sound. Passing the string
          // 'default' makes expo-notifications look for a bundled custom sound
          // file named 'default' and log a dev warning when it's absent.
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
          // Accept both key casings: the backend sends snake_case `order_id` on
          // newer notifications (order_voided, #694) and camelCase `orderId` on
          // others. Reading either keeps a tap from silently falling through.
          const orderId = data?.order_id ?? data?.orderId;
          const type = data?.type ?? '';
          // Any order-scoped notification opens the order. Previously only the
          // (never-actually-sent) `order_update` type routed, so real order pushes
          // — status changes, and the void apology — landed nowhere. Match on the
          // presence of an order id + an order-ish type instead.
          if (orderId && (type === '' || type.startsWith('order'))) {
            router.push(`/order/${orderId}`);
          } else if (data?.chefId && type === 'weekly_menu_published') {
            // A favorited chef dropped a new weekly menu (#239) → open the chef.
            router.push(`/chef/${data.chefId}`);
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

  // Reconcile onboarding state with the server whenever the user is authenticated
  // but the device flag says not-onboarded. A returning user (who onboarded on a
  // previous session, then logged out — which clears the local flag) has
  // OnboardingCompleted=true server-side, so we restore the flag and skip setup
  // instead of forcing them through the wizard again and losing their details.
  useEffect(() => {
    let cancelled = false;
    if (isLoading || !isAuthenticated || onboardingComplete) {
      setOnboardingChecked(true);
      return;
    }
    setOnboardingChecked(false);
    api
      .get<{ onboardingCompleted?: boolean }>('/v1/customer/onboarding/status')
      .then((r) => {
        if (cancelled) return;
        if (r.data?.onboardingCompleted) {
          // Server says they're set up — restore the local flag; the gate below
          // then routes them straight to the app.
          setOnboardingComplete(true);
        }
      })
      .catch(() => {
        // On a transient failure, fall through to the wizard rather than block
        // the user — they can still finish (the server upserts idempotently).
      })
      .finally(() => {
        if (!cancelled) setOnboardingChecked(true);
      });
    return () => {
      cancelled = true;
    };
  }, [isAuthenticated, isLoading, onboardingComplete, setOnboardingComplete]);

  useEffect(() => {
    if (isLoading) return;
    if (!isAuthenticated) {
      router.replace('/(auth)/login');
    } else if (!onboardingComplete) {
      // Wait for the server reconciliation before deciding, so we never flash the
      // setup wizard at a returning user whose profile is already complete.
      if (!onboardingChecked) return;
      router.replace('/(onboarding)/user-info');
    } else {
      router.replace('/(tabs)');
    }
  }, [isAuthenticated, isLoading, onboardingComplete, onboardingChecked]);

  return (
    // GestureHandlerRootView must wrap the whole app so @gorhom/bottom-sheet's
    // GestureDetector (used by CartSheet on the chef detail screen) works.
    <GestureHandlerRootView style={{ flex: 1 }}>
      {/* BottomSheetModalProvider is required by the shared <Sheet> (a
          BottomSheetModal) — e.g. the "Show my plan" sheet on the chef page.
          Without it, mounting a BottomSheetModal crashes the app. */}
      <BottomSheetModalProvider>
        <AuthProvider
          bffUrl={process.env.EXPO_PUBLIC_BFF_URL ?? ''}
          tenantId={process.env.EXPO_PUBLIC_GIP_TENANT_ID ?? ''}
        >
          <QueryClientProvider client={queryClient}>
            <View style={{ flex: 1 }}>
              <OfflineBanner />
              <Stack screenOptions={{ headerShown: false }} />
              {!fontsLoaded && (
                <View
                  pointerEvents="auto"
                  style={{
                    position: 'absolute',
                    top: 0,
                    left: 0,
                    right: 0,
                    bottom: 0,
                    backgroundColor: customerColors.canvas,
                    alignItems: 'center',
                    justifyContent: 'center',
                  }}
                >
                  <ActivityIndicator color={customerColors.charcoal.DEFAULT} size="large" />
                </View>
              )}
            </View>
          </QueryClientProvider>
        </AuthProvider>
      </BottomSheetModalProvider>
    </GestureHandlerRootView>
  );
}
