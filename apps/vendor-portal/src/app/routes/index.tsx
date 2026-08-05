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
const ChefBookPage = lazyWithRetry(() => import('@/features/chefbook/pages/ChefBookPage'));
const RegisterPage = lazyWithRetry(() => import('@/features/auth/pages/RegisterPage'));
const ForgotPasswordPage = lazyWithRetry(() => import('@/features/auth/pages/ForgotPasswordPage'));

// Onboarding
const OnboardingPage = lazyWithRetry(() => import('@/features/onboarding/pages/OnboardingPage'));

// Feature pages
const DashboardPage = lazyWithRetry(() => import('@/features/dashboard/pages/DashboardPage'));
const MenuPage = lazyWithRetry(() => import('@/features/menu/pages/MenuPage'));
const MenuItemFormPage = lazyWithRetry(() => import('@/features/menu/pages/MenuItemFormPage'));
const MenuItemViewPage = lazyWithRetry(() => import('@/features/menu/pages/MenuItemViewPage'));
const CapacityPage = lazyWithRetry(() => import('@/features/capacity/pages/CapacityPage'));
const PrepPage = lazyWithRetry(() => import('@/features/meal-plans/pages/PrepPage'));
const CancellationRequestsPage = lazyWithRetry(
  () => import('@/features/cancellations/pages/CancellationRequestsPage'),
);
const LiveOrdersPage = lazyWithRetry(() => import('@/features/orders/pages/LiveOrdersPage'));
const OrderHistoryPage = lazyWithRetry(() => import('@/features/orders/pages/OrderHistoryPage'));
const EarningsPage = lazyWithRetry(() => import('@/features/earnings/pages/EarningsPage'));
const PayoutsPage = lazyWithRetry(() => import('@/features/earnings/pages/PayoutsPage'));
const ExpensesPage = lazyWithRetry(() => import('@/features/earnings/pages/ExpensesPage'));
const RewardsPage = lazyWithRetry(() => import('@/features/rewards/pages/RewardsPage'));
const ProfilePage = lazyWithRetry(() => import('@/features/profile/pages/ProfilePage'));
const SecurityPage = lazyWithRetry(() => import('@/features/profile/pages/SecurityPage'));
const KitchenSetupPage = lazyWithRetry(() => import('@/features/profile/pages/KitchenSetupPage'));
const ReviewsPage = lazyWithRetry(() => import('@/features/reviews/pages/ReviewsPage'));
const AnalyticsPage = lazyWithRetry(() => import('@/features/analytics/pages/AnalyticsPage'));
const WeeklyMenuPage = lazyWithRetry(() => import('@/features/meal-plans/pages/WeeklyMenuPage'));
const SubscriptionSetupPage = lazyWithRetry(() => import('@/features/subscriptions/pages/SubscriptionSetupPage'));
const SettingsPage = lazyWithRetry(() => import('@/features/settings/pages/SettingsPage'));
// These were ONE import: /admin-requests rendered NotificationsPage under an
// AdminRequestsPage alias, so the portal had no admin-requests screen at all and
// the notifications inbox was only reachable at the wrong URL.
const AdminRequestsPage = lazyWithRetry(
  () => import('@/features/admin-requests/pages/AdminRequestsPage'),
);
const NotificationsPage = lazyWithRetry(
  () => import('@/features/notifications/pages/NotificationsPage'),
);
const AccountLifecyclePage = lazyWithRetry(
  () => import('@/features/account/pages/AccountLifecyclePage'),
);
const TiffinPlansPage = lazyWithRetry(() => import('@/features/meal-plans/pages/TiffinPlansPage'));
const DailyMenuPage = lazyWithRetry(() => import('@/features/meal-plans/pages/DailyMenuPage'));
const RefundDecisionsPage = lazyWithRetry(
  () => import('@/features/meal-plans/pages/RefundDecisionsPage'),
);
const PlanRequestPage = lazyWithRetry(() => import('@/features/meal-plans/pages/PlanRequestPage'));
const DocumentsPage = lazyWithRetry(() => import('@/features/documents/pages/DocumentsPage'));
const FssaiPage = lazyWithRetry(() => import('@/features/fssai/pages/FssaiPage'));
const SupportPage = lazyWithRetry(() => import('@/features/support/pages/SupportPage'));
const SupportTicketPage = lazyWithRetry(
  () => import('@/features/support/pages/SupportTicketPage'),
);
const CateringPage = lazyWithRetry(() => import('@/features/catering/pages/CateringPage'));
const LegalPage = lazyWithRetry(() => import('@/features/legal/pages/LegalPage'));

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
        <Route path="/forgot-password" element={<PublicRoute><ForgotPasswordPage /></PublicRoute>} />

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
          <Route path="weekly-menu" element={<WeeklyMenuPage />} />
          <Route path="subscriptions" element={<SubscriptionSetupPage />} />
          <Route path="capacity" element={<CapacityPage />} />
          <Route path="prep" element={<PrepPage />} />
          <Route path="cancel-requests" element={<CancellationRequestsPage />} />
          <Route path="menu/new" element={<MenuItemFormPage />} />
          <Route path="menu/:id" element={<MenuItemViewPage />} />
          <Route path="menu/:id/edit" element={<MenuItemFormPage />} />
          <Route path="orders" element={<LiveOrdersPage />} />
          <Route path="orders/history" element={<OrderHistoryPage />} />
          <Route path="earnings" element={<EarningsPage />} />
          <Route path="earnings/payouts" element={<PayoutsPage />} />
          <Route path="earnings/expenses" element={<ExpensesPage />} />
          <Route path="rewards" element={<RewardsPage />} />
          <Route path="profile" element={<ProfilePage />} />
          <Route path="security" element={<SecurityPage />} />
          <Route path="profile/kitchen" element={<KitchenSetupPage />} />
          <Route path="reviews" element={<ReviewsPage />} />
          <Route path="admin-requests" element={<AdminRequestsPage />} />
          <Route path="notifications" element={<NotificationsPage />} />
          <Route path="account" element={<AccountLifecyclePage />} />
          <Route path="tiffin-plans" element={<TiffinPlansPage />} />
          <Route path="tiffin-plans/requests/:id" element={<PlanRequestPage />} />
          <Route path="daily-menu" element={<DailyMenuPage />} />
          <Route path="refund-requests" element={<RefundDecisionsPage />} />
          <Route path="documents" element={<DocumentsPage />} />
          <Route path="fssai" element={<FssaiPage />} />
          <Route path="support" element={<SupportPage />} />
          <Route path="support/:id" element={<SupportTicketPage />} />
          <Route path="catering" element={<CateringPage />} />
          <Route path="legal" element={<LegalPage />} />
          <Route path="analytics" element={<AnalyticsPage />} />
          <Route path="chefbook" element={<ChefBookPage />} />
          <Route path="settings" element={<SettingsPage />} />
        </Route>

        {/* Catch all */}
        <Route path="*" element={<Navigate to="/dashboard" replace />} />
      </Routes>
    </Suspense>
  );
}
