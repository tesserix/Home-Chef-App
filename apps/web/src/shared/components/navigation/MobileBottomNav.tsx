import { Link, useLocation } from 'react-router-dom';
import { Home, ChefHat, Heart, ShoppingCart, User } from 'lucide-react';
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

  const handleNavClick = () => {
    lightImpact();
  };

  return (
    <nav className="fixed bottom-0 left-0 right-0 z-50 border-t border-gray-200 bg-white/95 backdrop-blur-lg safe-bottom md:hidden">
      <div className="flex h-16 items-center justify-around px-2">
        {navItems.map((item) => {
          const Icon = item.icon;
          const active = isActive(item.href);

          return (
            <Link
              key={item.name}
              to={item.href}
              onClick={handleNavClick}
              className={cn(
                'relative flex flex-1 flex-col items-center justify-center py-2 transition-all duration-200',
                'touch-target',
                active ? 'text-brand-600' : 'text-gray-500'
              )}
            >
              <span className={cn(
                'relative flex h-7 w-7 items-center justify-center rounded-lg transition-all duration-200',
                active && 'bg-brand-100'
              )}>
                <Icon className={cn(
                  'h-5 w-5 transition-transform duration-200',
                  active && 'scale-110'
                )} />
                {item.badge && item.badge > 0 && (
                  <span className="absolute -right-1 -top-1 flex h-4 w-4 items-center justify-center rounded-full bg-brand-500 text-[10px] font-semibold text-white">
                    {item.badge > 9 ? '9+' : item.badge}
                  </span>
                )}
              </span>
              <span className={cn(
                'mt-0.5 text-[10px] font-medium transition-all duration-200',
                active ? 'text-brand-600' : 'text-gray-500'
              )}>
                {item.name}
              </span>
              {active && (
                <span className="absolute -top-0.5 left-1/2 h-0.5 w-8 -translate-x-1/2 rounded-full bg-brand-500" />
              )}
            </Link>
          );
        })}
      </div>
    </nav>
  );
}

export function MobileBottomNavSpacer() {
  return <div className="h-16 md:hidden" />;
}
