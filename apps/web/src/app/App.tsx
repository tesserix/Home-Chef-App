import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { ReactQueryDevtools } from '@tanstack/react-query-devtools';
import { BrowserRouter } from 'react-router-dom';
import { MotionConfig } from 'framer-motion';
import { AppRoutes } from './routes';
import { AuthProvider } from './providers/AuthProvider';
import { SkipLink } from '@/shared/components/a11y/SkipLink';
import { ThemeProvider, ThemedToaster } from '@/shared/theme';
import { ErrorBoundary } from '@/shared/components/error/ErrorBoundary';

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 1000 * 60 * 5, // 5 minutes
      retry: 1,
      refetchOnWindowFocus: false,
    },
  },
});

export function App() {
  return (
    <ErrorBoundary>
      <QueryClientProvider client={queryClient}>
        <ThemeProvider>
          <MotionConfig reducedMotion="user" transition={{ duration: 0.25, ease: [0.22, 1, 0.36, 1] }}>
            <BrowserRouter>
              <AuthProvider>
                <SkipLink />
                <AppRoutes />
                <ThemedToaster
                  position="bottom-right"
                  expand={false}
                  closeButton
                  toastOptions={{
                    duration: 4000,
                  }}
                />
              </AuthProvider>
            </BrowserRouter>
          </MotionConfig>
        </ThemeProvider>
        <ReactQueryDevtools initialIsOpen={false} />
      </QueryClientProvider>
    </ErrorBoundary>
  );
}
