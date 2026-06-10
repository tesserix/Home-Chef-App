import '../global.css';
import '../lib/i18n'; // side-effect: initialise i18next before first render

import { useEffect, useRef, useState } from 'react';
import { ActivityIndicator, Platform, Text, View } from 'react-native';
import { Stack, router, usePathname } from 'expo-router';
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
import { hydratePersistedLocale } from '../lib/i18n';
import { ErrorBoundary } from '../components/ErrorBoundary';
import { ToastProvider, UndoSnackbarProvider, useToast } from '@homechef/mobile-shared/ui';
import { useFonts } from 'expo-font';
import { Geist_600SemiBold } from '@expo-google-fonts/geist/600SemiBold';
import { Geist_700Bold } from '@expo-google-fonts/geist/700Bold';
import { Inter_400Regular } from '@expo-google-fonts/inter/400Regular';
import { Inter_500Medium } from '@expo-google-fonts/inter/500Medium';
import { Inter_600SemiBold } from '@expo-google-fonts/inter/600SemiBold';
import { useMinVersion } from '../hooks/useMinVersion';
import { initSentry, wrapWithSentry } from '../lib/sentry';

// Init Sentry as early as possible — at module scope, before any
// React component mounts — so a crash during initial render lands in
// Sentry instead of disappearing into Xcode-only crash logs.
// No-ops cleanly when EXPO_PUBLIC_SENTRY_DSN is unset.
initSentry();

interface OnboardingStatusResponse {
  status:
    | 'not_started'
    | 'in_progress'
    | 'pending_review'
    | 'submitted'
    | 'verified'
    | 'rejected'
    | 'info_requested';
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
// SDK 55 split the legacy `shouldShowAlert` into `shouldShowBanner` (heads-up)
// + `shouldShowList` (Notification Center); keep both true to match prior
// behavior. `shouldShowAlert` is still required by the type for back-compat.
Notifications.setNotificationHandler({
  handleNotification: async () => ({
    shouldShowAlert: true,
    shouldShowBanner: true,
    shouldShowList: true,
    shouldPlaySound: true,
    shouldSetBadge: true,
  }),
});

// API base URL for background fetch (no React context available in background).
const API_BASE_URL = process.env.EXPO_PUBLIC_API_URL ?? '';

// resolvePushRoute maps a notification's data payload to an in-app route for
// tap-through, covering every push category:
//   - order pushes carry `orderId`        → /orders/<id>
//   - other pushes carry a `deeplink`     → homechef-vendor:///<path> → /<path>
//   - and a `type` fallback map for older payloads without a deeplink.
// Returns null when nothing actionable is present (stay put).
function resolvePushRoute(data: Record<string, string> | undefined): string | null {
  if (!data) return null;
  if (data.orderId) return `/orders/${data.orderId}`;

  if (data.deeplink) {
    const m = data.deeplink.match(/^homechef-vendor:\/\/(.*)$/);
    if (m) {
      const path = m[1] && m[1] !== '/' ? m[1] : '/(tabs)';
      return path.startsWith('/') ? path : `/${path}`;
    }
  }

  switch (data.type) {
    case 'fssai_expiring':
      return '/documents/renew';
    case 'weekly_statement':
      return '/earnings';
    case 'availability_resumed':
      return '/(tabs)';
    case 'new_order':
      return '/(tabs)/orders';
    default:
      return null;
  }
}

// Maps the API's `step` field (count of completed wizard sections — see
// apps/api/handlers/upload.go:382-395) to the next screen the chef should
// land on. The API skips step=1 because personal-info isn't persisted
// server-side, so the array index lines up: step 0 → personal-info,
// step 2 → operations (kitchen-details just completed), etc.
const WIZARD_STEP_PATHS = [
  '/(onboarding)/personal-info',
  '/(onboarding)/kitchen-details',
  '/(onboarding)/operations',
  '/(onboarding)/documents',
  '/(onboarding)/policies',
  '/(onboarding)/review',
] as const;

function wizardPathForStep(step: number): string {
  const i = Math.max(0, Math.min(step, WIZARD_STEP_PATHS.length - 1));
  return WIZARD_STEP_PATHS[i];
}

function AppNavigator() {
  // Load Geist + Inter at boot. Until these resolve every Text falls back
  // to the platform System font, so we delay rendering until ready.
  // Naming convention: weight is encoded in the family key so consumers
  // pick by family name (`Inter-Medium`) instead of by fontWeight, which
  // RN doesn't reliably honor for non-system fonts.
  const [fontsLoaded] = useFonts({
    Geist: Geist_600SemiBold, // default Geist = 600 (semibold) for display
    'Geist-Bold': Geist_700Bold,
    Inter: Inter_400Regular, // default Inter = 400 (regular) for body
    'Inter-Medium': Inter_500Medium,
    'Inter-SemiBold': Inter_600SemiBold,
  });

  const { isAuthenticated, isLoading, hydrateFromStorage } = useAuthStore();

  // Cleanup ref for push subscription teardown.
  const pushCleanupRef = useRef<(() => void) | null>(null);

  useEffect(() => {
    hydrateFromStorage();
    // Apply the saved language choice (overrides the device default).
    hydratePersistedLocale();
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

  const {
    data: onboardingStatus,
    isLoading: onboardingLoading,
    isError: onboardingError,
  } = useQuery({
    queryKey: ['chef', 'onboarding', 'status'],
    queryFn: () => api.get<OnboardingStatusResponse>('/chef/onboarding/status'),
    enabled: isAuthenticated && !isLoading,
    // Force a refetch on every mount — a stale "verified" payload was
    // what let a chef whose application was actually rolled back to
    // "submitted" navigate freely into tabs that then 401'd. Cached
    // data is still returned for the first paint (so the splash isn't
    // any longer), but the network truth wins by the second tick.
    refetchOnMount: 'always',
    staleTime: 0,
    // A 401 from this endpoint means the stored token is dead. Retrying
    // costs 5-30s of splash time and can't fix it; the response
    // interceptor already cleared the token and flipped isAuthenticated
    // to false, so we just want the next render to route to /login.
    retry: false,
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

          // Default tap: route to wherever the notification points. Order
          // pushes open the order; FSSAI/statement/availability pushes carry
          // a `deeplink` (homechef-vendor:///<path>); everything else falls
          // back to a type→route map, then the dashboard.
          const path = resolvePushRoute(data);
          if (path) {
            router.push(path as never);
          }
        }
      );

      // Cold-start: if the app was launched by tapping a push (killed state),
      // the live listener above may have missed it — replay the last response.
      const last = await Notifications.getLastNotificationResponseAsync();
      if (last) {
        const data = last.notification.request.content.data as Record<
          string,
          string
        >;
        const path = resolvePushRoute(data);
        if (path) {
          router.push(path as never);
        }
      }

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

  // Watch for status transitioning into 'verified'. When admin approves
  // a chef while their app is open, the next status refetch flips the
  // bit — surface a single celebratory toast so the chef understands
  // why they suddenly have access to the kitchen tools.
  const prevStatusRef = useRef<string | null>(null);
  const { show: showToast } = useToast();
  useEffect(() => {
    const currentStatus = onboardingStatus?.data?.status ?? null;
    if (
      prevStatusRef.current &&
      prevStatusRef.current !== 'verified' &&
      currentStatus === 'verified'
    ) {
      showToast({
        message: 'Your kitchen is approved. Welcome aboard.',
        tone: 'success',
      });
    }
    prevStatusRef.current = currentStatus;
  }, [onboardingStatus, showToast]);

  // A coarse identifier for the auth state. Flips on real transitions
  // (login, logout, onboarding moving from in_progress → verified, etc).
  // We use it to decide when the splash should re-appear: it covers the
  // gap between auth state changing and the URL actually swapping under us.
  const authKey: string = (() => {
    if (isLoading) return 'resolving';
    if (!isAuthenticated) return 'unauth';
    if (onboardingLoading) return 'resolving';
    // Query errored (most often: token expired → 401). Drop to login
    // instead of holding the splash forever.
    if (onboardingError || !onboardingStatus) return 'unauth';
    return `auth:${onboardingStatus.data.status}`;
  })();

  // Where we believe the user should land right now. null while still
  // resolving — the splash stays up in that case.
  //
  // Status → screen rules:
  // - verified                            → tabs
  // - pending_review / submitted          → /pending (real "under review")
  // - not_started / in_progress / rejected
  //   / info_requested                    → wizard, resuming at the
  //                                         next incomplete step.
  //
  // The previous rule treated anything that wasn't `verified` or
  // `not_started` as "pending", which sent users mid-wizard (BusinessName
  // saved, no approval row yet — backend status `in_progress`) to the
  // application-under-review screen even though they hadn't submitted.
  const expectedPath: string | null = (() => {
    if (authKey === 'resolving') return null;
    if (authKey === 'unauth') return '/(auth)/login';
    if (!onboardingStatus) return null;
    const { status, step } = onboardingStatus.data;
    if (status === 'verified') return '/(tabs)';
    if (status === 'pending_review' || status === 'submitted') {
      return '/(onboarding)/pending';
    }
    return wizardPathForStep(step);
  })();

  // Force-upgrade gate. Polls /mobile/min-version while foregrounded;
  // when the running app is below the configured minimum, every other
  // routing decision below is short-circuited and the wall is pinned.
  // Backend ALSO returns 426 on any authenticated request from a
  // too-old client (defense in depth wired in lib/api.ts); both paths
  // land on the same /upgrade-required screen.
  const { upgradeRequired, minVersion, storeUrl } = useMinVersion();

  const pathname = usePathname();

  // The auth state we've last successfully routed for. While this lags
  // behind authKey we know the URL hasn't caught up to the latest auth
  // decision yet, so the splash should stay up.
  const routedFor = useRef<string | null>(null);

  // Normalize an Expo Router target path so it can be compared against
  // `usePathname()` (which strips group `(...)` segments and emits `/`
  // for the group root). Without this, `/(tabs)` would normalize to ''
  // while the actual landed pathname is `/`, and the splash latch never
  // fires.
  function normalize(p: string): string {
    const stripped = p.replace(/\/\([^)]+\)/g, '');
    return stripped === '' ? '/' : stripped;
  }

  // Paths inside the (onboarding) group. The backend status stays
  // 'not_started' for the whole 6-step wizard (chef_profile only gets
  // persisted at /review submit), so the guard must treat ANY of these
  // as "user is where we want them" — otherwise pushing from step 1 to
  // step 2 immediately snaps back to step 1.
  const ONBOARDING_STEPS = [
    '/personal-info',
    '/kitchen-details',
    '/operations',
    '/documents',
    '/policies',
    '/review',
  ];

  // Single routing effect — fires on every pathname / expectedPath
  // change. If the user is where they should be we update routedFor
  // and lift the splash. If they've drifted (back gesture, deep link,
  // status change mid-session) we replace back to expectedPath. This is
  // the hard lock that keeps a non-verified chef out of the tabs.
  useEffect(() => {
    if (upgradeRequired) {
      if (normalize(pathname || '') !== '/upgrade-required') {
        const qs = new URLSearchParams();
        if (minVersion) qs.set('minVersion', minVersion);
        if (storeUrl) qs.set('storeUrl', storeUrl);
        const href = qs.toString()
          ? `/upgrade-required?${qs.toString()}`
          : '/upgrade-required';
        router.replace(href as never);
      }
      // Pretend routing settled so the splash lifts; otherwise the
      // wall renders under a permanent spinner.
      routedFor.current = authKey;
      return;
    }
    if (!expectedPath) return;
    const target = normalize(expectedPath);
    const here = normalize(pathname || '');
    const matched = target === '/'
      ? !here.startsWith('/login') &&
        !here.startsWith('/personal-info') &&
        !here.startsWith('/pending')
      : ONBOARDING_STEPS.includes(target) && ONBOARDING_STEPS.includes(here)
        ? true // any onboarding step counts as "on track" while wizard is mid-flight
        : here === target;
    if (matched) {
      routedFor.current = authKey;
    } else {
      router.replace(expectedPath as never);
    }
  }, [pathname, expectedPath, authKey, upgradeRequired, minVersion, storeUrl]);

  // Keep splash up until the OTF/TTF files are available — otherwise we
  // render a frame of System font then snap to Geist/Inter when fonts
  // arrive, which is jarringly visible on the login + dashboard.
  //
  // Splash safety net: if any of the auth/routing conditions never
  // settle (status webhook race, slow network, edge case in the routing
  // effect), force-lift after 5s instead of holding the user hostage on
  // a spinner. Worst case is a brief flash of the wrong screen, which
  // the routing effect will then correct.
  const [forceLift, setForceLift] = useState(false);
  useEffect(() => {
    const t = setTimeout(() => setForceLift(true), 5000);
    return () => clearTimeout(t);
  }, []);

  const isRouting =
    !forceLift &&
    (!fontsLoaded ||
      authKey === 'resolving' ||
      routedFor.current !== authKey);

  // Surface what we're waiting on so the splash reads as "in progress",
  // not "stuck". After a Google sign-in the auth → session → onboarding
  // chain can take 2-3s and a bare spinner makes that feel longer.
  const splashLabel = !fontsLoaded
    ? null
    : isAuthenticated && (onboardingLoading || routedFor.current !== authKey)
      ? 'Setting up your kitchen…'
      : 'Loading…';

  return (
    <View style={{ flex: 1 }}>
      <OfflineBanner />
      <Stack screenOptions={{ headerShown: false }} />
      {isRouting && (
        <View
          pointerEvents="auto"
          style={{
            position: 'absolute',
            top: 0,
            left: 0,
            right: 0,
            bottom: 0,
            backgroundColor: '#FFFFFF',
            alignItems: 'center',
            justifyContent: 'center',
          }}
        >
          <ActivityIndicator color="#0E0E0C" size="large" />
          {splashLabel && (
            <Text
              style={{
                marginTop: 16,
                fontFamily: 'Inter',
                fontSize: 13,
                color: '#888888',
                letterSpacing: 0.2,
              }}
            >
              {splashLabel}
            </Text>
          )}
        </View>
      )}
    </View>
  );
}

function RootLayout() {
  return (
    <ErrorBoundary>
      <AuthProvider
        bffUrl={process.env.EXPO_PUBLIC_BFF_URL ?? ''}
        tenantId={process.env.EXPO_PUBLIC_GIP_TENANT_ID ?? ''}
      >
        <GestureHandlerRootView style={{ flex: 1 }}>
          <QueryClientProvider client={queryClient}>
            <BottomSheetModalProvider>
              <ToastProvider>
                <UndoSnackbarProvider>
                  <AppNavigator />
                </UndoSnackbarProvider>
              </ToastProvider>
            </BottomSheetModalProvider>
          </QueryClientProvider>
        </GestureHandlerRootView>
      </AuthProvider>
    </ErrorBoundary>
  );
}

// Sentry.wrap installs the navigation tracer + error boundary. No-op
// when Sentry didn't initialize (DSN unset).
export default wrapWithSentry(RootLayout);
