import { Link, useLocation } from 'react-router';
import { RAIL_NAV, isNavItemActive } from './nav-items';

// The fixed navigation rail (desktop only). Sticky rather than `position:
// fixed` so it sits naturally under the sticky header without having to
// recompute offsets when the offline banner pushes the header down.
//
// Browse destinations only — account lives in the drawer, and the two lists
// never overlap (see nav-items.ts). Everything here is public, so the rail
// renders identically signed in or out.
//
// Mobile keeps the existing bottom nav; this rail is hidden below lg, and the
// same destinations remain reachable there via the drawer's browse group.

export function AppSidebar() {
  const location = useLocation();

  return (
    <aside className="sticky top-16 hidden h-[calc(100vh-4rem)] w-60 shrink-0 overflow-y-auto border-r border-mist bg-bone lg:block">
      <nav aria-label="Main" className="flex flex-col gap-1 p-3">
        {RAIL_NAV.map((item) => {
          const Icon = item.icon;
          const active = isNavItemActive(item.href, location.pathname);
          return (
            <Link
              key={item.href}
              to={item.href}
              aria-current={active ? 'page' : undefined}
              className={`flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-herb/40 ${
                active
                  ? 'bg-herb-tint font-medium text-herb'
                  : 'text-ink-soft hover:bg-mist hover:text-ink'
              }`}
            >
              <Icon aria-hidden="true" className="h-5 w-5 shrink-0" />
              <span className="truncate">{item.name}</span>
            </Link>
          );
        })}
      </nav>
    </aside>
  );
}
