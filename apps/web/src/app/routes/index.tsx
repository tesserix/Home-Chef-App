import { Routes, Route, Navigate } from 'react-router-dom';
import { Suspense, lazy } from 'react';
import { useAuth } from '../providers/AuthProvider';
import { LoadingScreen } from '@/shared/components/LoadingScreen';
import { MainLayout } from '@/shared/components/layout/MainLayout';
import { ChefLayout } from '@/shared/components/layout/ChefLayout';
import { AdminLayout } from '@/shared/components/layout/AdminLayout';
import { DeliveryLayout } from '@/shared/components/layout/DeliveryLayout';

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

// Lazy load pages for code splitting
const HomePage = lazyWithRetry(() => import('@/features/customer/pages/HomePage'));
const BrowseChefsPage = lazyWithRetry(() => import('@/features/customer/pages/BrowseChefsPage'));
const ChefDetailPage = lazyWithRetry(() => import('@/features/customer/pages/ChefDetailPage'));
const CartPage = lazyWithRetry(() => import('@/features/customer/pages/CartPage'));
const CheckoutPage = lazyWithRetry(() => import('@/features/customer/pages/CheckoutPage'));
const OrdersPage = lazyWithRetry(() => import('@/features/customer/pages/OrdersPage'));
const OrderDetailPage = lazyWithRetry(() => import('@/features/customer/pages/OrderDetailPage'));
const ProfilePage = lazyWithRetry(() => import('@/features/customer/pages/ProfilePage'));
const SocialFeedPage = lazyWithRetry(() => import('@/features/social/pages/SocialFeedPage'));
const FavoritesPage = lazyWithRetry(() => import('@/features/customer/pages/FavoritesPage'));
const CateringRequestPage = lazyWithRetry(() => import('@/features/catering/pages/CateringRequestPage'));
const CateringQuotesPage = lazyWithRetry(() => import('@/features/catering/pages/CateringQuotesPage'));

// Onboarding
const UserInfoPage = lazyWithRetry(() => import('@/features/onboarding/pages/UserInfoPage'));

// Auth pages
const LoginPage = lazyWithRetry(() => import('@/features/auth/pages/LoginPage'));
const RegisterPage = lazyWithRetry(() => import('@/features/auth/pages/RegisterPage'));

// Chef pages
const ChefDashboardPage = lazyWithRetry(() => import('@/features/chef/pages/DashboardPage'));
const ChefMenuPage = lazyWithRetry(() => import('@/features/chef/pages/MenuPage'));
const ChefOrdersPage = lazyWithRetry(() => import('@/features/chef/pages/OrdersPage'));
const ChefEarningsPage = lazyWithRetry(() => import('@/features/chef/pages/EarningsPage'));
const ChefProfilePage = lazyWithRetry(() => import('@/features/chef/pages/ProfilePage'));
const ChefSocialPage = lazyWithRetry(() => import('@/features/chef/pages/SocialPage'));
const ChefCateringPage = lazyWithRetry(() => import('@/features/chef/pages/CateringPage'));

// Admin pages
const AdminDashboardPage = lazyWithRetry(() => import('@/features/admin/pages/DashboardPage'));
const AdminUsersPage = lazyWithRetry(() => import('@/features/admin/pages/UsersPage'));
const AdminChefsPage = lazyWithRetry(() => import('@/features/admin/pages/ChefsPage'));
const AdminOrdersPage = lazyWithRetry(() => import('@/features/admin/pages/OrdersPage'));
const AdminAnalyticsPage = lazyWithRetry(() => import('@/features/admin/pages/AnalyticsPage'));
const AdminSettingsPage = lazyWithRetry(() => import('@/features/admin/pages/SettingsPage'));

// Delivery pages
const DeliveryDashboardPage = lazyWithRetry(() => import('@/features/delivery/pages/DashboardPage'));
const DeliveryOrdersPage = lazyWithRetry(() => import('@/features/delivery/pages/OrdersPage'));
const DeliveryEarningsPage = lazyWithRetry(() => import('@/features/delivery/pages/EarningsPage'));

// Protected route wrapper
function ProtectedRoute({
  children,
  roles,
}: {
  children: React.ReactNode;
  roles?: string[];
}) {
  const { user, isLoading, isAuthenticated } = useAuth();

  if (isLoading) {
    return <LoadingScreen />;
  }

  if (!isAuthenticated) {
    return <Navigate to="/login" replace />;
  }

  if (roles && user && !roles.some(r => user.roles?.includes(r))) {
    return <Navigate to="/" replace />;
  }

  return <>{children}</>;
}

export function AppRoutes() {
  return (
    <Suspense fallback={<LoadingScreen />}>
      <Routes>
        {/* Public routes */}
        <Route path="/login" element={<LoginPage />} />
        <Route path="/register" element={<RegisterPage />} />

        {/* Customer onboarding (no MainLayout — standalone page) */}
        <Route
          path="user-info"
          element={
            <ProtectedRoute>
              <UserInfoPage />
            </ProtectedRoute>
          }
        />

        {/* Customer routes */}
        <Route element={<MainLayout />}>
          <Route index element={<HomePage />} />
          <Route path="chefs" element={<BrowseChefsPage />} />
          <Route path="chefs/:id" element={<ChefDetailPage />} />
          <Route path="feed" element={<SocialFeedPage />} />
          <Route path="favorites" element={<FavoritesPage />} />
          <Route
            path="cart"
            element={
              <ProtectedRoute>
                <CartPage />
              </ProtectedRoute>
            }
          />
          <Route
            path="checkout"
            element={
              <ProtectedRoute>
                <CheckoutPage />
              </ProtectedRoute>
            }
          />
          <Route
            path="orders"
            element={
              <ProtectedRoute>
                <OrdersPage />
              </ProtectedRoute>
            }
          />
          <Route
            path="orders/:id"
            element={
              <ProtectedRoute>
                <OrderDetailPage />
              </ProtectedRoute>
            }
          />
          <Route
            path="profile"
            element={
              <ProtectedRoute>
                <ProfilePage />
              </ProtectedRoute>
            }
          />
          <Route
            path="catering"
            element={
              <ProtectedRoute>
                <CateringRequestPage />
              </ProtectedRoute>
            }
          />
          <Route
            path="catering/quotes"
            element={
              <ProtectedRoute>
                <CateringQuotesPage />
              </ProtectedRoute>
            }
          />
        </Route>

        {/* Chef routes */}
        <Route
          path="chef"
          element={
            <ProtectedRoute roles={['chef']}>
              <ChefLayout />
            </ProtectedRoute>
          }
        >
          <Route index element={<Navigate to="dashboard" replace />} />
          <Route path="dashboard" element={<ChefDashboardPage />} />
          <Route path="menu" element={<ChefMenuPage />} />
          <Route path="orders" element={<ChefOrdersPage />} />
          <Route path="earnings" element={<ChefEarningsPage />} />
          <Route path="profile" element={<ChefProfilePage />} />
          <Route path="social" element={<ChefSocialPage />} />
          <Route path="catering" element={<ChefCateringPage />} />
        </Route>

        {/* Admin routes */}
        <Route
          path="admin"
          element={
            <ProtectedRoute roles={['admin', 'super_admin']}>
              <AdminLayout />
            </ProtectedRoute>
          }
        >
          <Route index element={<Navigate to="dashboard" replace />} />
          <Route path="dashboard" element={<AdminDashboardPage />} />
          <Route path="users" element={<AdminUsersPage />} />
          <Route path="chefs" element={<AdminChefsPage />} />
          <Route path="orders" element={<AdminOrdersPage />} />
          <Route path="analytics" element={<AdminAnalyticsPage />} />
          <Route path="settings" element={<AdminSettingsPage />} />
        </Route>

        {/* Delivery partner routes */}
        <Route
          path="delivery"
          element={
            <ProtectedRoute roles={['delivery']}>
              <DeliveryLayout />
            </ProtectedRoute>
          }
        >
          <Route index element={<Navigate to="dashboard" replace />} />
          <Route path="dashboard" element={<DeliveryDashboardPage />} />
          <Route path="orders" element={<DeliveryOrdersPage />} />
          <Route path="earnings" element={<DeliveryEarningsPage />} />
        </Route>

        {/* 404 */}
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </Suspense>
  );
}
