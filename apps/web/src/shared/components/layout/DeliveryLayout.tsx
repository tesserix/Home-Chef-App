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
    <div className="flex min-h-screen flex-col bg-gray-50">
      {/* Top bar */}
      <header className="sticky top-0 z-40 border-b border-gray-200 bg-white">
        <div className="flex h-16 items-center justify-between px-4">
          {/* Logo */}
          <Link to="/delivery/dashboard" className="flex items-center gap-2">
            <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-brand-500">
              <span className="text-lg font-bold text-white">H</span>
            </div>
            <div>
              <span className="text-lg font-bold text-gray-900">Fe3dr</span>
              <p className="text-xs text-gray-500">Delivery</p>
            </div>
          </Link>

          {/* Right side */}
          <div className="flex items-center gap-2">
            {/* Online toggle */}
            <button className="flex items-center gap-2 rounded-full bg-green-100 px-3 py-1.5 text-sm font-medium text-green-700">
              <span className="h-2 w-2 animate-pulse rounded-full bg-green-500" />
              Online
            </button>

            {/* Notifications */}
            <button className="relative rounded-lg p-2 hover:bg-gray-100">
              <Bell className="h-5 w-5 text-gray-600" />
              <span className="absolute right-1 top-1 h-2 w-2 rounded-full bg-red-500" />
            </button>

            {/* User */}
            <button className="flex items-center gap-2 rounded-lg p-2 hover:bg-gray-100">
              {user?.avatar ? (
                <img
                  src={user.avatar}
                  alt={user.firstName}
                  className="h-8 w-8 rounded-full object-cover"
                />
              ) : (
                <div className="flex h-8 w-8 items-center justify-center rounded-full bg-brand-100">
                  <User className="h-4 w-4 text-brand-600" />
                </div>
              )}
            </button>
          </div>
        </div>
      </header>

      {/* Main content */}
      <main className="flex-1 p-4 pb-20">
        <Outlet />
      </main>

      {/* Mobile Bottom Navigation */}
      <DeliveryBottomNav />

      {/* Offline Banner */}
      {!isOnline && (
        <div className="fixed inset-x-0 top-0 z-50 bg-amber-500 px-4 py-2 text-center text-sm font-medium text-white safe-top">
          You're offline. Deliveries will sync when connected.
        </div>
      )}
    </div>
  );
}
