import { useEffect, useRef } from 'react';
import { Link } from 'react-router';
import { AnimatePresence, motion } from 'framer-motion';
import { User, LogOut, X, ExternalLink } from 'lucide-react';
import { useAuth } from '@/app/providers/AuthProvider';
import { useLockBodyScroll } from '@/shared/hooks/useMobile';
import { Button } from '@/shared/components/ui';
import {
  ACCOUNT_NAV,
  ACCOUNT_SECONDARY_NAV,
  LEGAL_NAV,
  PARTNER_NAV,
  RAIL_NAV,
  type NavItem,
} from './nav-items';

// The account drawer behind the hamburger — the web counterpart of mobile's
// Profile tab, reachable from every page instead of being a destination you
// navigate away to.
//
// Below lg the fixed rail is hidden, so the drawer also carries the rail's
// browse destinations; otherwise closing the rail would strand them behind the
// five-item bottom nav.

interface AccountDrawerProps {
  open: boolean;
  onClose: () => void;
  /** Focus returns here on close, so keyboard users don't lose their place. */
  returnFocusRef?: React.RefObject<HTMLElement | null>;
}

const EASE = [0.22, 1, 0.36, 1] as const;

const ROW =
  'flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm text-ink-soft transition-colors hover:bg-mist hover:text-ink focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-herb/40';

function DrawerLink({ item, onClose }: { item: NavItem; onClose: () => void }) {
  const Icon = item.icon;
  const body = (
    <>
      <Icon aria-hidden="true" className="h-5 w-5 shrink-0" />
      <span className="truncate">{item.name}</span>
      {item.external && (
        <ExternalLink aria-hidden="true" className="ml-auto h-3.5 w-3.5 shrink-0 text-ink-muted" />
      )}
    </>
  );

  // Another origin (the vendor portal) can't go through the router, and it
  // opens in its own tab so the customer doesn't lose their place here.
  if (item.external) {
    return (
      <a href={item.href} target="_blank" rel="noopener noreferrer" onClick={onClose} className={ROW}>
        {body}
        <span className="sr-only">(opens in a new tab)</span>
      </a>
    );
  }

  return (
    <Link to={item.href} onClick={onClose} className={ROW}>
      {body}
    </Link>
  );
}

export function AccountDrawer({ open, onClose, returnFocusRef }: AccountDrawerProps) {
  const { user, isAuthenticated, logout } = useAuth();
  const panelRef = useRef<HTMLDivElement>(null);
  const closeButtonRef = useRef<HTMLButtonElement>(null);

  useLockBodyScroll(open);

  // Move focus into the panel on open and hand it back to the trigger on
  // close. Without the hand-back, closing the drawer drops focus onto <body>
  // and a keyboard user restarts from the top of the document.
  useEffect(() => {
    if (!open) return;
    const previous = document.activeElement as HTMLElement | null;
    closeButtonRef.current?.focus();
    return () => {
      const target = returnFocusRef?.current ?? previous;
      target?.focus?.();
    };
  }, [open, returnFocusRef]);

  // Escape closes; Tab is trapped inside the panel while it's open, which is
  // what makes this a dialog rather than a floating div a user can tab behind.
  useEffect(() => {
    if (!open) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        onClose();
        return;
      }
      if (e.key !== 'Tab') return;
      const focusables = panelRef.current?.querySelectorAll<HTMLElement>(
        'a[href], button:not([disabled]), input, select, textarea, [tabindex]:not([tabindex="-1"])',
      );
      if (!focusables || focusables.length === 0) return;
      const first = focusables[0]!;
      const last = focusables[focusables.length - 1]!;
      if (e.shiftKey && document.activeElement === first) {
        e.preventDefault();
        last.focus();
      } else if (!e.shiftKey && document.activeElement === last) {
        e.preventDefault();
        first.focus();
      }
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [open, onClose]);

  return (
    <AnimatePresence>
      {open && (
        <>
          <motion.div
            aria-hidden="true"
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            transition={{ duration: 0.15, ease: EASE }}
            onClick={onClose}
            className="fixed inset-0 z-50 bg-ink/40"
          />
          <motion.div
            ref={panelRef}
            role="dialog"
            aria-modal="true"
            aria-label="Account menu"
            initial={{ x: '-100%' }}
            animate={{ x: 0 }}
            exit={{ x: '-100%' }}
            transition={{ duration: 0.25, ease: EASE }}
            className="fixed inset-y-0 left-0 z-50 flex w-[19rem] max-w-[85vw] flex-col overflow-y-auto bg-bone shadow-3"
          >
            <div className="flex items-start justify-between gap-2 p-4">
              {isAuthenticated ? (
                <Link
                  to="/profile"
                  onClick={onClose}
                  className="flex min-w-0 items-center gap-3 rounded-lg p-1 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-herb/40"
                >
                  {user?.avatar ? (
                    <img
                      src={user.avatar}
                      alt=""
                      className="h-12 w-12 shrink-0 rounded-full object-cover"
                      draggable={false}
                      loading="lazy"
                      decoding="async"
                    />
                  ) : (
                    <div
                      aria-hidden="true"
                      className="flex h-12 w-12 shrink-0 items-center justify-center rounded-full bg-herb-tint text-herb"
                    >
                      <User className="h-6 w-6" />
                    </div>
                  )}
                  <div className="min-w-0">
                    <p className="truncate font-medium text-ink">
                      {user?.firstName} {user?.lastName}
                    </p>
                    <p className="text-sm text-herb">Manage account</p>
                  </div>
                </Link>
              ) : (
                <p className="p-1 font-medium text-ink">Welcome to Fe3dr</p>
              )}

              <Button
                ref={closeButtonRef}
                variant="ghost"
                size="icon"
                onClick={onClose}
                aria-label="Close menu"
              >
                <X aria-hidden="true" className="h-5 w-5" />
              </Button>
            </div>

            {isAuthenticated ? (
              <nav aria-label="Account" className="flex flex-col gap-1 px-3 pb-2">
                {ACCOUNT_NAV.map((item) => (
                  <DrawerLink key={item.href} item={item} onClose={onClose} />
                ))}
              </nav>
            ) : (
              // Styled as links rather than <Button asChild>: Slot-based
              // buttons drop their classes in this app, so anything built on
              // asChild renders as bare text. These keep link semantics
              // (middle-click, open in new tab) and are guaranteed to style.
              // Sign up leads, log in follows — a signed-out visitor is more
              // often new than returning, and it matches the pattern the
              // category's apps have trained people on.
              <div className="flex flex-col gap-2 px-4 pb-2">
                <Link
                  to="/register"
                  onClick={onClose}
                  className="flex w-full items-center justify-center rounded-lg bg-herb px-4 py-2.5 text-sm font-medium text-paper transition-colors hover:bg-herb-soft focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-herb/40"
                >
                  Sign up
                </Link>
                <Link
                  to="/login"
                  onClick={onClose}
                  className="flex w-full items-center justify-center rounded-lg bg-mist px-4 py-2.5 text-sm font-medium text-ink transition-colors hover:bg-mist/70 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-herb/40"
                >
                  Log in
                </Link>
              </div>
            )}

            {/* Below lg the rail is hidden, so browse destinations live here
                too — otherwise the hamburger would be the only chrome on screen
                and half the app would be unreachable from it. */}
            <div className="lg:hidden">
              <hr className="mx-3 my-2 border-mist" />
              <nav aria-label="Browse" className="flex flex-col gap-1 px-3 pb-2">
                {RAIL_NAV.map((item) => (
                  <DrawerLink key={`browse-${item.href}`} item={item} onClose={onClose} />
                ))}
              </nav>
            </div>

            {/* The other ways in. Signed-in customers don't need pitching at,
                so this only shows when signed out. */}
            {!isAuthenticated && (
              <>
                <hr className="mx-3 my-2 border-mist" />
                <nav aria-label="Partner with us" className="flex flex-col gap-1 px-3 pb-2">
                  {PARTNER_NAV.map((item) => (
                    <DrawerLink key={item.href} item={item} onClose={onClose} />
                  ))}
                </nav>
              </>
            )}

            <hr className="mx-3 my-2 border-mist" />
            <nav
              aria-label={isAuthenticated ? 'Settings' : 'Legal'}
              className="flex flex-col gap-1 px-3 pb-2"
            >
              {(isAuthenticated ? ACCOUNT_SECONDARY_NAV : LEGAL_NAV).map((item) => (
                <DrawerLink key={item.href} item={item} onClose={onClose} />
              ))}
            </nav>

            {isAuthenticated && (
              <div className="mt-auto border-t border-mist p-3">
                <button
                  type="button"
                  onClick={() => {
                    onClose();
                    logout();
                  }}
                  className="flex w-full items-center gap-3 rounded-lg px-3 py-2.5 text-sm text-paprika transition-colors hover:bg-paprika-tint focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-herb/40"
                >
                  <LogOut aria-hidden="true" className="h-5 w-5 shrink-0" />
                  Sign out
                </button>
              </div>
            )}
          </motion.div>
        </>
      )}
    </AnimatePresence>
  );
}
