import React, { useEffect, useState } from 'react';
import { ActivityIndicator, Alert, Pressable, Text, TextInput, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { z } from 'zod';
import { useForm, Controller } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';

import { customerColors } from '@homechef/mobile-shared/theme';
import { KeyboardAwareScrollView } from '@homechef/mobile-shared/ui';
import { useProfile, useUpdateProfile } from '../../hooks/useProfile';
import { friendlyErrorMessage } from '../../lib/errors';
import { ScreenHeader } from '../../components/ScreenHeader';

// Edit profile — the name and phone form that used to sit at the TOP of the
// profile tab, pushing every navigational destination below the fold. The
// profile is a hub; editing is a task, and a task belongs on its own screen.

const profileSchema = z.object({
  firstName: z.string().min(1, 'First name is required').max(50),
  lastName: z.string().min(1, 'Last name is required').max(50),
  phone: z
    .string()
    .regex(/^\+?[0-9]{7,15}$/, 'Invalid phone number')
    .optional()
    .or(z.literal('')),
});

type ProfileFormValues = z.infer<typeof profileSchema>;

const CTA_RIPPLE = `${customerColors.canvas}33`;

function SectionLabel({ children }: { children: string }) {
  return (
    <Text className="text-xs font-semibold text-charcoal-soft px-4 pt-5 pb-2">{children}</Text>
  );
}

export default function EditProfileScreen() {
  const { data: profile, isLoading } = useProfile();
  const updateProfile = useUpdateProfile();

  // Visible 2px coral focus ring on the fields below (R9 / Input parity).
  const [focusedField, setFocusedField] = useState<'firstName' | 'lastName' | 'phone' | null>(null);

  const {
    control,
    handleSubmit,
    reset,
    formState: { errors, isDirty },
  } = useForm<ProfileFormValues>({
    resolver: zodResolver(profileSchema),
    defaultValues: { firstName: '', lastName: '', phone: '' },
  });

  useEffect(() => {
    if (profile) {
      reset({
        firstName: profile.firstName ?? '',
        lastName: profile.lastName ?? '',
        phone: profile.phone ?? '',
      });
    }
  }, [profile, reset]);

  function onSavePersonalInfo(values: ProfileFormValues) {
    updateProfile.mutate(
      {
        firstName: values.firstName,
        lastName: values.lastName,
        phone: values.phone ?? undefined,
      },
      {
        onSuccess: () => Alert.alert('Saved', 'Profile updated successfully.'),
        onError: (error) =>
          Alert.alert(
            'Error',
            friendlyErrorMessage(error, 'Could not update profile. Please try again.'),
          ),
      },
    );
  }

  if (isLoading) {
    return (
      <SafeAreaView className="flex-1 bg-canvas" edges={['top', 'left', 'right']}>
        <ScreenHeader title="Edit profile" />
        <View className="flex-1 items-center justify-center">
          <ActivityIndicator size="large" color={customerColors.coral.DEFAULT} />
        </View>
      </SafeAreaView>
    );
  }

  return (
    <SafeAreaView className="flex-1 bg-canvas" edges={['top', 'left', 'right']}>
      <ScreenHeader title="Edit profile" />
      <KeyboardAwareScrollView contentContainerStyle={{ paddingBottom: 40 }}>
        {/* ═══════════════════════════════════════════════════════════════════
            Section — Personal Info
        ═══════════════════════════════════════════════════════════════════ */}
        <SectionLabel>Personal Info</SectionLabel>

        <View className="bg-canvas mx-4 rounded-xl overflow-hidden border border-hairline">
          {/* First Name */}
          <View className="px-4 pt-3 pb-1">
            <Text className="text-xs font-semibold text-charcoal-soft mb-1">
              First Name
            </Text>
            <Controller
              control={control}
              name="firstName"
              render={({ field: { onChange, value, onBlur } }) => (
                <TextInput
                  className="text-base text-charcoal bg-transparent pb-2"
                  style={
                    errors.firstName
                      ? { borderBottomWidth: 1, borderBottomColor: customerColors.destructive.DEFAULT }
                      : focusedField === 'firstName'
                        ? { borderBottomWidth: 2, borderBottomColor: customerColors.coral.DEFAULT }
                        : { borderBottomWidth: 0 }
                  }
                  value={value}
                  onChangeText={onChange}
                  onFocus={() => setFocusedField('firstName')}
                  onBlur={() => {
                    setFocusedField(null);
                    onBlur();
                  }}
                  placeholder="First name"
                  placeholderTextColor={customerColors.charcoal.soft}
                  autoCapitalize="words"
                  accessibilityLabel="First name"
                />
              )}
            />
            {errors.firstName ? (
              <Text className="text-xs text-destructive mb-1">{errors.firstName.message}</Text>
            ) : null}
          </View>

          <View className="h-px bg-hairline mx-4" />

          {/* Last Name */}
          <View className="px-4 pt-3 pb-1">
            <Text className="text-xs font-semibold text-charcoal-soft mb-1">
              Last Name
            </Text>
            <Controller
              control={control}
              name="lastName"
              render={({ field: { onChange, value, onBlur } }) => (
                <TextInput
                  className="text-base text-charcoal bg-transparent pb-2"
                  style={
                    errors.lastName
                      ? { borderBottomWidth: 1, borderBottomColor: customerColors.destructive.DEFAULT }
                      : focusedField === 'lastName'
                        ? { borderBottomWidth: 2, borderBottomColor: customerColors.coral.DEFAULT }
                        : { borderBottomWidth: 0 }
                  }
                  value={value}
                  onChangeText={onChange}
                  onFocus={() => setFocusedField('lastName')}
                  onBlur={() => {
                    setFocusedField(null);
                    onBlur();
                  }}
                  placeholder="Last name"
                  placeholderTextColor={customerColors.charcoal.soft}
                  autoCapitalize="words"
                  accessibilityLabel="Last name"
                />
              )}
            />
            {errors.lastName ? (
              <Text className="text-xs text-destructive mb-1">{errors.lastName.message}</Text>
            ) : null}
          </View>

          <View className="h-px bg-hairline mx-4" />

          {/* Phone */}
          <View className="px-4 pt-3 pb-3">
            <Text className="text-xs font-semibold text-charcoal-soft mb-1">
              Phone
            </Text>
            <Controller
              control={control}
              name="phone"
              render={({ field: { onChange, value, onBlur } }) => (
                <TextInput
                  className="text-base text-charcoal bg-transparent pb-2"
                  style={
                    errors.phone
                      ? { borderBottomWidth: 1, borderBottomColor: customerColors.destructive.DEFAULT }
                      : focusedField === 'phone'
                        ? { borderBottomWidth: 2, borderBottomColor: customerColors.coral.DEFAULT }
                        : { borderBottomWidth: 0 }
                  }
                  value={value ?? ''}
                  onChangeText={onChange}
                  onFocus={() => setFocusedField('phone')}
                  onBlur={() => {
                    setFocusedField(null);
                    onBlur();
                  }}
                  placeholder="+91 9876543210"
                  placeholderTextColor={customerColors.charcoal.soft}
                  keyboardType="phone-pad"
                  accessibilityLabel="Phone number"
                />
              )}
            />
            {errors.phone ? (
              <Text className="text-xs text-destructive mt-1">{errors.phone.message}</Text>
            ) : null}
          </View>
        </View>

        {/* Save Changes CTA — only shown when form is dirty */}
        {isDirty ? (
          <Pressable
            onPress={() => void handleSubmit(onSavePersonalInfo)()}
            disabled={updateProfile.isPending}
            accessibilityRole="button"
            accessibilityLabel="Save changes"
            android_ripple={{ color: CTA_RIPPLE, borderless: false }}
          >
            {({ pressed }) => (
              <View
                className={`mx-4 mt-3 rounded-lg min-h-[52px] items-center justify-center ${
                  pressed ? 'opacity-90' : ''
                } bg-coral`}
              >
                {updateProfile.isPending ? (
                  <ActivityIndicator size="small" color={customerColors.canvas} />
                ) : (
                  <Text className="text-canvas font-semibold text-base">
                    Save Changes
                  </Text>
                )}
              </View>
            )}
          </Pressable>
        ) : null}

      </KeyboardAwareScrollView>
    </SafeAreaView>
  );
}
