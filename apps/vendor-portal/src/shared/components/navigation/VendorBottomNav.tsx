import { Link, useLocation } from 'react-router-dom';
import {
  LayoutDashboard,
  UtensilsCrossed,
  ClipboardList,
  DollarSign,
  User,
} from 'lucide-react';

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
    <nav className="fixed bottom-0 left-0 right-0 z-50 border-t border-gray-200 bg-white safe-bottom md:hidden">
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
              className={`flex flex-1 flex-col items-center justify-center gap-1 py-2 text-xs font-medium transition-colors ${
                isActive ? 'text-brand-600' : 'text-gray-500'
              }`}
            >
              <Icon className="h-5 w-5" />
              <span>{item.name}</span>
            </Link>
          );
        })}
      </div>
    </nav>
  );
}
