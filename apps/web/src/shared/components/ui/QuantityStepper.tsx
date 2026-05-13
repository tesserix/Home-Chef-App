import { forwardRef } from 'react';
import { Minus, Plus } from 'lucide-react';
import { cva, type VariantProps } from 'class-variance-authority';
import { cn } from '@/shared/utils/cn';

const quantityStepperVariants = cva(
  'inline-flex items-center',
  {
    variants: {
      variant: {
        default: 'bg-secondary rounded-xl',
        outlined: 'border border-border rounded-xl',
        filled: 'bg-primary/10 rounded-xl',
      },
      size: {
        sm: 'h-8 gap-1',
        md: 'h-10 gap-2',
        lg: 'h-12 gap-2',
      },
    },
    defaultVariants: {
      variant: 'default',
      size: 'md',
    },
  }
);

const stepperButtonVariants = cva(
  [
    'flex items-center justify-center',
    'transition-all duration-200',
    'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/50',
    'disabled:opacity-40 disabled:cursor-not-allowed',
  ],
  {
    variants: {
      variant: {
        default: 'text-secondary-foreground hover:bg-secondary/80 hover:text-foreground',
        outlined: 'text-secondary-foreground hover:bg-secondary hover:text-foreground',
        filled: 'text-primary hover:bg-primary/20 hover:text-primary',
      },
      size: {
        sm: 'w-7 h-7 rounded-lg [&_svg]:h-3 [&_svg]:w-3',
        md: 'w-9 h-9 rounded-lg [&_svg]:h-4 [&_svg]:w-4',
        lg: 'w-11 h-11 rounded-xl [&_svg]:h-5 [&_svg]:w-5',
      },
    },
    defaultVariants: {
      variant: 'default',
      size: 'md',
    },
  }
);

export interface QuantityStepperProps
  extends Omit<React.HTMLAttributes<HTMLDivElement>, 'onChange'>,
    VariantProps<typeof quantityStepperVariants> {
  value: number;
  onChange: (value: number) => void;
  min?: number;
  max?: number;
  step?: number;
  disabled?: boolean;
  /** Describes what the stepper controls. Required for screen-reader users. */
  'aria-label'?: string;
}

const QuantityStepper = forwardRef<HTMLDivElement, QuantityStepperProps>(
  (
    {
      className,
      variant,
      size,
      value,
      onChange,
      min = 0,
      max = 99,
      step = 1,
      disabled = false,
      'aria-label': ariaLabel = 'Quantity',
      ...props
    },
    ref
  ) => {
    const handleDecrement = () => {
      const newValue = Math.max(min, value - step);
      onChange(newValue);
    };

    const handleIncrement = () => {
      const newValue = Math.min(max, value + step);
      onChange(newValue);
    };

    // min-w lets the display grow past 999 without clipping while still
    // anchoring the layout at the configured size for the common case.
    const displaySizes = {
      sm: 'min-w-[1.75rem] px-1 text-sm',
      md: 'min-w-[2.25rem] px-1.5 text-base',
      lg: 'min-w-[2.75rem] px-2 text-lg',
    };

    return (
      <div
        ref={ref}
        role="group"
        aria-label={ariaLabel}
        className={cn(quantityStepperVariants({ variant, size }), className)}
        {...props}
      >
        <button
          type="button"
          onClick={handleDecrement}
          disabled={disabled || value <= min}
          className={cn(stepperButtonVariants({ variant, size }))}
          aria-label={`Decrease ${ariaLabel.toLowerCase()}`}
        >
          <Minus aria-hidden="true" />
        </button>
        <span
          aria-live="polite"
          aria-atomic="true"
          className={cn(
            'font-medium text-center tabular-nums text-foreground',
            displaySizes[size || 'md']
          )}
        >
          {value}
        </span>
        <button
          type="button"
          onClick={handleIncrement}
          disabled={disabled || value >= max}
          className={cn(stepperButtonVariants({ variant, size }))}
          aria-label={`Increase ${ariaLabel.toLowerCase()}`}
        >
          <Plus aria-hidden="true" />
        </button>
      </div>
    );
  }
);

QuantityStepper.displayName = 'QuantityStepper';

// Compact version
const CompactQuantityStepper = forwardRef<HTMLDivElement, QuantityStepperProps>(
  (
    {
      className,
      value,
      onChange,
      min = 0,
      max = 99,
      disabled = false,
      'aria-label': ariaLabel = 'Quantity',
      ...props
    },
    ref
  ) => (
    <div
      ref={ref}
      role="group"
      aria-label={ariaLabel}
      className={cn('inline-flex items-center gap-2', className)}
      {...props}
    >
      <button
        type="button"
        onClick={() => onChange(Math.max(min, value - 1))}
        disabled={disabled || value <= min}
        className="flex h-8 w-8 items-center justify-center rounded-lg bg-secondary text-secondary-foreground hover:bg-secondary/80 disabled:opacity-40 disabled:cursor-not-allowed transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/50"
        aria-label={`Decrease ${ariaLabel.toLowerCase()}`}
      >
        <Minus aria-hidden="true" className="h-4 w-4" />
      </button>
      <span
        aria-live="polite"
        aria-atomic="true"
        className="min-w-[2rem] px-1 text-center font-medium text-foreground tabular-nums"
      >
        {value}
      </span>
      <button
        type="button"
        onClick={() => onChange(Math.min(max, value + 1))}
        disabled={disabled || value >= max}
        className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary text-primary-foreground hover:bg-primary/90 disabled:opacity-40 disabled:cursor-not-allowed transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/50"
        aria-label={`Increase ${ariaLabel.toLowerCase()}`}
      >
        <Plus aria-hidden="true" className="h-4 w-4" />
      </button>
    </div>
  )
);

CompactQuantityStepper.displayName = 'CompactQuantityStepper';

// Add to cart button with quantity
interface AddToCartQuantityProps {
  quantity: number;
  onQuantityChange: (quantity: number) => void;
  onAddToCart: () => void;
  isInCart?: boolean;
  disabled?: boolean;
  className?: string;
}

const AddToCartQuantity = ({
  quantity,
  onQuantityChange,
  onAddToCart,
  isInCart = false,
  disabled = false,
  className,
}: AddToCartQuantityProps) => {
  if (isInCart && quantity > 0) {
    return (
      <QuantityStepper
        variant="filled"
        size="sm"
        value={quantity}
        onChange={onQuantityChange}
        min={0}
        disabled={disabled}
        className={className}
      />
    );
  }

  return (
    <button
      type="button"
      onClick={onAddToCart}
      disabled={disabled}
      className={cn(
        'flex h-9 w-9 items-center justify-center',
        'rounded-xl bg-primary text-primary-foreground',
        'hover:bg-primary/90 active:scale-95',
        'transition-all duration-200',
        'disabled:opacity-50 disabled:cursor-not-allowed',
        className
      )}
      aria-label="Add to cart"
    >
      <Plus className="h-5 w-5"  aria-hidden="true" />
    </button>
  );
};

export { QuantityStepper, CompactQuantityStepper, AddToCartQuantity, quantityStepperVariants };
