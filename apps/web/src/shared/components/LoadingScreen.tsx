import { Loader2 } from 'lucide-react';

interface LoadingScreenProps {
  message?: string;
}

export function LoadingScreen({ message = 'Loading...' }: LoadingScreenProps) {
  return (
    <div className="fixed inset-0 z-50 flex flex-col items-center justify-center bg-white">
      <div className="flex flex-col items-center gap-4">
        {/* Logo */}
        <div className="flex items-center gap-2">
          <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-brand-500">
            <span className="text-2xl font-bold text-white">H</span>
          </div>
          <span className="text-2xl font-bold text-gray-900">Fe3dr</span>
        </div>

        {/* Spinner */}
        <Loader2 className="h-8 w-8 animate-spin text-brand-500" />

        {/* Message */}
        <p className="text-sm text-gray-500">{message}</p>
      </div>
    </div>
  );
}

export function LoadingSpinner({ className = '' }: { className?: string }) {
  return (
    <Loader2
      className={`h-5 w-5 animate-spin text-brand-500 ${className}`}
    />
  );
}

export function PageLoader() {
  return (
    <div className="flex h-[50vh] items-center justify-center">
      <LoadingSpinner className="h-8 w-8" />
    </div>
  );
}
