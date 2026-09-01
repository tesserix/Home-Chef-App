import {
  createContext,
  useContext,
  useEffect,
  useCallback,
  useState,
  useRef,
  type ReactNode,
} from 'react';
import { useNavigate, useLocation } from 'react-router';
import { useQueryClient } from '@tanstack/react-query';
import { useAuthStore } from '../store/auth-store';
import type { SessionUser } from '@/shared/types/auth';

const BFF_URL = (() => {
  const env = import.meta.env.VITE_BFF_URL;
  if (env) return env;
  if (typeof window !== 'undefined' && window.location.hostname !== 'localhost') {
    return `${window.location.origin}/bff`;
  }
  return '/bff';
})();

interface AuthContextValue {
  user: SessionUser | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  csrfToken: string | null;
  needsOnboarding: boolean;
  onboardingStatus: string;
  adminNotes: string;
  login: (options?: { returnTo?: string }) => Promise<void>;
  register: () => Promise<void>;
  logout: () => Promise<void>;
}

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const location = useLocation();
  // Wipe the cache whenever the signed-in identity changes, not only on an
  // explicit logout. A session that expires and is replaced by a different
  // chef never runs logout(), and the QueryClient lives at module scope, so
  // the previous account's data would otherwise be served to the new one.
  // Same guard the customer app keeps in its root layout.
  const signedInUserId = useAuthStore((s) => s.user?.id ?? null);
  const prevUserIdRef = useRef<string | null>(signedInUserId);
  useEffect(() => {
    if (prevUserIdRef.current !== signedInUserId) {
      queryClient.clear();
      prevUserIdRef.current = signedInUserId;
    }
  }, [signedInUserId, queryClient]);
  const {
    user,
    isAuthenticated,
    isLoading,
    csrfToken,
    clearAuth,
    initialize,
  } = useAuthStore();
  const [needsOnboarding, setNeedsOnboarding] = useState(false);
  const [onboardingChecked, setOnboardingChecked] = useState(false);
  const [onboardingStatus, setOnboardingStatus] = useState('');
  const [adminNotes, setAdminNotes] = useState('');

  useEffect(() => {
    initialize();
  }, [initialize]);

  // No session on load? Drop any persisted onboarding draft.
  //
  // logout() already clears it, but that only helps people who actually log
  // out. A draft left behind by an abandoned signup — or by someone who just
  // closed the tab — otherwise survives in localStorage and rehydrates for
  // whoever opens the browser next, showing a stranger's name, phone, address
  // and email. On a shared or family device that is a straightforward PII leak,
  // and it is also why a logged-out visitor could see a half-filled form
  // instead of a clean one.
  useEffect(() => {
    if (isLoading || isAuthenticated) return;
    import('@/app/store/onboarding-store').then(({ useOnboardingStore }) => {
      useOnboardingStore.getState().reset();
    });
  }, [isAuthenticated, isLoading]);

  // After auth, check if chef profile exists and hydrate form data from server
  useEffect(() => {
    if (!isAuthenticated || isLoading || onboardingChecked) return;

    const checkOnboarding = async () => {
      try {
        const res = await fetch(`${BFF_URL}/api/v1/chef/onboarding/status`, {
          credentials: 'include',
        });
        if (!res.ok) {
          setNeedsOnboarding(true);
          if (!location.pathname.startsWith('/onboarding')) {
            navigate('/onboarding', { replace: true });
          }
          return;
        }
        const status = await res.json();
        setOnboardingStatus(status.status || '');
        setAdminNotes(status.adminNotes || '');
        if (status.completed) {
          setNeedsOnboarding(false);
        } else {
          // Only redirect to onboarding for first-time users who haven't submitted yet.
          // For rejected/info_requested, the chef has already submitted their profile —
          // let them stay on dashboard and see the notification in Admin Requests.
          const shouldRedirectToOnboarding =
            status.status === 'not_started' || status.status === 'in_progress';

          if (shouldRedirectToOnboarding) {
            setNeedsOnboarding(true);

            // Hydrate the onboarding form with server data.
            // First check if the local store has data from a different user and reset if so.
            const { useOnboardingStore } = await import('@/app/store/onboarding-store');
            const localEmail = useOnboardingStore.getState().data.email;
            const currentEmail = user?.email || status.profile?.email || '';
            if (localEmail && currentEmail && localEmail !== currentEmail) {
              // Different user logged in - clear stale data
              useOnboardingStore.getState().reset();
            }
            if (status.profile) {
              useOnboardingStore.getState().hydrateFromServer(status.step || 0, status.profile);
            }

            if (!location.pathname.startsWith('/onboarding')) {
              navigate('/onboarding', { replace: true });
            }
          } else {
            // rejected or info_requested — profile already submitted, no redirect
            setNeedsOnboarding(false);
          }
        }
      } catch {
        setNeedsOnboarding(false);
      } finally {
        setOnboardingChecked(true);
      }
    };

    checkOnboarding();
  }, [isAuthenticated, isLoading, onboardingChecked, navigate, location.pathname, user?.email]);

  // Login and registration are full-page redirects into the hosted Zitadel
  // pages via the BFF; credentials never touch this app any more.
  const login = useCallback(async (options?: { returnTo?: string }) => {
    const svc = await import('@/features/auth/services/auth-service');
    svc.redirectToLogin({ returnTo: options?.returnTo });
  }, []);

  const register = useCallback(async () => {
    const svc = await import('@/features/auth/services/auth-service');
    svc.redirectToLogin({ register: true });
  }, []);

  const logout = useCallback(async () => {
    const { authService } = await import('@/features/auth/services/auth-service');
    await authService.logout();
    clearAuth();
    // Drop every cached server response for the account that just left.
    //
    // The QueryClient is created once at module scope and logout is an SPA
    // navigation, so without this the cache OUTLIVES the session: the next chef
    // to sign in on this browser is served the previous one's data until each
    // query happens to refetch — and the default staleTime here is five
    // minutes. That is a cross-user leak of business name, orders, earnings and
    // profile, and the account menu now renders another chef's kitchen name in
    // the header on every page.
    queryClient.clear();
    setOnboardingChecked(false);
    setNeedsOnboarding(false);
    // Clear onboarding form data to prevent cross-user contamination
    import('@/app/store/onboarding-store').then(({ useOnboardingStore }) => {
      useOnboardingStore.getState().reset();
    });
    navigate('/login');
  }, [clearAuth, navigate, queryClient]);

  const value: AuthContextValue = {
    user,
    isAuthenticated,
    isLoading,
    csrfToken,
    needsOnboarding,
    onboardingStatus,
    adminNotes,
    login,
    register,
    logout,
  };

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
}
