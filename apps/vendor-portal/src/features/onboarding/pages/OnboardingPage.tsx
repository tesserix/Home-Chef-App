import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { motion, AnimatePresence } from 'framer-motion';
import { Stepper, StepperItem } from '@tesserix/web';
import { toast } from 'sonner';
import {
  ChefHat,
  ArrowLeft,
  ArrowRight,
  Send,
  User,
  UtensilsCrossed,
  Clock,
  FileText,
  Shield,
  CheckCircle2,
} from 'lucide-react';
import { useOnboardingStore } from '@/app/store/onboarding-store';
import { useAuth } from '@/app/providers/AuthProvider';
import { apiClient } from '@/shared/services/api-client';
import { Button } from '@/shared/components/ui/Button';
import { StepPersonalInfo } from '../components/StepPersonalInfo';
import { StepKitchenDetails } from '../components/StepKitchenDetails';
import { StepOperations } from '../components/StepOperations';
import { StepDocuments } from '../components/StepDocuments';
import { StepPolicies } from '../components/StepPolicies';
import { StepReview } from '../components/StepReview';

const STEPS = [
  { title: 'Personal Info', description: 'Your details & address', icon: User },
  { title: 'Kitchen Details', description: 'About your kitchen', icon: UtensilsCrossed },
  { title: 'Operations', description: 'Hours & pricing', icon: Clock },
  { title: 'Documents', description: 'ID & kitchen photos', icon: FileText },
  { title: 'Policies & Review', description: 'Agreements & submit', icon: Shield },
];

// We combine step 5 (policies) and review into one final step
const TOTAL_DISPLAY_STEPS = 5;

function validateStep(step: number, data: ReturnType<typeof useOnboardingStore.getState>['data']): Record<string, string> {
  const errors: Record<string, string> = {};

  if (step === 0) {
    if (!data.fullName.trim()) errors.fullName = 'Full name is required';
    if (!data.phone.trim()) errors.phone = 'Phone number is required';
    if (!data.email.trim()) errors.email = 'Email is required';
    if (!data.kitchenAddress.line1.trim()) errors['kitchenAddress.line1'] = 'Address is required';
    if (!data.kitchenAddress.city.trim()) errors['kitchenAddress.city'] = 'City is required';
    if (!data.kitchenAddress.state.trim()) errors['kitchenAddress.state'] = 'State is required';
    if (!data.kitchenAddress.postalCode.trim()) errors['kitchenAddress.postalCode'] = 'PIN code is required';
  }

  if (step === 1) {
    if (!data.businessName.trim()) errors.businessName = 'Kitchen name is required';
    if (data.description.length < 20) errors.description = 'Description must be at least 20 characters';
    if (data.cuisines.length === 0) errors.cuisines = 'Select at least one cuisine';
  }

  if (step === 2) {
    if (data.serviceRadius < 1) errors.serviceRadius = 'Minimum 1 km radius';
  }

  return errors;
}

export default function OnboardingPage() {
  const navigate = useNavigate();
  const { user, onboardingStatus, adminNotes } = useAuth();
  const {
    currentStep,
    data,
    setStep,
    nextStep,
    prevStep,
    reset,
  } = useOnboardingStore();
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [isCreatingProfile, setIsCreatingProfile] = useState(false);
  const [showReview, setShowReview] = useState(false);

  // Always pre-fill email/name from current logged-in user's session
  // This ensures a new user never sees a previous user's data
  useEffect(() => {
    if (user?.email) {
      const updates: Partial<typeof data> = { email: user.email };
      const name = [user.firstName, user.lastName].filter(Boolean).join(' ');
      if (name) updates.fullName = name;
      useOnboardingStore.getState().updateData(updates);
    }
  }, [user?.email, user?.firstName, user?.lastName]); // eslint-disable-line react-hooks/exhaustive-deps

  const ensureProfile = async (): Promise<boolean> => {
    setIsCreatingProfile(true);
    try {
      await apiClient.post('/chef/onboarding', data);
      return true;
    } catch {
      toast.error('Failed to save your details. Please try again.');
      return false;
    } finally {
      setIsCreatingProfile(false);
    }
  };

  const handleNext = async () => {
    const validationErrors = validateStep(currentStep, data);
    setErrors(validationErrors);

    if (Object.keys(validationErrors).length > 0) {
      toast.error('Please fill in all required fields');
      return;
    }

    // Create/update chef profile before documents step so file uploads have a chef ID
    if (currentStep === 2) {
      const ok = await ensureProfile();
      if (!ok) return;
    }

    if (currentStep === TOTAL_DISPLAY_STEPS - 1) {
      // Last step — show review
      setShowReview(true);
    } else {
      nextStep();
      setErrors({});
    }
  };

  const handleBack = () => {
    if (showReview) {
      setShowReview(false);
    } else {
      prevStep();
      setErrors({});
    }
  };

  const handleEditFromReview = (step: number) => {
    setShowReview(false);
    setStep(step);
  };

  const handleSubmit = async () => {
    // Final validation
    if (!data.acceptedTerms || !data.acceptedHygienePolicy || !data.acceptedCancellationPolicy) {
      toast.error('Please accept all required policies');
      setShowReview(false);
      setStep(4);
      return;
    }

    setIsSubmitting(true);
    try {
      await apiClient.post('/chef/onboarding', data);
      toast.success('Application submitted! We\'ll review and get back to you within 24-48 hours.');
      reset();
      navigate('/dashboard');
    } catch {
      toast.error('Failed to submit application. Please try again.');
    } finally {
      setIsSubmitting(false);
    }
  };

  const displayStep = showReview ? TOTAL_DISPLAY_STEPS : currentStep;

  return (
    <div className="min-h-screen bg-background">
      {/* Header */}
      <header className="sticky top-0 z-40 border-b bg-card/95 backdrop-blur supports-[backdrop-filter]:bg-card/60">
        <div className="mx-auto flex h-16 max-w-4xl items-center gap-3 px-4">
          <ChefHat className="h-7 w-7 text-primary" />
          <div>
            <h1 className="text-lg font-bold text-foreground">Set Up Your Kitchen</h1>
            <p className="text-xs text-muted-foreground">
              {showReview
                ? 'Review your application'
                : `Step ${currentStep + 1} of ${TOTAL_DISPLAY_STEPS}`}
            </p>
          </div>
        </div>
      </header>

      <div className="mx-auto max-w-4xl px-4 py-8">
        {/* Rejection/Info Request Banner */}
        {onboardingStatus === 'rejected' && (
          <div className="mb-6 rounded-xl border border-destructive/30 bg-destructive/5 p-4">
            <h3 className="font-semibold text-destructive">Application Rejected</h3>
            <p className="mt-1 text-sm text-destructive/80">
              Your previous application was not approved. Please review the feedback below and re-submit.
            </p>
            {adminNotes && (
              <div className="mt-3 rounded-lg bg-destructive/10 p-3">
                <p className="text-xs font-medium text-destructive">Admin Notes:</p>
                <p className="mt-1 text-sm text-foreground">{adminNotes}</p>
              </div>
            )}
          </div>
        )}
        {onboardingStatus === 'info_requested' && (
          <div className="mb-6 rounded-xl border border-warning/30 bg-warning/5 p-4">
            <h3 className="font-semibold text-warning">More Information Needed</h3>
            <p className="mt-1 text-sm text-warning/80">
              The admin team needs additional information before approving your application.
            </p>
            {adminNotes && (
              <div className="mt-3 rounded-lg bg-warning/10 p-3">
                <p className="text-xs font-medium text-warning">Admin Notes:</p>
                <p className="mt-1 text-sm text-foreground">{adminNotes}</p>
              </div>
            )}
          </div>
        )}

        {/* Stepper */}
        <div className="mb-8 hidden sm:block">
          <Stepper currentStep={displayStep + 1} totalSteps={TOTAL_DISPLAY_STEPS + 1}>
            {STEPS.map((step, i) => (
              <StepperItem
                key={i}
                step={i + 1}
                title={step.title}
                description={step.description}
              />
            ))}
            <StepperItem
              step={TOTAL_DISPLAY_STEPS + 1}
              title="Review"
              description="Submit application"
            />
          </Stepper>
        </div>

        {/* Mobile step indicator */}
        <div className="mb-6 sm:hidden">
          <div className="flex items-center justify-between text-sm">
            <span className="font-medium text-foreground">
              {showReview ? 'Review & Submit' : STEPS[currentStep]?.title}
            </span>
            <span className="text-muted-foreground">
              {showReview ? `${TOTAL_DISPLAY_STEPS + 1}` : `${currentStep + 1}`}/{TOTAL_DISPLAY_STEPS + 1}
            </span>
          </div>
          <div className="mt-2 h-2 rounded-full bg-secondary">
            <div
              className="h-full rounded-full bg-primary transition-all duration-300"
              style={{ width: `${((showReview ? TOTAL_DISPLAY_STEPS + 1 : currentStep + 1) / (TOTAL_DISPLAY_STEPS + 1)) * 100}%` }}
            />
          </div>
        </div>

        {/* Step Content */}
        <AnimatePresence mode="wait">
          <motion.div
            key={showReview ? 'review' : currentStep}
            initial={{ opacity: 0, x: 20 }}
            animate={{ opacity: 1, x: 0 }}
            exit={{ opacity: 0, x: -20 }}
            transition={{ duration: 0.2 }}
          >
            {showReview ? (
              <StepReview onEdit={handleEditFromReview} />
            ) : (
              <>
                {currentStep === 0 && <StepPersonalInfo errors={errors} />}
                {currentStep === 1 && <StepKitchenDetails errors={errors} />}
                {currentStep === 2 && <StepOperations errors={errors} />}
                {currentStep === 3 && <StepDocuments errors={errors} />}
                {currentStep === 4 && <StepPolicies errors={errors} />}
              </>
            )}
          </motion.div>
        </AnimatePresence>

        {/* Navigation */}
        <div className="mt-8 flex items-center justify-between border-t pt-6">
          <Button
            type="button"
            variant="ghost"
            onClick={handleBack}
            disabled={currentStep === 0 && !showReview}
            leftIcon={<ArrowLeft className="h-4 w-4" />}
          >
            Back
          </Button>

          <div className="flex gap-3">
            {showReview ? (
              <Button
                type="button"
                variant="primary"
                size="lg"
                onClick={handleSubmit}
                isLoading={isSubmitting}
                leftIcon={<Send className="h-4 w-4" />}
                disabled={
                  !data.acceptedTerms ||
                  !data.acceptedHygienePolicy ||
                  !data.acceptedCancellationPolicy
                }
              >
                Submit Application
              </Button>
            ) : (
              <Button
                type="button"
                variant="primary"
                size="lg"
                onClick={handleNext}
                isLoading={isCreatingProfile}
                rightIcon={
                  currentStep === TOTAL_DISPLAY_STEPS - 1 ? (
                    <CheckCircle2 className="h-4 w-4" />
                  ) : (
                    <ArrowRight className="h-4 w-4" />
                  )
                }
              >
                {currentStep === TOTAL_DISPLAY_STEPS - 1 ? 'Review Application' : 'Continue'}
              </Button>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
