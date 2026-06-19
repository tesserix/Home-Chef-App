import { Outlet, Link, useLocation } from 'react-router-dom';
import {
  LayoutDashboard,
  UtensilsCrossed,
  ClipboardList,
  DollarSign,
  User,
  Star,
  BarChart3,
  Gauge,
  Settings,
  Menu,
  X,
  LogOut,
  Bell,
  ChevronDown,
  ChefHat,
  Sparkles,
  CalendarDays,
} from 'lucide-react';
import { useState, useEffect } from 'react';
import { toast } from 'sonner';
import { useAuth } from '@/app/providers/AuthProvider';
import { VendorBottomNav } from '@/shared/components/navigation';
import { useIsMobile, useOnlineStatus } from '@/shared/hooks/useMobile';
import { useNotificationsWS } from '@/shared/hooks/useNotificationsWS';
import { ErrorBoundary } from '@/shared/components/ErrorBoundary';
import { ThemeToggle } from '@/shared/theme';

const navigation = [
  { name: 'Dashboard', href: '/dashboard', icon: LayoutDashboard },
  { name: 'Menu', href: '/menu', icon: UtensilsCrossed },
  { name: 'Weekly Menu', href: '/weekly-menu', icon: CalendarDays },
  { name: 'Capacity', href: '/capacity', icon: Gauge },
  { name: 'Orders', href: '/orders', icon: ClipboardList },
  { name: 'Earnings', href: '/earnings', icon: DollarSign },
  { name: 'Admin Requests', href: '/admin-requests', icon: ClipboardList },
  { name: 'Reviews', href: '/reviews', icon: Star },
  { name: 'Analytics', href: '/analytics', icon: BarChart3 },
  { name: 'Premium', href: '/premium', icon: Sparkles },
  { name: 'Profile', href: '/profile', icon: User },
  { name: 'Settings', href: '/settings', icon: Settings },
];

export function VendorLayout() {
  const location = useLocation();
  const { user, logout } = useAuth();
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const [userMenuOpen, setUserMenuOpen] = useState(false);
  const isMobile = useIsMobile('lg');
  const isOnline = useOnlineStatus();

  const { unreadCount, lastNotification } = useNotificationsWS();

  // Show toast when a new real-time notification arrives
  useEffect(() => {
    if (lastNotification?.title) {
      toast(lastNotification.title, {
        description: lastNotification.message,
      });
    }
  }, [lastNotification]);

  const isActive = (href: string) =>
    location.pathname === href || location.pathname.startsWith(href + '/');

  const displayName = user?.name || [user?.firstName, user?.lastName].filter(Boolean).join(' ') || user?.email || 'Vendor';

  return (
    <div className="flex h-screen bg-background">
      {/* Sidebar - Desktop */}
      <aside className="hidden w-64 flex-shrink-0 border-r border-border bg-card lg:block">
        <div className="flex h-full flex-col">
          {/* Logo */}
          <div className="flex h-16 items-center gap-2 border-b border-border px-6">
            <div className="logo-icon">
              <span><ChefHat className="h-5 w-5" /></span>
            </div>
            <div>
              <span className="logo-text">Fe3dr</span>
              <p className="logo-tagline">Vendor Portal</p>
            </div>
          </div>

          {/* Navigation */}
          <nav className="flex-1 space-y-1 p-4">
            {navigation.map((item) => {
              const Icon = item.icon;
              return (
                <Link
                  key={item.name}
                  to={item.href}
                  className={`flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium transition-colors ${
                    isActive(item.href)
                      ? 'bg-primary/10 text-primary'
                      : 'text-muted-foreground hover:bg-secondary hover:text-foreground'
                  }`}
                >
                  <Icon className="h-5 w-5" />
                  {item.name}
                  {item.name === 'Admin Requests' && unreadCount > 0 && (
                    <span className="ml-auto flex h-5 min-w-5 items-center justify-center rounded-full bg-destructive px-1.5 text-xs font-medium text-destructive-foreground">
                      {unreadCount > 99 ? '99+' : unreadCount}
                    </span>
                  )}
                </Link>
              );
            })}
          </nav>

          {/* User section */}
          <div className="border-t border-border p-4">
            <button
              type="button"
              aria-haspopup="menu"
              aria-expanded={userMenuOpen}
              aria-label={userMenuOpen ? 'Close user menu' : 'Open user menu'}
              onClick={() => setUserMenuOpen(!userMenuOpen)}
              className="flex w-full items-center gap-3 rounded-lg p-2 transition-colors hover:bg-secondary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
            >
              <div className="flex h-10 w-10 items-center justify-center rounded-full bg-primary/10">
                <User aria-hidden="true" className="h-5 w-5 text-primary" />
              </div>
              <div className="flex-1 text-left">
                <p className="text-sm font-medium text-foreground truncate">
                  {displayName}
                </p>
                <p className="text-xs text-muted-foreground">Vendor</p>
              </div>
              <ChevronDown aria-hidden="true" className="h-4 w-4 text-muted-foreground" />
            </button>

            {userMenuOpen && (
              <div role="menu" className="mt-2 space-y-1">
                <Link
                  to="/settings"
                  role="menuitem"
                  className="flex items-center gap-2 rounded-lg px-3 py-2 text-sm text-muted-foreground transition-colors hover:bg-secondary hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
                >
                  <Settings aria-hidden="true" className="h-4 w-4" />
                  Settings
                </Link>
                <div className="flex items-center justify-between rounded-lg px-3 py-2 text-sm text-muted-foreground">
                  <span id="vendor-theme-label">Theme</span>
                  <ThemeToggle size="sm" />
                </div>
                <button
                  type="button"
                  role="menuitem"
                  onClick={logout}
                  className="flex w-full items-center gap-2 rounded-lg px-3 py-2 text-sm text-destructive transition-colors hover:bg-destructive/10 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-destructive focus-visible:ring-offset-2"
                >
                  <LogOut aria-hidden="true" className="h-4 w-4" />
                  Logout
                </button>
              </div>
            )}
          </div>
        </div>
      </aside>

      {/* Mobile sidebar */}
      {sidebarOpen && (
        <>
          <div
            className="fixed inset-0 z-40 bg-foreground/50 lg:hidden"
            onClick={() => setSidebarOpen(false)}
          />
          <aside
            id="vendor-mobile-sidebar"
            aria-label="Main navigation"
            className="fixed inset-y-0 left-0 z-50 w-64 bg-card lg:hidden"
          >
            <div className="flex h-full flex-col">
              <div className="flex h-16 items-center justify-between border-b border-border px-4">
                <Link to="/dashboard" className="logo">
                  <div className="logo-icon">
                    <span><ChefHat className="h-5 w-5" /></span>
                  </div>
                  <span className="logo-text">Fe3dr</span>
                </Link>
                <button
                  type="button"
                  aria-label="Close navigation"
                  onClick={() => setSidebarOpen(false)}
                  className="rounded-lg p-1 transition-colors hover:bg-secondary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
                >
                  <X aria-hidden="true" className="h-6 w-6 text-muted-foreground" />
                </button>
              </div>
              <nav className="flex-1 space-y-1 p-4">
                {navigation.map((item) => {
                  const Icon = item.icon;
                  return (
                    <Link
                      key={item.name}
                      to={item.href}
                      onClick={() => setSidebarOpen(false)}
                      className={`flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium ${
                        isActive(item.href)
                          ? 'bg-primary/10 text-primary'
                          : 'text-muted-foreground hover:bg-secondary'
                      }`}
                    >
                      <Icon className="h-5 w-5" />
                      {item.name}
                    </Link>
                  );
                })}
              </nav>
            </div>
          </aside>
        </>
      )}

      {/* Main content */}
      <div className="flex flex-1 flex-col overflow-hidden">
        {/* Top bar */}
        <header className="flex h-16 items-center justify-between border-b border-border bg-card px-4 lg:px-8">
          <button
            type="button"
            aria-label="Open navigation"
            aria-expanded={sidebarOpen}
            aria-controls="vendor-mobile-sidebar"
            onClick={() => setSidebarOpen(true)}
            className="rounded-lg p-2 transition-colors hover:bg-secondary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 lg:hidden"
          >
            <Menu aria-hidden="true" className="h-5 w-5" />
          </button>

          {/* Notifications */}
          <Link
            to="/admin-requests"
            aria-label={unreadCount > 0 ? `Notifications, ${unreadCount} unread` : 'Notifications'}
            className="relative ml-auto rounded-lg p-2 transition-colors hover:bg-secondary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
          >
            <Bell aria-hidden="true" className="h-5 w-5 text-muted-foreground" />
            {unreadCount > 0 && <span aria-hidden="true" className="notification-dot" />}
          </Link>
        </header>

        {/* Page content */}
        <main id="main" className={`flex-1 overflow-y-auto p-4 lg:p-8 ${isMobile ? 'pb-20' : ''}`}>
          <ErrorBoundary>
            <Outlet />
          </ErrorBoundary>
        </main>
      </div>

      {/* Mobile Bottom Navigation */}
      {isMobile && <VendorBottomNav />}

      {/* Offline Banner */}
      {!isOnline && (
        <div className="fixed inset-x-0 top-0 z-50 bg-warning px-4 py-2 text-center text-sm font-medium text-warning-foreground safe-top">
          You're offline. Orders will sync when connected.
        </div>
      )}
    </div>
  );
}
