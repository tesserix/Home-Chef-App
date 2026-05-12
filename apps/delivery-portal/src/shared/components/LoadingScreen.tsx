import { Loader2 } from 'lucide-react';

interface LoadingScreenProps {
  message?: string;
}

export function LoadingScreen({ message = 'Loading...' }: LoadingScreenProps) {
  return (
    <div
      role="status"
      aria-live="polite"
      aria-busy="true"
      className="fixed inset-0 z-50 flex flex-col items-center justify-center bg-bone"
    >
      <div className="flex flex-col items-center gap-4">
        <div className="flex items-center gap-2" aria-hidden="true">
          <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-herb">
            <span className="font-display text-2xl font-semibold text-paper">D</span>
          </div>
          <span className="font-display text-2xl font-semibold text-ink">Fe3dr</span>
        </div>
        <Loader2 aria-hidden="true" className="h-8 w-8 animate-spin text-herb" />
        <p className="text-sm text-ink-muted">{message}</p>
      </div>
      <span className="sr-only">{message}</span>
    </div>
  );
}

export function LoadingSpinner({
  className = '',
  label,
}: {
  className?: string;
  label?: string;
}) {
  return (
    <>
      <Loader2 aria-hidden="true" className={`h-5 w-5 animate-spin text-herb ${className}`} />
      {label ? <span className="sr-only">{label}</span> : null}
    </>
  );
}

export function PageLoader({ label = 'Loading...' }: { label?: string } = {}) {
  return (
    <div
      role="status"
      aria-live="polite"
      aria-busy="true"
      className="flex h-[50vh] items-center justify-center"
    >
      <LoadingSpinner className="h-8 w-8" label={label} />
    </div>
  );
}
