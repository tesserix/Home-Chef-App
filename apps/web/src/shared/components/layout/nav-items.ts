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
  Store,
  type LucideIcon,
} from 'lucide-react';

// Single source of truth for the customer navigation, shared by the fixed
// sidebar and the account drawer.
//
// The two surfaces are deliberately DISJOINT — no destination appears in both:
//
//   RAIL_NAV     browse: where you go to find food. Public, no session needed.
//   ACCOUNT_NAV  account: your own stuff. Requires a session.
//
// An earlier cut repeated Orders/Saved/Plans/Rewards/Invite across both, which
// made the rail and the drawer read as two half-copies of one menu. Keeping the
// split clean means each surface answers exactly one question: "what can I
// order?" vs "what's mine?".
//
// Every href is a route that exists in app/routes/index.tsx. Mobile also has
// blocked-accounts and support-chat screens the web has no route for, so they
// are omitted rather than linked into a 404. The footer separately links to
// /help, /about, /become-chef and /chef-resources, which 404 today — those are
// deliberately NOT repeated here.

export interface NavItem {
  name: string;
  href: string;
  icon: LucideIcon;
  /** Renders as a real <a> to another origin rather than a router Link. */
  external?: boolean;
}

/** The fixed rail — browse and discovery only. */
export const RAIL_NAV: NavItem[] = [
  { name: 'Home', href: '/', icon: Home },
  { name: 'Browse Chefs', href: '/chefs', icon: ChefHat },
  { name: 'Catering', href: '/catering', icon: Utensils },
  { name: 'Social Feed', href: '/feed', icon: Newspaper },
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

/** Legal, shown to everyone — signed in or out. */
export const LEGAL_NAV: NavItem[] = [
  { name: 'Privacy & data', href: '/data-privacy', icon: ShieldCheck },
  { name: 'Terms', href: '/terms', icon: FileText },
];

/**
 * Drawer footer group for a signed-in visitor. Settings sits here rather than
 * in LEGAL_NAV because it configures *your* account — it means nothing to
 * someone who hasn't signed in, so the signed-out drawer shows LEGAL_NAV only.
 */
export const ACCOUNT_SECONDARY_NAV: NavItem[] = [
  { name: 'Settings', href: '/settings', icon: Settings },
  ...LEGAL_NAV,
];

/**
 * The other ways in — our counterpart to Uber Eats' "Add your restaurant" /
 * "Sign up to deliver" block. Only genuinely reachable destinations belong
 * here: the vendor portal is live, whereas delivery.fe3dr.com currently 404s,
 * so driver signup is deliberately absent rather than linked into a dead end.
 */
export const PARTNER_NAV: NavItem[] = [
  { name: 'Add your kitchen', href: 'https://vendors.fe3dr.com', icon: Store, external: true },
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
