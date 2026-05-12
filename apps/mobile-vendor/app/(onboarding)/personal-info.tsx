import { View, Text, TextInput, TouchableOpacity, ScrollView, Alert } from 'react-native';
import { useForm, Controller } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { router } from 'expo-router';
import { useAuthStore } from '../../store/auth-store';
import { useVendorOnboardingStore } from '../../store/onboarding-store';

const schema = z.object({
  fullName: z.string().min(2, 'Full name must be at least 2 characters'),
  phone: z.string().regex(/^[6-9]\d{9}$/, 'Enter a valid 10-digit Indian mobile number'),
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

  return (
    <ScrollView className="flex-1 bg-bone">
      <View className="px-6 pt-4 pb-2">
        <View className="h-1.5 rounded-full bg-mist mb-6">
          <View className="h-1.5 rounded-full bg-herb" style={{ width: `${(1 / 6) * 100}%` }} />
        </View>

        <Text className="font-display text-2xl font-semibold text-ink mb-1">Personal Information</Text>
        <Text className="text-sm text-ink-muted mb-6">Tell us about yourself</Text>

        <Text className="text-sm font-medium text-ink-soft mb-1">Full Name *</Text>
        <Controller
          control={control}
          name="fullName"
          render={({ field: { onChange, onBlur, value } }) => (
            <TextInput
              className="border border-mist-strong rounded-lg px-4 py-3 text-base text-ink mb-1"
              placeholder="Enter your full name"
              onBlur={onBlur}
              onChangeText={onChange}
              value={value}
              autoCapitalize="words"
            />
          )}
        />
        {errors.fullName && (
          <Text className="text-paprika text-xs mb-3">{errors.fullName.message}</Text>
        )}
        {!errors.fullName && <View className="mb-3" />}

        <Text className="text-sm font-medium text-ink-soft mb-1">Phone Number *</Text>
        <Controller
          control={control}
          name="phone"
          render={({ field: { onChange, onBlur, value } }) => (
            <TextInput
              className="border border-mist-strong rounded-lg px-4 py-3 text-base text-ink mb-1"
              placeholder="10-digit mobile number"
              onBlur={onBlur}
              onChangeText={onChange}
              value={value}
              keyboardType="phone-pad"
              maxLength={10}
            />
          )}
        />
        {errors.phone && (
          <Text className="text-paprika text-xs mb-3">{errors.phone.message}</Text>
        )}
        {!errors.phone && <View className="mb-3" />}

        <Text className="text-sm font-medium text-ink-soft mb-1">Email Address</Text>
        <Controller
          control={control}
          name="email"
          render={({ field: { value } }) => (
            <TextInput
              className="border border-mist rounded-lg px-4 py-3 text-base text-ink-muted bg-paper mb-1"
              value={value}
              editable={false}
            />
          )}
        />
        <Text className="text-xs text-ink-muted mb-6">Email is pre-filled from your account</Text>

        <TouchableOpacity
          className="bg-herb rounded-xl py-4 items-center"
          onPress={handleSubmit(onSubmit, (errs) => {
            const firstError = Object.values(errs)[0];
            if (firstError?.message) Alert.alert('Validation Error', firstError.message);
          })}
        >
          <Text className="text-paper font-semibold text-base">Next</Text>
        </TouchableOpacity>
      </View>
    </ScrollView>
  );
}
