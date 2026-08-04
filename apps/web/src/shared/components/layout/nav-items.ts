import {
  Home,
  ChefHat,
  Heart,
  Utensils,
  CalendarDays,
  Repeat,
  Award,
  Gift,
  Package,
  Wallet,
  ShieldCheck,
  FileText,
  Store,
  type LucideIcon,
  BookOpen,
} from 'lucide-react';
import { VENDOR_CTA_LABEL, VENDOR_PORTAL_URL } from '@/shared/config/partner-sites';
import {
  CATERING_ENABLED,
  TIFFIN_ENABLED,
  WALLET_ENABLED,
  REWARDS_ENABLED,
  REFERRAL_ENABLED,
} from '@/shared/config/features';

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
// are omitted rather than linked into a 404. The footer used to link /help,
// /about, /become-chef and /chef-resources, which 404'd; those have since been
// removed or repointed, so the whole app now links only to real routes.

export interface NavItem {
  name: string;
  href: string;
  icon: LucideIcon;
  /** Renders as a real <a> to another origin rather than a router Link. */
  external?: boolean;
}

/**
 * The fixed rail — browse and discovery only.
 *
 * Catering is gated: the mobile app hides it for v1, and advertising it on web
 * meant leading customers into a flow the product has deferred. It reappears
 * the moment the flag flips.
 */
export const RAIL_NAV: NavItem[] = [
  { name: 'Home', href: '/', icon: Home },
  { name: 'Browse Chefs', href: '/chefs', icon: ChefHat },
  ...(CATERING_ENABLED ? [{ name: 'Catering', href: '/catering', icon: Utensils }] : []),
  { name: 'ChefBook', href: '/chefbook', icon: BookOpen },
];

/** Account drawer — mirrors the quick tiles and list rows on mobile Profile. */
export const ACCOUNT_NAV: NavItem[] = [
  { name: 'Orders', href: '/orders', icon: Package },
  { name: 'Saved', href: '/favorites', icon: Heart },
  ...(WALLET_ENABLED ? [{ name: 'Wallet', href: '/wallet', icon: Wallet }] : []),
  // Meal Plans and Subscriptions are DIFFERENT products: a plan is a one-off
  // pre-booked week paid as an advance, a subscription is a recurring daily
  // tiffin. "Meal Plans" used to point at /subscriptions, so a customer with a
  // live plan was shown "No tiffin subscriptions yet" and had no way to reach it.
  ...(TIFFIN_ENABLED ? [{ name: 'Meal Plans', href: '/meal-plans', icon: CalendarDays }] : []),
  ...(TIFFIN_ENABLED ? [{ name: 'Subscriptions', href: '/subscriptions', icon: Repeat }] : []),
  ...(REWARDS_ENABLED ? [{ name: 'Rewards', href: '/loyalty', icon: Award }] : []),
  ...(REFERRAL_ENABLED ? [{ name: 'Invite friends', href: '/referral', icon: Gift }] : []),
];

/** Legal, shown to everyone — signed in or out. */
export const LEGAL_NAV: NavItem[] = [
  { name: 'Privacy & data', href: '/data-privacy', icon: ShieldCheck },
  { name: 'Terms', href: '/terms', icon: FileText },
];

/**
 * Drawer footer group for a signed-in visitor.
 *
 * This used to lead with a Settings row pointing at /settings. No such route
 * exists — the only `settings` path in the app belonged to the embedded admin
 * tree (/admin/settings), which has since moved out to its own app — so every
 * signed-in customer got a menu row that dumped them on the home page. Account
 * settings live on /profile, which the drawer header already links to.
 */
export const ACCOUNT_SECONDARY_NAV: NavItem[] = [...LEGAL_NAV];

/**
 * The other ways in — our counterpart to Uber Eats' "Add your restaurant" /
 * "Sign up to deliver" block. Only genuinely reachable destinations belong
 * here: the vendor portal is live, whereas delivery.fe3dr.com currently 404s,
 * so driver signup is deliberately absent rather than linked into a dead end.
 */
export const PARTNER_NAV: NavItem[] = [
  { name: VENDOR_CTA_LABEL, href: VENDOR_PORTAL_URL, icon: Store, external: true },
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
