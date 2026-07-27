import { Routes, Route, Navigate } from 'react-router-dom';
import { Suspense, lazy } from 'react';
import { useAuth } from '../providers/AuthProvider';
import { LoadingScreen } from '@/shared/components/LoadingScreen';
import { MainLayout } from '@/shared/components/layout/MainLayout';
import { SOCIAL_ENABLED } from '@/shared/config/features';

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
const MealSubscribePage = lazyWithRetry(() => import('@/features/customer/pages/MealSubscribePage'));
const SubscriptionsPage = lazyWithRetry(() => import('@/features/customer/pages/SubscriptionsPage'));
const CartPage = lazyWithRetry(() => import('@/features/customer/pages/CartPage'));
const CheckoutPage = lazyWithRetry(() => import('@/features/customer/pages/CheckoutPage'));
const OrdersPage = lazyWithRetry(() => import('@/features/customer/pages/OrdersPage'));
const OrderDetailPage = lazyWithRetry(() => import('@/features/customer/pages/OrderDetailPage'));
const ReviewPage = lazyWithRetry(() => import('@/features/customer/pages/ReviewPage'));
const TipPage = lazyWithRetry(() => import('@/features/customer/pages/TipPage'));
const GroupOrderPage = lazyWithRetry(() => import('@/features/customer/pages/GroupOrderPage'));
const GroupInvitePage = lazyWithRetry(() => import('@/features/customer/pages/GroupInvitePage'));
const ProfilePage = lazyWithRetry(() => import('@/features/customer/pages/ProfilePage'));
const DataPrivacyPage = lazyWithRetry(() => import('@/features/customer/pages/DataPrivacyPage'));
const WalletPage = lazyWithRetry(() => import('@/features/customer/pages/WalletPage'));
const LoyaltyPage = lazyWithRetry(() => import('@/features/customer/pages/LoyaltyPage'));
const ReferralPage = lazyWithRetry(() => import('@/features/customer/pages/ReferralPage'));
const SocialFeedPage = lazyWithRetry(() => import('@/features/social/pages/SocialFeedPage'));
const ChefBookFeedPage = lazyWithRetry(() => import('@/features/chefbook/pages/ChefBookFeedPage'));
const ArticlePage = lazyWithRetry(() => import('@/features/chefbook/pages/ArticlePage'));
const FavoritesPage = lazyWithRetry(() => import('@/features/customer/pages/FavoritesPage'));
const CateringRequestPage = lazyWithRetry(() => import('@/features/catering/pages/CateringRequestPage'));
const CateringQuotesPage = lazyWithRetry(() => import('@/features/catering/pages/CateringQuotesPage'));

// Onboarding
const UserInfoPage = lazyWithRetry(() => import('@/features/onboarding/pages/UserInfoPage'));

// Auth pages
const LoginPage = lazyWithRetry(() => import('@/features/auth/pages/LoginPage'));
const RegisterPage = lazyWithRetry(() => import('@/features/auth/pages/RegisterPage'));
const ForgotPasswordPage = lazyWithRetry(() => import('@/features/auth/pages/ForgotPasswordPage'));

// Legal pages
const TermsPage = lazyWithRetry(() => import('@/features/legal/pages/TermsPage'));
const PrivacyPolicyPage = lazyWithRetry(() => import('@/features/legal/pages/PrivacyPolicyPage'));
const RefundPolicyPage = lazyWithRetry(() => import('@/features/legal/pages/RefundPolicyPage'));
const CookiePolicyPage = lazyWithRetry(() => import('@/features/legal/pages/CookiePolicyPage'));

// Protected route wrapper.
//
// Deliberately has no role check: every route in this app is a customer route,
// so a session is the only gate. The chef/admin/delivery role guards that used
// to live here went with those route trees to their own apps.
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

export function AppRoutes() {
  return (
    <Suspense fallback={<LoadingScreen />}>
      <Routes>
        {/* Public routes */}
        <Route path="/login" element={<LoginPage />} />
        <Route path="/register" element={<RegisterPage />} />
        <Route path="/forgot-password" element={<ForgotPasswordPage />} />

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
          <Route path="chefs/:id/subscribe" element={<MealSubscribePage />} />
          <Route path="subscriptions" element={<SubscriptionsPage />} />
          {/* Gated to match the nav and footer, which already hide the feed
              behind SOCIAL_ENABLED. The route was left ungated, so the page
              stayed reachable by URL while the product had deferred it. */}
          {SOCIAL_ENABLED && <Route path="feed" element={<SocialFeedPage />} />}
          {/* ChefBook is public — an article is meant to be shareable to
              someone who has never opened the app. */}
          {SOCIAL_ENABLED && <Route path="chefbook" element={<ChefBookFeedPage />} />}
          {SOCIAL_ENABLED && <Route path="chefbook/:slug" element={<ArticlePage />} />}
          <Route path="favorites" element={<FavoritesPage />} />

          {/* Legal pages — public, under MainLayout shell */}
          <Route path="terms" element={<TermsPage />} />
          <Route path="privacy" element={<PrivacyPolicyPage />} />
          <Route path="refund" element={<RefundPolicyPage />} />
          <Route path="cookies" element={<CookiePolicyPage />} />
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
            path="orders/:id/review"
            element={
              <ProtectedRoute>
                <ReviewPage />
              </ProtectedRoute>
            }
          />
          {/* Post-delivery tip (#45) */}
          <Route
            path="orders/:id/tip"
            element={
              <ProtectedRoute>
                <TipPage />
              </ProtectedRoute>
            }
          />
          {/* Group / office orders (#46) — shared cart hub + invite landing */}
          <Route
            path="group-orders/:id"
            element={
              <ProtectedRoute>
                <GroupOrderPage />
              </ProtectedRoute>
            }
          />
          <Route
            path="group/:token"
            element={
              <ProtectedRoute>
                <GroupInvitePage />
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
            path="data-privacy"
            element={
              <ProtectedRoute>
                <DataPrivacyPage />
              </ProtectedRoute>
            }
          />
          <Route
            path="wallet"
            element={
              <ProtectedRoute>
                <WalletPage />
              </ProtectedRoute>
            }
          />
          <Route
            path="loyalty"
            element={
              <ProtectedRoute>
                <LoyaltyPage />
              </ProtectedRoute>
            }
          />
          <Route
            path="referral"
            element={
              <ProtectedRoute>
                <ReferralPage />
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

        {/* Chef, admin and delivery-partner surfaces deliberately live in their
            own apps — vendor-portal / mobile-vendor, mobile-admin, and
            delivery-portal / mobile-delivery. This app is customer-only, which
            is what mobile-customer ships. */}

        {/* 404 */}
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </Suspense>
  );
}
