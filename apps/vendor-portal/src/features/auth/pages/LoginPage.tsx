import { Link, useSearchParams } from 'react-router';
import { motion } from 'framer-motion';
import { ChefHat, Check, ArrowRight } from 'lucide-react';
import { useAuth } from '@/app/providers/AuthProvider';
import { Button } from '@/shared/components/ui/Button';
import { fadeInUp, staggerContainer } from '@/shared/utils/animations';

const FEATURES = [
  'Real-time order management',
  'Earnings & analytics dashboard',
  'Menu management with categories',
  'Customer reviews & ratings',
  'Document verification & compliance',
];

// Sign-in lives on the hosted Zitadel pages (email/password, Google, Apple
// all in one place); this page is just the branded doorway. Keeping a
// deliberate click here — rather than auto-redirecting — avoids a loop when
// the callback bounces back with ?error=….
export default function LoginPage() {
  const { login } = useAuth();
  const [searchParams] = useSearchParams();

  const authError = searchParams.get('error');
  const accessDenied = authError === 'access-denied';
  const sessionExpired = authError === 'session_expired' || authError === 'invalid_state';
  const rawReturnTo = searchParams.get('returnTo');
  const returnTo =
    rawReturnTo && rawReturnTo.startsWith('/') && !rawReturnTo.startsWith('//')
      ? rawReturnTo
      : undefined;

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

      {/* Right side - Sign-in door */}
      <div className="flex flex-1 flex-col justify-center px-6 py-12 lg:max-w-xl lg:px-16 xl:px-20">
        <motion.div
          variants={staggerContainer}
          initial="hidden"
          animate="visible"
          className="mx-auto w-full max-w-sm"
        >
          {/* Logo */}
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
              For those who love to eat — and those who love to cook.
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

          {/* Hosted sign-in hand-off */}
          <motion.div variants={fadeInUp} className="space-y-4">
            <Button
              variant="default"
              size="xl"
              fullWidth
              onClick={() => void login({ returnTo })}
              className="justify-center rounded-xl"
            >
              Sign in
              <ArrowRight aria-hidden="true" className="ml-2 h-4 w-4" />
            </Button>
            <p className="text-center text-sm text-muted-foreground">
              You'll sign in securely on our account page — email, Google and
              Apple all work there.
            </p>
            <p className="text-center text-sm">
              <Link to="/forgot-password" className="font-medium text-primary hover:underline">
                Forgot password?
              </Link>
            </p>
          </motion.div>

          {/* Register link */}
          <motion.div variants={fadeInUp} className="mt-8 text-center">
            <p className="text-sm text-muted-foreground">
              Want to start selling?{' '}
              <Link
                to="/register"
                className="rounded font-semibold text-primary transition-colors hover:text-primary/80 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
              >
                Register as a chef
              </Link>
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
