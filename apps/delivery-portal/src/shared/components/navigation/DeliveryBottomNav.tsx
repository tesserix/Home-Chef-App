import { Link, useLocation } from 'react-router-dom';
import { LayoutDashboard, Navigation, Package, DollarSign, User } from 'lucide-react';
import { motion } from 'framer-motion';

const items = [
  { name: 'Home', href: '/dashboard', icon: LayoutDashboard },
  { name: 'Active', href: '/active', icon: Navigation },
  { name: 'Available', href: '/available', icon: Package },
  { name: 'Earnings', href: '/earnings', icon: DollarSign },
  { name: 'Profile', href: '/profile', icon: User },
];

export function DeliveryBottomNav() {
  const location = useLocation();
  const isActive = (href: string) =>
    location.pathname === href || location.pathname.startsWith(href + '/');

  return (
    <nav aria-label="Driver navigation" className="bottom-nav">
      <div className="flex">
        {items.map((item) => {
          const Icon = item.icon;
          const active = isActive(item.href);
          return (
            <Link
              key={item.name}
              to={item.href}
              aria-current={active ? 'page' : undefined}
              className={`${active ? 'bottom-nav-item-active' : 'bottom-nav-item'} relative touch-target-lg`}
            >
              <Icon aria-hidden="true" className="h-5 w-5" />
              <span>{item.name}</span>
              {active && (
                <motion.span
                  layoutId="deliveryPortalNavIndicator"
                  transition={{ duration: 0.25, ease: [0.22, 1, 0.36, 1] }}
                  className="absolute -top-px left-1/2 h-0.5 w-8 -translate-x-1/2 rounded-full bg-herb"
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
