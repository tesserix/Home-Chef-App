import { Alert, View } from 'react-native';
import { useForm, Controller } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { router } from 'expo-router';
import { Input, OnboardingScaffold } from '@homechef/mobile-shared/ui';
import { useAuthStore } from '../../store/auth-store';
import { useVendorOnboardingStore } from '../../store/onboarding-store';

const schema = z.object({
  fullName: z.string().min(2, 'Full name must be at least 2 characters'),
  phone: z
    .string()
    .regex(/^[6-9]\d{9}$/, 'Enter a valid 10-digit Indian mobile number'),
  email: z.string().email('Invalid email address'),
});

type FormValues = z.infer<typeof schema>;

export default function PersonalInfoScreen() {
  const { user } = useAuthStore();
  const { updatePersonalInfo, setStep } = useVendorOnboardingStore();

  const {
    control,
    handleSubmit,
    formState: { errors },
  } = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: {
      fullName: '',
      phone: '',
      email: user?.email ?? '',
    },
  });

  function onSubmit(data: FormValues): void {
    updatePersonalInfo(data);
    setStep(2);
    router.push('/(onboarding)/kitchen-details');
  }

  const onInvalid = (errs: typeof errors) => {
    const firstError = Object.values(errs)[0];
    if (firstError?.message) Alert.alert('Check your details', firstError.message);
  };

  return (
    <OnboardingScaffold
      step={1}
      total={6}
      title="Tell us about you"
      subtitle="The chef behind the kitchen — your customers will see this."
      primaryLabel="Continue"
      onPrimary={handleSubmit(onSubmit, onInvalid)}
    >
      <Controller
        control={control}
        name="fullName"
        render={({ field: { onChange, onBlur, value } }) => (
          <Input
            label="Full name"
            placeholder="Enter your full name"
            onBlur={onBlur}
            onChangeText={onChange}
            value={value}
            autoCapitalize="words"
            error={errors.fullName?.message}
          />
        )}
      />

      <Controller
        control={control}
        name="phone"
        render={({ field: { onChange, onBlur, value } }) => (
          <Input
            label="Phone number"
            placeholder="10-digit mobile number"
            onBlur={onBlur}
            onChangeText={onChange}
            value={value}
            keyboardType="phone-pad"
            maxLength={10}
            helper="We'll use this for delivery coordination only."
            error={errors.phone?.message}
          />
        )}
      />

      <Controller
        control={control}
        name="email"
        render={({ field: { value } }) => (
          <Input
            label="Email"
            value={value}
            editable={false}
            helper="Pre-filled from your account."
          />
        )}
      />

      <View />
    </OnboardingScaffold>
  );
}
