import { useEffect } from 'react';
import { Link, useSearchParams, useNavigate } from 'react-router';
import { motion } from 'framer-motion';
import { ChefHat, ArrowRight } from 'lucide-react';
import { useAuth } from '@/app/providers/AuthProvider';
import { Button } from '@/shared/components/ui';
import { fadeInLeft, fadeInRight } from '@/shared/utils/animations';
import { redirectToLogin } from '@/features/auth/services/auth-service';

// Sign-in now lives on the hosted Zitadel pages (email/password, Google,
// Apple all in one place); this page is just the branded doorway. Keeping a
// deliberate click here — rather than auto-redirecting — avoids a loop when
// the callback bounces back with ?error=….
export default function LoginPage() {
  const { isAuthenticated, isLoading } = useAuth();
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const authError = searchParams.get('error');
  const sessionExpired = authError === 'session_expired' || authError === 'invalid_state';
  const returnTo = searchParams.get('returnTo');

  // Already signed in? The BFF cookie survives new tabs and reloads; don't
  // show a sign-in door to someone who is already inside.
  useEffect(() => {
    if (isLoading || !isAuthenticated) return;
    const target = returnTo && returnTo.startsWith('/') && !returnTo.startsWith('//')
      ? returnTo
      : '/';
    navigate(target === '/login' ? '/' : target, { replace: true });
  }, [isAuthenticated, isLoading, returnTo, navigate]);

  const startLogin = () => {
    redirectToLogin({
      returnTo:
        returnTo && returnTo.startsWith('/') && !returnTo.startsWith('//')
          ? returnTo
          : undefined,
    });
  };

  return (
    <div className="flex min-h-screen">
      {/* Left side - Sign-in door */}
      <motion.div
        initial="hidden"
        animate="visible"
        variants={fadeInLeft}
        transition={{ duration: 0.5 }}
        className="flex flex-1 flex-col justify-center px-4 py-12 sm:px-6 lg:flex-none lg:px-20 xl:px-24"
      >
        <div className="mx-auto w-full max-w-sm lg:w-96">
          <Link to="/" className="inline-flex items-center gap-2 group">
            <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-herb shadow-1 group-hover:shadow-2 transition-shadow">
              <ChefHat aria-hidden="true" className="h-5 w-5 text-paper" />
            </div>
            <span className="font-display text-2xl font-semibold text-ink">Fe3dr</span>
          </Link>
          <p className="mt-3 text-sm text-ink-soft">
            For those who love to eat — and those who love to cook.
          </p>

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
            <Button
              variant="primary"
              size="lg"
              onClick={startLogin}
              className="w-full rounded-full"
            >
              Sign in
              <ArrowRight aria-hidden="true" className="ml-2 h-4 w-4" />
            </Button>
            <p className="text-center text-sm text-ink-muted">
              You'll sign in securely on our account page — email, Google and
              Apple all work there.
            </p>
            <p className="text-center text-sm">
              <Link to="/forgot-password" className="font-medium text-herb hover:text-herb">
                Forgot password?
              </Link>
            </p>
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
