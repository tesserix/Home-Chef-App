/**
 * ErrorBoundary — Top-level safety net.
 *
 * Catches uncaught render errors in any descendant component tree and shows
 * a fallback UI instead of a white screen. Errors are surfaced to whatever
 * error monitoring you wire up via the `onError` prop.
 *
 * Usage:
 *   <ErrorBoundary>
 *     <App />
 *   </ErrorBoundary>
 *
 *   // Or with a custom fallback:
 *   <ErrorBoundary fallback={<CustomError />}>...</ErrorBoundary>
 */

import { Component, type ErrorInfo, type ReactNode } from 'react';

interface ErrorBoundaryProps {
  children: ReactNode;
  /** Custom fallback UI. Omit to use the default. */
  fallback?: ReactNode;
  /** Called on every caught error — wire up to Sentry / Datadog / etc. */
  onError?: (error: Error, info: ErrorInfo) => void;
}

interface ErrorBoundaryState {
  error: Error | null;
}

export class ErrorBoundary extends Component<ErrorBoundaryProps, ErrorBoundaryState> {
  state: ErrorBoundaryState = { error: null };

  static getDerivedStateFromError(error: Error): ErrorBoundaryState {
    return { error };
  }

  componentDidCatch(error: Error, info: ErrorInfo): void {
    this.props.onError?.(error, info);
  }

  reset = (): void => {
    this.setState({ error: null });
  };

  render(): ReactNode {
    if (this.state.error) {
      return this.props.fallback ?? <DefaultErrorFallback error={this.state.error} onRetry={this.reset} />;
    }
    return this.props.children;
  }
}

interface DefaultErrorFallbackProps {
  error: Error;
  onRetry: () => void;
}

function DefaultErrorFallback({ error, onRetry }: DefaultErrorFallbackProps) {
  return (
    <div className="flex min-h-screen items-center justify-center bg-paper px-6 py-12">
      <div className="max-w-md text-center">
        <p className="font-display text-sm font-medium uppercase tracking-wider text-ink-muted">
          Unexpected error
        </p>
        <h1 className="mt-3 font-display text-2xl font-semibold text-foreground sm:text-3xl">
          Something broke on our end.
        </h1>
        <p className="mt-3 text-ink-soft">
          We've been notified and we're looking into it. You can try the action
          again, or refresh the page to start over.
        </p>
        {import.meta.env?.DEV && (
          <details className="mx-auto mt-6 max-w-md rounded-md border border-mist bg-bone p-3 text-left text-xs">
            <summary className="cursor-pointer font-medium text-ink-soft">
              Developer details
            </summary>
            <pre className="mt-2 overflow-auto whitespace-pre-wrap break-words text-paprika">
              {error.name}: {error.message}
              {error.stack ? `\n\n${error.stack}` : null}
            </pre>
          </details>
        )}
        <div className="mt-8 flex flex-col items-center justify-center gap-3 sm:flex-row">
          <button type="button" onClick={onRetry} className="btn-primary">
            Try again
          </button>
          <button
            type="button"
            onClick={() => window.location.reload()}
            className="btn-secondary"
          >
            Refresh page
          </button>
        </div>
      </div>
    </div>
  );
}
