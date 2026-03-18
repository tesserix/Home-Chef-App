import { Outlet, Link, useLocation, useNavigate } from 'react-router-dom';
import {
  Home,
  Search,
  ShoppingCart,
  User,
  Menu,
  X,
  ChefHat,
  Heart,
  LogOut,
  Settings,
  Package,
  Utensils,
} from 'lucide-react';
import { useState } from 'react';
import { useAuth } from '@/app/providers/AuthProvider';
import { useCartStore } from '@/app/store/cart-store';
import { MobileBottomNav, MobileBottomNavSpacer } from '@/shared/components/navigation';
import { Logo } from '@/shared/components/brand';
import { CurrencySelector } from '@/shared/components/CurrencySelector';
import { useIsMobile, useOnlineStatus } from '@/shared/hooks/useMobile';

export function MainLayout() {
  const location = useLocation();
  const navigate = useNavigate();
  const { user, isAuthenticated, logout } = useAuth();
  const cartItemCount = useCartStore((state) => state.getItemCount());
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);
  const [userMenuOpen, setUserMenuOpen] = useState(false);
  const isMobile = useIsMobile();
  const isOnline = useOnlineStatus();

  const navigation = [
    { name: 'Home', href: '/', icon: Home },
    { name: 'Browse Chefs', href: '/chefs', icon: ChefHat },
    { name: 'Favorites', href: '/favorites', icon: Heart },
    { name: 'Catering', href: '/catering', icon: Utensils },
  ];

  const isActive = (href: string) => {
    if (href === '/') return location.pathname === '/';
    return location.pathname.startsWith(href);
  };

  return (
    <div className="min-h-screen bg-gray-50">
      {/* Offline Banner */}
      {!isOnline && (
        <div className="fixed inset-x-0 top-0 z-50 bg-amber-500 px-4 py-2 text-center text-sm font-medium text-white safe-top">
          You're offline. Some features may be unavailable.
        </div>
      )}

      {/* Header */}
      <header className={`sticky top-0 z-40 border-b border-gray-200 bg-white/95 backdrop-blur ${!isOnline ? 'mt-10' : ''}`}>
        <nav className="container-app">
          <div className="flex h-16 items-center justify-between">
            {/* Logo */}
            <Logo size="sm" />

            {/* Desktop Navigation */}
            <div className="hidden items-center gap-1 md:flex">
              {navigation.map((item) => {
                const Icon = item.icon;
                return (
                  <Link
                    key={item.name}
                    to={item.href}
                    className={`flex items-center gap-2 rounded-lg px-3 py-2 text-sm font-medium transition-colors ${
                      isActive(item.href)
                        ? 'bg-brand-50 text-brand-600'
                        : 'text-gray-600 hover:bg-gray-100 hover:text-gray-900'
                    }`}
                  >
                    <Icon className="h-4 w-4" />
                    {item.name}
                  </Link>
                );
              })}
            </div>

            {/* Right side */}
            <div className="flex items-center gap-2">
              {/* Currency selector */}
              <CurrencySelector />

              {/* Search button */}
              <button
                onClick={() => navigate('/chefs')}
                className="btn-ghost hidden p-2 md:flex"
              >
                <Search className="h-5 w-5" />
              </button>

              {/* Cart */}
              <Link
                to="/cart"
                className="btn-ghost relative p-2"
              >
                <ShoppingCart className="h-5 w-5" />
                {cartItemCount > 0 && (
                  <span className="absolute -right-1 -top-1 flex h-5 w-5 items-center justify-center rounded-full bg-brand-500 text-xs font-medium text-white">
                    {cartItemCount}
                  </span>
                )}
              </Link>

              {/* User menu */}
              {isAuthenticated ? (
                <div className="relative">
                  <button
                    onClick={() => setUserMenuOpen(!userMenuOpen)}
                    className="flex items-center gap-2 rounded-lg p-1.5 hover:bg-gray-100"
                  >
                    {user?.avatar ? (
                      <img
                        src={user.avatar}
                        alt={user.firstName}
                        className="h-8 w-8 rounded-full object-cover"
                        draggable={false}
                        onContextMenu={(e) => e.preventDefault()}
                      />
                    ) : (
                      <div className="flex h-8 w-8 items-center justify-center rounded-full bg-brand-100 text-brand-600">
                        <User className="h-4 w-4" />
                      </div>
                    )}
                  </button>

                  {/* Dropdown */}
                  {userMenuOpen && (
                    <>
                      <div
                        className="fixed inset-0 z-40"
                        onClick={() => setUserMenuOpen(false)}
                      />
                      <div className="absolute right-0 z-50 mt-2 w-56 rounded-xl border border-gray-200 bg-white py-2 shadow-lg">
                        <div className="border-b border-gray-100 px-4 py-2">
                          <p className="font-medium text-gray-900">
                            {user?.firstName} {user?.lastName}
                          </p>
                          <p className="text-sm text-gray-500">{user?.email}</p>
                        </div>
                        <div className="py-1">
                          <Link
                            to="/profile"
                            onClick={() => setUserMenuOpen(false)}
                            className="flex items-center gap-3 px-4 py-2 text-sm text-gray-700 hover:bg-gray-50"
                          >
                            <User className="h-4 w-4" />
                            Profile
                          </Link>
                          <Link
                            to="/favorites"
                            onClick={() => setUserMenuOpen(false)}
                            className="flex items-center gap-3 px-4 py-2 text-sm text-gray-700 hover:bg-gray-50"
                          >
                            <Heart className="h-4 w-4" />
                            Favorites
                          </Link>
                          <Link
                            to="/orders"
                            onClick={() => setUserMenuOpen(false)}
                            className="flex items-center gap-3 px-4 py-2 text-sm text-gray-700 hover:bg-gray-50"
                          >
                            <Package className="h-4 w-4" />
                            My Orders
                          </Link>
                          <Link
                            to="/profile?tab=security"
                            onClick={() => setUserMenuOpen(false)}
                            className="flex items-center gap-3 px-4 py-2 text-sm text-gray-700 hover:bg-gray-50"
                          >
                            <Settings className="h-4 w-4" />
                            Settings
                          </Link>
                        </div>
                        <div className="border-t border-gray-100 py-1">
                          <button
                            onClick={() => {
                              setUserMenuOpen(false);
                              logout();
                            }}
                            className="flex w-full items-center gap-3 px-4 py-2 text-sm text-red-600 hover:bg-red-50"
                          >
                            <LogOut className="h-4 w-4" />
                            Logout
                          </button>
                        </div>
                      </div>
                    </>
                  )}
                </div>
              ) : (
                <div className="flex items-center gap-2">
                  <Link to="/login" className="btn-ghost">
                    Login
                  </Link>
                  <Link to="/register" className="btn-primary">
                    Sign Up
                  </Link>
                </div>
              )}

              {/* Mobile menu button */}
              <button
                onClick={() => setMobileMenuOpen(!mobileMenuOpen)}
                className="btn-ghost p-2 md:hidden"
              >
                {mobileMenuOpen ? (
                  <X className="h-5 w-5" />
                ) : (
                  <Menu className="h-5 w-5" />
                )}
              </button>
            </div>
          </div>
        </nav>

        {/* Mobile menu */}
        {mobileMenuOpen && (
          <div className="border-t border-gray-200 bg-white md:hidden">
            <div className="container-app py-4">
              <div className="flex flex-col gap-1">
                {navigation.map((item) => {
                  const Icon = item.icon;
                  return (
                    <Link
                      key={item.name}
                      to={item.href}
                      onClick={() => setMobileMenuOpen(false)}
                      className={`flex items-center gap-3 rounded-lg px-4 py-3 text-sm font-medium ${
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
              </div>
            </div>
          </div>
        )}
      </header>

      {/* Main content */}
      <main>
        <Outlet />
      </main>

      {/* Mobile Bottom Navigation */}
      {isMobile && <MobileBottomNav />}

      {/* Footer - hidden on mobile when bottom nav is shown */}
      <footer className="border-t border-gray-200 bg-white hidden md:block">
        <div className="container-app py-12">
          <div className="grid gap-8 md:grid-cols-4">
            {/* Brand */}
            <div className="md:col-span-1">
              <Logo showTagline />
              <p className="mt-4 text-sm text-gray-500">
                Connecting you with home chefs for authentic, homemade food delivered to your doorstep.
              </p>
            </div>

            {/* Links */}
            <div>
              <h3 className="font-semibold text-gray-900">For Customers</h3>
              <ul className="mt-4 space-y-2 text-sm">
                <li>
                  <Link to="/chefs" className="text-gray-500 hover:text-gray-900">
                    Browse Chefs
                  </Link>
                </li>
                <li>
                  <Link to="/catering" className="text-gray-500 hover:text-gray-900">
                    Catering
                  </Link>
                </li>
                <li>
                  <Link to="/feed" className="text-gray-500 hover:text-gray-900">
                    Food Feed
                  </Link>
                </li>
              </ul>
            </div>

            <div>
              <h3 className="font-semibold text-gray-900">For Chefs</h3>
              <ul className="mt-4 space-y-2 text-sm">
                <li>
                  <Link to="/become-chef" className="text-gray-500 hover:text-gray-900">
                    Become a Chef
                  </Link>
                </li>
                <li>
                  <Link to="/chef-resources" className="text-gray-500 hover:text-gray-900">
                    Resources
                  </Link>
                </li>
              </ul>
            </div>

            <div>
              <h3 className="font-semibold text-gray-900">Company</h3>
              <ul className="mt-4 space-y-2 text-sm">
                <li>
                  <Link to="/about" className="text-gray-500 hover:text-gray-900">
                    About Us
                  </Link>
                </li>
                <li>
                  <Link to="/help" className="text-gray-500 hover:text-gray-900">
                    Help Center
                  </Link>
                </li>
                <li>
                  <Link to="/privacy" className="text-gray-500 hover:text-gray-900">
                    Privacy Policy
                  </Link>
                </li>
                <li>
                  <Link to="/terms" className="text-gray-500 hover:text-gray-900">
                    Terms of Service
                  </Link>
                </li>
              </ul>
            </div>
          </div>

          <div className="mt-12 border-t border-gray-200 pt-8 text-center text-sm text-gray-500">
            <p>&copy; {new Date().getFullYear()} Fe3dr. All rights reserved.</p>
          </div>
        </div>
      </footer>

      {/* Spacer for mobile bottom navigation */}
      {isMobile && <MobileBottomNavSpacer />}
    </div>
  );
}
