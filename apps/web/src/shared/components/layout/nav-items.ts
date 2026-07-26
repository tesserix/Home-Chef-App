import {
  Home,
  ChefHat,
  Heart,
  Utensils,
  Newspaper,
  CalendarDays,
  Award,
  Gift,
  Package,
  Wallet,
  Settings,
  ShieldCheck,
  FileText,
  type LucideIcon,
} from 'lucide-react';

// Single source of truth for the customer navigation, shared by the fixed
// sidebar and the account drawer so the two can never drift apart.
//
// The groupings deliberately mirror the mobile customer app so the two clients
// teach the same mental model:
//   - RAIL_NAV       ← mobile's five bottom tabs (Home/Orders/Plans/Saved/…)
//   - DISCOVER_NAV   ← the browse rows on mobile's Profile screen
//   - ACCOUNT_NAV    ← mobile Profile's quick tiles + list rows
//
// Every href is a route that exists in app/routes/index.tsx. Mobile also has
// blocked-accounts and support-chat screens which the web has no route for, so
// they are omitted rather than linked into a 404. The footer separately links
// to /help, /about, /become-chef and /chef-resources, which 404 today — those
// are deliberately NOT repeated here.

export interface NavItem {
  name: string;
  href: string;
  icon: LucideIcon;
  /** Only render when the visitor is signed in. */
  authOnly?: boolean;
}

/**
 * The fixed rail — the web equivalent of mobile's bottom tab bar, plus the two
 * browse destinations that a wide viewport has room to promote.
 */
export const RAIL_NAV: NavItem[] = [
  { name: 'Home', href: '/', icon: Home },
  { name: 'Browse Chefs', href: '/chefs', icon: ChefHat },
  { name: 'Orders', href: '/orders', icon: Package, authOnly: true },
  { name: 'Plans', href: '/subscriptions', icon: CalendarDays, authOnly: true },
  { name: 'Saved', href: '/favorites', icon: Heart },
];

/** Secondary discovery, below a divider in the rail. */
export const DISCOVER_NAV: NavItem[] = [
  { name: 'Catering', href: '/catering', icon: Utensils },
  { name: 'Social Feed', href: '/feed', icon: Newspaper },
  { name: 'Rewards', href: '/loyalty', icon: Award, authOnly: true },
];

/** Bottom of the rail — set apart so it reads as an offer, not a section. */
export const OFFERS_NAV: NavItem[] = [
  { name: 'Invite & Earn', href: '/referral', icon: Gift, authOnly: true },
];

/** Account drawer — mirrors the quick tiles and list rows on mobile Profile. */
export const ACCOUNT_NAV: NavItem[] = [
  { name: 'Orders', href: '/orders', icon: Package },
  { name: 'Saved', href: '/favorites', icon: Heart },
  { name: 'Wallet', href: '/wallet', icon: Wallet },
  { name: 'Meal Plans', href: '/subscriptions', icon: CalendarDays },
  { name: 'Rewards', href: '/loyalty', icon: Award },
  { name: 'Invite friends', href: '/referral', icon: Gift },
];

/** Drawer footer group — settings, privacy and legal sit apart from the list. */
export const ACCOUNT_SECONDARY_NAV: NavItem[] = [
  { name: 'Settings', href: '/settings', icon: Settings },
  { name: 'Privacy & data', href: '/data-privacy', icon: ShieldCheck },
  { name: 'Terms', href: '/terms', icon: FileText },
];

/**
 * Active-route test. `/` must match exactly or it would light up on every
 * page; everything else matches by prefix so nested routes (e.g. /orders/123)
 * keep their parent highlighted.
 */
export function isNavItemActive(href: string, pathname: string): boolean {
  if (href === '/') return pathname === '/';
  return pathname === href || pathname.startsWith(`${href}/`);
}
