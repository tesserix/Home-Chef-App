import { useEffect } from 'react';
import { Stack, router } from 'expo-router';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { useQuery } from '@tanstack/react-query';
import { GestureHandlerRootView } from 'react-native-gesture-handler';
import { BottomSheetModalProvider } from '@gorhom/bottom-sheet';
import { useAuthStore } from '../store/auth-store';
import { useBiometricLock } from '@homechef/mobile-shared/hooks';
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

function AppNavigator() {
  const { isAuthenticated, isLoading, hydrateFromStorage } = useAuthStore();

  useEffect(() => {
    hydrateFromStorage();
  }, []);

  // useBiometricLock internally waits for isLoading === false before registering
  // the AppState listener — safe to call unconditionally here.
  useBiometricLock({
    onLockout: () => router.replace('/(auth)/login'),
  });

  const { data: onboardingStatus, isLoading: onboardingLoading } = useQuery({
    queryKey: ['chef', 'onboarding', 'status'],
    queryFn: () => api.get<OnboardingStatusResponse>('/chef/onboarding/status'),
    enabled: isAuthenticated && !isLoading,
    staleTime: 60_000,
  });

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

  return <Stack screenOptions={{ headerShown: false }} />;
}

export default function RootLayout() {
  return (
    <GestureHandlerRootView style={{ flex: 1 }}>
      <QueryClientProvider client={queryClient}>
        <BottomSheetModalProvider>
          <AppNavigator />
        </BottomSheetModalProvider>
      </QueryClientProvider>
    </GestureHandlerRootView>
  );
}
