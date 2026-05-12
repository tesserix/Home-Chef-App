import { Link, useLocation } from 'react-router-dom';
import { Home, ChefHat, Heart, ShoppingCart, User } from 'lucide-react';
import { AnimatePresence, motion } from 'framer-motion';
import { useCartStore } from '@/app/store/cart-store';
import { useAuth } from '@/app/providers/AuthProvider';
import { useHaptics } from '@/shared/hooks/useMobile';
import { cn } from '@/shared/utils/cn';

interface NavItem {
  name: string;
  href: string;
  icon: React.ComponentType<{ className?: string }>;
  badge?: number;
}

export function MobileBottomNav() {
  const location = useLocation();
  const { isAuthenticated } = useAuth();
  const cartItemCount = useCartStore((state) => state.getItemCount());
  const { lightImpact } = useHaptics();

  const navItems: NavItem[] = [
    { name: 'Home', href: '/', icon: Home },
    { name: 'Chefs', href: '/chefs', icon: ChefHat },
    { name: 'Favorites', href: '/favorites', icon: Heart },
    { name: 'Cart', href: '/cart', icon: ShoppingCart, badge: cartItemCount },
    { name: 'Account', href: isAuthenticated ? '/profile' : '/login', icon: User },
  ];

  const isActive = (href: string) => {
    if (href === '/') return location.pathname === '/';
    return location.pathname.startsWith(href);
  };

  return (
    <nav
      aria-label="Primary"
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
                'touch-target',
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
                      aria-label={`${item.badge} items in cart`}
                    >
                      {item.badge > 99 ? '99+' : item.badge}
                    </motion.span>
                  ) : null}
                </AnimatePresence>
              </span>
              <span className="mt-0.5 text-[10px] font-medium">{item.name}</span>
              {active && (
                <motion.span
                  layoutId="bottomNavIndicator"
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

export function MobileBottomNavSpacer() {
  return <div className="h-16 lg:hidden" />;
}
