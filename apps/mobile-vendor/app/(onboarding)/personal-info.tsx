import { Alert, StyleSheet, Text, View } from 'react-native';
import { useForm, Controller } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { router } from 'expo-router';
import { Input, OnboardingScaffold } from '@homechef/mobile-shared/ui';
import { theme } from '@homechef/mobile-shared/theme';
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

  function onInvalid(errs: typeof errors): void {
    const firstError = Object.values(errs)[0];
    if (firstError?.message) Alert.alert('Check your details', firstError.message);
  }

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
          <View style={styles.lockedFieldWrap}>
            <Input
              label="Email"
              value={value}
              editable={false}
            />
            <Text style={styles.lockedHint}>Signed in with Google — cannot be changed.</Text>
          </View>
        )}
      />

      <View style={styles.bottomSpacer} />
    </OnboardingScaffold>
  );
}

const styles = StyleSheet.create({
  // Locked email field — wraps the Input with a small caption below it.
  lockedFieldWrap: {
    gap: theme.spacing[1],
  },

  lockedHint: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.caption.size,
    color: theme.colors.ink.muted,
    paddingHorizontal: theme.spacing[1],
  },

  bottomSpacer: {
    height: theme.spacing[2],
  },
});
