import { useState } from 'react';
import {
  View,
  Text,
  TextInput,
  TouchableOpacity,
  ScrollView,
  Alert,
  ActivityIndicator,
} from 'react-native';
import { useForm, Controller } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { router } from 'expo-router';
import { SafeAreaView } from 'react-native-safe-area-context';
import { api } from '../../lib/api';
import { useDriverOnboardingStore } from '../../store/onboarding-store';

const VEHICLE_TYPES = [
  { label: 'Bike', value: 'bike' },
  { label: 'Scooter', value: 'scooter' },
  { label: 'Car', value: 'car' },
  { label: 'Van', value: 'van' },
] as const;

const indianMobileRegex = /^[6-9]\d{9}$/;

const personalSchema = z.object({
  city: z.string().min(2, 'City must be at least 2 characters'),
  vehicleType: z.enum(['bike', 'scooter', 'car', 'van']),
  emergencyContact: z.string().min(2, 'Emergency contact name is required'),
  emergencyPhone: z
    .string()
    .regex(indianMobileRegex, 'Enter a valid 10-digit Indian mobile number'),
  dateOfBirth: z.string().optional(),
});

type PersonalFormData = z.infer<typeof personalSchema>;

export default function PersonalScreen() {
  const [isSubmitting, setIsSubmitting] = useState(false);
  const { personalInfo, updatePersonalInfo, setStep } = useDriverOnboardingStore();

  const {
    control,
    handleSubmit,
    formState: { errors },
  } = useForm<PersonalFormData>({
    resolver: zodResolver(personalSchema),
    defaultValues: {
      city: personalInfo.city,
      vehicleType: personalInfo.vehicleType,
      emergencyContact: personalInfo.emergencyContact,
      emergencyPhone: personalInfo.emergencyPhone,
      dateOfBirth: personalInfo.dateOfBirth,
    },
  });

  const onSubmit = async (data: PersonalFormData) => {
    setIsSubmitting(true);
    try {
      await api.post('/driver/onboarding/personal', {
        city: data.city,
        emergencyContact: data.emergencyContact,
        emergencyPhone: data.emergencyPhone,
        vehicleType: data.vehicleType,
        dateOfBirth: data.dateOfBirth || undefined,
      });
      updatePersonalInfo(data);
      setStep(2);
      router.push('/(onboarding)/vehicle');
    } catch (err: unknown) {
      const message =
        err instanceof Error ? err.message : 'Failed to save personal info. Please try again.';
      Alert.alert('Error', message);
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <SafeAreaView className="flex-1 bg-bone" edges={['bottom']}>
      <ScrollView className="flex-1 px-6" keyboardShouldPersistTaps="handled">
        {/* Progress bar */}
        <View className="mt-4 mb-6 h-1 bg-mist rounded-full">
          <View className="h-1 bg-herb rounded-full" style={{ width: '16.67%' }} />
        </View>

        <Text className="font-display text-2xl font-semibold text-ink mb-2">Personal Information</Text>
        <Text className="text-ink-muted mb-6">Tell us about yourself to get started</Text>

        {/* City */}
        <View className="mb-4">
          <Text className="text-sm font-medium text-ink-soft mb-1">
            City <Text className="text-paprika">*</Text>
          </Text>
          <Controller
            control={control}
            name="city"
            render={({ field: { onChange, onBlur, value } }) => (
              <TextInput
                className={`border rounded-lg px-4 py-3 text-ink ${errors.city ? 'border-paprika' : 'border-mist-strong'}`}
                placeholder="Enter your city"
                onBlur={onBlur}
                onChangeText={onChange}
                value={value}
              />
            )}
          />
          {errors.city && (
            <Text className="text-paprika text-sm mt-1">{errors.city.message}</Text>
          )}
        </View>

        {/* Vehicle Type */}
        <View className="mb-4">
          <Text className="text-sm font-medium text-ink-soft mb-2">
            Vehicle Type <Text className="text-paprika">*</Text>
          </Text>
          <Controller
            control={control}
            name="vehicleType"
            render={({ field: { onChange, value } }) => (
              <View className="flex-row gap-2 flex-wrap">
                {VEHICLE_TYPES.map((type) => (
                  <TouchableOpacity
                    key={type.value}
                    onPress={() => onChange(type.value)}
                    className={`px-4 py-2 rounded-full border ${
                      value === type.value
                        ? 'bg-herb border-herb'
                        : 'bg-bone border-mist-strong'
                    }`}
                  >
                    <Text
                      className={`font-medium ${
                        value === type.value ? 'text-paper' : 'text-ink-soft'
                      }`}
                    >
                      {type.label}
                    </Text>
                  </TouchableOpacity>
                ))}
              </View>
            )}
          />
          {errors.vehicleType && (
            <Text className="text-paprika text-sm mt-1">{errors.vehicleType.message}</Text>
          )}
        </View>

        {/* Emergency Contact Name */}
        <View className="mb-4">
          <Text className="text-sm font-medium text-ink-soft mb-1">
            Emergency Contact Name <Text className="text-paprika">*</Text>
          </Text>
          <Controller
            control={control}
            name="emergencyContact"
            render={({ field: { onChange, onBlur, value } }) => (
              <TextInput
                className={`border rounded-lg px-4 py-3 text-ink ${errors.emergencyContact ? 'border-paprika' : 'border-mist-strong'}`}
                placeholder="Full name of emergency contact"
                onBlur={onBlur}
                onChangeText={onChange}
                value={value}
              />
            )}
          />
          {errors.emergencyContact && (
            <Text className="text-paprika text-sm mt-1">{errors.emergencyContact.message}</Text>
          )}
        </View>

        {/* Emergency Phone */}
        <View className="mb-4">
          <Text className="text-sm font-medium text-ink-soft mb-1">
            Emergency Contact Phone <Text className="text-paprika">*</Text>
          </Text>
          <Controller
            control={control}
            name="emergencyPhone"
            render={({ field: { onChange, onBlur, value } }) => (
              <TextInput
                className={`border rounded-lg px-4 py-3 text-ink ${errors.emergencyPhone ? 'border-paprika' : 'border-mist-strong'}`}
                placeholder="10-digit mobile number"
                keyboardType="phone-pad"
                maxLength={10}
                onBlur={onBlur}
                onChangeText={onChange}
                value={value}
              />
            )}
          />
          {errors.emergencyPhone && (
            <Text className="text-paprika text-sm mt-1">{errors.emergencyPhone.message}</Text>
          )}
        </View>

        {/* Date of Birth (optional) */}
        <View className="mb-8">
          <Text className="text-sm font-medium text-ink-soft mb-1">
            Date of Birth <Text className="text-ink-muted">(optional)</Text>
          </Text>
          <Controller
            control={control}
            name="dateOfBirth"
            render={({ field: { onChange, onBlur, value } }) => (
              <TextInput
                className="border border-mist-strong rounded-lg px-4 py-3 text-ink"
                placeholder="MM/DD/YYYY"
                onBlur={onBlur}
                onChangeText={onChange}
                value={value}
              />
            )}
          />
        </View>
      </ScrollView>

      {/* Next Button */}
      <View className="px-6 py-4 border-t border-mist">
        <TouchableOpacity
          onPress={handleSubmit(onSubmit)}
          disabled={isSubmitting}
          className={`w-full py-4 rounded-xl items-center ${isSubmitting ? 'bg-herb-soft' : 'bg-herb'}`}
        >
          {isSubmitting ? (
            <ActivityIndicator color="white" />
          ) : (
            <Text className="text-paper font-semibold text-base">Next</Text>
          )}
        </TouchableOpacity>
      </View>
    </SafeAreaView>
  );
}
