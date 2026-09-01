import { useState } from 'react';
import { Link } from 'react-router';
import { motion } from 'framer-motion';
import { ChefHat, Check, ArrowRight } from 'lucide-react';
import { useAuth } from '@/app/providers/AuthProvider';
import { Button } from '@/shared/components/ui';
import { fadeInLeft, fadeInRight } from '@/shared/utils/animations';

const BENEFITS = [
  'Access to 500+ home chefs',
  'Authentic homemade food',
  'Fast & reliable delivery',
  'Support local home chefs',
];

// Account creation happens on the hosted Zitadel registration page; this page
// captures the choices that belong to Fe3dr, not to the identity provider —
// the DPDP §6 marketing opt-in and agreement to our terms — then hands off.
// A pending ?ref=CODE referral is applied post-signup by the onboarding flow.
export default function RegisterPage() {
  const { register } = useAuth();
  const [error, setError] = useState('');
  const [agreeTerms, setAgreeTerms] = useState(false);
  const [confirmAge, setConfirmAge] = useState(false);
  const [optInMarketing, setOptInMarketing] = useState(false);

  const handleContinue = async () => {
    if (!agreeTerms) {
      setError('Please agree to the Terms, Privacy Policy, and Refund Policy.');
      return;
    }
    if (!confirmAge) {
      setError('You must be at least 18 to use Fe3dr.');
      return;
    }
    setError('');
    await register({ marketingConsent: optInMarketing });
  };

  return (
    <div className="flex min-h-screen">
      {/* Left side - Image & Benefits */}
      <motion.div
        initial="hidden"
        animate="visible"
        variants={fadeInRight}
        transition={{ duration: 0.6 }}
        className="relative hidden w-0 flex-1 lg:block"
      >
        <img
          className="absolute inset-0 h-full w-full object-cover"
          src="https://images.unsplash.com/photo-1543352634-a1c51d9f1fa7?w=1200&h=900&fit=crop"
          alt="Home chef preparing food"
          fetchPriority="high"
          decoding="async"
        />
        <div aria-hidden="true" className="absolute inset-0 bg-gradient-to-t from-ink/60 via-ink/20 to-transparent" />
        <div className="absolute bottom-0 left-0 right-0 p-12">
          <ul className="space-y-3">
            {BENEFITS.map((benefit) => (
              <li key={benefit} className="flex items-center gap-3 text-on-photo">
                <span className="flex h-6 w-6 items-center justify-center rounded-full bg-herb">
                  <Check aria-hidden="true" className="h-3.5 w-3.5 text-paper" />
                </span>
                <span className="font-medium">{benefit}</span>
              </li>
            ))}
          </ul>
        </div>
      </motion.div>

      {/* Right side - Consent + hand-off */}
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

          <div className="mt-8">
            <h2 className="font-display text-display-xs text-ink">Create your account</h2>
            <p className="mt-2 text-ink-soft">
              Already have an account?{' '}
              <Link to="/login" className="font-medium text-herb hover:text-herb transition-colors">
                Sign in
              </Link>
            </p>
          </div>

          {error && (
            <div
              role="alert"
              className="mt-6 rounded-lg border border-paprika/30 bg-paprika-tint p-3 text-sm text-paprika"
            >
              {error}
            </div>
          )}

          <div className="mt-8 space-y-4">
            <label className="flex items-start gap-3">
              <input
                type="checkbox"
                checked={agreeTerms}
                onChange={(e) => setAgreeTerms(e.target.checked)}
                className="mt-1 h-4 w-4 rounded border-mist-strong text-herb focus-visible:ring-2 focus-visible:ring-herb/30"
              />
              <span className="text-sm text-ink-soft">
                I agree to the{' '}
                <Link to="/terms" className="text-herb hover:underline">Terms of Service</Link>,{' '}
                <Link to="/privacy" className="text-herb hover:underline">Privacy Policy</Link> and{' '}
                <Link to="/refund" className="text-herb hover:underline">Refund Policy</Link>.
              </span>
            </label>

            <label className="flex items-start gap-3">
              <input
                type="checkbox"
                checked={confirmAge}
                onChange={(e) => setConfirmAge(e.target.checked)}
                className="mt-1 h-4 w-4 rounded border-mist-strong text-herb focus-visible:ring-2 focus-visible:ring-herb/30"
              />
              <span className="text-sm text-ink-soft">I confirm that I am at least 18 years old.</span>
            </label>

            <label className="flex items-start gap-3">
              <input
                type="checkbox"
                checked={optInMarketing}
                onChange={(e) => setOptInMarketing(e.target.checked)}
                className="mt-1 h-4 w-4 rounded border-mist-strong text-herb focus-visible:ring-2 focus-visible:ring-herb/30"
              />
              <span className="text-sm text-ink-soft">
                Send me offers, new-chef announcements and food inspiration. Optional —
                you can change this any time.
              </span>
            </label>

            <Button
              variant="primary"
              size="lg"
              onClick={handleContinue}
              className="w-full rounded-full"
            >
              Continue to sign up
              <ArrowRight aria-hidden="true" className="ml-2 h-4 w-4" />
            </Button>
            <p className="text-center text-sm text-ink-muted">
              You'll create your account securely on our account page — email,
              Google and Apple all work there.
            </p>
          </div>
        </div>
      </motion.div>
    </div>
  );
}
