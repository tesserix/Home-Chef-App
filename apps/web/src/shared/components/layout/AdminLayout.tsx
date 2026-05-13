import { Outlet, Link, useLocation } from 'react-router-dom';
import {
  LayoutDashboard,
  Users,
  ChefHat,
  ClipboardList,
  BarChart3,
  Settings,
  Menu,
  X,
  Bell,
  LogOut,
  Search,
  Truck,
  Shield,
  MessageSquare,
} from 'lucide-react';
import { useState } from 'react';
import { useAuth } from '@/app/providers/AuthProvider';

const navigation = [
  { name: 'Dashboard', href: '/admin/dashboard', icon: LayoutDashboard },
  { name: 'Users', href: '/admin/users', icon: Users },
  { name: 'Chefs', href: '/admin/chefs', icon: ChefHat },
  { name: 'Orders', href: '/admin/orders', icon: ClipboardList },
  { name: 'Delivery', href: '/admin/delivery', icon: Truck },
  { name: 'Moderation', href: '/admin/moderation', icon: Shield },
  { name: 'Support', href: '/admin/support', icon: MessageSquare },
  { name: 'Analytics', href: '/admin/analytics', icon: BarChart3 },
  { name: 'Settings', href: '/admin/settings', icon: Settings },
];

export function AdminLayout() {
  const location = useLocation();
  const { user, logout } = useAuth();
  const [sidebarOpen, setSidebarOpen] = useState(false);

  const isActive = (href: string) => location.pathname === href;

  return (
    <div className="flex h-screen bg-mist">
      {/* Sidebar - Desktop */}
      <aside className="hidden w-64 flex-shrink-0 bg-ink lg:block">
        <div className="flex h-full flex-col">
          {/* Logo */}
          <div className="flex h-16 items-center gap-2 px-6">
            <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-herb">
              <span className="text-lg font-medium text-paper">H</span>
            </div>
            <div>
              <span className="text-lg font-medium text-paper">Fe3dr</span>
              <p className="text-xs text-ink-muted">Admin Panel</p>
            </div>
          </div>

          {/* Navigation */}
          <nav className="flex-1 space-y-1 px-3 py-4">
            {navigation.map((item) => {
              const Icon = item.icon;
              return (
                <Link
                  key={item.name}
                  to={item.href}
                  className={`flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium transition-colors ${
                    isActive(item.href)
                      ? 'bg-ink text-paper'
                      : 'text-ink-muted hover:bg-ink hover:text-paper'
                  }`}
                >
                  <Icon className="h-5 w-5" />
                  {item.name}
                </Link>
              );
            })}
          </nav>

          {/* User section */}
          <div className="border-t border-ink p-4">
            <div className="flex items-center gap-3">
              <div className="flex h-10 w-10 items-center justify-center rounded-full bg-ink-soft">
                <span className="text-sm font-medium text-paper">
                  {user?.firstName?.[0]}
                  {user?.lastName?.[0]}
                </span>
              </div>
              <div className="flex-1">
                <p className="text-sm font-medium text-paper">
                  {user?.firstName} {user?.lastName}
                </p>
                <p className="text-xs text-ink-muted capitalize">{user?.roles?.[0]}</p>
              </div>
              <button
                type="button"
                aria-label="Log out"
                onClick={logout}
                className="rounded-lg p-2 text-ink-muted transition-colors hover:bg-ink hover:text-paper focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-herb focus-visible:ring-offset-2"
              >
                <LogOut aria-hidden="true" className="h-5 w-5" />
              </button>
            </div>
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
            id="admin-mobile-sidebar"
            aria-label="Main navigation"
            className="fixed inset-y-0 left-0 z-50 w-64 bg-ink lg:hidden"
          >
            <div className="flex h-full flex-col">
              <div className="flex h-16 items-center justify-between px-4">
                <span className="text-lg font-medium text-paper">Fe3dr Admin</span>
                <button
                  type="button"
                  aria-label="Close navigation"
                  onClick={() => setSidebarOpen(false)}
                  className="rounded-lg p-1 transition-colors hover:bg-ink-soft focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-herb focus-visible:ring-offset-2"
                >
                  <X aria-hidden="true" className="h-6 w-6 text-ink-muted" />
                </button>
              </div>
              <nav className="flex-1 space-y-1 px-3 py-4">
                {navigation.map((item) => {
                  const Icon = item.icon;
                  return (
                    <Link
                      key={item.name}
                      to={item.href}
                      onClick={() => setSidebarOpen(false)}
                      className={`flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium ${
                        isActive(item.href)
                          ? 'bg-ink text-paper'
                          : 'text-ink-muted hover:bg-ink'
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
        <header className="flex h-16 items-center justify-between border-b border-mist bg-bone px-4 shadow-1 lg:px-8">
          <div className="flex items-center gap-4">
            <button
              type="button"
              aria-label="Open navigation"
              aria-expanded={sidebarOpen}
              aria-controls="admin-mobile-sidebar"
              onClick={() => setSidebarOpen(true)}
              className="rounded-lg p-2 transition-colors hover:bg-mist focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-herb focus-visible:ring-offset-2 lg:hidden"
            >
              <Menu aria-hidden="true" className="h-5 w-5" />
            </button>

            {/* Search */}
            <div className="hidden items-center gap-2 rounded-lg border border-mist-strong bg-paper px-3 py-2 md:flex">
              <Search aria-hidden="true" className="h-4 w-4 text-ink-muted" />
              <label htmlFor="admin-global-search" className="sr-only">Search admin</label>
              <input
                id="admin-global-search"
                type="search"
                placeholder="Search..."
                className="w-64 bg-transparent text-sm outline-none placeholder:text-ink-muted"
              />
            </div>
          </div>

          <div className="flex items-center gap-4">
            {/* Notifications */}
            <button
              type="button"
              aria-label="Notifications, 3 unread"
              className="relative rounded-lg p-2 transition-colors hover:bg-mist focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-herb focus-visible:ring-offset-2"
            >
              <Bell aria-hidden="true" className="h-5 w-5 text-ink-soft" />
              <span aria-hidden="true" className="absolute right-1 top-1 flex h-4 w-4 items-center justify-center rounded-full bg-paprika text-xs text-paper">
                3
              </span>
            </button>
          </div>
        </header>

        {/* Page content */}
        <main id="main" className="flex-1 overflow-y-auto p-4 lg:p-8">
          <Outlet />
        </main>
      </div>
    </div>
  );
}
