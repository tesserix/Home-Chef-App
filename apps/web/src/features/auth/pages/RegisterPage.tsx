import { Link } from 'react-router-dom';
import { motion } from 'framer-motion';
import { ChefHat, Check } from 'lucide-react';
import { useAuth } from '@/app/providers/AuthProvider';
import { Button } from '@/shared/components/ui';
import { fadeInLeft, fadeInRight } from '@/shared/utils/animations';

const BENEFITS = [
  'Access to 500+ home chefs',
  'Authentic homemade food',
  'Fast & reliable delivery',
  'Support local home chefs',
];

export default function RegisterPage() {
  const { register, login } = useAuth();

  return (
    <div className="flex min-h-screen">
      {/* Left side - Image & Benefits */}
      <motion.div
        initial="hidden"
        animate="visible"
        variants={fadeInLeft}
        transition={{ duration: 0.6 }}
        className="relative hidden w-0 flex-1 lg:block"
      >
        <img
          className="absolute inset-0 h-full w-full object-cover"
          src="https://images.unsplash.com/photo-1543352634-a1c51d9f1fa7?w=1200&h=900&fit=crop"
          alt="Home chef preparing food"
        />
        <div className="absolute inset-0 bg-gradient-to-br from-brand-600/40 to-spice-600/30" />
        <div className="absolute inset-0 flex items-center justify-center p-12">
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: 0.4 }}
            className="max-w-lg text-center text-white"
          >
            <h2 className="font-display text-display-sm font-bold">Join Fe3dr Today</h2>
            <p className="mt-4 text-lg text-white/90">
              Get access to hundreds of home chefs serving authentic, homemade food in your area.
            </p>
            <div className="mt-8 space-y-3">
              {BENEFITS.map((benefit, index) => (
                <motion.div
                  key={benefit}
                  initial={{ opacity: 0, x: -20 }}
                  animate={{ opacity: 1, x: 0 }}
                  transition={{ delay: 0.5 + index * 0.1 }}
                  className="flex items-center gap-3 justify-center"
                >
                  <div className="flex h-6 w-6 items-center justify-center rounded-full bg-white/20 backdrop-blur-sm">
                    <Check className="h-4 w-4" />
                  </div>
                  <span className="text-white/95">{benefit}</span>
                </motion.div>
              ))}
            </div>
          </motion.div>
        </div>
      </motion.div>

      {/* Right side - Sign up options */}
      <motion.div
        initial="hidden"
        animate="visible"
        variants={fadeInRight}
        transition={{ duration: 0.5, delay: 0.1 }}
        className="flex flex-1 flex-col justify-center px-4 py-12 sm:px-6 lg:flex-none lg:px-20 xl:px-24"
      >
        <div className="mx-auto w-full max-w-sm lg:w-96">
          {/* Logo */}
          <Link to="/" className="inline-flex items-center gap-2 group">
            <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-gradient-to-br from-brand-500 to-brand-600 shadow-md group-hover:shadow-lg transition-shadow">
              <ChefHat className="h-5 w-5 text-white" />
            </div>
            <span className="font-display text-2xl font-bold text-gray-900">Fe3dr</span>
          </Link>

          <motion.div
            initial={{ opacity: 0, y: 10 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: 0.2 }}
            className="mt-8"
          >
            <h2 className="font-display text-display-xs text-gray-900">Create your account</h2>
            <p className="mt-2 text-gray-600">
              Already have an account?{' '}
              <Link to="/login" className="font-medium text-brand-600 hover:text-brand-500 transition-colors">
                Sign in
              </Link>
            </p>
          </motion.div>

          <motion.div
            initial={{ opacity: 0, y: 10 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: 0.3 }}
            className="mt-8 space-y-4"
          >
            {/* Social signup buttons */}
            <Button
              variant="outline"
              size="lg"
              onClick={() => login('google')}
              className="w-full justify-center gap-3 py-3"
            >
              <svg className="h-5 w-5" viewBox="0 0 24 24">
                <path fill="#4285F4" d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z" />
                <path fill="#34A853" d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z" />
                <path fill="#FBBC05" d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z" />
                <path fill="#EA4335" d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z" />
              </svg>
              Sign up with Google
            </Button>

            <Button
              variant="outline"
              size="lg"
              onClick={() => login('facebook')}
              className="w-full justify-center gap-3 py-3"
            >
              <svg className="h-5 w-5 text-blue-600" fill="currentColor" viewBox="0 0 24 24">
                <path d="M24 12.073c0-6.627-5.373-12-12-12s-12 5.373-12 12c0 5.99 4.388 10.954 10.125 11.854v-8.385H7.078v-3.47h3.047V9.43c0-3.007 1.792-4.669 4.533-4.669 1.312 0 2.686.235 2.686.235v2.953H15.83c-1.491 0-1.956.925-1.956 1.874v2.25h3.328l-.532 3.47h-2.796v8.385C19.612 23.027 24 18.062 24 12.073z" />
              </svg>
              Sign up with Facebook
            </Button>

            <div className="relative mt-6">
              <div className="absolute inset-0 flex items-center">
                <div className="w-full border-t border-gray-200" />
              </div>
              <div className="relative flex justify-center text-sm">
                <span className="bg-white px-3 text-gray-500">Or</span>
              </div>
            </div>

            <Button
              variant="primary"
              size="lg"
              onClick={() => register()}
              className="w-full"
            >
              Sign up with email
            </Button>

            <p className="mt-4 text-center text-xs text-gray-500">
              By signing up, you agree to our{' '}
              <Link to="/terms" className="text-brand-600 hover:text-brand-500 transition-colors">
                Terms of Service
              </Link>{' '}
              and{' '}
              <Link to="/privacy" className="text-brand-600 hover:text-brand-500 transition-colors">
                Privacy Policy
              </Link>
            </p>
          </motion.div>
        </div>
      </motion.div>
    </div>
  );
}
