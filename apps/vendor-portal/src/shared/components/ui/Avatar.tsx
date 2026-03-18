import { forwardRef } from 'react';
import { Avatar as DSAvatar } from '@tesserix/web';
import { cn } from '@tesserix/web';
import { cva, type VariantProps } from 'class-variance-authority';
import * as AvatarPrimitive from '@radix-ui/react-avatar';

const avatarVariants = cva(
  [
    'relative flex shrink-0 overflow-hidden',
    'bg-muted',
  ],
  {
    variants: {
      size: {
        xs: 'h-6 w-6 text-xs',
        sm: 'h-8 w-8 text-sm',
        md: 'h-10 w-10 text-base',
        lg: 'h-12 w-12 text-lg',
        xl: 'h-16 w-16 text-xl',
        '2xl': 'h-20 w-20 text-2xl',
      },
      shape: {
        circle: 'rounded-full',
        square: 'rounded-xl',
      },
      ring: {
        none: '',
        default: 'ring-2 ring-card',
        brand: 'ring-2 ring-primary',
        golden: 'ring-2 ring-warning',
        success: 'ring-2 ring-success',
      },
    },
    defaultVariants: {
      size: 'md',
      shape: 'circle',
      ring: 'none',
    },
  }
);

export interface AvatarProps
  extends React.ComponentPropsWithoutRef<typeof AvatarPrimitive.Root>,
    VariantProps<typeof avatarVariants> {
  src?: string;
  alt?: string;
  fallback?: string;
  fallbackClassName?: string;
}

const Avatar = forwardRef<HTMLDivElement, AvatarProps>(
  ({ className, size, shape, ring, src, alt, fallback, fallbackClassName, ...props }, ref) => {
    // For simple DS-compatible usage (circle, no ring, standard sizes, no custom fallback)
    const dsCompatibleSizes = ['sm', 'md', 'lg', 'xl'] as const;
    type DSSizeType = typeof dsCompatibleSizes[number];
    const isDSSize = !size || dsCompatibleSizes.includes(size as DSSizeType);
    const isSimple = isDSSize && shape !== 'square' && ring === 'none' && !fallbackClassName;

    if (isSimple) {
      return (
        <DSAvatar
          ref={ref}
          src={src}
          alt={alt}
          fallback={fallback || getInitials(alt)}
          size={size === 'md' || !size ? 'default' : (size as 'sm' | 'lg' | 'xl')}
          className={className}
          {...props}
        />
      );
    }

    return (
      <AvatarPrimitive.Root
        ref={ref}
        className={cn(avatarVariants({ size, shape, ring }), className)}
        {...props}
      >
        <AvatarPrimitive.Image
          className="h-full w-full object-cover"
          src={src}
          alt={alt}
        />
        <AvatarPrimitive.Fallback
          className={cn(
            'flex h-full w-full items-center justify-center bg-muted font-medium text-muted-foreground',
            fallbackClassName
          )}
        >
          {fallback || getInitials(alt)}
        </AvatarPrimitive.Fallback>
      </AvatarPrimitive.Root>
    );
  }
);

Avatar.displayName = 'Avatar';

function getInitials(name?: string): string {
  if (!name) return '?';
  const parts = name.trim().split(' ').filter(Boolean);
  if (parts.length === 0) return '?';
  const first = parts[0] ?? '';
  const last = parts[parts.length - 1] ?? '';
  if (parts.length === 1) return first.charAt(0).toUpperCase();
  return (first.charAt(0) + last.charAt(0)).toUpperCase();
}

// Avatar Group
interface AvatarGroupProps extends React.HTMLAttributes<HTMLDivElement> {
  max?: number;
  size?: VariantProps<typeof avatarVariants>['size'];
  children: React.ReactNode;
}

const AvatarGroup = forwardRef<HTMLDivElement, AvatarGroupProps>(
  ({ className, max = 4, size = 'md', children, ...props }, ref) => {
    const childArray = Array.isArray(children) ? children : [children];
    const visibleAvatars = childArray.slice(0, max);
    const remainingCount = childArray.length - max;

    const overlapClasses = {
      xs: '-ml-1.5',
      sm: '-ml-2',
      md: '-ml-2.5',
      lg: '-ml-3',
      xl: '-ml-4',
      '2xl': '-ml-5',
    };

    return (
      <div
        ref={ref}
        className={cn('flex items-center', className)}
        {...props}
      >
        {visibleAvatars.map((child, index) => (
          <div
            key={index}
            className={cn(index > 0 && overlapClasses[size || 'md'])}
            style={{ zIndex: visibleAvatars.length - index }}
          >
            {child}
          </div>
        ))}
        {remainingCount > 0 && (
          <div
            className={cn(overlapClasses[size || 'md'])}
            style={{ zIndex: 0 }}
          >
            <div
              className={cn(
                avatarVariants({ size, shape: 'circle' }),
                'flex items-center justify-center bg-secondary font-medium text-secondary-foreground'
              )}
            >
              +{remainingCount}
            </div>
          </div>
        )}
      </div>
    );
  }
);

AvatarGroup.displayName = 'AvatarGroup';

// Avatar with Status indicator
interface AvatarWithStatusProps extends AvatarProps {
  status?: 'online' | 'offline' | 'away' | 'busy';
  statusPosition?: 'top-right' | 'bottom-right';
}

const AvatarWithStatus = forwardRef<HTMLDivElement, AvatarWithStatusProps>(
  ({ status, statusPosition = 'bottom-right', size = 'md', ...props }, ref) => {
    const statusColors = {
      online: 'bg-success',
      offline: 'bg-muted-foreground',
      away: 'bg-warning',
      busy: 'bg-destructive',
    };

    const statusSizes = {
      xs: 'h-1.5 w-1.5',
      sm: 'h-2 w-2',
      md: 'h-2.5 w-2.5',
      lg: 'h-3 w-3',
      xl: 'h-3.5 w-3.5',
      '2xl': 'h-4 w-4',
    };

    const positionClasses = {
      'top-right': 'top-0 right-0',
      'bottom-right': 'bottom-0 right-0',
    };

    return (
      <div className="relative inline-block">
        <Avatar ref={ref} size={size} {...props} />
        {status && (
          <span
            className={cn(
              'absolute rounded-full ring-2 ring-card',
              statusColors[status],
              statusSizes[size || 'md'],
              positionClasses[statusPosition]
            )}
          />
        )}
      </div>
    );
  }
);

AvatarWithStatus.displayName = 'AvatarWithStatus';

export { Avatar, AvatarGroup, AvatarWithStatus, avatarVariants };
