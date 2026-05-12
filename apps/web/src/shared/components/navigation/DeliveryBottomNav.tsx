import { Link, useLocation } from 'react-router-dom';
import { LayoutDashboard, Package, DollarSign, MapPin, User } from 'lucide-react';
import { AnimatePresence, motion } from 'framer-motion';
import { useHaptics } from '@/shared/hooks/useMobile';
import { cn } from '@/shared/utils/cn';

interface NavItem {
  name: string;
  href: string;
  icon: React.ComponentType<{ className?: string }>;
  badge?: number;
}

interface DeliveryBottomNavProps {
  activeDeliveries?: number;
}

export function DeliveryBottomNav({ activeDeliveries }: DeliveryBottomNavProps) {
  const location = useLocation();
  const { lightImpact } = useHaptics();

  const navItems: NavItem[] = [
    { name: 'Dashboard', href: '/delivery/dashboard', icon: LayoutDashboard },
    { name: 'Deliveries', href: '/delivery/orders', icon: Package, badge: activeDeliveries },
    { name: 'Map', href: '/delivery/map', icon: MapPin },
    { name: 'Earnings', href: '/delivery/earnings', icon: DollarSign },
    { name: 'Profile', href: '/delivery/profile', icon: User },
  ];

  const isActive = (href: string) => location.pathname === href;

  return (
    <nav
      aria-label="Delivery navigation"
      className="fixed bottom-0 left-0 right-0 z-50 border-t border-mist bg-bone safe-bottom lg:hidden"
    >
      <div className="flex h-16 items-center justify-around px-2">
        {navItems.map((item) => {
          const Icon = item.icon;
          const active = isActive(item.href);

          return (
            <Link
              key={item.name}
              to={item.href}
              onClick={lightImpact}
              aria-current={active ? 'page' : undefined}
              className={cn(
                'relative flex flex-1 flex-col items-center justify-center py-2 transition-colors',
                'touch-target-lg',
                active ? 'text-foreground' : 'text-ink-muted hover:text-foreground'
              )}
            >
              <span className="relative flex h-7 w-7 items-center justify-center">
                <Icon className="h-5 w-5" />
                <AnimatePresence>
                  {item.badge && item.badge > 0 ? (
                    <motion.span
                      key={item.badge}
                      initial={{ scale: 0.6, opacity: 0 }}
                      animate={{ scale: 1, opacity: 1 }}
                      exit={{ scale: 0.6, opacity: 0 }}
                      transition={{ duration: 0.15, ease: [0.22, 1, 0.36, 1] }}
                      className="absolute -right-1.5 -top-1.5 flex h-4 min-w-[1rem] items-center justify-center rounded-full bg-herb px-1 text-[10px] font-medium tabular-nums text-paper"
                      aria-label={`${item.badge} active deliveries`}
                    >
                      {item.badge > 9 ? '9+' : item.badge}
                    </motion.span>
                  ) : null}
                </AnimatePresence>
              </span>
              <span className="mt-0.5 text-[10px] font-medium">{item.name}</span>
              {active && (
                <motion.span
                  layoutId="deliveryNavIndicator"
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

export function DeliveryBottomNavSpacer() {
  return <div className="h-16 lg:hidden" />;
}
