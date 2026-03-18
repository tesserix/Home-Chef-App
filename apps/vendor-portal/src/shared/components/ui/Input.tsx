import { forwardRef } from 'react';
import { Input as DSInput, Label as DSLabel } from '@tesserix/web';
import { cn } from '@tesserix/web';
import { cva, type VariantProps } from 'class-variance-authority';

const inputVariants = cva(
  [
    'w-full',
    'text-foreground placeholder:text-muted-foreground',
    'transition-all duration-200 ease-premium',
    'focus-visible:outline-none focus-visible:ring-4 focus-visible:ring-ring/20',
    'disabled:cursor-not-allowed disabled:opacity-50 disabled:bg-muted',
    'file:border-0 file:bg-transparent file:text-sm file:font-medium file:text-foreground',
  ],
  {
    variants: {
      variant: {
        default: [
          'border-2 border-input bg-background shadow-sm',
          'hover:border-primary/30',
          'focus-visible:border-ring',
        ],
        filled: [
          'border-2 border-transparent bg-secondary',
          'hover:bg-secondary/80',
          'focus-visible:bg-background focus-visible:border-ring',
        ],
        ghost: [
          'border-2 border-transparent bg-transparent',
          'hover:bg-secondary',
          'focus-visible:bg-background focus-visible:border-ring',
        ],
      },
      inputSize: {
        sm: 'h-9 px-3 text-sm rounded-md',
        md: 'h-10 px-4 text-sm rounded-lg',
        lg: 'h-11 px-4 text-base rounded-lg',
        xl: 'h-12 px-5 text-base rounded-xl',
      },
      hasError: {
        true: [
          'border-destructive text-destructive',
          'focus-visible:border-destructive focus-visible:ring-destructive/20',
          'placeholder:text-destructive/50',
        ],
        false: '',
      },
    },
    defaultVariants: {
      variant: 'default',
      inputSize: 'md',
      hasError: false,
    },
  }
);

export interface InputProps
  extends Omit<React.InputHTMLAttributes<HTMLInputElement>, 'size'>,
    VariantProps<typeof inputVariants> {
  leftIcon?: React.ReactNode;
  rightIcon?: React.ReactNode;
  error?: string;
  label?: string;
  hint?: string;
}

const Input = forwardRef<HTMLInputElement, InputProps>(
  (
    {
      className,
      variant,
      inputSize,
      hasError,
      leftIcon,
      rightIcon,
      error,
      label,
      hint,
      id,
      ...props
    },
    ref
  ) => {
    const inputId = id || label?.toLowerCase().replace(/\s+/g, '-');
    const showError = error || hasError;
    const needsWrapper = label || error || hint || leftIcon || rightIcon;

    // Simple case: no wrapper needed, delegate to DS Input
    if (!needsWrapper && variant === 'default' && !inputSize) {
      return (
        <DSInput
          id={inputId}
          ref={ref}
          className={cn(showError && 'border-destructive', className)}
          {...props}
        />
      );
    }

    // Extended case: wrap DS input with label/error/hint/icons
    return (
      <div className="w-full">
        {label && (
          <DSLabel
            htmlFor={inputId}
            className="mb-1.5 block text-sm font-medium text-foreground"
          >
            {label}
          </DSLabel>
        )}
        <div className="relative">
          {leftIcon && (
            <div className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground">
              {leftIcon}
            </div>
          )}
          <input
            id={inputId}
            className={cn(
              inputVariants({ variant, inputSize, hasError: !!showError }),
              leftIcon && 'pl-10',
              rightIcon && 'pr-10',
              className
            )}
            ref={ref}
            {...props}
          />
          {rightIcon && (
            <div className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground">
              {rightIcon}
            </div>
          )}
        </div>
        {error && (
          <p className="mt-1.5 text-sm text-destructive">{error}</p>
        )}
        {hint && !error && (
          <p className="mt-1.5 text-sm text-muted-foreground">{hint}</p>
        )}
      </div>
    );
  }
);

Input.displayName = 'Input';

// Textarea - kept local as DS may not have an equivalent
const textareaVariants = cva(
  [
    'w-full min-h-[80px] resize-y',
    'text-foreground placeholder:text-muted-foreground',
    'transition-all duration-200 ease-premium',
    'focus-visible:outline-none focus-visible:ring-4 focus-visible:ring-ring/20',
    'disabled:cursor-not-allowed disabled:opacity-50 disabled:bg-muted',
  ],
  {
    variants: {
      variant: {
        default: [
          'border-2 border-input bg-background shadow-sm',
          'hover:border-primary/30',
          'focus-visible:border-ring',
        ],
        filled: [
          'border-2 border-transparent bg-secondary',
          'hover:bg-secondary/80',
          'focus-visible:bg-background focus-visible:border-ring',
        ],
      },
      hasError: {
        true: [
          'border-destructive',
          'focus-visible:border-destructive focus-visible:ring-destructive/20',
        ],
        false: '',
      },
    },
    defaultVariants: {
      variant: 'default',
      hasError: false,
    },
  }
);

export interface TextareaProps
  extends React.TextareaHTMLAttributes<HTMLTextAreaElement>,
    VariantProps<typeof textareaVariants> {
  error?: string;
  label?: string;
  hint?: string;
}

const Textarea = forwardRef<HTMLTextAreaElement, TextareaProps>(
  ({ className, variant, hasError, error, label, hint, id, ...props }, ref) => {
    const inputId = id || label?.toLowerCase().replace(/\s+/g, '-');
    const showError = error || hasError;

    return (
      <div className="w-full">
        {label && (
          <DSLabel
            htmlFor={inputId}
            className="mb-1.5 block text-sm font-medium text-foreground"
          >
            {label}
          </DSLabel>
        )}
        <textarea
          id={inputId}
          className={cn(
            textareaVariants({ variant, hasError: !!showError }),
            'p-4 rounded-lg',
            className
          )}
          ref={ref}
          {...props}
        />
        {error && (
          <p className="mt-1.5 text-sm text-destructive">{error}</p>
        )}
        {hint && !error && (
          <p className="mt-1.5 text-sm text-muted-foreground">{hint}</p>
        )}
      </div>
    );
  }
);

Textarea.displayName = 'Textarea';

export { Input, Textarea, inputVariants, textareaVariants };
