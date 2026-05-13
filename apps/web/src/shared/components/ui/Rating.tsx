import { forwardRef, type KeyboardEvent } from 'react';
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

function safeToFixed(value: number, digits = 1): string {
  return Number.isFinite(value) ? value.toFixed(digits) : '—';
}

function clamp(value: number, min: number, max: number): number {
  return Math.min(max, Math.max(min, value));
}

function buildAriaLabel(value: number, max: number, count?: number): string {
  const stars = `${safeToFixed(value, 1)} out of ${max} stars`;
  if (count !== undefined && Number.isFinite(count)) {
    return `${stars}, ${count.toLocaleString()} review${count === 1 ? '' : 's'}`;
  }
  return stars;
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
      'aria-label': ariaLabelOverride,
      ...props
    },
    ref
  ) => {
    const safeValue = Number.isFinite(value) ? value : 0;
    const fullStars = Math.floor(safeValue);
    const hasHalfStar = safeValue % 1 >= 0.5;
    const ariaLabel = ariaLabelOverride ?? buildAriaLabel(safeValue, max, count);

    const stars: React.ReactNode[] = [];

    for (let i = 1; i <= max; i++) {
      const isFull = i <= fullStars;
      const isHalf = i === fullStars + 1 && hasHalfStar;

      const visualStar = isFull ? (
        <Star className="fill-warning text-warning"  aria-hidden="true" />
      ) : isHalf ? (
        <span className="relative inline-flex">
          <Star className="text-border"  aria-hidden="true" />
          <span className="absolute inset-0 overflow-hidden w-1/2">
            <Star className="fill-warning text-warning"  aria-hidden="true" />
          </span>
        </span>
      ) : (
        <Star className="text-border"  aria-hidden="true" />
      );

      if (readonly) {
        stars.push(
          <span key={i} aria-hidden="true" className="inline-flex">
            {visualStar}
          </span>
        );
      } else {
        const ariaChecked = Math.round(safeValue) === i;
        const onKey = (e: KeyboardEvent<HTMLButtonElement>) => {
          let next: number | null = null;
          if (e.key === 'ArrowRight' || e.key === 'ArrowUp') next = clamp(i + 1, 1, max);
          else if (e.key === 'ArrowLeft' || e.key === 'ArrowDown') next = clamp(i - 1, 1, max);
          else if (e.key === 'Home') next = 1;
          else if (e.key === 'End') next = max;
          else if (e.key === ' ' || e.key === 'Enter') next = i;
          if (next !== null) {
            e.preventDefault();
            onValueChange?.(next);
          }
        };
        stars.push(
          <button
            key={i}
            type="button"
            role="radio"
            aria-checked={ariaChecked}
            aria-label={`Rate ${i} of ${max}`}
            tabIndex={ariaChecked || (i === 1 && safeValue === 0) ? 0 : -1}
            onClick={() => onValueChange?.(i)}
            onKeyDown={onKey}
            className="relative inline-flex h-6 min-w-[24px] items-center justify-center rounded-md transition-transform focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-1 active:scale-95"
          >
            {visualStar}
          </button>
        );
      }
    }

    if (readonly) {
      return (
        <div
          ref={ref}
          role="img"
          aria-label={ariaLabel}
          className={cn(ratingVariants({ size }), 'items-center', className)}
          {...props}
        >
          {stars}
          {showValue && (
            <span aria-hidden="true" className="ml-1.5 font-medium text-foreground tabular-nums">
              {safeToFixed(safeValue, 1)}
            </span>
          )}
          {showCount && count !== undefined && Number.isFinite(count) && (
            <span aria-hidden="true" className="ml-1 text-muted-foreground tabular-nums">
              ({count.toLocaleString()})
            </span>
          )}
        </div>
      );
    }

    return (
      <div
        ref={ref}
        role="radiogroup"
        aria-label={ariaLabel}
        className={cn(ratingVariants({ size }), 'items-center', className)}
        {...props}
      >
        {stars}
        {showValue && (
          <span aria-hidden="true" className="ml-1.5 font-medium text-foreground tabular-nums">
            {safeToFixed(safeValue, 1)}
          </span>
        )}
        {showCount && count !== undefined && Number.isFinite(count) && (
          <span aria-hidden="true" className="ml-1 text-muted-foreground tabular-nums">
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
  ({ className, size, value, showCount, count, ...props }, ref) => {
    const ariaLabel = buildAriaLabel(value, 5, count);
    const safeCount = Number.isFinite(count) ? count : undefined;
    return (
      <div
        ref={ref}
        role="img"
        aria-label={ariaLabel}
        className={cn(ratingBadgeVariants({ size }), className)}
        {...props}
      >
        <Star aria-hidden="true" className="fill-warning text-warning" />
        <span aria-hidden="true" className="tabular-nums">
          {safeToFixed(value, 1)}
        </span>
        {showCount && safeCount !== undefined && (
          <span aria-hidden="true" className="opacity-70 tabular-nums">
            ({safeCount.toLocaleString()})
          </span>
        )}
      </div>
    );
  }
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
  const safe = Number.isFinite(rating) ? rating : 0;

  return (
    <div
      role="img"
      aria-label={buildAriaLabel(safe, 5)}
      className={cn('inline-flex items-center gap-0.5', className)}
    >
      {[1, 2, 3, 4, 5].map((star) => (
        <Star
          key={star}
          aria-hidden="true"
          className={cn(
            sizeClasses[size],
            star <= safe
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
  <div
    role="img"
    aria-label={buildAriaLabel(value, 5)}
    className={cn('inline-flex items-center gap-1', className)}
  >
    <Star aria-hidden="true" className="h-4 w-4 fill-warning text-warning" />
    <span aria-hidden="true" className="font-medium text-foreground tabular-nums">
      {safeToFixed(value, 1)}
    </span>
  </div>
);

export { Rating, RatingBadge, StarRating, CompactRating, ratingVariants, ratingBadgeVariants };
