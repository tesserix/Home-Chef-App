import { useState, type FormEvent } from 'react';
import { useSearchParams, useNavigate } from 'react-router-dom';
import { motion, AnimatePresence } from 'framer-motion';
import { Shield, Check, Loader2, Eye, EyeOff } from 'lucide-react';
import { useAuth } from '@/app/providers/AuthProvider';
import { Button } from '@/shared/components/ui/Button';
import { fadeInUp, staggerContainer } from '@/shared/utils/animations';

// MFA is deferred per the migration spec; the admin allowlist is enforced
// server-side at the BFF, so the 2FA challenge step has been removed.

function GoogleIcon({ className }: { className?: string }) {
  return (
    <svg className={className} viewBox="0 0 24 24" fill="none">
      <path d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92a5.06 5.06 0 01-2.2 3.32v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.1z" fill="#4285F4" />
      <path d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z" fill="#34A853" />
      <path d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18A10.96 10.96 0 001 12c0 1.77.43 3.45 1.18 4.93l2.85-2.22.81-.62z" fill="#FBBC05" />
      <path d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z" fill="#EA4335" />
    </svg>
  );
}

const FEATURES = [
  'User & role management',
  'Chef verification & approvals',
  'Order monitoring & refunds',
  'Platform analytics & insights',
  'Revenue tracking & payouts',
  'Content moderation tools',
];

export default function LoginPage() {
  const { login, loginWithEmail } = useAuth();
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

  const handleGoogleLogin = async () => {
    setError('');
    setLoading(true);
    try {
      await login('google');
      navigate('/dashboard');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Sign-in failed');
    } finally {
      setLoading(false);
    }
  };

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
    <div className="flex min-h-screen bg-background">
      {/* Left side - Image & Features */}
      <div className="relative hidden flex-1 lg:block">
        <img
          src="https://images.unsplash.com/photo-1600565193348-f74bd3c7ccdf?w=1200&h=900&fit=crop&q=80"
          alt="Overhead view of home-cooked Indian food spread on a wooden table"
          fetchPriority="high"
          decoding="async"
          className="absolute inset-0 h-full w-full object-cover"
        />
        <div className="absolute inset-0 scrim-bottom" />

        {/* Content overlay */}
        <div className="relative flex h-full flex-col justify-end p-10 xl:p-14">
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: 0.3, duration: 0.6 }}
          >
            <div className="flex items-center gap-2.5 mb-4">
              <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-white/15 backdrop-blur-sm">
                <Shield className="h-5 w-5 text-on-photo" />
              </div>
              <span className="text-on-photo text-xl font-semibold font-display">Fe3dr</span>
            </div>

            <h2 className="text-on-photo max-w-md font-display text-3xl font-semibold tabular-nums leading-tight xl:text-4xl">
              Platform Administration
            </h2>
            <p className="text-on-photo-soft mt-3 max-w-md text-base">
              Manage users, chefs, orders, and analytics for the Fe3dr platform.
            </p>

            <div className="mt-8 grid grid-cols-2 gap-x-6 gap-y-3">
              {FEATURES.map((feature) => (
                <div key={feature} className="flex items-center gap-2.5">
                  <div className="flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-white/15 backdrop-blur-sm">
                    <Check className="h-3 w-3 text-on-photo" />
                  </div>
                  <span className="text-on-photo-soft text-sm">{feature}</span>
                </div>
              ))}
            </div>
          </motion.div>
        </div>
      </div>

      {/* Right side - Login */}
      <div className="flex flex-1 flex-col justify-center px-6 py-12 lg:max-w-xl lg:px-16 xl:px-20">
        <motion.div
          variants={staggerContainer}
          initial="hidden"
          animate="visible"
          className="mx-auto w-full max-w-sm"
        >
          {/* Logo (mobile) */}
          <motion.div variants={fadeInUp} className="mb-10">
            <div className="flex items-center gap-2.5">
              <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-primary shadow-md">
                <Shield className="h-5 w-5 text-primary-foreground" />
              </div>
              <div>
                <h1 className="text-xl font-semibold text-foreground font-display">Fe3dr</h1>
                <p className="text-xs text-muted-foreground">Admin Portal</p>
              </div>
            </div>
          </motion.div>

          {/* Heading */}
          <motion.div variants={fadeInUp} className="mb-8">
            <h2 className="font-display text-2xl font-semibold tracking-tight text-foreground font-display sm:text-3xl">
              Admin Sign In
            </h2>
            <p className="mt-2 text-muted-foreground">
              Access the administration dashboard with your internal credentials
            </p>
          </motion.div>

          {/* Error messages */}
          {accessDenied && (
            <motion.div
              variants={fadeInUp}
              className="mb-6 rounded-xl border border-destructive/30 bg-destructive/5 p-4 text-sm text-destructive"
            >
              Access denied. Only administrators can sign in to this portal.
            </motion.div>
          )}

          {sessionExpired && (
            <motion.div
              variants={fadeInUp}
              className="mb-6 rounded-xl border border-warning/30 bg-warning/5 p-4 text-sm text-warning"
            >
              Your session has expired. Please sign in again.
            </motion.div>
          )}

          {authError && !accessDenied && !sessionExpired && (
            <motion.div
              variants={fadeInUp}
              className="mb-6 rounded-xl border border-destructive/30 bg-destructive/5 p-4 text-sm text-destructive"
            >
              Something went wrong. Please try again.
            </motion.div>
          )}

          {/* Social login buttons — admin portal supports Google only. */}
          <motion.div variants={fadeInUp} className="space-y-3">
            <Button
              variant="outline"
              size="xl"
              fullWidth
              leftIcon={<GoogleIcon className="h-5 w-5" />}
              onClick={handleGoogleLogin}
              disabled={loading}
              className="justify-center rounded-xl border-border hover:bg-secondary/60"
            >
              Continue with Google
            </Button>
          </motion.div>

          {/* Divider */}
          <motion.div variants={fadeInUp} className="my-6 flex items-center gap-4">
            <div className="h-px flex-1 bg-border" />
            <span className="text-sm text-muted-foreground">or</span>
            <div className="h-px flex-1 bg-border" />
          </motion.div>

          {/* Email login */}
          <motion.div variants={fadeInUp}>
            <AnimatePresence mode="wait">
              {!showEmailForm ? (
                <motion.div key="email-btn" exit={{ opacity: 0, height: 0 }}>
                  <Button
                    variant="default"
                    size="xl"
                    fullWidth
                    onClick={() => setShowEmailForm(true)}
                    className="justify-center rounded-xl"
                  >
                    Sign in with Email
                  </Button>
                </motion.div>
              ) : (
                <motion.form
                  key="email-form"
                  initial={{ opacity: 0, height: 0 }}
                  animate={{ opacity: 1, height: 'auto' }}
                  transition={{ duration: 0.3 }}
                  onSubmit={handleEmailLogin}
                  className="space-y-4"
                >
                  {error && (
                    <div className="rounded-lg border border-destructive/30 bg-destructive/5 p-3 text-sm text-destructive">
                      {error}
                    </div>
                  )}

                  <div>
                    <label htmlFor="login-email" className="block text-sm font-medium text-foreground">
                      Email
                    </label>
                    <input
                      id="login-email"
                      type="email"
                      autoComplete="email"
                      required
                      aria-required="true"
                      value={email}
                      onChange={(e) => setEmail(e.target.value)}
                      className="mt-1 block w-full rounded-xl border border-border bg-card px-3 py-2.5 text-foreground shadow-sm placeholder:text-muted-foreground focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
                      placeholder="admin@fe3dr.com"
                    />
                  </div>

                  <div>
                    <label htmlFor="login-password" className="block text-sm font-medium text-foreground">
                      Password
                    </label>
                    <div className="relative mt-1">
                      <input
                        id="login-password"
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
                        className="absolute inset-y-0 right-0 flex items-center pr-3 text-muted-foreground hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 rounded"
                      >
                        {showPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                      </button>
                    </div>
                  </div>

                  <Button
                    type="submit"
                    variant="default"
                    size="xl"
                    fullWidth
                    disabled={loading}
                    className="justify-center rounded-xl"
                  >
                    {loading ? (
                      <>
                        <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                        Signing in...
                      </>
                    ) : (
                      'Sign in'
                    )}
                  </Button>
                </motion.form>
              )}
            </AnimatePresence>
          </motion.div>

          {/* Security notice */}
          <motion.div variants={fadeInUp} className="mt-8">
            <div className="rounded-xl border border-border bg-muted/50 p-4">
              <div className="flex items-start gap-3">
                <Shield className="h-5 w-5 text-primary mt-0.5 shrink-0" />
                <div>
                  <p className="text-sm font-medium text-foreground">Restricted Access</p>
                  <p className="mt-1 text-xs text-muted-foreground">
                    Only pre-authorized administrators can sign in. New registrations are not allowed.
                  </p>
                </div>
              </div>
            </div>
          </motion.div>

          {/* Footer */}
          <motion.div variants={fadeInUp} className="mt-12">
            <p className="text-center text-xs text-muted-foreground">
              Fe3dr Administration Portal &middot; Internal Use Only
            </p>
          </motion.div>
        </motion.div>
      </div>
    </div>
  );
}
