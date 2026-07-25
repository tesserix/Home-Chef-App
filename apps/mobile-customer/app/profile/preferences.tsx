import React, { useEffect, useState } from 'react';
import { ActivityIndicator, Alert, Platform, Pressable, Text, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';

import { customerColors } from '@homechef/mobile-shared/theme';
import { KeyboardAwareScrollView } from '@homechef/mobile-shared/ui';
import { DIET_OPTIONS, ALLERGEN_OPTIONS } from '@homechef/mobile-shared/dietary';
import { useProfile, useUpdateProfile } from '../../hooks/useProfile';
import { friendlyErrorMessage } from '../../lib/errors';
import { ScreenHeader } from '../../components/ScreenHeader';

// Food preferences and dietary profile — moved off the profile tab so that tab
// can be a hub of destinations rather than a stack of forms.

const CUISINE_OPTIONS = [
  'North Indian',
  'South Indian',
  'Chinese',
  'Continental',
  'Italian',
  'Healthy',
  'Desserts',
  'Street Food',
];

const CHIP_RIPPLE = `${customerColors.charcoal.DEFAULT}14`;
const CTA_RIPPLE = `${customerColors.canvas}33`;
// Allergen chips are the one place a destructive tint is meaningful: they mark
// what the customer must NOT be served.
const DESTRUCTIVE_RIPPLE = `${customerColors.destructive.DEFAULT}14`;

function SectionLabel({ children }: { children: string }) {
  return (
    <Text className="text-xs font-semibold text-charcoal-soft px-4 pt-5 pb-2">{children}</Text>
  );
}

export default function FoodPreferencesScreen() {
  const { data: profile, isLoading } = useProfile();
  const updateProfile = useUpdateProfile();

  const [cuisinePrefs, setCuisinePrefs] = useState<string[]>([]);
  const [dietPrefs, setDietPrefs] = useState<string[]>([]);
  const [allergyPrefs, setAllergyPrefs] = useState<string[]>([]);

  useEffect(() => {
    if (profile) {
      setCuisinePrefs(profile.cuisinePreferences ?? []);
      setDietPrefs(profile.dietaryPreferences ?? []);
      setAllergyPrefs(profile.foodAllergies ?? []);
    }
  }, [profile]);

  // Save buttons appear only when something actually changed.
  const sameSet = (a: string[], b: string[]) =>
    a.length === b.length && a.every((v) => b.includes(v));
  const cuisineDirty = profile ? !sameSet(cuisinePrefs, profile.cuisinePreferences ?? []) : false;
  const dietaryDirty = profile
    ? !sameSet(dietPrefs, profile.dietaryPreferences ?? []) ||
      !sameSet(allergyPrefs, profile.foodAllergies ?? [])
    : false;

  function toggleCuisine(cuisine: string) {
    setCuisinePrefs((prev) =>
      prev.includes(cuisine) ? prev.filter((c) => c !== cuisine) : [...prev, cuisine],
    );
  }

  const toggleFrom = (set: React.Dispatch<React.SetStateAction<string[]>>) => (value: string) =>
    set((prev) => (prev.includes(value) ? prev.filter((v) => v !== value) : [...prev, value]));

  function saveCuisinePrefs() {
    updateProfile.mutate(
      { cuisinePreferences: cuisinePrefs },
      {
        onSuccess: () => Alert.alert('Saved', 'Cuisine preferences updated.'),
        onError: (error) =>
          Alert.alert('Error', friendlyErrorMessage(error, 'Could not save preferences.')),
      },
    );
  }

  function saveDietaryProfile() {
    updateProfile.mutate(
      { dietaryPreferences: dietPrefs, foodAllergies: allergyPrefs },
      {
        onSuccess: () => Alert.alert('Saved', 'Dietary profile updated.'),
        onError: (error) =>
          Alert.alert('Error', friendlyErrorMessage(error, 'Could not save dietary profile.')),
      },
    );
  }

  if (isLoading) {
    return (
      <SafeAreaView className="flex-1 bg-canvas" edges={['top', 'left', 'right']}>
        <ScreenHeader title="Food preferences" />
        <View className="flex-1 items-center justify-center">
          <ActivityIndicator size="large" color={customerColors.coral.DEFAULT} />
        </View>
      </SafeAreaView>
    );
  }

  return (
    <SafeAreaView className="flex-1 bg-canvas" edges={['top', 'left', 'right']}>
      <ScreenHeader title="Food preferences" />
      <KeyboardAwareScrollView contentContainerStyle={{ paddingBottom: 40 }}>
        {/* ═══════════════════════════════════════════════════════════════════
            Section — Food Preferences
        ═══════════════════════════════════════════════════════════════════ */}
        <SectionLabel>Food Preferences</SectionLabel>

        <View className="px-4">
          <View className="flex-row flex-wrap gap-2 mb-3">
            {CUISINE_OPTIONS.map((cuisine) => {
              const isSelected = cuisinePrefs.includes(cuisine);
              return (
                /* iOS Pressable pattern: visual styles on inner View */
                <Pressable
                  key={cuisine}
                  onPress={() => toggleCuisine(cuisine)}
                  accessibilityRole="checkbox"
                  accessibilityLabel={cuisine}
                  accessibilityState={{ checked: isSelected }}
                  android_ripple={{ color: CHIP_RIPPLE, borderless: false }}
                >
                  {({ pressed }) => (
                    <View
                      className={`px-4 py-2 rounded-full ${
                        isSelected ? 'bg-coral-tint' : 'bg-surface-soft'
                      }`}
                      style={pressed && Platform.OS === 'ios' ? { opacity: 0.7 } : undefined}
                    >
                      <Text
                        className={`text-sm font-medium ${
                          isSelected ? 'text-coral font-semibold' : 'text-charcoal-soft'
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

          {/* Save Preferences CTA — only when selections changed (mirrors the
              Personal Info dirty-gated pattern; no permanent giant coral block) */}
          {cuisineDirty && (
          <Pressable
            onPress={saveCuisinePrefs}
            disabled={updateProfile.isPending}
            accessibilityRole="button"
            accessibilityLabel="Save preferences"
            android_ripple={{ color: CTA_RIPPLE, borderless: false }}
          >
            {({ pressed }) => (
              <View
                className={`rounded-lg min-h-[52px] items-center justify-center bg-coral ${pressed ? 'opacity-90' : ''}`}
              >
                <Text className="text-canvas font-semibold text-base">
                  Save Preferences
                </Text>
              </View>
            )}
          </Pressable>
          )}
        </View>

        {/* ═══════════════════════════════════════════════════════════════════
            Section — Dietary Profile (#41)
        ═══════════════════════════════════════════════════════════════════ */}
        <SectionLabel>Dietary Profile</SectionLabel>

        <View className="px-4">
          <Text className="text-xs text-charcoal-soft mb-2">
            We'll flag dishes that don't match your diet or contain allergens you avoid.
          </Text>

          <Text className="text-sm font-semibold text-charcoal mb-2">Diet</Text>
          <View className="flex-row flex-wrap gap-2 mb-4">
            {DIET_OPTIONS.map((opt) => {
              const isSelected = dietPrefs.includes(opt.value);
              return (
                <Pressable
                  key={opt.value}
                  onPress={() => toggleFrom(setDietPrefs)(opt.value)}
                  accessibilityRole="checkbox"
                  accessibilityLabel={opt.label}
                  accessibilityState={{ checked: isSelected }}
                  android_ripple={{ color: CHIP_RIPPLE, borderless: false }}
                >
                  {({ pressed }) => (
                    <View
                      className={`px-4 py-2 rounded-full ${
                        isSelected ? 'bg-coral-tint' : 'bg-surface-soft'
                      }`}
                      style={pressed && Platform.OS === 'ios' ? { opacity: 0.7 } : undefined}
                    >
                      <Text
                        className={`text-sm font-medium ${
                          isSelected ? 'text-coral font-semibold' : 'text-charcoal-soft'
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

          <Text className="text-sm font-semibold text-charcoal mb-2">Allergies to avoid</Text>
          <View className="flex-row flex-wrap gap-2 mb-3">
            {ALLERGEN_OPTIONS.map((opt) => {
              const isSelected = allergyPrefs.includes(opt.value);
              return (
                <Pressable
                  key={opt.value}
                  onPress={() => toggleFrom(setAllergyPrefs)(opt.value)}
                  accessibilityRole="checkbox"
                  accessibilityLabel={opt.label}
                  accessibilityState={{ checked: isSelected }}
                  android_ripple={{ color: DESTRUCTIVE_RIPPLE, borderless: false }}
                >
                  {({ pressed }) => (
                    <View
                      className={`px-4 py-2 rounded-full ${
                        isSelected ? 'bg-destructive-tint' : 'bg-surface-soft'
                      }`}
                      style={pressed && Platform.OS === 'ios' ? { opacity: 0.7 } : undefined}
                    >
                      <Text
                        className={`text-sm font-medium ${
                          isSelected ? 'text-destructive font-semibold' : 'text-charcoal-soft'
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

          {/* Dirty-gated like the other saves — appears only when changed */}
          {dietaryDirty && (
          <Pressable
            onPress={saveDietaryProfile}
            disabled={updateProfile.isPending}
            accessibilityRole="button"
            accessibilityLabel="Save dietary profile"
            android_ripple={{ color: CTA_RIPPLE, borderless: false }}
          >
            {({ pressed }) => (
              <View
                className={`rounded-lg min-h-[52px] items-center justify-center bg-coral ${pressed ? 'opacity-90' : ''}`}
              >
                <Text className="text-canvas font-semibold text-base">Save Dietary Profile</Text>
              </View>
            )}
          </Pressable>
          )}
        </View>

      </KeyboardAwareScrollView>
    </SafeAreaView>
  );
}
