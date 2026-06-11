import { useState } from 'react';
import {
  View,
  Text,
  Pressable,
  ScrollView,
  Alert,
  ActivityIndicator,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { router, useLocalSearchParams } from 'expo-router';
import { useAuthStore } from '../../store/auth-store';
import { api } from '../../lib/api';
import { friendlyErrorMessage } from '../../lib/errors';
import { customerColors } from '@homechef/mobile-shared/theme';

const CUISINE_OPTIONS = [
  'North Indian',
  'South Indian',
  'Chinese',
  'Continental',
  'Italian',
  'Healthy',
  'Street Food',
  'Desserts',
] as const;

export default function PreferencesScreen() {
  const [selected, setSelected] = useState<string[]>([]);
  const [isSubmitting, setIsSubmitting] = useState(false);

  const params = useLocalSearchParams<{
    firstName: string;
    lastName: string;
    phone: string;
    label: string;
    addressLine1: string;
    addressLine2: string;
    city: string;
    state: string;
    pincode: string;
    latitude: string;
    longitude: string;
  }>();

  const setOnboardingComplete = useAuthStore(
    (s) => s.setOnboardingComplete
  );

  const toggleChip = (cuisine: string) => {
    setSelected((prev) =>
      prev.includes(cuisine)
        ? prev.filter((c) => c !== cuisine)
        : [...prev, cuisine]
    );
  };

  const onFinish = async () => {
    setIsSubmitting(true);
    try {
      // Backend (CompleteOnboarding) reads FLAT address fields, not a nested
      // `address` object — addressCity / addressState / addressPostalCode.
      // Sending a nested object silently dropped the address before.
      await api.post('/v1/customer/onboarding/complete', {
        firstName: params.firstName,
        lastName: params.lastName,
        phone: params.phone,
        addressLabel: params.label || 'Home',
        addressLine1: params.addressLine1,
        addressLine2: params.addressLine2 ?? '',
        addressCity: params.city,
        addressState: params.state,
        addressPostalCode: params.pincode,
        addressCountry: 'IN',
        // Geocoded from the address autocomplete pick; 0 when the user typed
        // the address manually (server then uses a flat fee + skips zones).
        addressLatitude: params.latitude ? Number(params.latitude) : 0,
        addressLongitude: params.longitude ? Number(params.longitude) : 0,
        cuisinePreferences: selected,
      });

      await setOnboardingComplete(true);
      router.replace('/(tabs)');
    } catch (error: unknown) {
      Alert.alert(
        'Setup failed',
        friendlyErrorMessage(
          error,
          "We couldn't finish setting up your account. Please try again.",
        ),
      );
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <SafeAreaView className="flex-1 bg-canvas" edges={['top', 'left', 'right']}>
      <ScrollView
        className="flex-1"
        contentContainerStyle={{ padding: 24, paddingTop: 40, paddingBottom: 48 }}
        keyboardShouldPersistTaps="handled"
      >
        {/* ── Step progress ── */}
        <Text className="text-[13px] text-charcoal-soft mb-2">
          Step 3 of 3
        </Text>
        <View className="h-1 bg-hairline rounded-full mb-8 overflow-hidden">
          <View className="h-1 bg-coral rounded-full" style={{ width: '100%' }} />
        </View>

        {/* ── Heading ── */}
        <Text className="text-[26px] font-bold text-charcoal tracking-tight font-display mb-2">
          What do you love to eat?
        </Text>
        <Text className="text-[15px] text-charcoal-soft mb-8">
          Select your favourite cuisines to get personalised recommendations.
        </Text>

        {/* ── Cuisine chips ── */}
        {/* iOS Pressable pattern: visual styles on inner View */}
        <View className="flex-row flex-wrap gap-2 mb-10">
          {CUISINE_OPTIONS.map((cuisine) => {
            const isActive = selected.includes(cuisine);
            return (
              <Pressable
                key={cuisine}
                onPress={() => toggleChip(cuisine)}
                accessibilityRole="checkbox"
                accessibilityLabel={cuisine}
                accessibilityState={{ checked: isActive }}
              >
                <View
                  className={`px-4 py-2 rounded-full border ${
                    isActive
                      ? 'bg-coral-tint border-coral'
                      : 'bg-canvas border-hairline'
                  }`}
                >
                  <Text
                    className={`text-sm font-medium ${
                      isActive ? 'text-coral font-semibold' : 'text-charcoal-soft'
                    }`}
                  >
                    {cuisine}
                  </Text>
                </View>
              </Pressable>
            );
          })}
        </View>

        {/* ── Primary CTA ── */}
        <Pressable
          onPress={() => void onFinish()}
          disabled={isSubmitting}
          accessibilityRole="button"
          accessibilityLabel="Finish setup"
        >
          {({ pressed }) => (
            <View
              className={`rounded-lg min-h-[52px] items-center justify-center bg-coral ${
                pressed || isSubmitting ? 'opacity-90' : ''
              }`}
            >
              {isSubmitting ? (
                <ActivityIndicator
                  size="small"
                  color={customerColors.canvas}
                />
              ) : (
                <Text className="text-canvas font-semibold text-base">
                  Finish Setup
                </Text>
              )}
            </View>
          )}
        </Pressable>
      </ScrollView>
    </SafeAreaView>
  );
}
