import { useEffect, useRef, useState } from 'react';
import { Link } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { ChevronDown, LogOut, Settings, User } from 'lucide-react';
import { apiClient } from '@/shared/services/api-client';
import { ThemeToggle } from '@/shared/theme';

// AccountMenu — who you are and what you can change about it, top right.
//
// This used to live at the BOTTOM of the left sidebar, which put the one
// control every other product in the world places top-right at the far corner
// of the page, below a nav list long enough to scroll. It also showed only the
// email, so a chef running the portal could not see WHICH kitchen they were
// signed in to — the thing they most need confirmed before accepting an order.

interface ChefProfileSummary {
  businessName?: string;
}

interface AccountMenuProps {
  /** Display name for the signed-in user; falls back to their email. */
  displayName: string;
  email?: string;
  /** Identity of the signed-in account. Scopes the cached kitchen name so one
   *  chef's kitchen can never be rendered under another chef's session. */
  userId?: string;
  onLogout: () => void;
}

export function AccountMenu({ displayName, email, userId, onLogout }: AccountMenuProps) {
  const [open, setOpen] = useState(false);
  const rootRef = useRef<HTMLDivElement>(null);

  // The kitchen's name. Cached hard — it changes about once ever, and this
  // renders on every page, so it must not become a request per navigation.
  const { data: chef } = useQuery<ChefProfileSummary>({
    // Keyed by account. AuthProvider already clears the cache when the identity
    // changes; this makes the component correct on its own terms too, so a
    // future provider refactor cannot silently reintroduce a cross-user leak in
    // the one control whose whole job is telling the chef who they are.
    queryKey: ['chef', 'profile', 'summary', userId ?? 'anonymous'],
    queryFn: () => apiClient.get<ChefProfileSummary>('/chef/profile'),
    staleTime: 5 * 60_000,
    retry: false,
    enabled: Boolean(userId),
  });
  const businessName = chef?.businessName?.trim();

  // Close on outside click and on Escape. Without both, a menu opened by
  // keyboard can only be dismissed by activating something in it.
  useEffect(() => {
    if (!open) return;
    const onPointer = (e: MouseEvent) => {
      if (!rootRef.current?.contains(e.target as Node)) setOpen(false);
    };
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setOpen(false);
    };
    document.addEventListener('mousedown', onPointer);
    document.addEventListener('keydown', onKey);
    return () => {
      document.removeEventListener('mousedown', onPointer);
      document.removeEventListener('keydown', onKey);
    };
  }, [open]);

  return (
    <div ref={rootRef} className="relative">
      <button
        type="button"
        aria-haspopup="menu"
        aria-expanded={open}
        aria-label={open ? 'Close account menu' : 'Open account menu'}
        onClick={() => setOpen((v) => !v)}
        className="flex max-w-[15rem] items-center gap-2 rounded-lg p-1.5 transition-colors hover:bg-secondary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
      >
        <span className="flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-full bg-primary/10">
          <User aria-hidden="true" className="h-4 w-4 text-primary" />
        </span>
        {/* The kitchen leads on wide screens — it is the identity that matters
            when accepting an order. Hidden on small screens, where the avatar
            alone has to carry it. */}
        <span className="hidden min-w-0 text-left sm:block">
          <span className="block truncate text-sm font-medium text-foreground">
            {businessName || displayName}
          </span>
          <span className="block truncate text-xs text-muted-foreground">
            {businessName ? displayName : 'Chef'}
          </span>
        </span>
        <ChevronDown aria-hidden="true" className="h-4 w-4 flex-shrink-0 text-muted-foreground" />
      </button>

      {open && (
        <div
          role="menu"
          aria-label="Account"
          className="absolute right-0 z-50 mt-2 w-64 overflow-hidden rounded-xl border border-border bg-card shadow-2"
        >
          <div className="border-b border-border px-4 py-3">
            <p className="truncate text-sm font-semibold text-foreground">
              {businessName || displayName}
            </p>
            <p className="truncate text-xs text-muted-foreground">{email || displayName}</p>
            {businessName && (
              <p className="mt-0.5 text-xs text-muted-foreground">Chef</p>
            )}
          </div>

          <div className="p-1">
            <Link
              to="/settings"
              role="menuitem"
              onClick={() => setOpen(false)}
              className="flex items-center gap-2 rounded-lg px-3 py-2 text-sm text-muted-foreground transition-colors hover:bg-secondary hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
            >
              <Settings aria-hidden="true" className="h-4 w-4" />
              Settings
            </Link>

            <div className="flex items-center justify-between rounded-lg px-3 py-2 text-sm text-muted-foreground">
              <span>Theme</span>
              <ThemeToggle size="sm" />
            </div>

            <button
              type="button"
              role="menuitem"
              onClick={() => {
                setOpen(false);
                onLogout();
              }}
              className="flex w-full items-center gap-2 rounded-lg px-3 py-2 text-sm text-destructive transition-colors hover:bg-destructive/10 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-destructive"
            >
              <LogOut aria-hidden="true" className="h-4 w-4" />
              Logout
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
