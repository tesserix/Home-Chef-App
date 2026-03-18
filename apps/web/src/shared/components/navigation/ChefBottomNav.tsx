import { Link, useLocation } from 'react-router-dom';
import { LayoutDashboard, UtensilsCrossed, ClipboardList, DollarSign, User } from 'lucide-react';
import { useHaptics } from '@/shared/hooks/useMobile';
import { cn } from '@/shared/utils/cn';

interface NavItem {
  name: string;
  href: string;
  icon: React.ComponentType<{ className?: string }>;
  badge?: number;
}

interface ChefBottomNavProps {
  pendingOrders?: number;
}

export function ChefBottomNav({ pendingOrders }: ChefBottomNavProps) {
  const location = useLocation();
  const { lightImpact } = useHaptics();

  const navItems: NavItem[] = [
    { name: 'Dashboard', href: '/chef/dashboard', icon: LayoutDashboard },
    { name: 'Menu', href: '/chef/menu', icon: UtensilsCrossed },
    { name: 'Orders', href: '/chef/orders', icon: ClipboardList, badge: pendingOrders },
    { name: 'Earnings', href: '/chef/earnings', icon: DollarSign },
    { name: 'Profile', href: '/chef/profile', icon: User },
  ];

  const isActive = (href: string) => location.pathname === href;

  const handleNavClick = () => {
    lightImpact();
  };

  return (
    <nav className="fixed bottom-0 left-0 right-0 z-50 border-t border-gray-200 bg-white/95 backdrop-blur-lg safe-bottom lg:hidden">
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
                  <span className="absolute -right-1 -top-1 flex h-4 w-4 items-center justify-center rounded-full bg-red-500 text-[10px] font-semibold text-white">
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

export function ChefBottomNavSpacer() {
  return <div className="h-16 lg:hidden" />;
}
