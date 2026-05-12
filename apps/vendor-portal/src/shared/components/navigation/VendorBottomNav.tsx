import { Link, useLocation } from 'react-router-dom';
import {
  LayoutDashboard,
  UtensilsCrossed,
  ClipboardList,
  DollarSign,
  User,
} from 'lucide-react';
import { motion } from 'framer-motion';

const bottomNavItems = [
  { name: 'Dashboard', href: '/dashboard', icon: LayoutDashboard },
  { name: 'Menu', href: '/menu', icon: UtensilsCrossed },
  { name: 'Orders', href: '/orders', icon: ClipboardList },
  { name: 'Earnings', href: '/earnings', icon: DollarSign },
  { name: 'Profile', href: '/profile', icon: User },
];

export function VendorBottomNav() {
  const location = useLocation();

  return (
    <nav
      aria-label="Vendor navigation"
      className="fixed bottom-0 left-0 right-0 z-50 border-t border-mist bg-bone safe-bottom md:hidden"
    >
      <div className="flex items-center">
        {bottomNavItems.map((item) => {
          const Icon = item.icon;
          const isActive =
            location.pathname === item.href ||
            location.pathname.startsWith(item.href + '/');

          return (
            <Link
              key={item.name}
              to={item.href}
              aria-current={isActive ? 'page' : undefined}
              className={`relative flex flex-1 flex-col items-center justify-center gap-1 py-2 text-xs font-medium transition-colors touch-target ${
                isActive ? 'text-foreground' : 'text-ink-muted hover:text-foreground'
              }`}
            >
              <Icon className="h-5 w-5" />
              <span>{item.name}</span>
              {isActive && (
                <motion.span
                  layoutId="vendorPortalNavIndicator"
                  transition={{ duration: 0.25, ease: [0.22, 1, 0.36, 1] }}
                  className="absolute -top-px h-0.5 w-8 rounded-full bg-herb"
                  aria-hidden
                />
              )}
            </Link>
          );
        })}
      </div>
    </nav>
  );
}
