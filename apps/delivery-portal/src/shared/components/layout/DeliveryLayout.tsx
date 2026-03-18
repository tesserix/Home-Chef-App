import { Outlet, Link, useLocation } from 'react-router-dom';
import {
  LayoutDashboard,
  Navigation,
  Package,
  History,
  DollarSign,
  User,
  UserCog,
  Users,
  Settings,
  Menu,
  X,
  LogOut,
  Bell,
  ChevronDown,
  Truck,
  MapPin,
} from 'lucide-react';
import { useState, useMemo } from 'react';
import { useQuery } from '@tanstack/react-query';
import { useAuth } from '@/app/providers/AuthProvider';
import { apiClient } from '@/shared/services/api-client';
import { useIsMobile, useOnlineStatus } from '@/shared/hooks/useMobile';
import { ErrorBoundary } from '@/shared/components/ErrorBoundary';
import { DeliveryBottomNav } from '@/shared/components/navigation/DeliveryBottomNav';

interface StaffProfile {
  id: string;
  staffRole: string;
  permissions: string[];
  isActive: boolean;
}

// Navigation for delivery partners (drivers)
const partnerNavigation = [
  { name: 'Dashboard', href: '/dashboard', icon: LayoutDashboard },
  { name: 'Active Delivery', href: '/active', icon: Navigation },
  { name: 'Available', href: '/available', icon: Package },
  { name: 'History', href: '/history', icon: History },
  { name: 'Earnings', href: '/earnings', icon: DollarSign },
  { name: 'Profile', href: '/profile', icon: User },
  { name: 'Settings', href: '/settings', icon: Settings },
];

// Navigation for staff members (fleet managers, delivery ops, super admins)
const staffNavigation = [
  { name: 'Dashboard', href: '/dashboard', icon: LayoutDashboard },
  { name: 'Fleet', href: '/fleet', icon: Truck },
  { name: 'Partners', href: '/fleet/partners', icon: Users },
  { name: 'Staff', href: '/staff', icon: UserCog },
  { name: 'Zones', href: '/fleet/zones', icon: MapPin },
  { name: 'Settings', href: '/settings', icon: Settings },
];

export function DeliveryLayout() {
  const location = useLocation();
  const { user, logout } = useAuth();
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const [userMenuOpen, setUserMenuOpen] = useState(false);
  const isMobile = useIsMobile('lg');
  const isOnline = useOnlineStatus();

  // Fetch staff profile to determine if user is a staff member
  const { data: staffProfile } = useQuery({
    queryKey: ['delivery-staff-me'],
    queryFn: () => apiClient.get<StaffProfile>('/delivery/staff/me'),
    retry: false,
    staleTime: 5 * 60 * 1000,
  });

  const isStaff = !!staffProfile?.id;
  const staffPermissions = staffProfile?.permissions ?? [];

  const navigation = useMemo(() => {
    if (!isStaff) return partnerNavigation;

    // Staff get a completely separate nav — filter by permissions
    return staffNavigation.filter((item) => {
      if (item.href === '/dashboard' || item.href === '/settings') return true;
      if (item.href === '/fleet' || item.href === '/fleet/partners') {
        return staffPermissions.includes('fleet:view');
      }
      if (item.href === '/staff') {
        return staffPermissions.includes('staff:view');
      }
      if (item.href === '/fleet/zones') {
        return staffPermissions.includes('zones:view');
      }
      return true;
    });
  }, [isStaff, staffPermissions]);

  const isActive = (href: string) =>
    location.pathname === href || location.pathname.startsWith(href + '/');

  const displayName = user?.name || [user?.firstName, user?.lastName].filter(Boolean).join(' ') || user?.email || 'Driver';
  const roleLabel = isStaff
    ? (staffProfile?.staffRole ?? '').split('_').map((w: string) => w.charAt(0).toUpperCase() + w.slice(1)).join(' ')
    : 'Delivery Partner';

  return (
    <div className="flex h-screen bg-background">
      {/* Sidebar - Desktop */}
      <aside className="hidden w-64 flex-shrink-0 border-r border-border bg-card lg:block">
        <div className="flex h-full flex-col">
          <div className="flex h-16 items-center gap-2 border-b border-border px-6">
            <div className="logo-icon">
              <span><Truck className="h-5 w-5" /></span>
            </div>
            <div>
              <span className="logo-text">Fe3dr</span>
              <p className="logo-tagline">Delivery</p>
            </div>
          </div>

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
                </Link>
              );
            })}
          </nav>

          <div className="border-t border-border p-4">
            <button
              onClick={() => setUserMenuOpen(!userMenuOpen)}
              className="flex w-full items-center gap-3 rounded-lg p-2 hover:bg-secondary"
            >
              <div className="flex h-10 w-10 items-center justify-center rounded-full bg-primary/10">
                <User className="h-5 w-5 text-primary" />
              </div>
              <div className="flex-1 text-left">
                <p className="text-sm font-medium text-foreground truncate">{displayName}</p>
                <p className="text-xs text-muted-foreground">{roleLabel}</p>
              </div>
              <ChevronDown className="h-4 w-4 text-muted-foreground" />
            </button>

            {userMenuOpen && (
              <div className="mt-2 space-y-1">
                <Link
                  to="/settings"
                  className="flex items-center gap-2 rounded-lg px-3 py-2 text-sm text-muted-foreground hover:bg-secondary hover:text-foreground"
                >
                  <Settings className="h-4 w-4" />
                  Settings
                </Link>
                <button
                  onClick={logout}
                  className="flex w-full items-center gap-2 rounded-lg px-3 py-2 text-sm text-destructive hover:bg-destructive/10"
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
          <div className="fixed inset-0 z-40 bg-foreground/50 lg:hidden" onClick={() => setSidebarOpen(false)} />
          <aside className="fixed inset-y-0 left-0 z-50 w-64 bg-card lg:hidden">
            <div className="flex h-full flex-col">
              <div className="flex h-16 items-center justify-between border-b border-border px-4">
                <Link to="/dashboard" className="logo">
                  <div className="logo-icon"><span><Truck className="h-5 w-5" /></span></div>
                  <span className="logo-text">Fe3dr</span>
                </Link>
                <button onClick={() => setSidebarOpen(false)}>
                  <X className="h-6 w-6 text-muted-foreground" />
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
        <header className="flex h-16 items-center justify-between border-b border-border bg-card px-4 lg:px-8">
          <button onClick={() => setSidebarOpen(true)} className="rounded-lg p-2 hover:bg-secondary lg:hidden">
            <Menu className="h-5 w-5" />
          </button>
          <div className="ml-auto flex items-center gap-2">
            {!isStaff && (
              <Link to="/available" className="relative rounded-lg p-2 hover:bg-secondary">
                <Bell className="h-5 w-5 text-muted-foreground" />
              </Link>
            )}
          </div>
        </header>

        <main className={`flex-1 overflow-y-auto p-4 lg:p-8 ${isMobile && !isStaff ? 'pb-20' : ''}`}>
          <ErrorBoundary>
            <Outlet />
          </ErrorBoundary>
        </main>
      </div>

      {/* Bottom nav only for delivery partners on mobile */}
      {isMobile && !isStaff && <DeliveryBottomNav />}

      {!isOnline && (
        <div className="fixed inset-x-0 top-0 z-50 bg-warning px-4 py-2 text-center text-sm font-medium text-warning-foreground safe-top">
          You're offline. Updates will sync when connected.
        </div>
      )}
    </div>
  );
}
