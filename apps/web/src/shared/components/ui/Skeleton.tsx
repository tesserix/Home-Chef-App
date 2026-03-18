import { Skeleton as DSSkeleton } from '@tesserix/web';
import { cn } from '@tesserix/web';

interface SkeletonProps extends React.HTMLAttributes<HTMLDivElement> {
  variant?: 'default' | 'circular' | 'rectangular' | 'text';
  width?: string | number;
  height?: string | number;
  animation?: 'pulse' | 'shimmer' | 'none';
}

function Skeleton({
  className,
  variant = 'default',
  width,
  height,
  animation = 'pulse',
  style,
  ...props
}: SkeletonProps) {
  const variantClasses = {
    default: 'rounded-lg',
    circular: 'rounded-full',
    rectangular: 'rounded-none',
    text: 'rounded h-4',
  };

  const animationClasses = {
    pulse: 'animate-pulse',
    shimmer: 'relative overflow-hidden before:absolute before:inset-0 before:-translate-x-full before:animate-shimmer before:bg-shimmer',
    none: '',
  };

  // For simple default usage, delegate to DS Skeleton
  if (variant === 'default' && animation === 'pulse' && !width && !height) {
    return <DSSkeleton className={className} style={style} {...props} />;
  }

  return (
    <div
      className={cn(
        'bg-muted',
        variantClasses[variant],
        animationClasses[animation],
        className
      )}
      style={{
        width: typeof width === 'number' ? `${width}px` : width,
        height: typeof height === 'number' ? `${height}px` : height,
        ...style,
      }}
      {...props}
    />
  );
}

function SkeletonText({
  lines = 3,
  className,
}: {
  lines?: number;
  className?: string;
}) {
  return (
    <div className={cn('space-y-2', className)}>
      {Array.from({ length: lines }).map((_, i) => (
        <Skeleton
          key={i}
          variant="text"
          className={cn(
            'h-4',
            i === lines - 1 && lines > 1 && 'w-4/5'
          )}
        />
      ))}
    </div>
  );
}

function SkeletonAvatar({
  size = 'md',
  className,
}: {
  size?: 'xs' | 'sm' | 'md' | 'lg' | 'xl' | '2xl';
  className?: string;
}) {
  const sizes = {
    xs: 'h-6 w-6',
    sm: 'h-8 w-8',
    md: 'h-10 w-10',
    lg: 'h-12 w-12',
    xl: 'h-16 w-16',
    '2xl': 'h-20 w-20',
  };

  return (
    <Skeleton
      variant="circular"
      className={cn(sizes[size], className)}
    />
  );
}

function SkeletonButton({
  size = 'md',
  fullWidth = false,
  className,
}: {
  size?: 'sm' | 'md' | 'lg';
  fullWidth?: boolean;
  className?: string;
}) {
  const sizes = {
    sm: 'h-8 w-20',
    md: 'h-10 w-24',
    lg: 'h-12 w-28',
  };

  return (
    <Skeleton
      className={cn(
        sizes[size],
        'rounded-lg',
        fullWidth && 'w-full',
        className
      )}
    />
  );
}

function SkeletonImage({
  aspectRatio = 'video',
  className,
}: {
  aspectRatio?: 'square' | 'video' | 'portrait';
  className?: string;
}) {
  const aspectClasses = {
    square: 'aspect-square',
    video: 'aspect-video',
    portrait: 'aspect-[3/4]',
  };

  return (
    <Skeleton
      className={cn('w-full rounded-xl', aspectClasses[aspectRatio], className)}
    />
  );
}

function SkeletonCard({ className }: { className?: string }) {
  return (
    <div className={cn('rounded-xl bg-card border border-border p-6 shadow-sm', className)}>
      <SkeletonImage aspectRatio="video" className="mb-4" />
      <Skeleton className="h-5 w-3/4 mb-2" />
      <Skeleton className="h-4 w-1/2 mb-4" />
      <div className="flex items-center gap-2">
        <SkeletonAvatar size="sm" />
        <div className="flex-1">
          <Skeleton className="h-3 w-24 mb-1" />
          <Skeleton className="h-3 w-16" />
        </div>
      </div>
    </div>
  );
}

function SkeletonFoodCard({ className }: { className?: string }) {
  return (
    <div className={cn('rounded-xl bg-card border border-border overflow-hidden shadow-sm', className)}>
      <SkeletonImage aspectRatio="square" className="rounded-none" />
      <div className="p-4">
        <div className="flex items-start justify-between mb-2">
          <Skeleton className="h-5 w-2/3" />
          <Skeleton className="h-5 w-12" />
        </div>
        <Skeleton className="h-4 w-full mb-1" />
        <Skeleton className="h-4 w-3/4 mb-3" />
        <div className="flex items-center justify-between">
          <Skeleton className="h-6 w-16" />
          <Skeleton className="h-9 w-9 rounded-lg" />
        </div>
      </div>
    </div>
  );
}

function SkeletonChefCard({ className }: { className?: string }) {
  return (
    <div className={cn('rounded-xl bg-card border border-border overflow-hidden shadow-sm', className)}>
      <Skeleton className="h-32 rounded-none" />
      <div className="p-4 pt-10 relative">
        <SkeletonAvatar
          size="xl"
          className="absolute -top-8 left-4 border-4 border-card"
        />
        <div className="flex items-start justify-between">
          <div className="flex-1 min-w-0">
            <Skeleton className="h-5 w-32 mb-2" />
            <Skeleton className="h-4 w-24" />
          </div>
          <Skeleton className="h-7 w-16 rounded-lg" />
        </div>
        <Skeleton className="h-4 w-full mt-3 mb-1" />
        <Skeleton className="h-4 w-4/5 mb-4" />
        <div className="flex items-center gap-4">
          <Skeleton className="h-4 w-16" />
          <Skeleton className="h-4 w-12" />
        </div>
      </div>
    </div>
  );
}

function SkeletonTableRow({
  columns = 5,
  className,
}: {
  columns?: number;
  className?: string;
}) {
  return (
    <tr className={className}>
      {Array.from({ length: columns }).map((_, i) => (
        <td key={i} className="py-3 px-4">
          <Skeleton className="h-4 w-full max-w-[120px]" />
        </td>
      ))}
    </tr>
  );
}

export {
  Skeleton,
  SkeletonText,
  SkeletonAvatar,
  SkeletonButton,
  SkeletonImage,
  SkeletonCard,
  SkeletonFoodCard,
  SkeletonChefCard,
  SkeletonTableRow,
};
