import { motion } from 'framer-motion';
import { ChefHat, Check } from 'lucide-react';
import { Link } from 'react-router-dom';
import { useAuth } from '@/app/providers/AuthProvider';
import { Button } from '@/shared/components/ui/Button';
import { fadeInUp, staggerContainer } from '@/shared/utils/animations';

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

function MetaIcon({ className }: { className?: string }) {
  return (
    <svg className={className} viewBox="0 0 24 24" fill="none">
      <path d="M12 2.04C6.5 2.04 2 6.53 2 12.06c0 5 3.66 9.13 8.44 9.88v-6.99H7.9v-2.89h2.54V9.85c0-2.5 1.49-3.89 3.78-3.89 1.09 0 2.24.2 2.24.2v2.46h-1.26c-1.24 0-1.63.77-1.63 1.56v1.88h2.78l-.45 2.89h-2.33v6.99C18.34 21.19 22 17.06 22 12.06c0-5.53-4.5-10.02-10-10.02z" fill="#1877F2" />
    </svg>
  );
}

const BENEFITS = [
  { title: 'Zero commission first month', desc: 'Get started completely risk-free' },
  { title: 'Easy menu management', desc: 'Upload photos, set prices, manage availability' },
  { title: 'Real-time order tracking', desc: 'Never miss an order with instant notifications' },
  { title: 'Weekly payouts', desc: 'Get paid directly to your bank account' },
];

export default function RegisterPage() {
  const { register, login } = useAuth();

  return (
    <div className="flex min-h-screen bg-background">
      {/* Left side - Image & Benefits */}
      <div className="relative hidden flex-1 lg:block">
        <img
          src="https://images.unsplash.com/photo-1606491956689-2ea866880049?w=1200&h=900&fit=crop&q=80"
          alt="Indian woman cooking in home kitchen"
          className="absolute inset-0 h-full w-full object-cover"
        />
        <div className="absolute inset-0 bg-gradient-to-t from-black/80 via-black/40 to-black/20" />

        {/* Content overlay */}
        <div className="relative flex h-full flex-col justify-end p-10 xl:p-14">
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: 0.3, duration: 0.6 }}
          >
            <h2 className="max-w-md text-3xl font-bold leading-tight text-white font-display xl:text-4xl">
              Start selling from your kitchen
            </h2>
            <p className="mt-3 max-w-md text-base text-white/80">
              Join thousands of home chefs earning with Fe3dr. Your kitchen, your recipes, your rules.
            </p>

            <div className="mt-8 space-y-4">
              {BENEFITS.map((item) => (
                <div key={item.title} className="flex items-start gap-3">
                  <div className="mt-0.5 flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-white/20 backdrop-blur-sm">
                    <Check className="h-3.5 w-3.5 text-white" />
                  </div>
                  <div>
                    <p className="font-semibold text-white">{item.title}</p>
                    <p className="text-sm text-white/70">{item.desc}</p>
                  </div>
                </div>
              ))}
            </div>
          </motion.div>
        </div>
      </div>

      {/* Right side - Registration options */}
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
                <h1 className="text-xl font-bold text-foreground font-display">Fe3dr</h1>
                <p className="text-xs text-muted-foreground">Vendor Portal</p>
              </div>
            </div>
          </motion.div>

          <motion.div variants={fadeInUp} className="mb-8">
            <h2 className="text-2xl font-bold tracking-tight text-foreground font-display sm:text-3xl">
              Register your kitchen
            </h2>
            <p className="mt-2 text-muted-foreground">
              Create your vendor account to start selling home-cooked meals
            </p>
          </motion.div>

          {/* Social registration */}
          <motion.div variants={fadeInUp} className="space-y-3">
            <Button
              variant="outline"
              size="xl"
              fullWidth
              leftIcon={<GoogleIcon className="h-5 w-5" />}
              onClick={() => login('google')}
              className="justify-center rounded-xl border-border hover:bg-secondary/60"
            >
              Sign up with Google
            </Button>

            <Button
              variant="outline"
              size="xl"
              fullWidth
              leftIcon={<MetaIcon className="h-5 w-5" />}
              onClick={() => login('facebook')}
              className="justify-center rounded-xl border-border hover:bg-secondary/60"
            >
              Sign up with Facebook
            </Button>
          </motion.div>

          {/* Divider */}
          <motion.div variants={fadeInUp} className="my-6 flex items-center gap-4">
            <div className="h-px flex-1 bg-border" />
            <span className="text-sm text-muted-foreground">or</span>
            <div className="h-px flex-1 bg-border" />
          </motion.div>

          {/* Email registration */}
          <motion.div variants={fadeInUp}>
            <Button
              variant="default"
              size="xl"
              fullWidth
              onClick={() => register()}
              className="justify-center rounded-xl"
            >
              Register with Email
            </Button>
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
              By registering, you agree to Fe3dr's Vendor Terms and Privacy Policy
            </p>
          </motion.div>
        </motion.div>
      </div>
    </div>
  );
}
