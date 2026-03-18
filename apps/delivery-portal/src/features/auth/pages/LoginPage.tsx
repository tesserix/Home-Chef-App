import { useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import { useAuth } from '@/app/providers/AuthProvider';
import { Truck, Shield, UserPlus, ArrowLeft } from 'lucide-react';

type LoginMode = 'choose' | 'staff' | 'driver';

export default function LoginPage() {
  const { loginStaff, loginDriver } = useAuth();
  const [searchParams] = useSearchParams();
  const [mode, setMode] = useState<LoginMode>('choose');

  const authError = searchParams.get('error');
  const accessDenied = authError === 'access-denied';
  const sessionExpired = authError === 'session_expired' || authError === 'invalid_state';

  return (
    <div className="flex min-h-screen items-center justify-center bg-background px-4">
      <div className="w-full max-w-md space-y-8">
        <div className="text-center">
          <div className="mx-auto flex h-16 w-16 items-center justify-center rounded-2xl bg-primary">
            <Truck className="h-8 w-8 text-primary-foreground" />
          </div>
          <h1 className="mt-6 text-3xl font-bold text-foreground">Fe3dr Delivery</h1>
          <p className="mt-2 text-muted-foreground">
            {mode === 'choose' && 'Sign in to get started'}
            {mode === 'staff' && 'Staff & fleet manager login'}
            {mode === 'driver' && 'Driver login or sign up'}
          </p>
        </div>

        {/* Error messages */}
        {accessDenied && (
          <div className="rounded-xl border border-destructive/30 bg-destructive/5 p-4 text-sm text-destructive">
            Access denied. Please check your credentials and try again.
          </div>
        )}
        {sessionExpired && (
          <div className="rounded-xl border border-warning/30 bg-warning/5 p-4 text-sm text-warning">
            Your session has expired. Please sign in again.
          </div>
        )}
        {authError && !accessDenied && !sessionExpired && (
          <div className="rounded-xl border border-destructive/30 bg-destructive/5 p-4 text-sm text-destructive">
            Something went wrong. Please try again.
          </div>
        )}

        {/* Mode: Choose */}
        {mode === 'choose' && (
          <div className="space-y-4">
            <button
              onClick={() => setMode('driver')}
              className="flex w-full items-center gap-4 rounded-xl border-2 border-primary bg-primary/5 p-5 text-left transition-all hover:bg-primary/10"
            >
              <div className="flex h-12 w-12 items-center justify-center rounded-full bg-primary/10">
                <Truck className="h-6 w-6 text-primary" />
              </div>
              <div>
                <p className="text-base font-semibold text-foreground">I'm a Driver</p>
                <p className="text-sm text-muted-foreground">
                  Login or sign up to deliver with Fe3dr
                </p>
              </div>
            </button>

            <button
              onClick={() => setMode('staff')}
              className="flex w-full items-center gap-4 rounded-xl border border-border bg-card p-5 text-left transition-all hover:bg-secondary"
            >
              <div className="flex h-12 w-12 items-center justify-center rounded-full bg-muted">
                <Shield className="h-6 w-6 text-muted-foreground" />
              </div>
              <div>
                <p className="text-base font-semibold text-foreground">I'm Staff</p>
                <p className="text-sm text-muted-foreground">
                  Fleet managers & delivery operations
                </p>
              </div>
            </button>
          </div>
        )}

        {/* Mode: Driver Login */}
        {mode === 'driver' && (
          <div className="space-y-4">
            <button
              onClick={() => setMode('choose')}
              className="flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground transition-colors"
            >
              <ArrowLeft className="h-4 w-4" />
              Back
            </button>

            <div className="space-y-3">
              <button
                onClick={() => loginDriver('google')}
                className="flex w-full items-center justify-center gap-3 rounded-xl border border-border bg-card px-4 py-3 text-sm font-medium text-foreground shadow-sm transition-colors hover:bg-secondary"
              >
                <svg className="h-5 w-5" viewBox="0 0 24 24">
                  <path d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92a5.06 5.06 0 01-2.2 3.32v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.1z" fill="#4285F4"/>
                  <path d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z" fill="#34A853"/>
                  <path d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z" fill="#FBBC05"/>
                  <path d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z" fill="#EA4335"/>
                </svg>
                Continue with Google
              </button>

              <button
                onClick={() => loginDriver()}
                className="w-full rounded-xl bg-primary px-4 py-3 text-sm font-medium text-primary-foreground shadow-sm transition-colors hover:bg-primary/90"
              >
                Sign in with Email
              </button>
            </div>

            <div className="rounded-xl border border-border bg-muted/50 p-4">
              <div className="flex items-start gap-3">
                <UserPlus className="h-5 w-5 text-primary mt-0.5 shrink-0" />
                <div>
                  <p className="text-sm font-medium text-foreground">New Driver?</p>
                  <p className="mt-1 text-xs text-muted-foreground">
                    Sign up with your email or Google account. You'll go through
                    a quick onboarding process to start delivering.
                  </p>
                </div>
              </div>
            </div>
          </div>
        )}

        {/* Mode: Staff Login */}
        {mode === 'staff' && (
          <div className="space-y-4">
            <button
              onClick={() => setMode('choose')}
              className="flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground transition-colors"
            >
              <ArrowLeft className="h-4 w-4" />
              Back
            </button>

            <div className="space-y-3">
              <button
                onClick={() => loginStaff('google')}
                className="flex w-full items-center justify-center gap-3 rounded-xl border border-border bg-card px-4 py-3 text-sm font-medium text-foreground shadow-sm transition-colors hover:bg-secondary"
              >
                <svg className="h-5 w-5" viewBox="0 0 24 24">
                  <path d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92a5.06 5.06 0 01-2.2 3.32v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.1z" fill="#4285F4"/>
                  <path d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z" fill="#34A853"/>
                  <path d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z" fill="#FBBC05"/>
                  <path d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z" fill="#EA4335"/>
                </svg>
                Continue with Google
              </button>

              <button
                onClick={() => loginStaff()}
                className="w-full rounded-xl bg-primary px-4 py-3 text-sm font-medium text-primary-foreground shadow-sm transition-colors hover:bg-primary/90"
              >
                Sign in with Email
              </button>
            </div>

            <div className="rounded-xl border border-border bg-muted/50 p-4">
              <div className="flex items-start gap-3">
                <Shield className="h-5 w-5 text-primary mt-0.5 shrink-0" />
                <div>
                  <p className="text-sm font-medium text-foreground">Staff Access</p>
                  <p className="mt-1 text-xs text-muted-foreground">
                    Only pre-authorized fleet managers and delivery operations
                    staff can sign in here. Contact an admin for access.
                  </p>
                </div>
              </div>
            </div>
          </div>
        )}

        <p className="text-center text-xs text-muted-foreground">
          Fe3dr Delivery Portal
        </p>
      </div>
    </div>
  );
}
