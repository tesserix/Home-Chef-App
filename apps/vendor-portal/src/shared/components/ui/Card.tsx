import { forwardRef } from 'react';
import { cva, type VariantProps } from 'class-variance-authority';
import { cn } from '@tesserix/web';

/**
 * Card components — kept as local implementation since the vendor portal
 * uses extended variants (elevated, outlined, filled, ghost, premium)
 * and additional props (padding, hover) not in the design system.
 */
const cardVariants = cva(
  ['rounded-xl transition-all duration-200 ease-premium'],
  {
    variants: {
      variant: {
        default: 'border border-border bg-card text-card-foreground shadow-sm',
        elevated: 'bg-card text-card-foreground shadow-elevated',
        outlined: 'bg-card text-card-foreground border border-border',
        filled: 'bg-secondary text-secondary-foreground',
        ghost: 'bg-transparent',
        premium: 'bg-card text-card-foreground shadow-soft-lg ring-1 ring-warning/20',
      },
      padding: {
        none: '',
        sm: 'p-4',
        md: 'p-6',
        lg: 'p-8',
      },
      hover: {
        none: '',
        lift: 'hover:shadow-card-hover hover:-translate-y-1 cursor-pointer',
        border: 'hover:border-primary/30 cursor-pointer',
      },
    },
    defaultVariants: {
      variant: 'default',
      padding: 'md',
      hover: 'none',
    },
  }
);

export interface CardProps
  extends React.HTMLAttributes<HTMLDivElement>,
    VariantProps<typeof cardVariants> {}

const Card = forwardRef<HTMLDivElement, CardProps>(
  ({ className, variant, padding, hover, ...props }, ref) => (
    <div
      ref={ref}
      className={cn(cardVariants({ variant, padding, hover, className }))}
      {...props}
    />
  )
);
Card.displayName = 'Card';

const CardHeader = forwardRef<
  HTMLDivElement,
  React.HTMLAttributes<HTMLDivElement>
>(({ className, ...props }, ref) => (
  <div
    ref={ref}
    className={cn('flex flex-col space-y-1.5', className)}
    {...props}
  />
));
CardHeader.displayName = 'CardHeader';

const CardTitle = forwardRef<
  HTMLHeadingElement,
  React.HTMLAttributes<HTMLHeadingElement>
>(({ className, ...props }, ref) => (
  <h3
    ref={ref}
    className={cn('text-2xl font-bold leading-tight tracking-tight', className)}
    {...props}
  />
));
CardTitle.displayName = 'CardTitle';

const CardDescription = forwardRef<
  HTMLParagraphElement,
  React.HTMLAttributes<HTMLParagraphElement>
>(({ className, ...props }, ref) => (
  <p
    ref={ref}
    className={cn('text-sm text-muted-foreground', className)}
    {...props}
  />
));
CardDescription.displayName = 'CardDescription';

const CardContent = forwardRef<
  HTMLDivElement,
  React.HTMLAttributes<HTMLDivElement>
>(({ className, ...props }, ref) => (
  <div ref={ref} className={cn('', className)} {...props} />
));
CardContent.displayName = 'CardContent';

const CardFooter = forwardRef<
  HTMLDivElement,
  React.HTMLAttributes<HTMLDivElement>
>(({ className, ...props }, ref) => (
  <div
    ref={ref}
    className={cn('flex items-center pt-4', className)}
    {...props}
  />
));
CardFooter.displayName = 'CardFooter';

const CardImage = forwardRef<
  HTMLDivElement,
  React.HTMLAttributes<HTMLDivElement> & { aspectRatio?: 'video' | 'square' | 'portrait' }
>(({ className, aspectRatio = 'video', children, ...props }, ref) => {
  const aspectClasses = {
    video: 'aspect-video',
    square: 'aspect-square',
    portrait: 'aspect-[3/4]',
  };

  return (
    <div
      ref={ref}
      className={cn(
        'relative overflow-hidden rounded-t-xl -mx-6 -mt-6 mb-4',
        aspectClasses[aspectRatio],
        className
      )}
      {...props}
    >
      {children}
    </div>
  );
});
CardImage.displayName = 'CardImage';

export {
  Card,
  CardHeader,
  CardTitle,
  CardDescription,
  CardContent,
  CardFooter,
  CardImage,
  cardVariants,
};
