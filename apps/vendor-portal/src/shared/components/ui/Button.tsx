import { forwardRef } from 'react';
import {
  Button as DSButton,
  buttonVariants as dsButtonVariants,
} from '@tesserix/web';
import { cn } from '@tesserix/web';
import { Slot } from '@radix-ui/react-slot';
import { cva, type VariantProps } from 'class-variance-authority';
import { Loader2 } from 'lucide-react';

/**
 * Extended Button variants that map to the design system and add
 * app-specific variants not present in @tesserix/web.
 */
const buttonVariants = cva(
  [
    'inline-flex items-center justify-center gap-2 whitespace-nowrap',
    'font-semibold transition-all duration-200 ease-premium',
    'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2',
    'disabled:pointer-events-none disabled:opacity-50',
    '[&_svg]:pointer-events-none [&_svg]:size-4 [&_svg]:shrink-0',
    'active:scale-[0.98]',
  ],
  {
    variants: {
      variant: {
        default: [
          'bg-primary text-primary-foreground',
          'hover:bg-primary/90',
          'shadow-md hover:shadow-lg',
        ],
        primary: [
          'bg-primary text-primary-foreground',
          'hover:bg-primary/90',
          'shadow-md hover:shadow-lg',
        ],
        destructive: [
          'bg-destructive text-destructive-foreground',
          'hover:bg-destructive/90',
          'shadow-md hover:shadow-lg',
        ],
        outline: [
          'border-2 border-border bg-transparent text-foreground',
          'hover:bg-secondary hover:border-primary/30',
        ],
        secondary: [
          'bg-secondary text-secondary-foreground',
          'hover:bg-secondary/80',
        ],
        ghost: [
          'text-foreground',
          'hover:bg-secondary',
        ],
        link: [
          'text-primary underline-offset-4',
          'hover:underline',
          'p-0 h-auto',
        ],
        danger: [
          'bg-destructive text-destructive-foreground',
          'hover:bg-destructive/90',
          'shadow-md hover:shadow-lg',
        ],
        success: [
          'bg-success text-success-foreground',
          'hover:bg-success/90',
          'shadow-md hover:shadow-lg',
        ],
        'brand-outline': [
          'border-2 border-primary bg-transparent text-primary',
          'hover:bg-primary/5',
        ],
      },
      size: {
        xs: 'h-7 px-2 text-xs rounded-md',
        sm: 'h-8 px-3 text-sm rounded-lg',
        md: 'h-10 px-4 text-sm rounded-lg',
        default: 'h-10 px-4 text-sm rounded-lg',
        lg: 'h-11 px-5 text-base rounded-xl',
        xl: 'h-12 px-6 text-base rounded-xl',
        icon: 'h-10 w-10 rounded-lg',
        'icon-sm': 'h-8 w-8 rounded-lg',
        'icon-lg': 'h-12 w-12 rounded-xl',
      },
      fullWidth: {
        true: 'w-full',
        false: '',
      },
    },
    defaultVariants: {
      variant: 'default',
      size: 'md',
      fullWidth: false,
    },
  }
);

/** Map local variant names to design system variant names */
type DSVariant = 'default' | 'destructive' | 'outline' | 'secondary' | 'ghost' | 'link';
type DSSize = 'default' | 'sm' | 'lg' | 'icon';

function mapVariant(variant?: string | null): DSVariant {
  switch (variant) {
    case 'primary':
    case 'default':
      return 'default';
    case 'danger':
    case 'destructive':
      return 'destructive';
    case 'outline':
    case 'brand-outline':
      return 'outline';
    case 'secondary':
      return 'secondary';
    case 'ghost':
      return 'ghost';
    case 'link':
      return 'link';
    default:
      return 'default';
  }
}

function mapSize(size?: string | null): DSSize {
  switch (size) {
    case 'xs':
    case 'sm':
    case 'icon-sm':
      return 'sm';
    case 'lg':
    case 'xl':
    case 'icon-lg':
      return 'lg';
    case 'icon':
      return 'icon';
    default:
      return 'default';
  }
}

export interface ButtonProps
  extends React.ButtonHTMLAttributes<HTMLButtonElement>,
    VariantProps<typeof buttonVariants> {
  asChild?: boolean;
  isLoading?: boolean;
  leftIcon?: React.ReactNode;
  rightIcon?: React.ReactNode;
}

const Button = forwardRef<HTMLButtonElement, ButtonProps>(
  (
    {
      className,
      variant,
      size,
      fullWidth,
      asChild = false,
      isLoading = false,
      leftIcon,
      rightIcon,
      children,
      disabled,
      ...props
    },
    ref
  ) => {
    // For simple cases that map cleanly to DS, delegate to DSButton
    const needsLocalStyling =
      variant === 'success' ||
      variant === 'brand-outline' ||
      variant === 'danger' ||
      variant === 'primary' ||
      size === 'xs' ||
      size === 'xl' ||
      size === 'icon-sm' ||
      size === 'icon-lg' ||
      size === 'md' ||
      fullWidth ||
      leftIcon ||
      rightIcon ||
      isLoading;

    if (!needsLocalStyling && !asChild) {
      return (
        <DSButton
          ref={ref}
          variant={mapVariant(variant)}
          size={mapSize(size)}
          className={className}
          disabled={disabled}
          {...props}
        >
          {children}
        </DSButton>
      );
    }

    // For extended variants/features, use local CVA with wrapper
    const Comp = asChild ? Slot : 'button';

    return (
      <Comp
        className={cn(buttonVariants({ variant, size, fullWidth, className }))}
        ref={ref}
        disabled={disabled || isLoading}
        {...props}
      >
        {isLoading ? (
          <>
            <Loader2 className="h-4 w-4 animate-spin" />
            <span>{children}</span>
          </>
        ) : (
          <>
            {leftIcon && <span className="shrink-0">{leftIcon}</span>}
            {children}
            {rightIcon && <span className="shrink-0">{rightIcon}</span>}
          </>
        )}
      </Comp>
    );
  }
);

Button.displayName = 'Button';

export { Button, buttonVariants, dsButtonVariants };
