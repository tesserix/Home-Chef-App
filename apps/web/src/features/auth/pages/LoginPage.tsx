import { useState, type FormEvent } from 'react';
import { Link, useSearchParams, useNavigate } from 'react-router-dom';
import { motion, AnimatePresence } from 'framer-motion';
import { ChefHat, Loader2, Eye, EyeOff } from 'lucide-react';
import { useAuth } from '@/app/providers/AuthProvider';
import { Button } from '@/shared/components/ui';
import { fadeInLeft, fadeInRight } from '@/shared/utils/animations';
import { GoogleSignInButton } from '@/features/auth/components/GoogleSignInButton';

export default function LoginPage() {
  const { login, loginWithEmail } = useAuth();
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const authError = searchParams.get('error');
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
      navigate('/');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Invalid email or password');
    } finally {
      setLoading(false);
    }
  };

  const handleSocialLogin = async (provider: 'google' | 'facebook') => {
    setError('');
    setLoading(true);
    try {
      await login(provider);
      navigate('/');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Sign-in failed');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="flex min-h-screen">
      {/* Left side - Form */}
      <motion.div
        initial="hidden"
        animate="visible"
        variants={fadeInLeft}
        transition={{ duration: 0.5 }}
        className="flex flex-1 flex-col justify-center px-4 py-12 sm:px-6 lg:flex-none lg:px-20 xl:px-24"
      >
        <div className="mx-auto w-full max-w-sm lg:w-96">
          {/* Logo */}
          <Link to="/" className="inline-flex items-center gap-2 group">
            <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-herb shadow-1 group-hover:shadow-2 transition-shadow">
              <ChefHat aria-hidden="true" className="h-5 w-5 text-paper" />
            </div>
            <span className="font-display text-2xl font-semibold text-ink">Fe3dr</span>
          </Link>

          <motion.div
            initial={{ opacity: 0, y: 10 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: 0.1 }}
            className="mt-8"
          >
            <h2 className="font-display text-display-xs text-ink">Welcome back</h2>
            <p className="mt-2 text-ink-soft">
              Don't have an account?{' '}
              <Link to="/register" className="font-medium text-herb hover:text-herb transition-colors">
                Sign up
              </Link>
            </p>
          </motion.div>

          {/* Error messages */}
          {sessionExpired && (
            <motion.div
              initial={{ opacity: 0, y: 10 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: 0.15 }}
              className="mt-6 rounded-lg border border-amber/30 bg-amber-tint p-3 text-sm text-amber"
            >
              Your session has expired. Please sign in again.
            </motion.div>
          )}
          {authError && !sessionExpired && (
            <motion.div
              initial={{ opacity: 0, y: 10 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: 0.15 }}
              className="mt-6 rounded-lg border border-paprika/30 bg-paprika-tint p-3 text-sm text-paprika"
            >
              Something went wrong. Please try again.
            </motion.div>
          )}

          <motion.div
            initial={{ opacity: 0, y: 10 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: 0.2 }}
            className="mt-8 space-y-4"
          >
            {/* Social login buttons — Google is rendered by GSI (inline FedCM modal, no popup) */}
            <GoogleSignInButton
              onSuccess={() => navigate('/')}
              onError={(msg) => setError(msg)}
              width={360}
            />

            <Button
              variant="outline"
              size="lg"
              onClick={() => handleSocialLogin('facebook')}
              className="mx-auto block w-full max-w-[360px] justify-center gap-3 rounded-full py-3"
            >
              <svg className="h-5 w-5 text-info" fill="currentColor" viewBox="0 0 24 24">
                <path d="M24 12.073c0-6.627-5.373-12-12-12s-12 5.373-12 12c0 5.99 4.388 10.954 10.125 11.854v-8.385H7.078v-3.47h3.047V9.43c0-3.007 1.792-4.669 4.533-4.669 1.312 0 2.686.235 2.686.235v2.953H15.83c-1.491 0-1.956.925-1.956 1.874v2.25h3.328l-.532 3.47h-2.796v8.385C19.612 23.027 24 18.062 24 12.073z" />
              </svg>
              Continue with Facebook
            </Button>

            <div className="relative mt-6">
              <div className="absolute inset-0 flex items-center">
                <div className="w-full border-t border-mist" />
              </div>
              <div className="relative flex justify-center text-sm">
                <span className="bg-bone px-3 text-ink-muted">Or</span>
              </div>
            </div>

            <AnimatePresence mode="wait">
              {!showEmailForm ? (
                <motion.div key="email-btn" exit={{ opacity: 0, height: 0 }}>
                  <Button
                    variant="primary"
                    size="lg"
                    onClick={() => setShowEmailForm(true)}
                    className="mx-auto block w-full max-w-[360px] rounded-full"
                  >
                    Sign in with email
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
                    <div
                      id="login-form-error"
                      role="alert"
                      className="rounded-lg border border-paprika/30 bg-paprika-tint p-3 text-sm text-paprika"
                    >
                      {error}
                    </div>
                  )}

                  <div>
                    <label htmlFor="login-email" className="block text-sm font-medium text-ink-soft">
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
                      aria-invalid={!!error || undefined}
                      aria-describedby={error ? 'login-form-error' : undefined}
                      className="mt-1 block w-full rounded-lg border border-mist-strong px-3 py-2.5 text-ink shadow-1 placeholder:text-ink-muted focus-visible:border-herb focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-herb/30"
                      placeholder="you@example.com"
                    />
                  </div>

                  <div>
                    <div className="flex items-center justify-between">
                      <label htmlFor="login-password" className="block text-sm font-medium text-ink-soft">
                        Password
                      </label>
                      <Link
                        to="/forgot-password"
                        className="text-sm text-herb hover:text-herb"
                      >
                        Forgot password?
                      </Link>
                    </div>
                    <div className="relative mt-1">
                      <input
                        id="login-password"
                        type={showPassword ? 'text' : 'password'}
                        autoComplete="current-password"
                        required
                        aria-required="true"
                        value={password}
                        onChange={(e) => setPassword(e.target.value)}
                        aria-invalid={!!error || undefined}
                        aria-describedby={error ? 'login-form-error' : undefined}
                        className="block w-full rounded-lg border border-mist-strong px-3 py-2.5 pr-10 text-ink shadow-1 placeholder:text-ink-muted focus-visible:border-herb focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-herb/30"
                        placeholder="Enter your password"
                      />
                      <button
                        type="button"
                        onClick={() => setShowPassword(!showPassword)}
                        aria-label={showPassword ? 'Hide password' : 'Show password'}
                        aria-pressed={showPassword}
                        className="absolute inset-y-0 right-0 flex items-center pr-3 text-ink-muted hover:text-ink-soft focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-herb/30 rounded-r-lg"
                      >
                        {showPassword ? <EyeOff aria-hidden="true" className="h-4 w-4" /> : <Eye aria-hidden="true" className="h-4 w-4" />}
                      </button>
                    </div>
                  </div>

                  <Button
                    type="submit"
                    variant="primary"
                    size="lg"
                    disabled={loading}
                    className="w-full"
                  >
                    {loading ? (
                      <>
                        <Loader2 className="mr-2 h-4 w-4 animate-spin"  aria-hidden="true" />
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
        </div>
      </motion.div>

      {/* Right side - Image */}
      <motion.div
        initial="hidden"
        animate="visible"
        variants={fadeInRight}
        transition={{ duration: 0.6, delay: 0.2 }}
        className="relative hidden w-0 flex-1 lg:block"
      >
        <img
          className="absolute inset-0 h-full w-full object-cover"
          src="https://images.unsplash.com/photo-1556909114-f6e7ad7d3136?w=1200&h=900&fit=crop"
          alt="Delicious homemade food"
          width={1200}
          height={900}
          loading="lazy"
          decoding="async"
        />
        <div aria-hidden="true" className="absolute inset-0 scrim-top" />
        <div className="absolute bottom-0 left-0 right-0 scrim-bottom p-12">
          <motion.blockquote
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: 0.6 }}
            className="text-on-photo"
          >
            <p className="font-display text-xl font-medium leading-relaxed">
              "Fe3dr has changed how I eat. Finally, real homemade food that
              reminds me of my mom's cooking!"
            </p>
            <footer className="mt-4">
              <p className="font-semibold">Sarah M.</p>
              <p className="text-on-photo-soft text-sm">Happy Customer</p>
            </footer>
          </motion.blockquote>
        </div>
      </motion.div>
    </div>
  );
}
