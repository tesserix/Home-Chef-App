import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { useMutation } from '@tanstack/react-query';
import {
  Calendar,
  Clock,
  MapPin,
  Users,
  UtensilsCrossed,
  DollarSign,

  ChefHat,
  ArrowRight,
  Info,
} from 'lucide-react';
import { toast } from 'sonner';
import { apiClient } from '@/shared/services/api-client';
import { Button } from '@/shared/components/ui';
import type { CateringRequest, CateringServiceType } from '@/shared/types';

const cateringSchema = z.object({
  eventDate: z.string().min(1, 'Event date is required'),
  eventTime: z.string().min(1, 'Event time is required'),
  guestCount: z.number().min(10, 'Minimum 10 guests required').max(500, 'Maximum 500 guests'),
  cuisinePreferences: z.array(z.string()).min(1, 'Select at least one cuisine'),
  dietaryRequirements: z.array(z.string()).optional(),
  serviceType: z.enum(['delivery_only', 'setup', 'full_service']),
  budgetMin: z.number().min(0).optional(),
  budgetMax: z.number().min(0).optional(),
  description: z.string().optional(),
  addressLine1: z.string().min(5, 'Address is required'),
  addressLine2: z.string().optional(),
  city: z.string().min(2, 'City is required'),
  state: z.string().min(2, 'State is required'),
  postalCode: z.string().min(5, 'Postal code is required'),
});

type CateringFormData = z.infer<typeof cateringSchema>;

const CUISINES = [
  'South Indian',
  'North Indian',
  'Italian',
  'Mexican',
  'Chinese',
  'Japanese',
  'Thai',
  'Mediterranean',
  'Continental',
  'American',
];

const DIETARY_OPTIONS = [
  'Vegetarian',
  'Vegan',
  'Gluten-Free',
  'Dairy-Free',
  'Nut-Free',
  'Halal',
  'Kosher',
];

const SERVICE_TYPES: { value: CateringServiceType; label: string; description: string }[] = [
  {
    value: 'delivery_only',
    label: 'Delivery Only',
    description: 'Food is delivered to your venue. You handle setup and serving.',
  },
  {
    value: 'setup',
    label: 'Delivery + Setup',
    description: 'Food is delivered and set up at your venue. You handle serving.',
  },
  {
    value: 'full_service',
    label: 'Full Service',
    description: 'Complete catering service including staff, setup, and serving.',
  },
];

export default function CateringRequestPage() {
  const navigate = useNavigate();
  const [step, setStep] = useState(1);

  const {
    register,
    handleSubmit,
    formState: { errors },
    watch,
    setValue,
    trigger,
  } = useForm<CateringFormData>({
    resolver: zodResolver(cateringSchema),
    defaultValues: {
      cuisinePreferences: [],
      dietaryRequirements: [],
      serviceType: 'delivery_only',
      guestCount: 50,
    },
  });

  const createRequestMutation = useMutation({
    mutationFn: (data: CateringFormData) =>
      apiClient.post<CateringRequest>('/catering/requests', data),
    onSuccess: () => {
      toast.success('Catering request submitted successfully!');
      navigate('/catering/quotes');
    },
    onError: () => {
      toast.error('Failed to submit request. Please try again.');
    },
  });

  const cuisines = watch('cuisinePreferences') || [];
  const dietary = watch('dietaryRequirements') || [];
  const serviceType = watch('serviceType');

  const toggleCuisine = (cuisine: string) => {
    if (cuisines.includes(cuisine)) {
      setValue('cuisinePreferences', cuisines.filter((c) => c !== cuisine));
    } else {
      setValue('cuisinePreferences', [...cuisines, cuisine]);
    }
  };

  const toggleDietary = (option: string) => {
    if (dietary.includes(option)) {
      setValue('dietaryRequirements', dietary.filter((d) => d !== option));
    } else {
      setValue('dietaryRequirements', [...dietary, option]);
    }
  };

  const nextStep = async () => {
    let fieldsToValidate: (keyof CateringFormData)[] = [];

    if (step === 1) {
      fieldsToValidate = ['eventDate', 'eventTime', 'guestCount'];
    } else if (step === 2) {
      fieldsToValidate = ['cuisinePreferences', 'serviceType'];
    }

    const isValid = await trigger(fieldsToValidate);
    if (isValid) {
      setStep(step + 1);
    }
  };

  const onSubmit = (data: CateringFormData) => {
    createRequestMutation.mutate(data);
  };

  return (
    <div className="min-h-screen bg-paper py-8">
      <div className="container-app max-w-3xl">
        {/* Header */}
        <div className="text-center">
          <div className="mx-auto flex h-16 w-16 items-center justify-center rounded-full bg-herb-tint">
            <UtensilsCrossed className="h-8 w-8 text-herb" />
          </div>
          <h1 className="mt-4 font-display text-2xl font-semibold text-ink md:text-3xl">
            Request Catering Quotes
          </h1>
          <p className="mt-2 text-ink-soft">
            Tell us about your event and receive quotes from our home chefs
          </p>
        </div>

        {/* Progress Steps */}
        <div className="mt-8 flex items-center justify-center gap-4">
          {[1, 2, 3].map((s) => (
            <div key={s} className="flex items-center">
              <div
                className={`flex h-10 w-10 items-center justify-center rounded-full font-medium ${
                  s < step
                    ? 'bg-herb text-paper'
                    : s === step
                    ? 'bg-herb text-paper ring-4 ring-herb/30'
                    : 'bg-mist text-ink-muted'
                }`}
              >
                {s}
              </div>
              {s < 3 && (
                <div
                  className={`h-1 w-12 md:w-24 ${
                    s < step ? 'bg-herb' : 'bg-mist'
                  }`}
                />
              )}
            </div>
          ))}
        </div>
        <div className="mt-2 flex justify-center gap-4 text-sm text-ink-muted">
          <span className="w-20 text-center">Event Details</span>
          <span className="w-24 md:w-32" />
          <span className="w-20 text-center">Preferences</span>
          <span className="w-24 md:w-32" />
          <span className="w-20 text-center">Location</span>
        </div>

        {/* Form */}
        <form onSubmit={handleSubmit(onSubmit)} className="mt-8">
          {/* Step 1: Event Details */}
          {step === 1 && (
            <div className="rounded-xl bg-bone p-6 shadow-1 md:p-8">
              <h2 className="text-xl font-semibold text-ink">Event Details</h2>
              <p className="mt-1 text-ink-soft">Tell us about your event</p>

              <div className="mt-6 space-y-6">
                <div className="grid gap-6 md:grid-cols-2">
                  <div>
                    <label className="block text-sm font-medium text-ink-soft">
                      <Calendar className="mr-2 inline h-4 w-4" />
                      Event Date
                    </label>
                    <input
                      type="date"
                      {...register('eventDate')}
                      min={new Date().toISOString().split('T')[0]}
                      className="input-base mt-1"
                    />
                    {errors.eventDate && (
                      <p className="mt-1 text-xs text-paprika">{errors.eventDate.message}</p>
                    )}
                  </div>

                  <div>
                    <label className="block text-sm font-medium text-ink-soft">
                      <Clock className="mr-2 inline h-4 w-4" />
                      Event Time
                    </label>
                    <input
                      type="time"
                      {...register('eventTime')}
                      className="input-base mt-1"
                    />
                    {errors.eventTime && (
                      <p className="mt-1 text-xs text-paprika">{errors.eventTime.message}</p>
                    )}
                  </div>
                </div>

                <div>
                  <label className="block text-sm font-medium text-ink-soft">
                    <Users className="mr-2 inline h-4 w-4" />
                    Number of Guests
                  </label>
                  <input
                    type="number"
                    {...register('guestCount', { valueAsNumber: true })}
                    min={10}
                    max={500}
                    className="input-base mt-1"
                  />
                  {errors.guestCount && (
                    <p className="mt-1 text-xs text-paprika">{errors.guestCount.message}</p>
                  )}
                  <p className="mt-1 text-sm text-ink-muted">Minimum 10, Maximum 500 guests</p>
                </div>

                <div className="grid gap-6 md:grid-cols-2">
                  <div>
                    <label className="block text-sm font-medium text-ink-soft">
                      <DollarSign className="mr-2 inline h-4 w-4" />
                      Budget Range (Optional)
                    </label>
                    <div className="mt-1 flex gap-2">
                      <input
                        type="number"
                        {...register('budgetMin', { valueAsNumber: true })}
                        placeholder="Min"
                        className="input-base"
                      />
                      <span className="flex items-center text-ink-muted">to</span>
                      <input
                        type="number"
                        {...register('budgetMax', { valueAsNumber: true })}
                        placeholder="Max"
                        className="input-base"
                      />
                    </div>
                  </div>
                </div>
              </div>

              <div className="mt-8 flex justify-end">
                <Button
                  type="button"
                  variant="primary"
                  onClick={nextStep}
                  rightIcon={<ArrowRight aria-hidden="true" className="h-5 w-5" />}
                >
                  Continue
                </Button>
              </div>
            </div>
          )}

          {/* Step 2: Preferences */}
          {step === 2 && (
            <div className="rounded-xl bg-bone p-6 shadow-1 md:p-8">
              <h2 className="text-xl font-semibold text-ink">Food Preferences</h2>
              <p className="mt-1 text-ink-soft">Select your cuisine and dietary preferences</p>

              <div className="mt-6 space-y-6">
                <div>
                  <label className="block text-sm font-medium text-ink-soft mb-3">
                    Cuisine Preferences (Select at least one)
                  </label>
                  <div className="flex flex-wrap gap-2">
                    {CUISINES.map((cuisine) => (
                      <button
                        key={cuisine}
                        type="button"
                        onClick={() => toggleCuisine(cuisine)}
                        className={`rounded-full px-4 py-2 text-sm font-medium transition-colors ${
                          cuisines.includes(cuisine)
                            ? 'bg-herb text-paper'
                            : 'bg-mist text-ink-soft hover:bg-mist'
                        }`}
                      >
                        {cuisine}
                      </button>
                    ))}
                  </div>
                  {errors.cuisinePreferences && (
                    <p className="mt-2 text-xs text-paprika">{errors.cuisinePreferences.message}</p>
                  )}
                </div>

                <div>
                  <label className="block text-sm font-medium text-ink-soft mb-3">
                    Dietary Requirements (Optional)
                  </label>
                  <div className="flex flex-wrap gap-2">
                    {DIETARY_OPTIONS.map((option) => (
                      <button
                        key={option}
                        type="button"
                        onClick={() => toggleDietary(option)}
                        className={`rounded-full px-4 py-2 text-sm font-medium transition-colors ${
                          dietary.includes(option)
                            ? 'bg-herb text-paper'
                            : 'bg-mist text-ink-soft hover:bg-mist'
                        }`}
                      >
                        {option}
                      </button>
                    ))}
                  </div>
                </div>

                <div>
                  <label className="block text-sm font-medium text-ink-soft mb-3">
                    Service Type
                  </label>
                  <div className="space-y-3">
                    {SERVICE_TYPES.map((type) => (
                      <label
                        key={type.value}
                        className={`flex cursor-pointer items-start gap-3 rounded-lg border p-4 transition-colors ${
                          serviceType === type.value
                            ? 'border-herb bg-herb-tint'
                            : 'border-mist hover:bg-paper'
                        }`}
                      >
                        <input
                          type="radio"
                          {...register('serviceType')}
                          value={type.value}
                          className="mt-1 h-4 w-4 text-herb focus-visible:ring-herb"
                        />
                        <div>
                          <span className="font-medium text-ink">{type.label}</span>
                          <p className="mt-1 text-sm text-ink-muted">{type.description}</p>
                        </div>
                      </label>
                    ))}
                  </div>
                </div>

                <div>
                  <label className="block text-sm font-medium text-ink-soft">
                    Additional Details (Optional)
                  </label>
                  <textarea
                    {...register('description')}
                    rows={4}
                    placeholder="Tell us more about your event, special requests, theme, etc."
                    className="input-base mt-1"
                  />
                </div>
              </div>

              <div className="mt-8 flex justify-between">
                <button type="button" onClick={() => setStep(1)} className="btn-outline">
                  Back
                </button>
                <Button
                  type="button"
                  variant="primary"
                  onClick={nextStep}
                  rightIcon={<ArrowRight aria-hidden="true" className="h-5 w-5" />}
                >
                  Continue
                </Button>
              </div>
            </div>
          )}

          {/* Step 3: Location */}
          {step === 3 && (
            <div className="rounded-xl bg-bone p-6 shadow-1 md:p-8">
              <h2 className="text-xl font-semibold text-ink">Event Location</h2>
              <p className="mt-1 text-ink-soft">Where should we deliver the catering?</p>

              <div className="mt-6 space-y-4">
                <div>
                  <label className="block text-sm font-medium text-ink-soft">
                    <MapPin className="mr-2 inline h-4 w-4" />
                    Street Address
                  </label>
                  <input
                    {...register('addressLine1')}
                    placeholder="123 Main Street"
                    className="input-base mt-1"
                  />
                  {errors.addressLine1 && (
                    <p className="mt-1 text-xs text-paprika">{errors.addressLine1.message}</p>
                  )}
                </div>

                <div>
                  <label className="block text-sm font-medium text-ink-soft">
                    Apartment, suite, etc. (Optional)
                  </label>
                  <input
                    {...register('addressLine2')}
                    placeholder="Suite 100"
                    className="input-base mt-1"
                  />
                </div>

                <div className="grid gap-4 md:grid-cols-3">
                  <div>
                    <label className="block text-sm font-medium text-ink-soft">City</label>
                    <input {...register('city')} className="input-base mt-1" />
                    {errors.city && (
                      <p className="mt-1 text-xs text-paprika">{errors.city.message}</p>
                    )}
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-ink-soft">State</label>
                    <input {...register('state')} className="input-base mt-1" />
                    {errors.state && (
                      <p className="mt-1 text-xs text-paprika">{errors.state.message}</p>
                    )}
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-ink-soft">Postal Code</label>
                    <input {...register('postalCode')} className="input-base mt-1" />
                    {errors.postalCode && (
                      <p className="mt-1 text-xs text-paprika">{errors.postalCode.message}</p>
                    )}
                  </div>
                </div>
              </div>

              {/* Info Box */}
              <div className="mt-6 flex items-start gap-3 rounded-lg bg-info/10 p-4">
                <Info className="h-5 w-5 flex-shrink-0 text-info" />
                <div className="text-sm text-info">
                  <p className="font-medium">What happens next?</p>
                  <p className="mt-1">
                    Once you submit your request, our home chefs will review it and send you
                    personalized quotes. You can then compare offers and choose the best one for
                    your event.
                  </p>
                </div>
              </div>

              <div className="mt-8 flex justify-between">
                <button type="button" onClick={() => setStep(2)} className="btn-outline">
                  Back
                </button>
                <Button
                  type="submit"
                  variant="primary"
                  isLoading={createRequestMutation.isPending}
                  disabled={createRequestMutation.isPending}
                  leftIcon={!createRequestMutation.isPending ? <ChefHat aria-hidden="true" className="h-5 w-5" /> : undefined}
                >
                  {createRequestMutation.isPending ? 'Submitting...' : 'Submit Request'}
                </Button>
              </div>
            </div>
          )}
        </form>
      </div>
    </div>
  );
}
