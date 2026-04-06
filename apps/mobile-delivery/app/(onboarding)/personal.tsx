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
    <SafeAreaView className="flex-1 bg-white" edges={['bottom']}>
      <ScrollView className="flex-1 px-6" keyboardShouldPersistTaps="handled">
        {/* Progress bar */}
        <View className="mt-4 mb-6 h-1 bg-gray-200 rounded-full">
          <View className="h-1 bg-orange-500 rounded-full" style={{ width: '16.67%' }} />
        </View>

        <Text className="text-2xl font-bold text-gray-900 mb-2">Personal Information</Text>
        <Text className="text-gray-500 mb-6">Tell us about yourself to get started</Text>

        {/* City */}
        <View className="mb-4">
          <Text className="text-sm font-medium text-gray-700 mb-1">
            City <Text className="text-red-500">*</Text>
          </Text>
          <Controller
            control={control}
            name="city"
            render={({ field: { onChange, onBlur, value } }) => (
              <TextInput
                className={`border rounded-lg px-4 py-3 text-gray-900 ${errors.city ? 'border-red-500' : 'border-gray-300'}`}
                placeholder="Enter your city"
                onBlur={onBlur}
                onChangeText={onChange}
                value={value}
              />
            )}
          />
          {errors.city && (
            <Text className="text-red-500 text-sm mt-1">{errors.city.message}</Text>
          )}
        </View>

        {/* Vehicle Type */}
        <View className="mb-4">
          <Text className="text-sm font-medium text-gray-700 mb-2">
            Vehicle Type <Text className="text-red-500">*</Text>
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
                        ? 'bg-orange-500 border-orange-500'
                        : 'bg-white border-gray-300'
                    }`}
                  >
                    <Text
                      className={`font-medium ${
                        value === type.value ? 'text-white' : 'text-gray-700'
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
            <Text className="text-red-500 text-sm mt-1">{errors.vehicleType.message}</Text>
          )}
        </View>

        {/* Emergency Contact Name */}
        <View className="mb-4">
          <Text className="text-sm font-medium text-gray-700 mb-1">
            Emergency Contact Name <Text className="text-red-500">*</Text>
          </Text>
          <Controller
            control={control}
            name="emergencyContact"
            render={({ field: { onChange, onBlur, value } }) => (
              <TextInput
                className={`border rounded-lg px-4 py-3 text-gray-900 ${errors.emergencyContact ? 'border-red-500' : 'border-gray-300'}`}
                placeholder="Full name of emergency contact"
                onBlur={onBlur}
                onChangeText={onChange}
                value={value}
              />
            )}
          />
          {errors.emergencyContact && (
            <Text className="text-red-500 text-sm mt-1">{errors.emergencyContact.message}</Text>
          )}
        </View>

        {/* Emergency Phone */}
        <View className="mb-4">
          <Text className="text-sm font-medium text-gray-700 mb-1">
            Emergency Contact Phone <Text className="text-red-500">*</Text>
          </Text>
          <Controller
            control={control}
            name="emergencyPhone"
            render={({ field: { onChange, onBlur, value } }) => (
              <TextInput
                className={`border rounded-lg px-4 py-3 text-gray-900 ${errors.emergencyPhone ? 'border-red-500' : 'border-gray-300'}`}
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
            <Text className="text-red-500 text-sm mt-1">{errors.emergencyPhone.message}</Text>
          )}
        </View>

        {/* Date of Birth (optional) */}
        <View className="mb-8">
          <Text className="text-sm font-medium text-gray-700 mb-1">
            Date of Birth <Text className="text-gray-400">(optional)</Text>
          </Text>
          <Controller
            control={control}
            name="dateOfBirth"
            render={({ field: { onChange, onBlur, value } }) => (
              <TextInput
                className="border border-gray-300 rounded-lg px-4 py-3 text-gray-900"
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
      <View className="px-6 py-4 border-t border-gray-100">
        <TouchableOpacity
          onPress={handleSubmit(onSubmit)}
          disabled={isSubmitting}
          className={`w-full py-4 rounded-xl items-center ${isSubmitting ? 'bg-orange-300' : 'bg-orange-500'}`}
        >
          {isSubmitting ? (
            <ActivityIndicator color="white" />
          ) : (
            <Text className="text-white font-semibold text-base">Next</Text>
          )}
        </TouchableOpacity>
      </View>
    </SafeAreaView>
  );
}
