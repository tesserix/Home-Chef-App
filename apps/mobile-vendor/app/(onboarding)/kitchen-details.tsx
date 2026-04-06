import { View, Text, TextInput, TouchableOpacity, ScrollView, Alert } from 'react-native';
import { useForm, Controller } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { router } from 'expo-router';
import { useVendorOnboardingStore } from '../../store/onboarding-store';

const CUISINE_OPTIONS = [
  'North Indian',
  'South Indian',
  'Chinese',
  'Continental',
  'Bakery',
  'Snacks',
  'Beverages',
  'Other',
] as const;

const schema = z.object({
  businessName: z.string().min(3, 'Business name must be at least 3 characters'),
  cuisines: z.array(z.string()).min(1, 'Select at least one cuisine type'),
  description: z
    .string()
    .min(50, 'Description must be at least 50 characters')
    .max(500, 'Description must be at most 500 characters'),
});

type FormValues = z.infer<typeof schema>;

export default function KitchenDetailsScreen() {
  const { updateKitchenDetails, setStep } = useVendorOnboardingStore();

  const {
    control,
    handleSubmit,
    watch,
    setValue,
    formState: { errors },
  } = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: {
      businessName: '',
      cuisines: [],
      description: '',
    },
  });

  const selectedCuisines = watch('cuisines');

  function toggleCuisine(cuisine: string): void {
    const current = selectedCuisines ?? [];
    if (current.includes(cuisine)) {
      setValue('cuisines', current.filter((c) => c !== cuisine));
    } else {
      setValue('cuisines', [...current, cuisine]);
    }
  }

  function onSubmit(data: FormValues): void {
    updateKitchenDetails(data);
    setStep(3);
    router.push('/(onboarding)/operations');
  }

  return (
    <ScrollView className="flex-1 bg-white">
      <View className="px-6 pt-4 pb-8">
        <View className="h-1.5 rounded-full bg-gray-200 mb-6">
          <View className="h-1.5 rounded-full bg-orange-500" style={{ width: `${(2 / 6) * 100}%` }} />
        </View>

        <Text className="text-2xl font-bold text-gray-900 mb-1">Kitchen Details</Text>
        <Text className="text-sm text-gray-500 mb-6">Tell us about your kitchen</Text>

        <Text className="text-sm font-medium text-gray-700 mb-1">Business Name *</Text>
        <Controller
          control={control}
          name="businessName"
          render={({ field: { onChange, onBlur, value } }) => (
            <TextInput
              className="border border-gray-300 rounded-lg px-4 py-3 text-base text-gray-900 mb-1"
              placeholder="Your kitchen / business name"
              onBlur={onBlur}
              onChangeText={onChange}
              value={value}
              autoCapitalize="words"
            />
          )}
        />
        {errors.businessName && (
          <Text className="text-red-500 text-xs mb-3">{errors.businessName.message}</Text>
        )}
        {!errors.businessName && <View className="mb-3" />}

        <Text className="text-sm font-medium text-gray-700 mb-2">Cuisine Types *</Text>
        <View className="flex-row flex-wrap gap-2 mb-1">
          {CUISINE_OPTIONS.map((cuisine) => {
            const selected = selectedCuisines?.includes(cuisine) ?? false;
            return (
              <TouchableOpacity
                key={cuisine}
                onPress={() => toggleCuisine(cuisine)}
                className={`px-3 py-1.5 rounded-full border ${
                  selected
                    ? 'bg-orange-500 border-orange-500'
                    : 'bg-white border-gray-300'
                }`}
              >
                <Text className={`text-sm ${selected ? 'text-white font-medium' : 'text-gray-700'}`}>
                  {cuisine}
                </Text>
              </TouchableOpacity>
            );
          })}
        </View>
        {errors.cuisines && (
          <Text className="text-red-500 text-xs mb-3">{errors.cuisines.message}</Text>
        )}
        {!errors.cuisines && <View className="mb-3" />}

        <Text className="text-sm font-medium text-gray-700 mb-1">Description *</Text>
        <Controller
          control={control}
          name="description"
          render={({ field: { onChange, onBlur, value } }) => (
            <TextInput
              className="border border-gray-300 rounded-lg px-4 py-3 text-base text-gray-900 mb-1"
              placeholder="Describe your kitchen, specialties, and cooking style (min 50 characters)"
              onBlur={onBlur}
              onChangeText={onChange}
              value={value}
              multiline
              numberOfLines={4}
              textAlignVertical="top"
              style={{ minHeight: 100 }}
            />
          )}
        />
        <Controller
          control={control}
          name="description"
          render={({ field: { value } }) => (
            <Text className="text-xs text-gray-400 text-right mb-1">
              {(value ?? '').length} / 500
            </Text>
          )}
        />
        {errors.description && (
          <Text className="text-red-500 text-xs mb-3">{errors.description.message}</Text>
        )}
        {!errors.description && <View className="mb-3" />}

        <TouchableOpacity
          className="bg-orange-500 rounded-xl py-4 items-center mt-2"
          onPress={handleSubmit(onSubmit, (errs) => {
            const firstError = Object.values(errs)[0];
            if (firstError?.message) Alert.alert('Validation Error', firstError.message);
          })}
        >
          <Text className="text-white font-semibold text-base">Next</Text>
        </TouchableOpacity>
      </View>
    </ScrollView>
  );
}
