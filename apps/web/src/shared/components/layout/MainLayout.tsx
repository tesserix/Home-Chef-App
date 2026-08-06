import { Outlet, Link, useLocation, useNavigate } from 'react-router';
import { Search, ShoppingCart, User, Menu, Store, ExternalLink } from 'lucide-react';
import { useEffect, useRef, useState } from 'react';
import { AnimatePresence, motion } from 'framer-motion';
import { useAuth } from '@/app/providers/AuthProvider';
import { ThemeToggleCompact } from '@/shared/theme';
import { useCartStore } from '@/app/store/cart-store';
import { MobileBottomNav, MobileBottomNavSpacer } from '@/shared/components/navigation';
import { Logo } from '@/shared/components/brand';
import { CurrencySelector } from '@/shared/components/CurrencySelector';
import { Button } from '@/shared/components/ui';
import { useIsMobile, useOnlineStatus } from '@/shared/hooks/useMobile';
import { CookieBanner } from '../cookie-banner/CookieBanner';
import { CATERING_ENABLED } from '@/shared/config/features';
import { VENDOR_CTA_LABEL, VENDOR_PORTAL_URL } from '@/shared/config/partner-sites';
import { AppSidebar } from './AppSidebar';
import { AccountDrawer } from './AccountDrawer';

export function MainLayout() {
  const location = useLocation();
  const navigate = useNavigate();
  const { user, isAuthenticated } = useAuth();
  const cartItemCount = useCartStore((state) => state.getItemCount());
  const [drawerOpen, setDrawerOpen] = useState(false);
  const isMobile = useIsMobile();
  const isOnline = useOnlineStatus();
  const menuButtonRef = useRef<HTMLButtonElement>(null);

  // Close the drawer on route change so navigating from it doesn't leave it
  // hanging open over the page you just landed on.
  useEffect(() => {
    setDrawerOpen(false);
  }, [location.pathname]);

  return (
    <div className="min-h-screen bg-paper">
      {/* Offline Banner */}
      {!isOnline && (
        <div className="fixed inset-x-0 top-0 z-50 bg-amber px-4 py-2 text-center text-sm font-medium text-paper safe-top">
          You're offline. Some features may be unavailable.
        </div>
      )}

      {/* Header — full-bleed rather than centred, so it lines up with the rail
          below it the way the mobile app's header lines up with its tab bar. */}
      <header className={`sticky top-0 z-40 border-b border-mist bg-bone ${!isOnline ? 'mt-10' : ''}`}>
        <div className="flex h-16 items-center gap-2 px-4 lg:px-6">
          <Button
            ref={menuButtonRef}
            variant="ghost"
            size="icon"
            onClick={() => setDrawerOpen(true)}
            aria-label="Open menu"
            aria-expanded={drawerOpen}
            aria-haspopup="dialog"
          >
            <Menu aria-hidden="true" className="h-5 w-5" />
          </Button>

          <Logo size="sm" />

          <div className="flex-1" />

          <div className="flex items-center gap-1 sm:gap-2">
            {/* A chef arriving on the customer storefront had no way to reach
                their own side of the platform without scrolling to a marketing
                block or opening the account drawer. This is the standing signpost.
                Hidden on mobile, where the bottom nav owns the space — the drawer's
                PARTNER_NAV carries it there. */}
            <a
              href={VENDOR_PORTAL_URL}
              className="hidden items-center gap-1.5 rounded-lg px-3 py-2 text-sm font-medium text-ink-muted transition-colors hover:bg-mist hover:text-ink focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-herb/40 md:inline-flex"
            >
              <Store aria-hidden="true" className="h-4 w-4" />
              {VENDOR_CTA_LABEL}
              <ExternalLink aria-hidden="true" className="h-3 w-3 opacity-60" />
              <span className="sr-only">(opens the chef portal)</span>
            </a>

            <CurrencySelector />
            <ThemeToggleCompact />

            <Button
              variant="ghost"
              size="icon"
              onClick={() => navigate('/chefs')}
              aria-label="Search chefs"
              className="hidden md:flex"
            >
              <Search aria-hidden="true" className="h-5 w-5" />
            </Button>

            {/* Cart */}
            <Button asChild variant="ghost" size="icon" className="relative">
              <Link
                to="/cart"
                aria-label={cartItemCount > 0 ? `Cart, ${cartItemCount} items` : 'Cart'}
              >
                <ShoppingCart aria-hidden="true" className="h-5 w-5" />
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
            </Button>

            {/* Account. The avatar opens the same drawer as the hamburger —
                one account surface instead of a dropdown that duplicated it. */}
            {isAuthenticated ? (
              <button
                type="button"
                onClick={() => setDrawerOpen(true)}
                aria-label={`Account menu for ${user?.firstName ?? 'user'}`}
                aria-haspopup="dialog"
                aria-expanded={drawerOpen}
                className="flex items-center rounded-lg p-1.5 hover:bg-mist focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-herb/40"
              >
                {user?.avatar ? (
                  <img
                    src={user.avatar}
                    alt=""
                    className="h-8 w-8 rounded-full object-cover"
                    draggable={false}
                    onContextMenu={(e) => e.preventDefault()}
                    loading="lazy"
                    decoding="async"
                  />
                ) : (
                  <div
                    aria-hidden="true"
                    className="flex h-8 w-8 items-center justify-center rounded-full bg-herb-tint text-herb"
                  >
                    <User className="h-4 w-4" aria-hidden="true" />
                  </div>
                )}
              </button>
            ) : (
              <div className="flex items-center gap-2">
                <Button asChild variant="ghost" className="hidden sm:inline-flex">
                  <Link to="/login">Login</Link>
                </Button>
                <Button asChild variant="primary">
                  <Link to="/register">Sign Up</Link>
                </Button>
              </div>
            )}
          </div>
        </div>
      </header>

      <AccountDrawer
        open={drawerOpen}
        onClose={() => setDrawerOpen(false)}
        returnFocusRef={menuButtonRef}
      />

      <div className="flex">
        <AppSidebar />

        <div className="flex min-w-0 flex-1 flex-col">
          {/* Main content */}
          <main id="main" className="flex-1">
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
                    {/* Same v1 gate as the nav — the footer was still
                        advertising the deferred surface. */}
                    {CATERING_ENABLED && (
                      <li>
                        <Link to="/catering" className="text-ink-muted hover:text-ink">
                          Catering
                        </Link>
                      </li>
                    )}
                    <li>
                      <Link to="/bakery" className="text-ink-muted hover:text-ink">
                        Bakery
                      </Link>
                    </li>
                    <li>
                      <Link to="/chefbook" className="text-ink-muted hover:text-ink">
                        ChefBook
                      </Link>
                    </li>
                  </ul>
                </div>

                {/* Chef signup lives on the vendor portal, the same destination
                    PARTNER_NAV uses. The old /become-chef and /chef-resources
                    links pointed at routes this app has never had, so they fell
                    through the catch-all and silently returned people home. */}
                <div>
                  <h3 className="font-semibold text-ink">For Chefs</h3>
                  <ul className="mt-4 space-y-2 text-sm">
                    <li>
                      <a
                        href={VENDOR_PORTAL_URL}
                        className="text-ink-muted hover:text-ink"
                      >
                        {VENDOR_CTA_LABEL}
                      </a>
                    </li>
                  </ul>
                </div>

                {/* About Us and Help Center were dead links too. The legal
                    pages below are the only Company routes that exist. */}
                <div>
                  <h3 className="font-semibold text-ink">Company</h3>
                  <ul className="mt-4 space-y-2 text-sm">
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
                    <li>
                      <Link to="/refund" className="text-ink-muted hover:text-ink">
                        Refund Policy
                      </Link>
                    </li>
                  </ul>
                </div>
              </div>

              {/* Grievance Officer + Data Fiduciary disclosure */}
              <div className="mt-8 border-t border-mist pt-6 text-xs text-ink-muted">
                <div className="grid gap-6 sm:grid-cols-2">
                  <div>
                    <h3 className="text-sm font-medium text-ink-soft mb-1">Data Fiduciary</h3>
                    <address className="not-italic">
                      Tesserix Pty Ltd<br />
                      ACN 694 070 865 · ABN 59 694 070 865<br />
                      Registered in New South Wales, Australia<br />
                      Operations: Mumbai, India · Sydney, Australia
                    </address>
                    {/* Zivana is the entity that actually collects payment, so it
                        is named here rather than only inside the terms — the name a
                        customer sees on their bank statement should be findable. */}
                    <p className="mt-2">
                      Payments are collected and settled by <strong>Zivana Innovations LLP</strong>,
                      part of Tesserix Pty Ltd.
                    </p>
                  </div>
                  <div>
                    <h3 className="text-sm font-medium text-ink-soft mb-1">Grievance Officer</h3>
                    {/* Email only — no phone is published, because none is staffed. */}
                    <address className="not-italic">
                      Samyak Rout<br />
                      <a href="mailto:grievance@fe3dr.com" className="text-herb hover:underline">grievance@fe3dr.com</a><br />
                      Response within 15 days (DPDP Act §13)
                    </address>
                  </div>
                </div>
                <p className="mt-4">
                  For consumer disputes you can also contact the National Consumer Helpline at{' '}
                  <a href="tel:1915" className="text-herb hover:underline">1915</a> or visit{' '}
                  <a
                    href="https://consumerhelpline.gov.in"
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-herb hover:underline"
                  >
                    consumerhelpline.gov.in
                  </a>
                  .
                </p>
              </div>

              <div className="mt-8 border-t border-mist pt-8 text-center text-sm text-ink-muted">
                <p>&copy; {new Date().getFullYear()} Fe3dr, a product of Tesserix Pty Ltd. All rights reserved.</p>
              </div>
            </div>
          </footer>

          {/* Spacer for mobile bottom navigation */}
          {isMobile && <MobileBottomNavSpacer />}
        </div>
      </div>

      {/* Cookie consent (DPDP Act + ePrivacy best practice) */}
      <CookieBanner />
    </div>
  );
}
