import { useState } from 'react';
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
  { title: 'Basic Info', description: 'Your details', icon: User },
  { title: 'Preferences', description: 'Food & dietary', icon: UtensilsCrossed },
  { title: 'Address', description: 'Delivery location', icon: MapPin },
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

  const handleComplete = async () => {
    setIsSubmitting(true);
    try {
      await apiClient.post('/customer/onboarding/complete', data);
      toast.success('Your preferences have been saved!');
      reset();
      useAuthStore.getState().setOnboardingCompleted(true);
      navigate('/', { replace: true });
    } catch {
      toast.error('Failed to save. Please try again.');
    } finally {
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

  return (
    <div className="min-h-screen bg-background">
      {/* Header */}
      <header className="sticky top-0 z-40 border-b bg-card/95 backdrop-blur supports-[backdrop-filter]:bg-card/60">
        <div className="mx-auto flex h-16 max-w-3xl items-center justify-between px-4">
          <div>
            <h1 className="text-lg font-bold text-foreground">Complete Your Profile</h1>
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
                  {i < currentStep ? <CheckCircle2 className="h-4 w-4" /> : i + 1}
                </div>
                <span className="hidden sm:inline text-sm font-medium text-foreground">
                  {step.title}
                </span>
              </div>
            ))}
          </div>
          <div className="h-2 rounded-full bg-secondary">
            <div
              className="h-full rounded-full bg-primary transition-all duration-300"
              style={{ width: `${((currentStep + 1) / TOTAL_STEPS) * 100}%` }}
            />
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
            leftIcon={<ArrowLeft className="h-4 w-4" />}
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
              leftIcon={<CheckCircle2 className="h-4 w-4" />}
            >
              Complete
            </Button>
          ) : (
            <Button
              type="button"
              variant="primary"
              size="lg"
              onClick={handleNext}
              rightIcon={<ArrowRight className="h-4 w-4" />}
            >
              Continue
            </Button>
          )}
        </div>
      </div>
    </div>
  );
}
