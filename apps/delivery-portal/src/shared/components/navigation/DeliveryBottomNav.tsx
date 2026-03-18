import { Link, useLocation } from 'react-router-dom';
import { LayoutDashboard, Navigation, Package, DollarSign, User } from 'lucide-react';

const items = [
  { name: 'Home', href: '/dashboard', icon: LayoutDashboard },
  { name: 'Active', href: '/active', icon: Navigation },
  { name: 'Available', href: '/available', icon: Package },
  { name: 'Earnings', href: '/earnings', icon: DollarSign },
  { name: 'Profile', href: '/profile', icon: User },
];

export function DeliveryBottomNav() {
  const location = useLocation();
  const isActive = (href: string) => location.pathname === href || location.pathname.startsWith(href + '/');

  return (
    <nav className="bottom-nav">
      <div className="flex">
        {items.map((item) => {
          const Icon = item.icon;
          return (
            <Link
              key={item.name}
              to={item.href}
              className={isActive(item.href) ? 'bottom-nav-item-active' : 'bottom-nav-item'}
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
