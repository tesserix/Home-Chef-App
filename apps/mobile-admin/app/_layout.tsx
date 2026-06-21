import '../global.css';

import { useEffect, useRef, useState } from 'react';
import { ActivityIndicator, Text, View } from 'react-native';
import { Stack, router, usePathname } from 'expo-router';
import { OfflineBanner } from '@homechef/mobile-shared';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { GestureHandlerRootView } from 'react-native-gesture-handler';
import { BottomSheetModalProvider } from '@gorhom/bottom-sheet';
import { AuthProvider } from '@homechef/mobile-shared/auth';
import { useBiometricLock } from '@homechef/mobile-shared/hooks';
import { ToastProvider, UndoSnackbarProvider } from '@homechef/mobile-shared/ui';
import { useFonts } from 'expo-font';
import { Geist_600SemiBold } from '@expo-google-fonts/geist/600SemiBold';
import { Geist_700Bold } from '@expo-google-fonts/geist/700Bold';
import { Inter_400Regular } from '@expo-google-fonts/inter/400Regular';
import { Inter_500Medium } from '@expo-google-fonts/inter/500Medium';
import { Inter_600SemiBold } from '@expo-google-fonts/inter/600SemiBold';
import { useAuthStore } from '../store/auth-store';
import { ErrorBoundary } from '../components/ErrorBoundary';

const queryClient = new QueryClient({
  defaultOptions: {
    queries: { staleTime: 1000 * 60 * 2, retry: 1 },
  },
});

function AppNavigator() {
  // Load Geist (display) + Inter (UI) at boot. Hold the splash until ready so
  // we never flash System font and snap.
  const [fontsLoaded] = useFonts({
    Geist: Geist_600SemiBold,
    'Geist-Bold': Geist_700Bold,
    Inter: Inter_400Regular,
    'Inter-Medium': Inter_500Medium,
    'Inter-SemiBold': Inter_600SemiBold,
  });

  const { isAuthenticated, isLoading, hydrateFromStorage } = useAuthStore();

  useEffect(() => {
    hydrateFromStorage();
  }, []);

  // Wipe the React Query cache whenever the signed-in identity changes so the
  // previous admin's data never leaks into the next session.
  const userId = useAuthStore((s) => s.user?.id ?? null);
  const prevUserIdRef = useRef<string | null>(userId);
  useEffect(() => {
    if (prevUserIdRef.current !== userId) {
      queryClient.clear();
      prevUserIdRef.current = userId;
    }
  }, [userId]);

  // Optional biometric re-lock when returning to the foreground. Safe to call
  // unconditionally — it waits for isLoading === false internally.
  useBiometricLock({
    onLockout: () => router.replace('/(auth)/login'),
  });

  // Coarse auth-state key. Drives when the splash re-appears between an auth
  // change and the URL actually swapping.
  const authKey: string = isLoading
    ? 'resolving'
    : isAuthenticated
      ? 'auth'
      : 'unauth';

  const expectedPath: string | null =
    authKey === 'resolving'
      ? null
      : authKey === 'auth'
        ? '/(tabs)'
        : '/(auth)/login';

  const pathname = usePathname();
  const routedFor = useRef<string | null>(null);

  function normalize(p: string): string {
    const stripped = p.replace(/\/\([^)]+\)/g, '');
    return stripped === '' ? '/' : stripped;
  }

  // Single routing effect — keeps an unauthenticated admin out of the tabs and
  // a signed-in admin out of the login screen.
  useEffect(() => {
    if (!expectedPath) return;
    const target = normalize(expectedPath);
    const here = normalize(pathname || '');
    const matched =
      target === '/' ? !here.startsWith('/login') : here === target;
    if (matched) {
      routedFor.current = authKey;
    } else {
      router.replace(expectedPath as never);
    }
  }, [pathname, expectedPath, authKey]);

  // Safety net: force-lift the splash after 5s if routing never settles.
  const [forceLift, setForceLift] = useState(false);
  useEffect(() => {
    const t = setTimeout(() => setForceLift(true), 5000);
    return () => clearTimeout(t);
  }, []);

  const isRouting =
    !forceLift &&
    (!fontsLoaded || authKey === 'resolving' || routedFor.current !== authKey);

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
          {fontsLoaded && (
            <Text
              style={{
                marginTop: 16,
                fontFamily: 'Inter',
                fontSize: 13,
                color: '#888888',
                letterSpacing: 0.2,
              }}
            >
              Loading…
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

export default RootLayout;
