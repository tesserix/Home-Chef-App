import { Routes, Route, Navigate } from 'react-router-dom';
import { Suspense, lazy } from 'react';
import { useAuth } from '../providers/AuthProvider';
import { LoadingScreen } from '@/shared/components/LoadingScreen';
import { DeliveryLayout } from '@/shared/components/layout/DeliveryLayout';

function lazyWithRetry(factory: () => Promise<{ default: React.ComponentType }>) {
  return lazy(() =>
    factory().catch((err: unknown) => {
      const key = 'chunk-reload';
      const hasReloaded = sessionStorage.getItem(key);
      if (!hasReloaded) {
        sessionStorage.setItem(key, '1');
        window.location.reload();
        return new Promise(() => {});
      }
      sessionStorage.removeItem(key);
      throw err;
    })
  );
}

// Auth pages (login only - no registration, uses internal Keycloak realm)
const LoginPage = lazyWithRetry(() => import('@/features/auth/pages/LoginPage'));

// Onboarding pages (outside DeliveryLayout)
const OnboardingPage = lazyWithRetry(() => import('@/features/onboarding/pages/OnboardingPage'));
const OnboardingStatusPage = lazyWithRetry(() => import('@/features/onboarding/pages/OnboardingStatusPage'));

// Feature pages
const DashboardPage = lazyWithRetry(() => import('@/features/dashboard/pages/DashboardPage'));
const ActiveDeliveryPage = lazyWithRetry(() => import('@/features/deliveries/pages/ActiveDeliveryPage'));
const AvailableDeliveriesPage = lazyWithRetry(() => import('@/features/deliveries/pages/AvailableDeliveriesPage'));
const DeliveryHistoryPage = lazyWithRetry(() => import('@/features/deliveries/pages/DeliveryHistoryPage'));
const EarningsPage = lazyWithRetry(() => import('@/features/earnings/pages/EarningsPage'));
const ProfilePage = lazyWithRetry(() => import('@/features/profile/pages/ProfilePage'));
const SettingsPage = lazyWithRetry(() => import('@/features/settings/pages/SettingsPage'));
const StaffPage = lazyWithRetry(() => import('@/features/staff/pages/StaffPage'));
const FleetOverviewPage = lazyWithRetry(() => import('@/features/fleet/pages/FleetOverviewPage'));
const PartnersPage = lazyWithRetry(() => import('@/features/fleet/pages/PartnersPage'));
const PartnerDetailPage = lazyWithRetry(() => import('@/features/fleet/pages/PartnerDetailPage'));

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

  if (isAuthenticated) {
    return <Navigate to="/dashboard" replace />;
  }

  return <>{children}</>;
}

export function AppRoutes() {
  return (
    <Suspense fallback={<LoadingScreen />}>
      <Routes>
        {/* Public routes - login only, no registration */}
        <Route path="/login" element={<PublicRoute><LoginPage /></PublicRoute>} />

        {/* Onboarding routes - outside DeliveryLayout (own minimal layout) */}
        <Route path="onboarding" element={<ProtectedRoute><OnboardingPage /></ProtectedRoute>} />
        <Route path="onboarding/status" element={<ProtectedRoute><OnboardingStatusPage /></ProtectedRoute>} />

        {/* Protected delivery routes */}
        <Route
          element={
            <ProtectedRoute>
              <DeliveryLayout />
            </ProtectedRoute>
          }
        >
          <Route index element={<Navigate to="/dashboard" replace />} />
          <Route path="dashboard" element={<DashboardPage />} />
          <Route path="active" element={<ActiveDeliveryPage />} />
          <Route path="available" element={<AvailableDeliveriesPage />} />
          <Route path="history" element={<DeliveryHistoryPage />} />
          <Route path="earnings" element={<EarningsPage />} />
          <Route path="staff" element={<StaffPage />} />
          <Route path="fleet" element={<FleetOverviewPage />} />
          <Route path="fleet/partners" element={<PartnersPage />} />
          <Route path="fleet/partners/:id" element={<PartnerDetailPage />} />
          <Route path="profile" element={<ProfilePage />} />
          <Route path="settings" element={<SettingsPage />} />
        </Route>

        {/* Catch all */}
        <Route path="*" element={<Navigate to="/dashboard" replace />} />
      </Routes>
    </Suspense>
  );
}
