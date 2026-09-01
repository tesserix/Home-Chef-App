import { Link } from 'react-router';
import { motion } from 'framer-motion';
import { ChefHat, Check, ArrowRight } from 'lucide-react';
import { useAuth } from '@/app/providers/AuthProvider';
import { Button } from '@/shared/components/ui/Button';
import { fadeInUp, staggerContainer } from '@/shared/utils/animations';

const BENEFITS = [
  { title: 'Zero commission first month', desc: 'Get started completely risk-free' },
  { title: 'Easy menu management', desc: 'Upload photos, set prices, manage availability' },
  { title: 'Real-time order tracking', desc: 'Never miss an order with instant notifications' },
  { title: 'Weekly payouts', desc: 'Get paid directly to your bank account' },
];

// Account creation happens on the hosted Zitadel registration page; this page
// is the branded doorway. After the callback, AuthProvider's onboarding check
// routes a brand-new chef into /onboarding automatically.
export default function RegisterPage() {
  const { register } = useAuth();

  return (
    <div className="flex min-h-screen bg-background">
      {/* Left side - Image & Benefits */}
      <div className="relative hidden flex-1 lg:block">
        <img
          src="https://images.unsplash.com/photo-1606491956689-2ea866880049?w=1200&h=900&fit=crop&q=80"
          alt="Indian woman cooking in home kitchen"
          className="absolute inset-0 h-full w-full object-cover"
          fetchPriority="high"
          decoding="async"
        />
        <div className="absolute inset-0 scrim-bottom" />

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
              Start selling from your kitchen
            </h2>
            <p className="text-on-photo-soft mt-3 max-w-md text-base">
              Join thousands of home chefs earning with Fe3dr. Your kitchen, your recipes, your rules.
            </p>

            <div className="mt-8 space-y-4">
              {BENEFITS.map((item) => (
                <div key={item.title} className="flex items-start gap-3">
                  <div className="mt-0.5 flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-white/15 backdrop-blur-sm">
                    <Check className="h-3.5 w-3.5 text-on-photo" />
                  </div>
                  <div>
                    <p className="text-on-photo font-semibold">{item.title}</p>
                    <p className="text-on-photo-soft text-sm">{item.desc}</p>
                  </div>
                </div>
              ))}
            </div>
          </motion.div>
        </div>
      </div>

      {/* Right side - Registration hand-off */}
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
              Register your kitchen
            </h2>
            <p className="mt-2 text-muted-foreground">
              Create your chef account to start selling home-cooked meals
            </p>
          </motion.div>

          {/* Hosted sign-up hand-off */}
          <motion.div variants={fadeInUp} className="space-y-4">
            <Button
              variant="default"
              size="xl"
              fullWidth
              onClick={() => void register()}
              className="justify-center rounded-xl"
            >
              Continue to sign up
              <ArrowRight aria-hidden="true" className="ml-2 h-4 w-4" />
            </Button>
            <p className="text-center text-sm text-muted-foreground">
              You'll create your account securely on our account page — email,
              Google and Apple all work there. Kitchen details come next, in
              onboarding.
            </p>
          </motion.div>

          {/* Login link */}
          <motion.div variants={fadeInUp} className="mt-8 text-center">
            <p className="text-sm text-muted-foreground">
              Already have an account?{' '}
              <Link
                to="/login"
                className="font-semibold text-primary hover:text-primary/80 transition-colors"
              >
                Sign in
              </Link>
            </p>
          </motion.div>

          {/* Footer */}
          <motion.div variants={fadeInUp} className="mt-12">
            <p className="text-center text-xs text-muted-foreground">
              By registering, you agree to Fe3dr's Chef Terms and Privacy Policy
            </p>
          </motion.div>
        </motion.div>
      </div>
    </div>
  );
}
