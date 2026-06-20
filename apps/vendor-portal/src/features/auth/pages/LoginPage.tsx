import { useState, type FormEvent } from 'react';
import { useSearchParams, useNavigate } from 'react-router-dom';
import { motion, AnimatePresence } from 'framer-motion';
import { ChefHat, Check, Loader2, Eye, EyeOff, Apple } from 'lucide-react';
import { useAuth } from '@/app/providers/AuthProvider';
import { Button } from '@/shared/components/ui/Button';
import { fadeInUp, staggerContainer } from '@/shared/utils/animations';
import { GoogleSignInButton } from '@/features/auth/components/GoogleSignInButton';

const FEATURES = [
  'Real-time order management',
  'Earnings & analytics dashboard',
  'Menu management with categories',
  'Customer reviews & ratings',
  'Document verification & compliance',
];

export default function LoginPage() {
  const { login, register, loginWithEmail } = useAuth();
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

  const handleSocialLogin = async (provider: 'google' | 'apple') => {
    setError('');
    setLoading(true);
    try {
      await login(provider);
      navigate('/dashboard');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Sign-in failed');
    } finally {
      setLoading(false);
    }
  };

  const handleRegisterClick = async () => {
    setError('');
    setLoading(true);
    try {
      await register();
      navigate('/dashboard');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Sign-up failed');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="flex min-h-screen bg-background">
      {/* Left side - Image & Features */}
      <div className="relative hidden flex-1 lg:block">
        <img
          src="https://images.unsplash.com/photo-1556910103-1c02745aae4d?w=1200&h=900&fit=crop&q=80"
          alt="Home chef preparing food in kitchen"
          className="absolute inset-0 h-full w-full object-cover"
          fetchPriority="high"
          decoding="async"
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
                <ChefHat className="h-5 w-5 text-on-photo" />
              </div>
              <span className="text-on-photo text-xl font-semibold font-display">Fe3dr</span>
            </div>

            <h2 className="text-on-photo max-w-md font-display text-3xl font-semibold tabular-nums leading-tight xl:text-4xl">
              Grow your home kitchen business
            </h2>
            <p className="text-on-photo-soft mt-3 max-w-md text-base">
              Manage menus, track orders, view earnings — all from one dashboard.
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

      {/* Right side - Login options */}
      <div className="flex flex-1 flex-col justify-center px-6 py-12 lg:max-w-xl lg:px-16 xl:px-20">
        <motion.div
          variants={staggerContainer}
          initial="hidden"
          animate="visible"
          className="mx-auto w-full max-w-sm"
        >
          {/* Logo (mobile only — desktop shows on image) */}
          <motion.div variants={fadeInUp} className="mb-10">
            <div className="flex items-center gap-2.5">
              <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-primary shadow-md">
                <ChefHat className="h-5 w-5 text-primary-foreground" />
              </div>
              <div>
                <h1 className="text-xl font-semibold text-foreground font-display">Fe3dr</h1>
                <p className="text-xs text-muted-foreground">Chef Portal</p>
              </div>
            </div>
          </motion.div>

          {/* Heading */}
          <motion.div variants={fadeInUp} className="mb-8">
            <h2 className="font-display text-2xl font-semibold tracking-tight text-foreground sm:text-3xl">
              Welcome back
            </h2>
            <p className="mt-2 text-muted-foreground">
              Sign in to manage your menu, orders, and earnings
            </p>
          </motion.div>

          {/* Error messages */}
          {accessDenied && (
            <motion.div
              variants={fadeInUp}
              className="mb-6 rounded-xl border border-destructive/30 bg-destructive/5 p-4 text-sm text-destructive"
            >
              This portal is only for chef accounts. Please use the Fe3dr customer app.
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

          {/* Social login buttons */}
          <motion.div variants={fadeInUp} className="space-y-3">
            <GoogleSignInButton
              onSuccess={() => navigate('/')}
              onError={(msg) => setError(msg)}
              width={360}
            />

            <Button
              variant="outline"
              size="xl"
              fullWidth
              leftIcon={<Apple className="h-5 w-5" />}
              onClick={() => handleSocialLogin('apple')}
              className="justify-center rounded-xl border-border hover:bg-secondary/60"
            >
              Continue with Apple
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
                    <div role="alert" aria-live="polite" className="rounded-lg border border-destructive/30 bg-destructive/5 p-3 text-sm text-destructive">
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
                      placeholder="you@example.com"
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
                        className="absolute inset-y-0 right-0 flex items-center rounded pr-3 text-muted-foreground transition-colors hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
                      >
                        {showPassword ? <EyeOff aria-hidden="true" className="h-4 w-4" /> : <Eye aria-hidden="true" className="h-4 w-4" />}
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

          {/* Register link */}
          <motion.div variants={fadeInUp} className="mt-8 text-center">
            <p className="text-sm text-muted-foreground">
              Want to start selling?{' '}
              <button
                type="button"
                onClick={handleRegisterClick}
                className="rounded font-semibold text-primary transition-colors hover:text-primary/80 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
              >
                Register as a chef
              </button>
            </p>
          </motion.div>

          {/* Footer */}
          <motion.div variants={fadeInUp} className="mt-12">
            <p className="text-center text-xs text-muted-foreground">
              By continuing, you agree to Fe3dr's Terms of Service and Privacy Policy
            </p>
          </motion.div>
        </motion.div>
      </div>
    </div>
  );
}
