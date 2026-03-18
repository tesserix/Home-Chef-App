import { forwardRef } from 'react';
import { badgeVariants as dsBadgeVariants } from '@tesserix/web';
import { cn } from '@tesserix/web';
import { cva, type VariantProps } from 'class-variance-authority';

const badgeVariants = cva(
  [
    'inline-flex items-center justify-center border',
    'font-semibold whitespace-nowrap rounded-full',
    'transition-colors duration-200',
    'focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2',
  ],
  {
    variants: {
      variant: {
        default: 'border-transparent bg-primary text-primary-foreground shadow hover:bg-primary/80',
        secondary: 'border-transparent bg-secondary text-secondary-foreground hover:bg-secondary/80',
        destructive: 'border-transparent bg-destructive text-destructive-foreground shadow hover:bg-destructive/80',
        outline: 'border-border text-foreground bg-transparent',
        success: 'border-transparent bg-success/10 text-success',
        warning: 'border-transparent bg-warning/10 text-warning',
        error: 'border-transparent bg-destructive/10 text-destructive',
        info: 'border-transparent bg-info/10 text-info',
        // Backward compat
        brand: 'border-transparent bg-primary/10 text-primary',
        'solid-brand': 'border-transparent bg-primary text-primary-foreground shadow',
        premium: 'border-transparent bg-accent text-accent-foreground',
      },
      size: {
        sm: 'h-5 px-2 text-[10px]',
        md: 'h-6 px-2.5 text-xs',
        lg: 'h-7 px-3 text-sm',
      },
    },
    defaultVariants: {
      variant: 'default',
      size: 'md',
    },
  }
);

export interface BadgeProps
  extends React.HTMLAttributes<HTMLSpanElement>,
    VariantProps<typeof badgeVariants> {
  dot?: boolean;
  dotColor?: string;
}

const Badge = forwardRef<HTMLSpanElement, BadgeProps>(
  ({ className, variant, size, dot, dotColor, children, ...props }, ref) => {
    const getDotColorClass = () => {
      if (dotColor) return dotColor;
      switch (variant) {
        case 'success':
          return 'bg-success';
        case 'destructive':
        case 'error':
          return 'bg-destructive';
        case 'warning':
          return 'bg-warning';
        case 'brand':
        case 'solid-brand':
          return 'bg-primary';
        case 'info':
          return 'bg-info';
        default:
          return 'bg-muted-foreground';
      }
    };

    return (
      <span
        ref={ref}
        className={cn(badgeVariants({ variant, size }), dot && 'pl-1.5', className)}
        {...props}
      >
        {dot && (
          <span className={cn('mr-1.5 h-1.5 w-1.5 rounded-full', getDotColorClass())} />
        )}
        {children}
      </span>
    );
  }
);

Badge.displayName = 'Badge';

// Status Badge
export interface StatusBadgeProps extends Omit<BadgeProps, 'variant'> {
  status: 'active' | 'inactive' | 'pending' | 'success' | 'error' | 'warning';
}

const StatusBadge = forwardRef<HTMLSpanElement, StatusBadgeProps>(
  ({ status, children, ...props }, ref) => {
    const statusConfig = {
      active: { variant: 'success' as const, label: 'Active', dot: true },
      inactive: { variant: 'default' as const, label: 'Inactive', dot: true },
      pending: { variant: 'warning' as const, label: 'Pending', dot: true },
      success: { variant: 'success' as const, label: 'Success', dot: false },
      error: { variant: 'destructive' as const, label: 'Error', dot: false },
      warning: { variant: 'warning' as const, label: 'Warning', dot: false },
    };

    const config = statusConfig[status];

    return (
      <Badge
        ref={ref}
        variant={config.variant}
        dot={config.dot}
        {...props}
      >
        {children || config.label}
      </Badge>
    );
  }
);

StatusBadge.displayName = 'StatusBadge';

// Order Status Badge
export interface OrderStatusBadgeProps extends Omit<BadgeProps, 'variant'> {
  status: string;
}

const OrderStatusBadge = forwardRef<HTMLSpanElement, OrderStatusBadgeProps>(
  ({ status, ...props }, ref) => {
    const statusConfig: Record<string, { variant: BadgeProps['variant']; label: string }> = {
      pending: { variant: 'warning', label: 'New' },
      accepted: { variant: 'info', label: 'Accepted' },
      preparing: { variant: 'brand', label: 'Preparing' },
      ready: { variant: 'success', label: 'Ready' },
      picked_up: { variant: 'info', label: 'Picked Up' },
      delivering: { variant: 'brand', label: 'Delivering' },
      delivered: { variant: 'default', label: 'Delivered' },
      cancelled: { variant: 'destructive', label: 'Cancelled' },
    };

    const config = statusConfig[status] || { variant: 'default', label: status };

    return (
      <Badge ref={ref} variant={config.variant} {...props}>
        {config.label}
      </Badge>
    );
  }
);

OrderStatusBadge.displayName = 'OrderStatusBadge';

export { Badge, StatusBadge, OrderStatusBadge, badgeVariants, dsBadgeVariants };
