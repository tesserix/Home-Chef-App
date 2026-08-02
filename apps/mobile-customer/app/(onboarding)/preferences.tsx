import { useState } from 'react';
import { View, Text, Platform, Pressable, ScrollView, ActivityIndicator } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { router } from 'expo-router';
import { useAuthStore } from '../../store/auth-store';
import { useCustomerOnboardingStore } from '../../store/onboarding-store';
import { useCancelOnboarding } from '../../lib/use-cancel-onboarding';
import { api } from '../../lib/api';
import { friendlyErrorMessage } from '../../lib/errors';
import { customerColors } from '@homechef/mobile-shared/theme';
import { useAlert } from '@homechef/mobile-shared/ui';
import {
  DIET_OPTIONS,
  ALLERGEN_OPTIONS,
  type DietaryOption,
} from '@homechef/mobile-shared/dietary';

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

// Android ripple tints — translucent tokens derived from existing colours,
// never a new literal colour (matches the ChefCard `withAlpha` convention).
const CHIP_RIPPLE = `${customerColors.charcoal.DEFAULT}14`;
const CTA_RIPPLE = `${customerColors.canvas}33`;

/** Immutable add/remove — never mutates the array held in the draft store. */
const toggleValue = (list: string[], value: string): string[] =>
  list.includes(value) ? list.filter((v) => v !== value) : [...list, value];

function SectionHeading({ title, helper }: { title: string; helper: string }) {
  return (
    <>
      <Text className="text-base font-semibold text-charcoal mb-1">{title}</Text>
      <Text className="text-[13px] text-charcoal-soft mb-3">{helper}</Text>
    </>
  );
}

/**
 * Chip group for the shared {value,label} dietary taxonomy. Same neutral coral
 * chips as the cuisine row above — allergens are deliberately not tinted
 * destructive here: this is an optional question during signup, not a warning.
 * iOS Pressable pattern: layout/visual classes live on the inner View.
 */
function OptionChips({
  options,
  selectedValues,
  onToggle,
}: {
  options: DietaryOption[];
  selectedValues: string[];
  onToggle: (value: string) => void;
}) {
  return (
    <View className="flex-row flex-wrap gap-2 mb-8">
      {options.map((opt) => {
        const isActive = selectedValues.includes(opt.value);
        return (
          <Pressable
            key={opt.value}
            onPress={() => onToggle(opt.value)}
            accessibilityRole="checkbox"
            accessibilityLabel={opt.label}
            accessibilityState={{ checked: isActive }}
            android_ripple={{ color: CHIP_RIPPLE, borderless: false }}
          >
            {({ pressed }) => (
              <View
                className={`px-4 py-2 rounded-full border ${
                  isActive ? 'bg-coral-tint border-coral' : 'bg-canvas border-hairline'
                }`}
                style={pressed && Platform.OS === 'ios' ? { opacity: 0.7 } : undefined}
              >
                <Text
                  className={`text-sm font-medium ${
                    isActive ? 'text-coral font-semibold' : 'text-charcoal-soft'
                  }`}
                >
                  {opt.label}
                </Text>
              </View>
            )}
          </Pressable>
        );
      })}
    </View>
  );
}

export default function PreferencesScreen() {
  const cancelOnboarding = useCancelOnboarding();
  const { showAlert } = useAlert();
  const draft = useCustomerOnboardingStore();
  const selected = draft.cuisinePreferences;
  const dietPrefs = draft.dietaryPreferences;
  const allergyPrefs = draft.foodAllergies;
  const [isSubmitting, setIsSubmitting] = useState(false);

  const setOnboardingComplete = useAuthStore(
    (s) => s.setOnboardingComplete
  );

  const toggleChip = (cuisine: string) => {
    draft.update({ cuisinePreferences: toggleValue(selected, cuisine) });
  };

  const toggleDiet = (value: string) => {
    draft.update({ dietaryPreferences: toggleValue(dietPrefs, value) });
  };

  const toggleAllergy = (value: string) => {
    draft.update({ foodAllergies: toggleValue(allergyPrefs, value) });
  };

  const onFinish = async () => {
    setIsSubmitting(true);
    try {
      // Backend (CompleteOnboarding) reads FLAT address fields, not a nested
      // `address` object — addressCity / addressState / addressPostalCode.
      // Sending a nested object silently dropped the address before.
      await api.post('/v1/customer/onboarding/complete', {
        firstName: draft.firstName,
        lastName: draft.lastName,
        phone: draft.phone,
        addressLabel: draft.label || 'Home',
        addressLine1: draft.addressLine1,
        addressLine2: draft.addressLine2 ?? '',
        addressCity: draft.city,
        addressState: draft.state,
        addressPostalCode: draft.pincode,
        addressCountry: 'IN',
        // Geocoded from the address autocomplete pick; 0 when the user typed
        // the address manually (server then uses a flat fee + skips zones).
        addressLatitude: draft.latitude ?? 0,
        addressLongitude: draft.longitude ?? 0,
        cuisinePreferences: selected,
        // Field names must match the Go binding tags on CompleteOnboarding —
        // the server already persists both; the client just never sent them (#912).
        dietaryPreferences: dietPrefs,
        foodAllergies: allergyPrefs,
      });

      // Application saved — clear the local draft so a future re-onboard
      // (or a different account on this device) starts clean.
      draft.reset();
      await setOnboardingComplete(true);
      router.replace('/(tabs)');
    } catch (error: unknown) {
      showAlert(
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
        <View className="mb-2 flex-row items-center justify-between">
          <Text className="text-[13px] text-charcoal-soft">Step 3 of 3</Text>
          <Pressable
            onPress={cancelOnboarding}
            hitSlop={12}
            accessibilityRole="button"
            accessibilityLabel="Cancel onboarding"
          >
            <Text className="text-[13px] font-medium text-charcoal-soft">Cancel</Text>
          </Pressable>
        </View>
        <View className="h-1 bg-hairline rounded-full mb-8 overflow-hidden">
          <View className="h-1 bg-coral rounded-full" style={{ width: '100%' }} />
        </View>

        {/* ── Heading ── */}
        <Text className="text-[26px] font-bold text-charcoal tracking-tight font-display mb-2">
          Tell us about your food
        </Text>
        <Text className="text-[15px] text-charcoal-soft mb-8">
          This helps us recommend dishes you'll love and flag the ones that don't
          suit you. All of it is optional.
        </Text>

        {/* ── Cuisine chips ── */}
        <SectionHeading
          title="Cuisines you love"
          helper="We'll show these first in your recommendations."
        />
        {/* iOS Pressable pattern: visual styles on inner View */}
        <View className="flex-row flex-wrap gap-2 mb-8">
          {CUISINE_OPTIONS.map((cuisine) => {
            const isActive = selected.includes(cuisine);
            return (
              <Pressable
                key={cuisine}
                onPress={() => toggleChip(cuisine)}
                accessibilityRole="checkbox"
                accessibilityLabel={cuisine}
                accessibilityState={{ checked: isActive }}
                android_ripple={{ color: CHIP_RIPPLE, borderless: false }}
              >
                {({ pressed }) => (
                  <View
                    className={`px-4 py-2 rounded-full border ${
                      isActive
                        ? 'bg-coral-tint border-coral'
                        : 'bg-canvas border-hairline'
                    }`}
                    style={pressed && Platform.OS === 'ios' ? { opacity: 0.7 } : undefined}
                  >
                    <Text
                      className={`text-sm font-medium ${
                        isActive ? 'text-coral font-semibold' : 'text-charcoal-soft'
                      }`}
                    >
                      {cuisine}
                    </Text>
                  </View>
                )}
              </Pressable>
            );
          })}
        </View>

        {/* ── Diet chips (#912) ── */}
        <SectionHeading
          title="Diet"
          helper="We'll flag dishes that don't match how you eat."
        />
        <OptionChips
          options={DIET_OPTIONS}
          selectedValues={dietPrefs}
          onToggle={toggleDiet}
        />

        {/* ── Allergen chips (#912) ── */}
        <SectionHeading
          title="Allergies to avoid"
          helper="Optional — we'll warn you before you order a dish containing these. You can change this any time in Profile → Food preferences."
        />
        <OptionChips
          options={ALLERGEN_OPTIONS}
          selectedValues={allergyPrefs}
          onToggle={toggleAllergy}
        />

        {/* ── Primary CTA ── */}
        <Pressable
          onPress={() => void onFinish()}
          disabled={isSubmitting}
          accessibilityRole="button"
          accessibilityLabel="Finish setup"
          android_ripple={{ color: CTA_RIPPLE, borderless: false }}
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
