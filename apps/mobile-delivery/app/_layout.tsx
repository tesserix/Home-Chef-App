import { useEffect } from 'react';
import { Stack, router } from 'expo-router';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { useAuthStore } from '../store/auth-store';

const queryClient = new QueryClient({
  defaultOptions: {
    queries: { staleTime: 1000 * 60 * 5, retry: 2 },
  },
});

export default function RootLayout() {
  const { isAuthenticated, isLoading, hydrateFromStorage } = useAuthStore();

  useEffect(() => {
    hydrateFromStorage();
  }, []);

  useEffect(() => {
    if (!isLoading) {
      if (isAuthenticated) {
        router.replace('/(tabs)');
      } else {
        router.replace('/(auth)/login');
      }
    }
  }, [isAuthenticated, isLoading]);

  return (
    <QueryClientProvider client={queryClient}>
      <Stack screenOptions={{ headerShown: false }} />
    </QueryClientProvider>
  );
}
