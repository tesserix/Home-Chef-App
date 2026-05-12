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
import { AnimatePresence, motion } from 'framer-motion';
import { useAuth } from '@/app/providers/AuthProvider';
import { ThemeToggleCompact } from '@/shared/theme';
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
    <div className="min-h-screen bg-paper">
      {/* Offline Banner */}
      {!isOnline && (
        <div className="fixed inset-x-0 top-0 z-50 bg-amber px-4 py-2 text-center text-sm font-medium text-paper safe-top">
          You're offline. Some features may be unavailable.
        </div>
      )}

      {/* Header */}
      <header className={`sticky top-0 z-40 border-b border-mist bg-bone ${!isOnline ? 'mt-10' : ''}`}>
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
                        ? 'bg-herb-tint text-herb'
                        : 'text-ink-soft hover:bg-mist hover:text-ink'
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

              {/* Theme toggle */}
              <ThemeToggleCompact />

              {/* Search button */}
              <button
                onClick={() => navigate('/chefs')}
                aria-label="Search chefs"
                className="btn-ghost hidden p-2 md:flex"
              >
                <Search className="h-5 w-5" />
              </button>

              {/* Cart */}
              <Link
                to="/cart"
                aria-label={cartItemCount > 0 ? `Cart, ${cartItemCount} items` : 'Cart'}
                className="btn-ghost relative p-2"
              >
                <ShoppingCart className="h-5 w-5" />
                <AnimatePresence>
                  {cartItemCount > 0 && (
                    <motion.span
                      key={cartItemCount}
                      initial={{ scale: 0.6, opacity: 0 }}
                      animate={{ scale: 1, opacity: 1 }}
                      exit={{ scale: 0.6, opacity: 0 }}
                      transition={{ duration: 0.15, ease: [0.22, 1, 0.36, 1] }}
                      className="absolute -right-1 -top-1 flex h-5 min-w-[1.25rem] items-center justify-center rounded-full bg-herb px-1 text-xs font-medium tabular-nums text-paper"
                    >
                      {cartItemCount}
                    </motion.span>
                  )}
                </AnimatePresence>
              </Link>

              {/* User menu */}
              {isAuthenticated ? (
                <div className="relative">
                  <button
                    onClick={() => setUserMenuOpen(!userMenuOpen)}
                    className="flex items-center gap-2 rounded-lg p-1.5 hover:bg-mist"
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
                      <div className="flex h-8 w-8 items-center justify-center rounded-full bg-herb-tint text-herb">
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
                      <div className="absolute right-0 z-50 mt-2 w-56 rounded-xl border border-mist bg-bone py-2 shadow-lg">
                        <div className="border-b border-mist px-4 py-2">
                          <p className="font-medium text-ink">
                            {user?.firstName} {user?.lastName}
                          </p>
                          <p className="text-sm text-ink-muted">{user?.email}</p>
                        </div>
                        <div className="py-1">
                          <Link
                            to="/profile"
                            onClick={() => setUserMenuOpen(false)}
                            className="flex items-center gap-3 px-4 py-2 text-sm text-ink-soft hover:bg-paper"
                          >
                            <User className="h-4 w-4" />
                            Profile
                          </Link>
                          <Link
                            to="/favorites"
                            onClick={() => setUserMenuOpen(false)}
                            className="flex items-center gap-3 px-4 py-2 text-sm text-ink-soft hover:bg-paper"
                          >
                            <Heart className="h-4 w-4" />
                            Favorites
                          </Link>
                          <Link
                            to="/orders"
                            onClick={() => setUserMenuOpen(false)}
                            className="flex items-center gap-3 px-4 py-2 text-sm text-ink-soft hover:bg-paper"
                          >
                            <Package className="h-4 w-4" />
                            My Orders
                          </Link>
                          <Link
                            to="/profile?tab=security"
                            onClick={() => setUserMenuOpen(false)}
                            className="flex items-center gap-3 px-4 py-2 text-sm text-ink-soft hover:bg-paper"
                          >
                            <Settings className="h-4 w-4" />
                            Settings
                          </Link>
                        </div>
                        <div className="border-t border-mist py-1">
                          <button
                            onClick={() => {
                              setUserMenuOpen(false);
                              logout();
                            }}
                            className="flex w-full items-center gap-3 px-4 py-2 text-sm text-paprika hover:bg-paprika-tint"
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
          <div className="border-t border-mist bg-bone md:hidden">
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
                          ? 'bg-herb-tint text-herb'
                          : 'text-ink-soft hover:bg-paper'
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
      <main id="main">
        <Outlet />
      </main>

      {/* Mobile Bottom Navigation */}
      {isMobile && <MobileBottomNav />}

      {/* Footer - hidden on mobile when bottom nav is shown */}
      <footer className="border-t border-mist bg-bone hidden md:block">
        <div className="container-app py-12">
          <div className="grid gap-8 md:grid-cols-4">
            {/* Brand */}
            <div className="md:col-span-1">
              <Logo showTagline />
              <p className="mt-4 text-sm text-ink-muted">
                Connecting you with home chefs for authentic, homemade food delivered to your doorstep.
              </p>
            </div>

            {/* Links */}
            <div>
              <h3 className="font-semibold text-ink">For Customers</h3>
              <ul className="mt-4 space-y-2 text-sm">
                <li>
                  <Link to="/chefs" className="text-ink-muted hover:text-ink">
                    Browse Chefs
                  </Link>
                </li>
                <li>
                  <Link to="/catering" className="text-ink-muted hover:text-ink">
                    Catering
                  </Link>
                </li>
                <li>
                  <Link to="/feed" className="text-ink-muted hover:text-ink">
                    Food Feed
                  </Link>
                </li>
              </ul>
            </div>

            <div>
              <h3 className="font-semibold text-ink">For Chefs</h3>
              <ul className="mt-4 space-y-2 text-sm">
                <li>
                  <Link to="/become-chef" className="text-ink-muted hover:text-ink">
                    Become a Chef
                  </Link>
                </li>
                <li>
                  <Link to="/chef-resources" className="text-ink-muted hover:text-ink">
                    Resources
                  </Link>
                </li>
              </ul>
            </div>

            <div>
              <h3 className="font-semibold text-ink">Company</h3>
              <ul className="mt-4 space-y-2 text-sm">
                <li>
                  <Link to="/about" className="text-ink-muted hover:text-ink">
                    About Us
                  </Link>
                </li>
                <li>
                  <Link to="/help" className="text-ink-muted hover:text-ink">
                    Help Center
                  </Link>
                </li>
                <li>
                  <Link to="/privacy" className="text-ink-muted hover:text-ink">
                    Privacy Policy
                  </Link>
                </li>
                <li>
                  <Link to="/terms" className="text-ink-muted hover:text-ink">
                    Terms of Service
                  </Link>
                </li>
              </ul>
            </div>
          </div>

          <div className="mt-12 border-t border-mist pt-8 text-center text-sm text-ink-muted">
            <p>&copy; {new Date().getFullYear()} Fe3dr. All rights reserved.</p>
          </div>
        </div>
      </footer>

      {/* Spacer for mobile bottom navigation */}
      {isMobile && <MobileBottomNavSpacer />}
    </div>
  );
}
