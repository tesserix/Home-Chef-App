import { Routes, Route, Navigate } from 'react-router-dom';
import { Suspense, lazy } from 'react';
import { useAuth } from '../providers/AuthProvider';
import { LoadingScreen } from '@/shared/components/LoadingScreen';
import { VendorLayout } from '@/shared/components/layout/VendorLayout';

/**
 * Wraps a dynamic import with retry + full-page reload on failure.
 * After a new deployment, browsers may cache stale HTML that references
 * old JS chunk filenames that no longer exist on the server. This helper
 * catches the resulting TypeError and reloads the page once so the browser
 * fetches the fresh index.html with updated chunk references.
 */
function lazyWithRetry(factory: () => Promise<{ default: React.ComponentType }>) {
  return lazy(() =>
    factory().catch((err: unknown) => {
      // Only auto-reload once per session to avoid infinite reload loops
      const key = 'chunk-reload';
      const hasReloaded = sessionStorage.getItem(key);
      if (!hasReloaded) {
        sessionStorage.setItem(key, '1');
        window.location.reload();
        // Return a never-resolving promise so React doesn't render stale state
        return new Promise(() => {});
      }
      // Already reloaded once this session — surface the real error
      sessionStorage.removeItem(key);
      throw err;
    })
  );
}

// Auth pages
const LoginPage = lazyWithRetry(() => import('@/features/auth/pages/LoginPage'));
const RegisterPage = lazyWithRetry(() => import('@/features/auth/pages/RegisterPage'));

// Onboarding
const OnboardingPage = lazyWithRetry(() => import('@/features/onboarding/pages/OnboardingPage'));

// Feature pages
const DashboardPage = lazyWithRetry(() => import('@/features/dashboard/pages/DashboardPage'));
const MenuPage = lazyWithRetry(() => import('@/features/menu/pages/MenuPage'));
const MenuItemFormPage = lazyWithRetry(() => import('@/features/menu/pages/MenuItemFormPage'));
const MenuItemViewPage = lazyWithRetry(() => import('@/features/menu/pages/MenuItemViewPage'));
const LiveOrdersPage = lazyWithRetry(() => import('@/features/orders/pages/LiveOrdersPage'));
const OrderHistoryPage = lazyWithRetry(() => import('@/features/orders/pages/OrderHistoryPage'));
const EarningsPage = lazyWithRetry(() => import('@/features/earnings/pages/EarningsPage'));
const PayoutsPage = lazyWithRetry(() => import('@/features/earnings/pages/PayoutsPage'));
const ProfilePage = lazyWithRetry(() => import('@/features/profile/pages/ProfilePage'));
const KitchenSetupPage = lazyWithRetry(() => import('@/features/profile/pages/KitchenSetupPage'));
const ReviewsPage = lazyWithRetry(() => import('@/features/reviews/pages/ReviewsPage'));
const AnalyticsPage = lazyWithRetry(() => import('@/features/analytics/pages/AnalyticsPage'));
const SettingsPage = lazyWithRetry(() => import('@/features/settings/pages/SettingsPage'));
const AdminRequestsPage = lazyWithRetry(() => import('@/features/notifications/pages/NotificationsPage'));

function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const { isLoading, isAuthenticated } = useAuth();

  if (isLoading) {
    return <LoadingScreen />;
  }

  if (!isAuthenticated) {
    return <Navigate to="/login" replace />;
  }

  return <>{children}</>;
}

function PublicRoute({ children }: { children: React.ReactNode }) {
  const { isLoading, isAuthenticated } = useAuth();

  if (isLoading) {
    return <LoadingScreen />;
  }

  // If already authenticated, redirect to dashboard
  if (isAuthenticated) {
    return <Navigate to="/dashboard" replace />;
  }

  return <>{children}</>;
}

export function AppRoutes() {
  return (
    <Suspense fallback={<LoadingScreen />}>
      <Routes>
        {/* Public routes - redirect to dashboard if already logged in */}
        <Route path="/login" element={<PublicRoute><LoginPage /></PublicRoute>} />
        <Route path="/register" element={<PublicRoute><RegisterPage /></PublicRoute>} />

        {/* Onboarding - authenticated but no layout (standalone fullscreen wizard) */}
        <Route
          path="/onboarding"
          element={
            <ProtectedRoute>
              <OnboardingPage />
            </ProtectedRoute>
          }
        />

        {/* Protected vendor routes */}
        <Route
          element={
            <ProtectedRoute>
              <VendorLayout />
            </ProtectedRoute>
          }
        >
          <Route index element={<Navigate to="/dashboard" replace />} />
          <Route path="dashboard" element={<DashboardPage />} />
          <Route path="menu" element={<MenuPage />} />
          <Route path="menu/new" element={<MenuItemFormPage />} />
          <Route path="menu/:id" element={<MenuItemViewPage />} />
          <Route path="menu/:id/edit" element={<MenuItemFormPage />} />
          <Route path="orders" element={<LiveOrdersPage />} />
          <Route path="orders/history" element={<OrderHistoryPage />} />
          <Route path="earnings" element={<EarningsPage />} />
          <Route path="earnings/payouts" element={<PayoutsPage />} />
          <Route path="profile" element={<ProfilePage />} />
          <Route path="profile/kitchen" element={<KitchenSetupPage />} />
          <Route path="reviews" element={<ReviewsPage />} />
          <Route path="admin-requests" element={<AdminRequestsPage />} />
          <Route path="analytics" element={<AnalyticsPage />} />
          <Route path="settings" element={<SettingsPage />} />
        </Route>

        {/* Catch all */}
        <Route path="*" element={<Navigate to="/dashboard" replace />} />
      </Routes>
    </Suspense>
  );
}
