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
import { router } from 'expo-router';
import { customerColors } from '@homechef/mobile-shared/theme';

const schema = z.object({
  firstName: z.string().min(2, 'First name must be at least 2 characters'),
  lastName: z.string().min(2, 'Last name must be at least 2 characters'),
  phone: z
    .string()
    .regex(/^[6-9]\d{9}$/, 'Enter a valid 10-digit Indian mobile number'),
});

type UserInfoForm = z.infer<typeof schema>;

export default function UserInfoScreen() {
  const {
    control,
    handleSubmit,
    formState: { errors },
  } = useForm<UserInfoForm>({
    resolver: zodResolver(schema),
    defaultValues: { firstName: '', lastName: '', phone: '' },
  });

  const onSubmit = (data: UserInfoForm) => {
    router.push({
      pathname: '/(onboarding)/address',
      params: {
        firstName: data.firstName,
        lastName: data.lastName,
        phone: data.phone,
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
            Step 1 of 3
          </Text>
          <View className="h-1 bg-hairline rounded-full mb-8 overflow-hidden">
            <View className="h-1 bg-coral rounded-full" style={{ width: '33%' }} />
          </View>

          {/* ── Heading ── */}
          <Text className="text-[26px] font-bold text-charcoal tracking-tight font-display mb-2">
            Tell us about yourself
          </Text>
          <Text className="text-[15px] text-charcoal-soft mb-8">
            We need a few details to set up your account.
          </Text>

          {/* ── First Name ── */}
          <Text className="text-sm font-medium text-charcoal mb-1">
            First Name
          </Text>
          <Controller
            control={control}
            name="firstName"
            render={({ field: { onChange, onBlur, value } }) => (
              <TextInput
                className="h-12 bg-surface-soft rounded-lg px-4 text-base text-charcoal mb-1"
                style={{
                  borderWidth: errors.firstName ? 1.5 : 0,
                  borderColor: errors.firstName
                    ? customerColors.destructive.DEFAULT
                    : 'transparent',
                }}
                placeholder="Enter your first name"
                placeholderTextColor={customerColors.charcoal.soft}
                onBlur={onBlur}
                onChangeText={onChange}
                value={value}
                autoCapitalize="words"
                returnKeyType="next"
                accessibilityLabel="First name"
              />
            )}
          />
          {errors.firstName ? (
            <Text className="text-xs text-destructive mb-3">
              {errors.firstName.message}
            </Text>
          ) : (
            <View className="mb-4" />
          )}

          {/* ── Last Name ── */}
          <Text className="text-sm font-medium text-charcoal mb-1">
            Last Name
          </Text>
          <Controller
            control={control}
            name="lastName"
            render={({ field: { onChange, onBlur, value } }) => (
              <TextInput
                className="h-12 bg-surface-soft rounded-lg px-4 text-base text-charcoal mb-1"
                style={{
                  borderWidth: errors.lastName ? 1.5 : 0,
                  borderColor: errors.lastName
                    ? customerColors.destructive.DEFAULT
                    : 'transparent',
                }}
                placeholder="Enter your last name"
                placeholderTextColor={customerColors.charcoal.soft}
                onBlur={onBlur}
                onChangeText={onChange}
                value={value}
                autoCapitalize="words"
                returnKeyType="next"
                accessibilityLabel="Last name"
              />
            )}
          />
          {errors.lastName ? (
            <Text className="text-xs text-destructive mb-3">
              {errors.lastName.message}
            </Text>
          ) : (
            <View className="mb-4" />
          )}

          {/* ── Phone ── */}
          <Text className="text-sm font-medium text-charcoal mb-1">
            Phone Number
          </Text>
          <Controller
            control={control}
            name="phone"
            render={({ field: { onChange, onBlur, value } }) => (
              <TextInput
                className="h-12 bg-surface-soft rounded-lg px-4 text-base text-charcoal mb-1"
                style={{
                  borderWidth: errors.phone ? 1.5 : 0,
                  borderColor: errors.phone
                    ? customerColors.destructive.DEFAULT
                    : 'transparent',
                }}
                placeholder="10-digit mobile number"
                placeholderTextColor={customerColors.charcoal.soft}
                onBlur={onBlur}
                onChangeText={onChange}
                value={value}
                keyboardType="phone-pad"
                maxLength={10}
                returnKeyType="done"
                accessibilityLabel="Phone number"
              />
            )}
          />
          {errors.phone ? (
            <Text className="text-xs text-destructive mb-3">
              {errors.phone.message}
            </Text>
          ) : (
            <View className="mb-4" />
          )}

          {/* ── Primary CTA ── */}
          {/* iOS Pressable pattern: visual styles on inner View */}
          <Pressable
            onPress={() => void handleSubmit(onSubmit)()}
            accessibilityRole="button"
            accessibilityLabel="Continue to address"
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
