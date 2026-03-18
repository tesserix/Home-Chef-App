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
    <div className="flex h-screen bg-gray-100">
      {/* Sidebar - Desktop */}
      <aside className="hidden w-64 flex-shrink-0 bg-gray-900 lg:block">
        <div className="flex h-full flex-col">
          {/* Logo */}
          <div className="flex h-16 items-center gap-2 px-6">
            <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-brand-500">
              <span className="text-lg font-bold text-white">H</span>
            </div>
            <div>
              <span className="text-lg font-bold text-white">Fe3dr</span>
              <p className="text-xs text-gray-400">Admin Panel</p>
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
                      ? 'bg-gray-800 text-white'
                      : 'text-gray-400 hover:bg-gray-800 hover:text-white'
                  }`}
                >
                  <Icon className="h-5 w-5" />
                  {item.name}
                </Link>
              );
            })}
          </nav>

          {/* User section */}
          <div className="border-t border-gray-800 p-4">
            <div className="flex items-center gap-3">
              <div className="flex h-10 w-10 items-center justify-center rounded-full bg-gray-700">
                <span className="text-sm font-medium text-white">
                  {user?.firstName?.[0]}
                  {user?.lastName?.[0]}
                </span>
              </div>
              <div className="flex-1">
                <p className="text-sm font-medium text-white">
                  {user?.firstName} {user?.lastName}
                </p>
                <p className="text-xs text-gray-400 capitalize">{user?.roles?.[0]}</p>
              </div>
              <button
                onClick={logout}
                className="rounded-lg p-2 text-gray-400 hover:bg-gray-800 hover:text-white"
              >
                <LogOut className="h-5 w-5" />
              </button>
            </div>
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
          <aside className="fixed inset-y-0 left-0 z-50 w-64 bg-gray-900 lg:hidden">
            <div className="flex h-full flex-col">
              <div className="flex h-16 items-center justify-between px-4">
                <span className="text-lg font-bold text-white">Fe3dr Admin</span>
                <button onClick={() => setSidebarOpen(false)}>
                  <X className="h-6 w-6 text-gray-400" />
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
                          ? 'bg-gray-800 text-white'
                          : 'text-gray-400 hover:bg-gray-800'
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
        <header className="flex h-16 items-center justify-between border-b border-gray-200 bg-white px-4 shadow-sm lg:px-8">
          <div className="flex items-center gap-4">
            <button
              onClick={() => setSidebarOpen(true)}
              className="rounded-lg p-2 hover:bg-gray-100 lg:hidden"
            >
              <Menu className="h-5 w-5" />
            </button>

            {/* Search */}
            <div className="hidden items-center gap-2 rounded-lg border border-gray-300 bg-gray-50 px-3 py-2 md:flex">
              <Search className="h-4 w-4 text-gray-400" />
              <input
                type="text"
                placeholder="Search..."
                className="w-64 bg-transparent text-sm outline-none placeholder:text-gray-400"
              />
            </div>
          </div>

          <div className="flex items-center gap-4">
            {/* Notifications */}
            <button className="relative rounded-lg p-2 hover:bg-gray-100">
              <Bell className="h-5 w-5 text-gray-600" />
              <span className="absolute right-1 top-1 flex h-4 w-4 items-center justify-center rounded-full bg-red-500 text-xs text-white">
                3
              </span>
            </button>
          </div>
        </header>

        {/* Page content */}
        <main className="flex-1 overflow-y-auto p-4 lg:p-8">
          <Outlet />
        </main>
      </div>
    </div>
  );
}
