import { Link, useLocation } from 'react-router-dom';
import { useAuth } from '@/app/providers/AuthProvider';
import {
  RAIL_NAV,
  DISCOVER_NAV,
  OFFERS_NAV,
  isNavItemActive,
  type NavItem,
} from './nav-items';

// The fixed navigation rail (desktop only). Sticky rather than `position:
// fixed` so it sits naturally under the sticky header without having to
// recompute offsets when the offline banner pushes the header down.
//
// Mobile keeps the existing bottom nav; this rail is hidden below lg, and the
// same destinations remain reachable there via the bottom nav and the drawer.

function NavRow({ item, active }: { item: NavItem; active: boolean }) {
  const Icon = item.icon;
  return (
    <Link
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
}

export function AppSidebar() {
  const location = useLocation();
  const { isAuthenticated } = useAuth();

  const visible = (items: NavItem[]) =>
    items.filter((i) => !i.authOnly || isAuthenticated);

  const discover = visible(DISCOVER_NAV);
  const offers = visible(OFFERS_NAV);

  return (
    <aside
      className="sticky top-16 hidden h-[calc(100vh-4rem)] w-60 shrink-0 overflow-y-auto border-r border-mist bg-bone lg:block"
    >
      <nav aria-label="Main" className="flex flex-col gap-1 p-3">
        {visible(RAIL_NAV).map((item) => (
          <NavRow
            key={item.href}
            item={item}
            active={isNavItemActive(item.href, location.pathname)}
          />
        ))}

        {discover.length > 0 && (
          <>
            <hr className="my-2 border-mist" />
            {discover.map((item) => (
              <NavRow
                key={item.href}
                item={item}
                active={isNavItemActive(item.href, location.pathname)}
              />
            ))}
          </>
        )}

        {offers.length > 0 && (
          <>
            <hr className="my-2 border-mist" />
            {offers.map((item) => (
              <NavRow
                key={item.href}
                item={item}
                active={isNavItemActive(item.href, location.pathname)}
              />
            ))}
          </>
        )}
      </nav>
    </aside>
  );
}
