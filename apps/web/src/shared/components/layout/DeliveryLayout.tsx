import { Outlet, Link } from 'react-router-dom';
import {
  User,
  Bell,
} from 'lucide-react';
import { useAuth } from '@/app/providers/AuthProvider';
import { DeliveryBottomNav } from '@/shared/components/navigation';
import { useOnlineStatus } from '@/shared/hooks/useMobile';

export function DeliveryLayout() {
  const { user } = useAuth();
  const isOnline = useOnlineStatus();

  return (
    <div className="flex min-h-screen flex-col bg-paper">
      {/* Top bar */}
      <header className="sticky top-0 z-40 border-b border-mist bg-bone">
        <div className="flex h-16 items-center justify-between px-4">
          {/* Logo */}
          <Link to="/delivery/dashboard" className="flex items-center gap-2">
            <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-herb">
              <span className="text-lg font-medium text-paper">H</span>
            </div>
            <div>
              <span className="text-lg font-medium text-ink">Fe3dr</span>
              <p className="text-xs text-ink-muted">Delivery</p>
            </div>
          </Link>

          {/* Right side */}
          <div className="flex items-center gap-2">
            {/* Online toggle */}
            <button
              type="button"
              aria-label="Currently online — tap to toggle availability"
              aria-pressed="true"
              className="flex items-center gap-2 rounded-full bg-herb-tint px-3 py-1.5 text-sm font-medium text-herb transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-herb focus-visible:ring-offset-2"
            >
              <span aria-hidden="true" className="h-2 w-2 animate-pulse rounded-full bg-herb" />
              Online
            </button>

            {/* Notifications */}
            <button
              type="button"
              aria-label="Notifications, unread"
              className="relative rounded-lg p-2 transition-colors hover:bg-mist focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-herb focus-visible:ring-offset-2"
            >
              <Bell aria-hidden="true" className="h-5 w-5 text-ink-soft" />
              <span aria-hidden="true" className="absolute right-1 top-1 h-2 w-2 rounded-full bg-paprika" />
            </button>

            {/* User */}
            <button
              type="button"
              aria-label={`Open menu for ${user?.firstName ?? 'driver'}`}
              aria-haspopup="menu"
              className="flex items-center gap-2 rounded-lg p-2 transition-colors hover:bg-mist focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-herb focus-visible:ring-offset-2"
            >
              {user?.avatar ? (
                <img
                  src={user.avatar}
                  alt=""
                  className="h-8 w-8 rounded-full object-cover"
                />
              ) : (
                <div className="flex h-8 w-8 items-center justify-center rounded-full bg-herb-tint">
                  <User aria-hidden="true" className="h-4 w-4 text-herb" />
                </div>
              )}
            </button>
          </div>
        </div>
      </header>

      {/* Main content */}
      <main id="main" className="flex-1 p-4 pb-20">
        <Outlet />
      </main>

      {/* Mobile Bottom Navigation */}
      <DeliveryBottomNav />

      {/* Offline Banner */}
      {!isOnline && (
        <div className="fixed inset-x-0 top-0 z-50 bg-amber px-4 py-2 text-center text-sm font-medium text-paper safe-top">
          You're offline. Deliveries will sync when connected.
        </div>
      )}
    </div>
  );
}
