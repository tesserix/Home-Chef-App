import { Link } from 'react-router-dom';
import { cn } from '@/shared/utils/cn';

interface LogoProps {
  variant?: 'default' | 'light' | 'icon-only';
  size?: 'sm' | 'md' | 'lg';
  showTagline?: boolean;
  className?: string;
  linkTo?: string;
}

const sizeClasses = {
  sm: {
    icon: 'h-8 w-8',
    iconText: 'text-sm',
    text: 'text-lg',
    tagline: 'text-[10px]',
  },
  md: {
    icon: 'h-10 w-10',
    iconText: 'text-lg',
    text: 'text-xl',
    tagline: 'text-xs',
  },
  lg: {
    icon: 'h-12 w-12',
    iconText: 'text-xl',
    text: 'text-2xl',
    tagline: 'text-sm',
  },
};

export function Logo({
  variant = 'default',
  size = 'md',
  showTagline = false,
  className,
  linkTo = '/',
}: LogoProps) {
  const sizes = sizeClasses[size];
  const isLight = variant === 'light';

  const logoContent = (
    <div className={cn('flex items-center gap-2', className)}>
      {/* Logo Icon - Chef Hat with Bowl */}
      <div
        className={cn(
          'flex items-center justify-center rounded-xl bg-gradient-to-br from-brand-500 to-brand-600 shadow-md',
          sizes.icon
        )}
      >
        <svg
          viewBox="0 0 32 32"
          fill="none"
          className={cn('h-2/3 w-2/3')}
          xmlns="http://www.w3.org/2000/svg"
        >
          {/* Chef Hat */}
          <path
            d="M16 4c-2 0-3.5 1.2-4 3-.3-1-1.3-1.8-2.5-1.8-2 0-3.5 1.8-3.5 4 0 1 .3 1.8 1 2.5v10h18v-10c.7-.7 1-1.5 1-2.5 0-2.2-1.5-4-3.5-4-1.2 0-2.2.8-2.5 1.8-.5-1.8-2-3-4-3z"
            fill="white"
          />
          {/* Bowl */}
          <path
            d="M6 24c0 3 4.5 5 10 5s10-2 10-5-4.5-1.5-10-1.5S6 21 6 24z"
            fill="white"
            opacity="0.85"
          />
        </svg>
      </div>

      {/* Logo Text */}
      {variant !== 'icon-only' && (
        <div className="flex flex-col">
          <span
            className={cn(
              'font-bold tracking-tight',
              sizes.text,
              isLight ? 'text-white' : 'text-gray-900'
            )}
          >
            Fe<span className="text-brand-500">3dr</span>
          </span>
          {showTagline && (
            <span
              className={cn(
                sizes.tagline,
                isLight ? 'text-white/70' : 'text-gray-500'
              )}
            >
              Homemade Food Delivered
            </span>
          )}
        </div>
      )}
    </div>
  );

  if (linkTo) {
    return (
      <Link to={linkTo} className="transition-opacity hover:opacity-90">
        {logoContent}
      </Link>
    );
  }

  return logoContent;
}

// Alternative text-based logo for special cases
export function LogoText({
  variant = 'default',
  className,
}: {
  variant?: 'default' | 'light';
  className?: string;
}) {
  const isLight = variant === 'light';

  return (
    <span
      className={cn(
        'font-display text-2xl font-bold tracking-tight',
        isLight ? 'text-white' : 'text-gray-900',
        className
      )}
    >
      Fe<span className="text-brand-500">3dr</span>
    </span>
  );
}

// Trademark footer text
export function LogoFooter() {
  return (
    <div className="flex items-center gap-2">
      <Logo size="sm" linkTo="/" />
      <span className="text-xs text-gray-400">
        &copy; {new Date().getFullYear()} Fe3dr. All rights reserved.
      </span>
    </div>
  );
}
