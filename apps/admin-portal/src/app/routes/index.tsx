import { Routes, Route, Navigate } from 'react-router-dom';
import { Suspense, lazy } from 'react';
import { useAuth } from '../providers/AuthProvider';
import { LoadingScreen } from '@/shared/components/LoadingScreen';
import { AdminLayout } from '@/shared/components/layout/AdminLayout';

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

// Auth pages
const LoginPage = lazyWithRetry(() => import('@/features/auth/pages/LoginPage'));

// Feature pages
const DashboardPage = lazyWithRetry(() => import('@/features/dashboard/pages/DashboardPage'));
const UsersPage = lazyWithRetry(() => import('@/features/users/pages/UsersPage'));
const ChefsPage = lazyWithRetry(() => import('@/features/chefs/pages/ChefsPage'));
const OrdersPage = lazyWithRetry(() => import('@/features/orders/pages/OrdersPage'));
const AnalyticsPage = lazyWithRetry(() => import('@/features/analytics/pages/AnalyticsPage'));
const SettingsPage = lazyWithRetry(() => import('@/features/settings/pages/SettingsPage'));
const UserDetailPage = lazyWithRetry(() => import('@/features/users/pages/UserDetailPage'));
const ApprovalsPage = lazyWithRetry(() => import('@/features/approvals/pages/ApprovalsPage'));
const ApprovalDetailPage = lazyWithRetry(() => import('@/features/approvals/pages/ApprovalDetailPage'));
const DeliveryPage = lazyWithRetry(() => import('@/features/delivery/pages/DeliveryPage'));
const ProvidersPage = lazyWithRetry(() => import('@/features/delivery/pages/ProvidersPage'));
const ProviderDetailPage = lazyWithRetry(() => import('@/features/delivery/pages/ProviderDetailPage'));
const ProviderCreatePage = lazyWithRetry(() => import('@/features/delivery/pages/ProviderCreatePage'));
const StaffPage = lazyWithRetry(() => import('@/features/staff/pages/StaffPage'));
const StaffDetailPage = lazyWithRetry(() => import('@/features/staff/pages/StaffDetailPage'));

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
        {/* Public routes */}
        <Route path="/login" element={<PublicRoute><LoginPage /></PublicRoute>} />

        {/* Protected admin routes */}
        <Route
          element={
            <ProtectedRoute>
              <AdminLayout />
            </ProtectedRoute>
          }
        >
          <Route index element={<Navigate to="/dashboard" replace />} />
          <Route path="dashboard" element={<DashboardPage />} />
          <Route path="users" element={<UsersPage />} />
          <Route path="users/:id" element={<UserDetailPage />} />
          <Route path="chefs" element={<ChefsPage />} />
          <Route path="orders" element={<OrdersPage />} />
          <Route path="approvals" element={<ApprovalsPage />} />
          <Route path="approvals/:id" element={<ApprovalDetailPage />} />
          <Route path="delivery" element={<DeliveryPage />} />
          <Route path="delivery/providers" element={<ProvidersPage />} />
          <Route path="delivery/providers/new" element={<ProviderCreatePage />} />
          <Route path="delivery/providers/:id" element={<ProviderDetailPage />} />
          <Route path="staff" element={<StaffPage />} />
          <Route path="staff/:id" element={<StaffDetailPage />} />
          <Route path="analytics" element={<AnalyticsPage />} />
          <Route path="settings" element={<SettingsPage />} />
        </Route>

        {/* Catch all */}
        <Route path="*" element={<Navigate to="/dashboard" replace />} />
      </Routes>
    </Suspense>
  );
}
