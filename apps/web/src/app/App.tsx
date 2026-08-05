import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { ReactQueryDevtools } from '@tanstack/react-query-devtools';
import { BrowserRouter } from 'react-router';
import { MotionConfig } from 'framer-motion';
import { AppRoutes } from './routes';
import { AuthProvider } from './providers/AuthProvider';
import { OttoChat } from '@/features/support/OttoChat';
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
                <OttoChat />
                {/* Top-right, tucked under the sticky header (~64px): toasts
                    can never collide with the Otto launcher or its open panel
                    in the bottom-right corner, and stay clear of the header
                    icons. The launcher itself is pinned to z-40 in globals.css
                    so overlays and payment modals always win the corner. */}
                <ThemedToaster
                  position="top-right"
                  offset="76px"
                  mobileOffset={{ top: '72px' }}
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
