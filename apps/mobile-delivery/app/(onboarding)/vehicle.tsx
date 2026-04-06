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

const currentYear = new Date().getFullYear();

const vehicleSchema = z.object({
  vehicleMake: z.string().min(1, 'Vehicle make is required'),
  vehicleModel: z.string().min(1, 'Vehicle model is required'),
  vehicleYear: z
    .string()
    .regex(/^\d{4}$/, 'Enter a valid 4-digit year')
    .refine(
      (val) => {
        const year = parseInt(val, 10);
        return year >= 2000 && year <= currentYear;
      },
      { message: `Year must be between 2000 and ${currentYear}` }
    ),
  vehicleColor: z.string().min(1, 'Vehicle color is required'),
  vehicleNumber: z
    .string()
    .regex(/^[A-Z]{2}[0-9]{2}[A-Z]{1,2}[0-9]{4}$/, 'Enter a valid vehicle number (e.g. MH12AB1234)'),
  licenseNumber: z.string().min(8, 'License number must be at least 8 characters'),
});

type VehicleFormData = z.infer<typeof vehicleSchema>;

export default function VehicleScreen() {
  const [isSubmitting, setIsSubmitting] = useState(false);
  const { personalInfo, vehicleDetails, updateVehicleDetails, setStep } =
    useDriverOnboardingStore();

  const {
    control,
    handleSubmit,
    formState: { errors },
  } = useForm<VehicleFormData>({
    resolver: zodResolver(vehicleSchema),
    defaultValues: {
      vehicleMake: vehicleDetails.vehicleMake,
      vehicleModel: vehicleDetails.vehicleModel,
      vehicleYear: vehicleDetails.vehicleYear,
      vehicleColor: vehicleDetails.vehicleColor,
      vehicleNumber: vehicleDetails.vehicleNumber,
      licenseNumber: vehicleDetails.licenseNumber,
    },
  });

  const onSubmit = async (data: VehicleFormData) => {
    setIsSubmitting(true);
    try {
      await api.post('/driver/onboarding/vehicle', {
        vehicleType: personalInfo.vehicleType,
        vehicleMake: data.vehicleMake,
        vehicleModel: data.vehicleModel,
        vehicleYear: parseInt(data.vehicleYear, 10),
        vehicleColor: data.vehicleColor,
        vehicleNumber: data.vehicleNumber.toUpperCase(),
        licenseNumber: data.licenseNumber,
      });
      updateVehicleDetails({
        ...data,
        vehicleType: personalInfo.vehicleType,
      });
      setStep(3);
      router.push('/(onboarding)/documents');
    } catch (err: unknown) {
      const message =
        err instanceof Error ? err.message : 'Failed to save vehicle details. Please try again.';
      Alert.alert('Error', message);
    } finally {
      setIsSubmitting(false);
    }
  };

  const fields: Array<{
    name: keyof VehicleFormData;
    label: string;
    placeholder: string;
    keyboardType?: 'default' | 'numeric';
    autoCapitalize?: 'none' | 'characters';
  }> = [
    { name: 'vehicleMake', label: 'Vehicle Make', placeholder: 'e.g. Honda' },
    { name: 'vehicleModel', label: 'Vehicle Model', placeholder: 'e.g. Activa 6G' },
    { name: 'vehicleYear', label: 'Vehicle Year', placeholder: 'e.g. 2022', keyboardType: 'numeric' },
    { name: 'vehicleColor', label: 'Vehicle Color', placeholder: 'e.g. Black' },
    {
      name: 'vehicleNumber',
      label: 'Vehicle Registration Number',
      placeholder: 'e.g. MH12AB1234',
      autoCapitalize: 'characters',
    },
    { name: 'licenseNumber', label: 'Driving License Number', placeholder: 'Min 8 characters' },
  ];

  return (
    <SafeAreaView className="flex-1 bg-white" edges={['bottom']}>
      <ScrollView className="flex-1 px-6" keyboardShouldPersistTaps="handled">
        {/* Progress bar */}
        <View className="mt-4 mb-6 h-1 bg-gray-200 rounded-full">
          <View className="h-1 bg-orange-500 rounded-full" style={{ width: '33.33%' }} />
        </View>

        <Text className="text-2xl font-bold text-gray-900 mb-2">Vehicle Details</Text>
        <Text className="text-gray-500 mb-6">
          Vehicle type: <Text className="font-semibold capitalize">{personalInfo.vehicleType}</Text>
        </Text>

        {fields.map((field) => (
          <View key={field.name} className="mb-4">
            <Text className="text-sm font-medium text-gray-700 mb-1">
              {field.label} <Text className="text-red-500">*</Text>
            </Text>
            <Controller
              control={control}
              name={field.name}
              render={({ field: { onChange, onBlur, value } }) => (
                <TextInput
                  className={`border rounded-lg px-4 py-3 text-gray-900 ${errors[field.name] ? 'border-red-500' : 'border-gray-300'}`}
                  placeholder={field.placeholder}
                  keyboardType={field.keyboardType ?? 'default'}
                  autoCapitalize={field.autoCapitalize ?? 'sentences'}
                  onBlur={onBlur}
                  onChangeText={onChange}
                  value={value}
                />
              )}
            />
            {errors[field.name] && (
              <Text className="text-red-500 text-sm mt-1">{errors[field.name]?.message}</Text>
            )}
          </View>
        ))}

        <View className="mb-8" />
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
