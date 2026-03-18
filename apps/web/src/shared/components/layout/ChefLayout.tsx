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
    <div className="flex h-screen bg-gray-50">
      {/* Sidebar - Desktop */}
      <aside className="hidden w-64 flex-shrink-0 border-r border-gray-200 bg-white lg:block">
        <div className="flex h-full flex-col">
          {/* Logo */}
          <div className="flex h-16 items-center gap-2 border-b border-gray-200 px-6">
            <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-brand-500">
              <span className="text-lg font-bold text-white">H</span>
            </div>
            <div>
              <span className="text-lg font-bold text-gray-900">Fe3dr</span>
              <p className="text-xs text-gray-500">Chef Dashboard</p>
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
                      ? 'bg-brand-50 text-brand-600'
                      : 'text-gray-600 hover:bg-gray-50 hover:text-gray-900'
                  }`}
                >
                  <Icon className="h-5 w-5" />
                  {item.name}
                </Link>
              );
            })}
          </nav>

          {/* User section */}
          <div className="border-t border-gray-200 p-4">
            <button
              onClick={() => setUserMenuOpen(!userMenuOpen)}
              className="flex w-full items-center gap-3 rounded-lg p-2 hover:bg-gray-50"
            >
              {user?.avatar ? (
                <img
                  src={user.avatar}
                  alt={user.firstName}
                  className="h-10 w-10 rounded-full object-cover"
                />
              ) : (
                <div className="flex h-10 w-10 items-center justify-center rounded-full bg-brand-100">
                  <User className="h-5 w-5 text-brand-600" />
                </div>
              )}
              <div className="flex-1 text-left">
                <p className="text-sm font-medium text-gray-900">
                  {user?.firstName} {user?.lastName}
                </p>
                <p className="text-xs text-gray-500">Chef</p>
              </div>
              <ChevronDown className="h-4 w-4 text-gray-400" />
            </button>

            {userMenuOpen && (
              <div className="mt-2 space-y-1">
                <Link
                  to="/chef/profile/settings"
                  className="flex items-center gap-2 rounded-lg px-3 py-2 text-sm text-gray-600 hover:bg-gray-50"
                >
                  <Settings className="h-4 w-4" />
                  Settings
                </Link>
                <button
                  onClick={logout}
                  className="flex w-full items-center gap-2 rounded-lg px-3 py-2 text-sm text-red-600 hover:bg-red-50"
                >
                  <LogOut className="h-4 w-4" />
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
            className="fixed inset-0 z-40 bg-black/50 lg:hidden"
            onClick={() => setSidebarOpen(false)}
          />
          <aside className="fixed inset-y-0 left-0 z-50 w-64 bg-white lg:hidden">
            <div className="flex h-full flex-col">
              <div className="flex h-16 items-center justify-between border-b border-gray-200 px-4">
                <Link to="/chef/dashboard" className="flex items-center gap-2">
                  <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-brand-500">
                    <span className="text-lg font-bold text-white">H</span>
                  </div>
                  <span className="text-lg font-bold text-gray-900">Fe3dr</span>
                </Link>
                <button onClick={() => setSidebarOpen(false)}>
                  <X className="h-6 w-6 text-gray-500" />
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
                          ? 'bg-brand-50 text-brand-600'
                          : 'text-gray-600 hover:bg-gray-50'
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
        <header className="flex h-16 items-center justify-between border-b border-gray-200 bg-white px-4 lg:px-8">
          <button
            onClick={() => setSidebarOpen(true)}
            className="rounded-lg p-2 hover:bg-gray-100 lg:hidden"
          >
            <Menu className="h-5 w-5" />
          </button>

          <div className="flex items-center gap-4">
            {/* Online/Offline toggle */}
            <div className="flex items-center gap-2">
              <span className="text-sm text-gray-600">Status:</span>
              <button className="flex items-center gap-2 rounded-full bg-green-100 px-3 py-1 text-sm font-medium text-green-700">
                <span className="h-2 w-2 rounded-full bg-green-500" />
                Online
              </button>
            </div>

            {/* Notifications */}
            <button className="relative rounded-lg p-2 hover:bg-gray-100">
              <Bell className="h-5 w-5 text-gray-600" />
              <span className="absolute right-1 top-1 h-2 w-2 rounded-full bg-red-500" />
            </button>
          </div>
        </header>

        {/* Page content */}
        <main className={`flex-1 overflow-y-auto p-4 lg:p-8 ${isMobile ? 'pb-20' : ''}`}>
          <Outlet />
        </main>
      </div>

      {/* Mobile Bottom Navigation */}
      {isMobile && <ChefBottomNav />}

      {/* Offline Banner */}
      {!isOnline && (
        <div className="fixed inset-x-0 top-0 z-50 bg-amber-500 px-4 py-2 text-center text-sm font-medium text-white safe-top">
          You're offline. Orders will sync when connected.
        </div>
      )}
    </div>
  );
}
