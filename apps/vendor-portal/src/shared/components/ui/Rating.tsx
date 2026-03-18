import { forwardRef } from 'react';
import { Star } from 'lucide-react';
import { cva, type VariantProps } from 'class-variance-authority';
import { cn } from '@tesserix/web';

const ratingVariants = cva(
  'inline-flex items-center gap-0.5',
  {
    variants: {
      size: {
        xs: '[&_svg]:h-3 [&_svg]:w-3',
        sm: '[&_svg]:h-4 [&_svg]:w-4',
        md: '[&_svg]:h-5 [&_svg]:w-5',
        lg: '[&_svg]:h-6 [&_svg]:w-6',
        xl: '[&_svg]:h-7 [&_svg]:w-7',
      },
    },
    defaultVariants: {
      size: 'md',
    },
  }
);

export interface RatingProps
  extends Omit<React.HTMLAttributes<HTMLDivElement>, 'onChange'>,
    VariantProps<typeof ratingVariants> {
  value: number;
  max?: number;
  showValue?: boolean;
  showCount?: boolean;
  count?: number;
  readonly?: boolean;
  onValueChange?: (value: number) => void;
}

const Rating = forwardRef<HTMLDivElement, RatingProps>(
  (
    {
      className,
      size,
      value,
      max = 5,
      showValue = false,
      showCount = false,
      count,
      readonly = true,
      onValueChange,
      ...props
    },
    ref
  ) => {
    const stars = [];
    const fullStars = Math.floor(value);
    const hasHalfStar = value % 1 >= 0.5;

    for (let i = 1; i <= max; i++) {
      if (i <= fullStars) {
        stars.push(
          <Star
            key={i}
            className={cn(
              'fill-warning text-warning',
              !readonly && 'cursor-pointer hover:scale-110 transition-transform'
            )}
            onClick={() => !readonly && onValueChange?.(i)}
          />
        );
      } else if (i === fullStars + 1 && hasHalfStar) {
        stars.push(
          <div key={i} className="relative">
            <Star className="text-border" />
            <div className="absolute inset-0 overflow-hidden w-1/2">
              <Star className="fill-warning text-warning" />
            </div>
          </div>
        );
      } else {
        stars.push(
          <Star
            key={i}
            className={cn(
              'text-border',
              !readonly && 'cursor-pointer hover:text-warning/60 transition-colors'
            )}
            onClick={() => !readonly && onValueChange?.(i)}
          />
        );
      }
    }

    return (
      <div
        ref={ref}
        className={cn(ratingVariants({ size }), 'items-center', className)}
        {...props}
      >
        {stars}
        {showValue && (
          <span className="ml-1.5 font-medium text-foreground">
            {value.toFixed(1)}
          </span>
        )}
        {showCount && count !== undefined && (
          <span className="ml-1 text-muted-foreground">
            ({count.toLocaleString()})
          </span>
        )}
      </div>
    );
  }
);

Rating.displayName = 'Rating';

// Rating Badge
const ratingBadgeVariants = cva(
  [
    'inline-flex items-center gap-1 font-medium',
    'bg-accent text-accent-foreground',
    'rounded-lg',
  ],
  {
    variants: {
      size: {
        sm: 'px-1.5 py-0.5 text-xs [&_svg]:h-3 [&_svg]:w-3',
        md: 'px-2 py-1 text-sm [&_svg]:h-4 [&_svg]:w-4',
        lg: 'px-2.5 py-1.5 text-base [&_svg]:h-5 [&_svg]:w-5',
      },
    },
    defaultVariants: {
      size: 'md',
    },
  }
);

export interface RatingBadgeProps
  extends React.HTMLAttributes<HTMLDivElement>,
    VariantProps<typeof ratingBadgeVariants> {
  value: number;
  showCount?: boolean;
  count?: number;
}

const RatingBadge = forwardRef<HTMLDivElement, RatingBadgeProps>(
  ({ className, size, value, showCount, count, ...props }, ref) => (
    <div
      ref={ref}
      className={cn(ratingBadgeVariants({ size }), className)}
      {...props}
    >
      <Star className="fill-warning text-warning" />
      <span>{value.toFixed(1)}</span>
      {showCount && count !== undefined && (
        <span className="opacity-70">({count})</span>
      )}
    </div>
  )
);

RatingBadge.displayName = 'RatingBadge';

// Simple Star Rating Display
interface StarRatingProps {
  rating: number;
  size?: 'xs' | 'sm' | 'md' | 'lg';
  className?: string;
}

const StarRating = ({ rating, size = 'sm', className }: StarRatingProps) => {
  const sizeClasses = {
    xs: 'h-3 w-3',
    sm: 'h-4 w-4',
    md: 'h-5 w-5',
    lg: 'h-6 w-6',
  };

  return (
    <div className={cn('inline-flex items-center gap-0.5', className)}>
      {[1, 2, 3, 4, 5].map((star) => (
        <Star
          key={star}
          className={cn(
            sizeClasses[size],
            star <= rating
              ? 'fill-warning text-warning'
              : 'text-border'
          )}
        />
      ))}
    </div>
  );
};

// Compact Rating
interface CompactRatingProps {
  value: number;
  className?: string;
}

const CompactRating = ({ value, className }: CompactRatingProps) => (
  <div className={cn('inline-flex items-center gap-1', className)}>
    <Star className="h-4 w-4 fill-warning text-warning" />
    <span className="font-medium text-foreground">{value.toFixed(1)}</span>
  </div>
);

export { Rating, RatingBadge, StarRating, CompactRating, ratingVariants, ratingBadgeVariants };
