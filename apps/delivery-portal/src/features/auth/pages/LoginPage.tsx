import { useState, type FormEvent } from 'react';
import { useSearchParams, useNavigate } from 'react-router-dom';
import { useAuth } from '@/app/providers/AuthProvider';
import { Truck, UserPlus, Loader2, Eye, EyeOff } from 'lucide-react';
import { GoogleSignInButton } from '@/features/auth/components/GoogleSignInButton';

/**
 * Delivery-portal login page. All users on this portal are drivers under the
 * business GIP pool; the previous staff/driver split has been removed.
 * Staff identification (fleet managers) is derived later from the
 * staff-profile API endpoint, not from a separate auth mode here.
 */
export default function LoginPage() {
  const { loginWithEmail } = useAuth();
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();

  const authError = searchParams.get('error');
  const accessDenied = authError === 'access-denied';
  const sessionExpired = authError === 'session_expired' || authError === 'invalid_state';

  const [showEmailForm, setShowEmailForm] = useState(false);
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  const handleEmailLogin = async (e: FormEvent) => {
    e.preventDefault();
    if (!email || !password) {
      setError('Please enter your email and password');
      return;
    }
    setLoading(true);
    setError('');
    try {
      await loginWithEmail(email, password);
      navigate('/dashboard');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Invalid email or password');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="flex min-h-screen items-center justify-center bg-background px-4">
      <div className="w-full max-w-md space-y-8">
        <div className="text-center">
          <div className="mx-auto flex h-16 w-16 items-center justify-center rounded-2xl bg-primary">
            <Truck className="h-8 w-8 text-primary-foreground" aria-hidden="true" />
          </div>
          <h1 className="mt-6 font-display text-3xl font-semibold tabular-nums text-foreground">Fe3dr Delivery</h1>
          <p className="mt-2 text-muted-foreground">Sign in or sign up to start delivering</p>
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

        <div className="space-y-3">
          <GoogleSignInButton
            onSuccess={() => navigate('/')}
            onError={(msg) => setError(msg)}
            width={360}
          />

          {!showEmailForm ? (
            <button
              type="button"
              onClick={() => setShowEmailForm(true)}
              className="w-full rounded-xl bg-primary px-4 py-3 text-sm font-medium text-primary-foreground shadow-sm transition-colors hover:bg-primary/90"
            >
              Sign in with Email
            </button>
          ) : (
            <form onSubmit={handleEmailLogin} className="space-y-3">
              {error && (
                <div role="alert" aria-live="polite" className="rounded-lg border border-destructive/30 bg-destructive/5 p-3 text-sm text-destructive">
                  {error}
                </div>
              )}
              <div>
                <label htmlFor="driver-email" className="block text-sm font-medium text-foreground">
                  Email
                </label>
                <input
                  id="driver-email"
                  type="email"
                  autoComplete="email"
                  required
                  aria-required="true"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  className="mt-1 block w-full rounded-xl border border-border bg-card px-3 py-2.5 text-foreground shadow-sm placeholder:text-muted-foreground focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
                  placeholder="you@example.com"
                />
              </div>
              <div>
                <label htmlFor="driver-password" className="block text-sm font-medium text-foreground">
                  Password
                </label>
                <div className="relative mt-1">
                  <input
                    id="driver-password"
                    type={showPassword ? 'text' : 'password'}
                    autoComplete="current-password"
                    required
                    aria-required="true"
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    className="block w-full rounded-xl border border-border bg-card px-3 py-2.5 pr-10 text-foreground shadow-sm placeholder:text-muted-foreground focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
                    placeholder="Enter your password"
                  />
                  <button
                    type="button"
                    aria-label={showPassword ? 'Hide password' : 'Show password'}
                    aria-pressed={showPassword}
                    onClick={() => setShowPassword(!showPassword)}
                    className="absolute inset-y-0 right-0 flex items-center rounded pr-3 text-muted-foreground transition-colors hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
                  >
                    {showPassword ? <EyeOff aria-hidden="true" className="h-4 w-4" /> : <Eye aria-hidden="true" className="h-4 w-4" />}
                  </button>
                </div>
              </div>
              <button
                type="submit"
                disabled={loading}
                className="flex w-full items-center justify-center rounded-xl bg-primary px-4 py-3 text-sm font-medium text-primary-foreground shadow-sm transition-colors hover:bg-primary/90 disabled:opacity-50"
              >
                {loading ? (
                  <>
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" aria-hidden="true" />
                    Signing in...
                  </>
                ) : (
                  'Sign in'
                )}
              </button>
            </form>
          )}
        </div>

        <div className="rounded-xl border border-border bg-muted/50 p-4">
          <div className="flex items-start gap-3">
            <UserPlus className="h-5 w-5 text-primary mt-0.5 shrink-0" aria-hidden="true" />
            <div>
              <p className="text-sm font-medium text-foreground">New Driver?</p>
              <p className="mt-1 text-xs text-muted-foreground">
                Sign up with your email or Google account. You'll go through
                a quick onboarding process to start delivering.
              </p>
            </div>
          </div>
        </div>

        <p className="text-center text-xs text-muted-foreground">Fe3dr Delivery Portal</p>
      </div>
    </div>
  );
}
