import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { motion, AnimatePresence } from 'framer-motion';
import { toast } from 'sonner';
import {
  ArrowLeft,
  ArrowRight,
  CheckCircle2,
  SkipForward,
  User,
  UtensilsCrossed,
  MapPin,
} from 'lucide-react';
import { useOnboardingStore } from '@/app/store/onboarding-store';
import { useAuth } from '@/app/providers/AuthProvider';
import { useAuthStore } from '@/app/store/auth-store';
import { apiClient } from '@/shared/services/api-client';
import { Button } from '@/shared/components/ui/Button';
import { StepBasicInfo } from '../components/StepBasicInfo';
import { StepPreferences } from '../components/StepPreferences';
import { StepAddress } from '../components/StepAddress';

const STEPS = [
  {
    title: 'Basic Info',
    description: 'Your details',
    icon: User,
    // One line per step saying what the customer GETS, not what we want from
    // them. A wizard that only counts fields feels like paperwork; saying why
    // the step exists is what makes finishing it worth the effort.
    encouragement: 'Just the basics — this takes about a minute.',
  },
  {
    title: 'Preferences',
    description: 'Food & dietary',
    icon: UtensilsCrossed,
    encouragement: 'Tell us how you eat and we’ll put the right chefs first.',
  },
  {
    title: 'Address',
    description: 'Delivery location',
    icon: MapPin,
    encouragement: 'Last one — then you can start ordering.',
  },
];

const TOTAL_STEPS = 3;

function validateStep(step: number, data: ReturnType<typeof useOnboardingStore.getState>['data']): Record<string, string> {
  const errors: Record<string, string> = {};
  if (step === 0) {
    if (!data.firstName.trim()) errors.firstName = 'First name is required';
    if (!data.lastName.trim()) errors.lastName = 'Last name is required';
  }
  return errors;
}

export default function UserInfoPage() {
  const navigate = useNavigate();
  const { user } = useAuth();
  const { currentStep, data, nextStep, prevStep, reset } = useOnboardingStore();
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [celebrating, setCelebrating] = useState(false);

  // Email-verification gate (#12). The API requires an OTP-verified email before
  // it will accept the completed profile; this UI had no step for it.
  const [needsEmailVerify, setNeedsEmailVerify] = useState(false);
  const [otpSent, setOtpSent] = useState(false);
  const [otpCode, setOtpCode] = useState('');
  const [otpBusy, setOtpBusy] = useState(false);
  const [cooldown, setCooldown] = useState(0);

  useEffect(() => {
    if (cooldown <= 0) return;
    const id = setInterval(() => setCooldown((s) => (s > 0 ? s - 1 : 0)), 1000);
    return () => clearInterval(id);
  }, [cooldown]);

  // Pre-fill from session user
  if (user?.firstName && !data.firstName) {
    useOnboardingStore.getState().updateData({
      firstName: user.firstName,
      lastName: user.lastName ?? '',
    });
  }

  const handleNext = () => {
    const validationErrors = validateStep(currentStep, data);
    setErrors(validationErrors);
    if (Object.keys(validationErrors).length > 0) {
      toast.error('Please fill in the required fields');
      return;
    }
    nextStep();
    setErrors({});
  };

  const submitOnboarding = async () => {
    await apiClient.post('/customer/onboarding/complete', data);
    reset();
    useAuthStore.getState().setOnboardingCompleted(true);
    // Land on a completion moment rather than snapping straight to the home
    // page. Finishing a multi-step form should feel like finishing something;
    // a toast that disappears mid-navigation doesn't read as an ending.
    setCelebrating(true);
    window.setTimeout(() => navigate('/', { replace: true }), 1600);
  };

  const handleComplete = async () => {
    setIsSubmitting(true);
    try {
      await submitOnboarding();
    } catch (err: unknown) {
      const e = err as { status?: number; error?: string };
      // 428 = the API's email-verification gate (EnsureLoginEmailVerified).
      // There was no way to satisfy it from this UI at all, so completing the
      // profile was impossible: every attempt failed and the generic "try
      // again" told people to repeat something that could never succeed.
      if (e?.status === 428) {
        setNeedsEmailVerify(true);
        void sendOtp();
        return;
      }
      // Otherwise show what the server actually said, not a blanket message.
      toast.error(e?.error || 'Failed to save. Please try again.');
    } finally {
      setIsSubmitting(false);
    }
  };

  const sendOtp = async () => {
    if (otpBusy || cooldown > 0) return;
    setOtpBusy(true);
    try {
      await apiClient.post('/account/email/otp/request', { email: user?.email });
      setOtpSent(true);
      setCooldown(60);
      toast.success('Verification code sent to your email');
    } catch (err: unknown) {
      const e = err as { error?: string };
      toast.error(e?.error || "Couldn't send the code. Please try again.");
    } finally {
      setOtpBusy(false);
    }
  };

  const verifyOtpAndFinish = async () => {
    if (otpCode.length !== 6) return;
    setOtpBusy(true);
    try {
      await apiClient.post('/account/email/otp/verify', {
        email: user?.email,
        code: otpCode,
      });
      setNeedsEmailVerify(false);
      setIsSubmitting(true);
      await submitOnboarding();
    } catch (err: unknown) {
      const e = err as { error?: string };
      toast.error(e?.error || 'That code did not work. Please try again.');
    } finally {
      setOtpBusy(false);
      setIsSubmitting(false);
    }
  };

  const handleSkip = async () => {
    try {
      await apiClient.post('/customer/onboarding/skip');
      reset();
      useAuthStore.getState().setOnboardingCompleted(true);
      navigate('/', { replace: true });
    } catch {
      toast.error('Something went wrong.');
    }
  };

  const isLastStep = currentStep === TOTAL_STEPS - 1;

  // Counts the step you're on, so arriving reads as 33% rather than 0%. The
  // vendor scaffold counts only *finished* steps and therefore opens at 0% —
  // defensible there across seven steps, but over three it makes a fresh start
  // look like no start, which is the opposite of encouraging.
  const percent = Math.round(((currentStep + 1) / TOTAL_STEPS) * 100);

  if (celebrating) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-background px-4">
        <motion.div
          initial={{ opacity: 0, y: 8 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.25, ease: [0.22, 1, 0.36, 1] }}
          className="text-center"
          role="status"
        >
          <div className="mx-auto flex h-16 w-16 items-center justify-center rounded-full bg-primary/10 text-primary">
            <CheckCircle2 aria-hidden="true" className="h-8 w-8" />
          </div>
          <p className="mt-4 text-xl font-semibold text-foreground">You&rsquo;re all set</p>
          <p className="mt-1 text-sm text-muted-foreground">
            Profile complete — taking you to the good stuff.
          </p>
        </motion.div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-background">
      {/* Header */}
      <header className="sticky top-0 z-40 border-b bg-card/95 backdrop-blur supports-[backdrop-filter]:bg-card/60">
        <div className="mx-auto flex h-16 max-w-3xl items-center justify-between px-4">
          <div>
            <h1 className="text-lg font-medium text-foreground">Complete Your Profile</h1>
            <p className="text-xs text-muted-foreground">
              Step {currentStep + 1} of {TOTAL_STEPS} &mdash; {STEPS[currentStep]?.title}
            </p>
          </div>
          <Button variant="ghost" size="sm" onClick={handleSkip}>
            <SkipForward className="h-4 w-4 mr-1" />
            Skip
          </Button>
        </div>
      </header>

      <div className="mx-auto max-w-3xl px-4 py-8">
        {/* Progress bar */}
        <div className="mb-8">
          <div className="flex items-center justify-between mb-2">
            {STEPS.map((step, i) => (
              <div key={i} className="flex items-center gap-2">
                <div
                  className={`flex h-8 w-8 items-center justify-center rounded-full text-sm font-semibold transition-colors ${
                    i <= currentStep
                      ? 'bg-primary text-primary-foreground'
                      : 'bg-secondary text-muted-foreground'
                  }`}
                >
                  {i < currentStep ? <CheckCircle2 className="h-4 w-4"  aria-hidden="true" /> : i + 1}
                </div>
                <span className="hidden sm:inline text-sm font-medium text-foreground">
                  {step.title}
                </span>
              </div>
            ))}
          </div>
          <div
            className="h-2 rounded-full bg-secondary"
            role="progressbar"
            aria-valuenow={percent}
            aria-valuemin={0}
            aria-valuemax={100}
            aria-label={`Profile ${percent} percent complete`}
          >
            <div
              className="h-full rounded-full bg-primary transition-all duration-300"
              style={{ width: `${percent}%` }}
            />
          </div>

          {/* The number and the line under it are the motivating part: the bar
              alone shows position, but "60% complete" plus what the next step
              buys you is what gets someone to finish. */}
          <div className="mt-2 flex items-baseline justify-between gap-3">
            <p className="text-sm text-muted-foreground">
              {STEPS[currentStep]?.encouragement}
            </p>
            <span className="shrink-0 text-sm font-semibold tabular-nums text-primary">
              {percent}%
            </span>
          </div>
        </div>

        {/* Step Content */}
        <AnimatePresence mode="wait">
          <motion.div
            key={currentStep}
            initial={{ opacity: 0, x: 20 }}
            animate={{ opacity: 1, x: 0 }}
            exit={{ opacity: 0, x: -20 }}
            transition={{ duration: 0.2 }}
          >
            {currentStep === 0 && <StepBasicInfo errors={errors} />}
            {currentStep === 1 && <StepPreferences />}
            {currentStep === 2 && <StepAddress />}
          </motion.div>
        </AnimatePresence>

        {/* Email verification — shown only once the API tells us it's required,
            so the common path stays a three-step wizard. */}
        {needsEmailVerify && (
          <div className="mt-6 rounded-lg border border-primary/30 bg-primary/5 p-4">
            <h2 className="text-sm font-semibold text-foreground">
              Verify your email to finish
            </h2>
            <p className="mt-1 text-sm text-muted-foreground">
              {otpSent
                ? `We sent a 6-digit code to ${user?.email ?? 'your email'}. Enter it below to save your profile.`
                : `We need to confirm ${user?.email ?? 'your email'} before saving your profile.`}
            </p>

            <div className="mt-3 flex flex-wrap items-center gap-2">
              <label htmlFor="otp-code" className="sr-only">
                Verification code
              </label>
              <input
                id="otp-code"
                inputMode="numeric"
                autoComplete="one-time-code"
                maxLength={6}
                value={otpCode}
                onChange={(e) => setOtpCode(e.target.value.replace(/\D/g, '').slice(0, 6))}
                placeholder="000000"
                className="w-32 rounded-md border bg-background px-3 py-2 text-center tracking-[0.3em] tabular-nums"
              />
              <Button
                type="button"
                variant="primary"
                onClick={verifyOtpAndFinish}
                isLoading={otpBusy}
                disabled={otpCode.length !== 6 || otpBusy}
              >
                Verify &amp; save
              </Button>
              <Button
                type="button"
                variant="ghost"
                onClick={sendOtp}
                disabled={cooldown > 0 || otpBusy}
              >
                {cooldown > 0 ? `Resend in ${cooldown}s` : 'Resend code'}
              </Button>
            </div>
          </div>
        )}

        {/* Navigation */}
        <div className="mt-8 flex items-center justify-between border-t pt-6">
          <Button
            type="button"
            variant="ghost"
            onClick={() => {
              prevStep();
              setErrors({});
            }}
            disabled={currentStep === 0}
            leftIcon={<ArrowLeft className="h-4 w-4"  aria-hidden="true" />}
          >
            Back
          </Button>

          {isLastStep ? (
            <Button
              type="button"
              variant="primary"
              size="lg"
              onClick={handleComplete}
              isLoading={isSubmitting}
              leftIcon={<CheckCircle2 className="h-4 w-4"  aria-hidden="true" />}
            >
              Complete
            </Button>
          ) : (
            <Button
              type="button"
              variant="primary"
              size="lg"
              onClick={handleNext}
              rightIcon={<ArrowRight className="h-4 w-4"  aria-hidden="true" />}
            >
              Continue
            </Button>
          )}
        </div>
      </div>
    </div>
  );
}
