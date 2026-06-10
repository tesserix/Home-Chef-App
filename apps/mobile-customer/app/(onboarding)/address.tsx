import {
  View,
  Text,
  TextInput,
  Pressable,
  KeyboardAvoidingView,
  Platform,
  ScrollView,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useForm, Controller } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { router, useLocalSearchParams } from 'expo-router';
import { customerColors } from '@homechef/mobile-shared/theme';

const schema = z.object({
  addressLine1: z.string().min(5, 'Address must be at least 5 characters'),
  city: z.string().min(1, 'City is required'),
  state: z.string().min(1, 'State is required'),
  pincode: z.string().regex(/^\d{6}$/, 'Enter a valid 6-digit pincode'),
});

type AddressForm = z.infer<typeof schema>;

export default function AddressScreen() {
  const params = useLocalSearchParams<{
    firstName: string;
    lastName: string;
    phone: string;
  }>();

  const {
    control,
    handleSubmit,
    formState: { errors },
  } = useForm<AddressForm>({
    resolver: zodResolver(schema),
    defaultValues: { addressLine1: '', city: '', state: '', pincode: '' },
  });

  const onSubmit = (data: AddressForm) => {
    router.push({
      pathname: '/(onboarding)/preferences',
      params: {
        firstName: params.firstName,
        lastName: params.lastName,
        phone: params.phone,
        addressLine1: data.addressLine1,
        city: data.city,
        state: data.state,
        pincode: data.pincode,
      },
    });
  };

  return (
    <SafeAreaView className="flex-1 bg-canvas" edges={['top', 'left', 'right']}>
      <KeyboardAvoidingView
        className="flex-1"
        behavior={Platform.OS === 'ios' ? 'padding' : 'height'}
      >
        <ScrollView
          className="flex-1"
          contentContainerStyle={{ padding: 24, paddingTop: 40 }}
          keyboardShouldPersistTaps="handled"
        >
          {/* ── Step progress ── */}
          <Text className="text-[13px] text-charcoal-soft mb-2">
            Step 2 of 3
          </Text>
          <View className="h-1 bg-hairline rounded-full mb-8 overflow-hidden">
            <View className="h-1 bg-coral rounded-full" style={{ width: '66%' }} />
          </View>

          {/* ── Heading ── */}
          <Text className="text-[26px] font-bold text-charcoal tracking-tight font-display mb-2">
            Your delivery address
          </Text>
          <Text className="text-[15px] text-charcoal-soft mb-8">
            Where should we deliver your orders?
          </Text>

          {/* ── Address Line 1 ── */}
          <Text className="text-sm font-medium text-charcoal mb-1">
            Address
          </Text>
          <Controller
            control={control}
            name="addressLine1"
            render={({ field: { onChange, onBlur, value } }) => (
              <TextInput
                className="h-12 bg-surface-soft rounded-lg px-4 text-base text-charcoal mb-1"
                style={{
                  borderWidth: errors.addressLine1 ? 1.5 : 0,
                  borderColor: errors.addressLine1
                    ? customerColors.destructive.DEFAULT
                    : 'transparent',
                }}
                placeholder="House no., street, area"
                placeholderTextColor={customerColors.charcoal.soft}
                onBlur={onBlur}
                onChangeText={onChange}
                value={value}
                autoCapitalize="words"
                returnKeyType="next"
                accessibilityLabel="Address line 1"
              />
            )}
          />
          {errors.addressLine1 ? (
            <Text className="text-xs text-destructive mb-3">
              {errors.addressLine1.message}
            </Text>
          ) : (
            <View className="mb-4" />
          )}

          {/* ── City ── */}
          <Text className="text-sm font-medium text-charcoal mb-1">City</Text>
          <Controller
            control={control}
            name="city"
            render={({ field: { onChange, onBlur, value } }) => (
              <TextInput
                className="h-12 bg-surface-soft rounded-lg px-4 text-base text-charcoal mb-1"
                style={{
                  borderWidth: errors.city ? 1.5 : 0,
                  borderColor: errors.city
                    ? customerColors.destructive.DEFAULT
                    : 'transparent',
                }}
                placeholder="Enter your city"
                placeholderTextColor={customerColors.charcoal.soft}
                onBlur={onBlur}
                onChangeText={onChange}
                value={value}
                autoCapitalize="words"
                returnKeyType="next"
                accessibilityLabel="City"
              />
            )}
          />
          {errors.city ? (
            <Text className="text-xs text-destructive mb-3">
              {errors.city.message}
            </Text>
          ) : (
            <View className="mb-4" />
          )}

          {/* ── State ── */}
          <Text className="text-sm font-medium text-charcoal mb-1">State</Text>
          <Controller
            control={control}
            name="state"
            render={({ field: { onChange, onBlur, value } }) => (
              <TextInput
                className="h-12 bg-surface-soft rounded-lg px-4 text-base text-charcoal mb-1"
                style={{
                  borderWidth: errors.state ? 1.5 : 0,
                  borderColor: errors.state
                    ? customerColors.destructive.DEFAULT
                    : 'transparent',
                }}
                placeholder="Enter your state"
                placeholderTextColor={customerColors.charcoal.soft}
                onBlur={onBlur}
                onChangeText={onChange}
                value={value}
                autoCapitalize="words"
                returnKeyType="next"
                accessibilityLabel="State"
              />
            )}
          />
          {errors.state ? (
            <Text className="text-xs text-destructive mb-3">
              {errors.state.message}
            </Text>
          ) : (
            <View className="mb-4" />
          )}

          {/* ── Pincode ── */}
          <Text className="text-sm font-medium text-charcoal mb-1">
            Pincode
          </Text>
          <Controller
            control={control}
            name="pincode"
            render={({ field: { onChange, onBlur, value } }) => (
              <TextInput
                className="h-12 bg-surface-soft rounded-lg px-4 text-base text-charcoal mb-1"
                style={{
                  borderWidth: errors.pincode ? 1.5 : 0,
                  borderColor: errors.pincode
                    ? customerColors.destructive.DEFAULT
                    : 'transparent',
                }}
                placeholder="6-digit pincode"
                placeholderTextColor={customerColors.charcoal.soft}
                onBlur={onBlur}
                onChangeText={onChange}
                value={value}
                keyboardType="numeric"
                maxLength={6}
                returnKeyType="done"
                accessibilityLabel="Pincode"
              />
            )}
          />
          {errors.pincode ? (
            <Text className="text-xs text-destructive mb-3">
              {errors.pincode.message}
            </Text>
          ) : (
            <View className="mb-4" />
          )}

          {/* ── Primary CTA ── */}
          {/* iOS Pressable pattern: visual styles on inner View */}
          <Pressable
            onPress={() => void handleSubmit(onSubmit)()}
            accessibilityRole="button"
            accessibilityLabel="Continue to preferences"
          >
            {({ pressed }) => (
              <View
                className={`rounded-lg min-h-[52px] items-center justify-center mt-4 ${
                  pressed ? 'opacity-90' : ''
                } bg-coral`}
              >
                <Text className="text-canvas font-semibold text-base">
                  Continue
                </Text>
              </View>
            )}
          </Pressable>
        </ScrollView>
      </KeyboardAvoidingView>
    </SafeAreaView>
  );
}
