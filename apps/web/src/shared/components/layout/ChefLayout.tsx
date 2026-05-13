import { Outlet, Link, useLocation } from 'react-router-dom';
import {
  LayoutDashboard,
  UtensilsCrossed,
  ClipboardList,
  DollarSign,
  User,
  Image,
  Calendar,
  Menu,
  X,
  LogOut,
  Bell,
  ChevronDown,
  Settings,
} from 'lucide-react';
import { useState } from 'react';
import { useAuth } from '@/app/providers/AuthProvider';
import { ChefBottomNav } from '@/shared/components/navigation';
import { useIsMobile, useOnlineStatus } from '@/shared/hooks/useMobile';
import { ThemeToggle } from '@/shared/theme';

const navigation = [
  { name: 'Dashboard', href: '/chef/dashboard', icon: LayoutDashboard },
  { name: 'Menu', href: '/chef/menu', icon: UtensilsCrossed },
  { name: 'Orders', href: '/chef/orders', icon: ClipboardList },
  { name: 'Catering', href: '/chef/catering', icon: Calendar },
  { name: 'Earnings', href: '/chef/earnings', icon: DollarSign },
  { name: 'Social', href: '/chef/social', icon: Image },
  { name: 'Profile', href: '/chef/profile', icon: User },
];

export function ChefLayout() {
  const location = useLocation();
  const { user, logout } = useAuth();
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const [userMenuOpen, setUserMenuOpen] = useState(false);
  const isMobile = useIsMobile('lg');
  const isOnline = useOnlineStatus();

  const isActive = (href: string) => location.pathname === href;

  return (
    <div className="flex h-screen bg-paper">
      {/* Sidebar - Desktop */}
      <aside className="hidden w-64 flex-shrink-0 border-r border-mist bg-bone lg:block">
        <div className="flex h-full flex-col">
          {/* Logo */}
          <div className="flex h-16 items-center gap-2 border-b border-mist px-6">
            <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-herb">
              <span className="text-lg font-medium text-paper">H</span>
            </div>
            <div>
              <span className="text-lg font-medium text-ink">Fe3dr</span>
              <p className="text-xs text-ink-muted">Chef Dashboard</p>
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
                      ? 'bg-herb-tint text-herb'
                      : 'text-ink-soft hover:bg-paper hover:text-ink'
                  }`}
                >
                  <Icon className="h-5 w-5" />
                  {item.name}
                </Link>
              );
            })}
          </nav>

          {/* User section */}
          <div className="border-t border-mist p-4">
            <button
              type="button"
              aria-haspopup="menu"
              aria-expanded={userMenuOpen}
              aria-label={userMenuOpen ? 'Close user menu' : 'Open user menu'}
              onClick={() => setUserMenuOpen(!userMenuOpen)}
              className="flex w-full items-center gap-3 rounded-lg p-2 transition-colors hover:bg-paper focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-herb focus-visible:ring-offset-2"
            >
              {user?.avatar ? (
                <img
                  src={user.avatar}
                  alt={user.firstName}
                  className="h-10 w-10 rounded-full object-cover"
                />
              ) : (
                <div className="flex h-10 w-10 items-center justify-center rounded-full bg-herb-tint">
                  <User aria-hidden="true" className="h-5 w-5 text-herb" />
                </div>
              )}
              <div className="flex-1 text-left">
                <p className="text-sm font-medium text-ink">
                  {user?.firstName} {user?.lastName}
                </p>
                <p className="text-xs text-ink-muted">Chef</p>
              </div>
              <ChevronDown aria-hidden="true" className="h-4 w-4 text-ink-muted" />
            </button>

            {userMenuOpen && (
              <div role="menu" className="mt-2 space-y-1">
                <Link
                  to="/chef/profile/settings"
                  role="menuitem"
                  className="flex items-center gap-2 rounded-lg px-3 py-2 text-sm text-ink-soft transition-colors hover:bg-paper focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-herb focus-visible:ring-offset-2"
                >
                  <Settings aria-hidden="true" className="h-4 w-4" />
                  Settings
                </Link>
                <div className="flex items-center justify-between rounded-lg px-3 py-2 text-sm text-ink-soft">
                  <span>Theme</span>
                  <ThemeToggle size="sm" />
                </div>
                <button
                  type="button"
                  role="menuitem"
                  onClick={logout}
                  className="flex w-full items-center gap-2 rounded-lg px-3 py-2 text-sm text-paprika transition-colors hover:bg-paprika-tint focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-paprika focus-visible:ring-offset-2"
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
            className="fixed inset-0 z-40 bg-ink/50 lg:hidden"
            onClick={() => setSidebarOpen(false)}
          />
          <aside
            id="chef-mobile-sidebar"
            aria-label="Main navigation"
            className="fixed inset-y-0 left-0 z-50 w-64 bg-bone lg:hidden"
          >
            <div className="flex h-full flex-col">
              <div className="flex h-16 items-center justify-between border-b border-mist px-4">
                <Link to="/chef/dashboard" className="flex items-center gap-2">
                  <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-herb">
                    <span aria-hidden="true" className="text-lg font-medium text-paper">H</span>
                  </div>
                  <span className="text-lg font-medium text-ink">Fe3dr</span>
                </Link>
                <button
                  type="button"
                  aria-label="Close navigation"
                  onClick={() => setSidebarOpen(false)}
                  className="rounded-lg p-1 transition-colors hover:bg-mist focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-herb focus-visible:ring-offset-2"
                >
                  <X aria-hidden="true" className="h-6 w-6 text-ink-muted" />
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
                          ? 'bg-herb-tint text-herb'
                          : 'text-ink-soft hover:bg-paper'
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
        <header className="flex h-16 items-center justify-between border-b border-mist bg-bone px-4 lg:px-8">
          <button
            type="button"
            aria-label="Open navigation"
            aria-expanded={sidebarOpen}
            aria-controls="chef-mobile-sidebar"
            onClick={() => setSidebarOpen(true)}
            className="rounded-lg p-2 transition-colors hover:bg-mist focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-herb focus-visible:ring-offset-2 lg:hidden"
          >
            <Menu aria-hidden="true" className="h-5 w-5" />
          </button>

          <div className="flex items-center gap-4">
            {/* Online/Offline toggle */}
            <div className="flex items-center gap-2">
              <span id="chef-status-label" className="text-sm text-ink-soft">Status:</span>
              <button
                type="button"
                aria-labelledby="chef-status-label"
                aria-pressed="true"
                className="flex items-center gap-2 rounded-full bg-herb-tint px-3 py-1 text-sm font-medium text-herb transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-herb focus-visible:ring-offset-2"
              >
                <span aria-hidden="true" className="h-2 w-2 rounded-full bg-herb" />
                Online
              </button>
            </div>

            {/* Notifications */}
            <button
              type="button"
              aria-label="Notifications, unread"
              className="relative rounded-lg p-2 transition-colors hover:bg-mist focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-herb focus-visible:ring-offset-2"
            >
              <Bell aria-hidden="true" className="h-5 w-5 text-ink-soft" />
              <span aria-hidden="true" className="absolute right-1 top-1 h-2 w-2 rounded-full bg-paprika" />
            </button>
          </div>
        </header>

        {/* Page content */}
        <main id="main" className={`flex-1 overflow-y-auto p-4 lg:p-8 ${isMobile ? 'pb-20' : ''}`}>
          <Outlet />
        </main>
      </div>

      {/* Mobile Bottom Navigation */}
      {isMobile && <ChefBottomNav />}

      {/* Offline Banner */}
      {!isOnline && (
        <div className="fixed inset-x-0 top-0 z-50 bg-amber px-4 py-2 text-center text-sm font-medium text-paper safe-top">
          You're offline. Orders will sync when connected.
        </div>
      )}
    </div>
  );
}
