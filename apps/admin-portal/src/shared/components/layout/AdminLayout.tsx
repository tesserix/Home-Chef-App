import { Outlet, Link, useLocation } from 'react-router-dom';
import {
  LayoutDashboard,
  Users,
  ChefHat,
  ShoppingBag,
  ClipboardCheck,
  Truck,
  UserCog,
  BarChart3,
  Settings,
  Shield,
  Menu,
  X,
  LogOut,
  Bell,
  ChevronDown,
} from 'lucide-react';
import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { useAuth } from '@/app/providers/AuthProvider';
import { apiClient } from '@/shared/services/api-client';
import { useIsMobile, useOnlineStatus } from '@/shared/hooks/useMobile';
import { ErrorBoundary } from '@/shared/components/ErrorBoundary';

const navigation = [
  { name: 'Dashboard', href: '/dashboard', icon: LayoutDashboard },
  { name: 'Users', href: '/users', icon: Users },
  { name: 'Chefs / Kitchens', href: '/chefs', icon: ChefHat },
  { name: 'Orders', href: '/orders', icon: ShoppingBag },
  { name: 'Reviews', href: '/approvals', icon: ClipboardCheck },
  { name: 'Delivery', href: '/delivery', icon: Truck },
  { name: 'Staff', href: '/staff', icon: UserCog },
  { name: 'Analytics', href: '/analytics', icon: BarChart3 },
  { name: 'Settings', href: '/settings', icon: Settings },
];

export function AdminLayout() {
  const location = useLocation();
  const { user, logout } = useAuth();
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const [userMenuOpen, setUserMenuOpen] = useState(false);
  const isMobile = useIsMobile('lg');
  const isOnline = useOnlineStatus();

  const { data: approvalCounts } = useQuery({
    queryKey: ['admin-approval-counts'],
    queryFn: () => apiClient.get<{ pending: number }>('/admin/approvals/counts'),
    refetchInterval: 30000,
  });
  const countsData = approvalCounts as unknown as { byStatus?: Record<string, number>; total?: number } | undefined;
  const pendingCount = countsData?.byStatus?.pending ?? 0;

  const isActive = (href: string) =>
    location.pathname === href || location.pathname.startsWith(href + '/');

  const displayName = user?.name || [user?.firstName, user?.lastName].filter(Boolean).join(' ') || user?.email || 'Admin';
  const isSuperAdmin = user?.roles?.includes('super_admin');

  return (
    <div className="flex h-screen bg-background">
      {/* Sidebar - Desktop */}
      <aside className="hidden w-64 flex-shrink-0 bg-sidebar lg:block">
        <div className="flex h-full flex-col">
          {/* Logo */}
          <div className="flex h-16 items-center gap-2 border-b border-sidebar-border px-6">
            <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-sidebar-primary">
              <Shield className="h-5 w-5 text-primary-foreground" />
            </div>
            <div>
              <span className="text-lg font-bold text-sidebar-foreground">Fe3dr</span>
              <p className="text-xs text-sidebar-foreground/60">Admin Portal</p>
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
                      ? 'bg-sidebar-primary text-primary-foreground'
                      : 'text-sidebar-foreground/70 hover:bg-sidebar-accent hover:text-sidebar-foreground'
                  }`}
                >
                  <Icon className="h-5 w-5" />
                  {item.name}
                  {item.name === 'Reviews' && pendingCount > 0 && (
                    <span className="ml-auto flex h-5 min-w-5 items-center justify-center rounded-full bg-destructive px-1.5 text-xs font-bold text-destructive-foreground">
                      {pendingCount > 99 ? '99+' : pendingCount}
                    </span>
                  )}
                </Link>
              );
            })}
          </nav>

          {/* User section */}
          <div className="border-t border-sidebar-border p-4">
            <button
              onClick={() => setUserMenuOpen(!userMenuOpen)}
              className="flex w-full items-center gap-3 rounded-lg p-2 hover:bg-sidebar-accent"
            >
              <div className="flex h-10 w-10 items-center justify-center rounded-full bg-sidebar-primary/20">
                <Shield className="h-5 w-5 text-sidebar-primary" />
              </div>
              <div className="flex-1 text-left">
                <p className="text-sm font-medium text-sidebar-foreground truncate">
                  {displayName}
                </p>
                <p className="text-xs text-sidebar-foreground/60">
                  {isSuperAdmin ? 'Super Admin' : 'Admin'}
                </p>
              </div>
              <ChevronDown className="h-4 w-4 text-sidebar-foreground/60" />
            </button>

            {userMenuOpen && (
              <div className="mt-2 space-y-1">
                <Link
                  to="/settings"
                  className="flex items-center gap-2 rounded-lg px-3 py-2 text-sm text-sidebar-foreground/70 hover:bg-sidebar-accent hover:text-sidebar-foreground"
                >
                  <Settings className="h-4 w-4" />
                  Settings
                </Link>
                <button
                  onClick={logout}
                  className="flex w-full items-center gap-2 rounded-lg px-3 py-2 text-sm text-red-400 hover:bg-red-500/10"
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
            className="fixed inset-0 z-40 bg-foreground/50 lg:hidden"
            onClick={() => setSidebarOpen(false)}
          />
          <aside className="fixed inset-y-0 left-0 z-50 w-64 bg-sidebar lg:hidden">
            <div className="flex h-full flex-col">
              <div className="flex h-16 items-center justify-between border-b border-sidebar-border px-4">
                <Link to="/dashboard" className="flex items-center gap-2">
                  <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-sidebar-primary">
                    <Shield className="h-5 w-5 text-primary-foreground" />
                  </div>
                  <span className="text-lg font-bold text-sidebar-foreground">Fe3dr</span>
                </Link>
                <button onClick={() => setSidebarOpen(false)}>
                  <X className="h-6 w-6 text-sidebar-foreground/60" />
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
                          ? 'bg-sidebar-primary text-primary-foreground'
                          : 'text-sidebar-foreground/70 hover:bg-sidebar-accent'
                      }`}
                    >
                      <Icon className="h-5 w-5" />
                      {item.name}
                      {item.name === 'Reviews' && pendingCount > 0 && (
                        <span className="ml-auto flex h-5 min-w-5 items-center justify-center rounded-full bg-destructive px-1.5 text-xs font-bold text-destructive-foreground">
                          {pendingCount > 99 ? '99+' : pendingCount}
                        </span>
                      )}
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
            onClick={() => setSidebarOpen(true)}
            className="rounded-lg p-2 hover:bg-secondary lg:hidden"
          >
            <Menu className="h-5 w-5" />
          </button>

          <div className="hidden lg:block">
            <h1 className="text-sm font-medium text-muted-foreground">
              Fe3dr Administration
            </h1>
          </div>

          <div className="flex items-center gap-2 ml-auto">
            <button className="relative rounded-lg p-2 hover:bg-secondary">
              <Bell className="h-5 w-5 text-muted-foreground" />
            </button>

            {/* Desktop user info */}
            <div className="hidden items-center gap-2 lg:flex">
              <div className="flex h-8 w-8 items-center justify-center rounded-full bg-primary/10">
                <Shield className="h-4 w-4 text-primary" />
              </div>
              <span className="text-sm font-medium text-foreground">{displayName}</span>
            </div>
          </div>
        </header>

        {/* Page content */}
        <main className={`flex-1 overflow-y-auto p-4 lg:p-8 ${isMobile ? 'pb-20' : ''}`}>
          <ErrorBoundary>
            <Outlet />
          </ErrorBoundary>
        </main>
      </div>

      {/* Offline Banner */}
      {!isOnline && (
        <div className="fixed inset-x-0 top-0 z-50 bg-warning px-4 py-2 text-center text-sm font-medium text-warning-foreground">
          You're offline. Data may not be up to date.
        </div>
      )}
    </div>
  );
}
