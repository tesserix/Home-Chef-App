import React, { useEffect, useState } from 'react';
import {
  ActivityIndicator,
  Alert,
  Pressable,
  ScrollView,
  Text,
  TextInput,
  View,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useRouter } from 'expo-router';
import { z } from 'zod';
import { useForm, Controller } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import {
  CalendarDays,
  ChevronRight,
  MessageSquare,
  UtensilsCrossed,
  User,
  FileText,
  Shield,
  Receipt,
  ScrollText,
  Wallet,
  Gift,
  Award,
  DatabaseZap,
} from 'lucide-react-native';
import { useProfile, useUpdateProfile } from '../../hooks/useProfile';
import { friendlyErrorMessage } from '../../lib/errors';
import {
  TIFFIN_ENABLED,
  CATERING_ENABLED,
  WALLET_ENABLED,
  REWARDS_ENABLED,
  REFERRAL_ENABLED,
  SOCIAL_ENABLED,
} from '../../lib/features';
import { useAuthStore } from '../../store/auth-store';
import { customerColors } from '@homechef/mobile-shared/theme';
import { DIET_OPTIONS, ALLERGEN_OPTIONS } from '@homechef/mobile-shared/dietary';
import { useDockClearance } from '../../components/navigation/Dock';

// Threat model T-02-05-01: Zod validates profile fields before PATCH
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

// ─── Section label (iOS grouped style) ───────────────────────────────────────

function SectionLabel({ children }: { children: string }) {
  return (
    <Text className="text-xs font-semibold text-charcoal-soft uppercase tracking-wide px-4 pt-5 pb-2">
      {children}
    </Text>
  );
}

// ─── Grouped list row ─────────────────────────────────────────────────────────
// iOS Pressable pattern: visual styles on the inner View; `flex-1` on a plain
// wrapper if the row must fill the width.

interface NavRowProps {
  icon: React.ReactNode;
  label: string;
  onPress: () => void;
  isLast?: boolean;
}

function NavRow({ icon, label, onPress, isLast = false }: NavRowProps) {
  return (
    <Pressable onPress={onPress} accessibilityRole="button" accessibilityLabel={label}>
      {({ pressed }) => (
        <View
          className={`flex-row items-center px-4 py-3 min-h-[52px] ${pressed ? 'bg-surface-soft' : 'bg-canvas'}`}
        >
          {/* Left icon circle */}
          <View className="w-9 h-9 rounded-full bg-surface-soft items-center justify-center mr-3">
            {icon}
          </View>
          {/* Label */}
          <Text className="flex-1 text-base text-charcoal font-sans">
            {label}
          </Text>
          {/* Chevron */}
          <ChevronRight size={16} color={customerColors.charcoal.soft} />
        </View>
      )}
    </Pressable>
  );
}

function NavRowDivider() {
  return <View className="h-px bg-hairline ml-[64px]" />;
}

// ─── Main screen ─────────────────────────────────────────────────────────────

export default function ProfileScreen() {
  const router = useRouter();
  const dockClearance = useDockClearance();
  const { data, isLoading } = useProfile();
  const updateProfile = useUpdateProfile();

  // The API returns the profile FLAT (no { data } envelope), so read it directly.
  const profile = data;

  const [cuisinePrefs, setCuisinePrefs] = useState<string[]>([]);
  // Dietary profile (#41) — diet types + allergens to avoid.
  const [dietPrefs, setDietPrefs] = useState<string[]>([]);
  const [allergyPrefs, setAllergyPrefs] = useState<string[]>([]);

  const {
    control,
    handleSubmit,
    reset,
    formState: { errors, isDirty },
  } = useForm<ProfileFormValues>({
    resolver: zodResolver(profileSchema),
    defaultValues: { firstName: '', lastName: '', phone: '' },
  });

  // Populate form when profile loads
  useEffect(() => {
    if (profile) {
      reset({
        firstName: profile.firstName ?? '',
        lastName: profile.lastName ?? '',
        phone: profile.phone ?? '',
      });
      setCuisinePrefs(profile.cuisinePreferences ?? []);
      setDietPrefs(profile.dietaryPreferences ?? []);
      setAllergyPrefs(profile.foodAllergies ?? []);
    }
  }, [profile, reset]);

  function toggleCuisine(cuisine: string) {
    setCuisinePrefs((prev) =>
      prev.includes(cuisine)
        ? prev.filter((c) => c !== cuisine)
        : [...prev, cuisine],
    );
  }

  const toggleFrom = (set: React.Dispatch<React.SetStateAction<string[]>>) => (value: string) =>
    set((prev) => (prev.includes(value) ? prev.filter((v) => v !== value) : [...prev, value]));

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

  function saveCuisinePrefs() {
    updateProfile.mutate(
      { cuisinePreferences: cuisinePrefs },
      {
        onSuccess: () =>
          Alert.alert('Saved', 'Cuisine preferences updated.'),
        onError: (error) =>
          Alert.alert('Error', friendlyErrorMessage(error, 'Could not save preferences.')),
      },
    );
  }

  function handleLogout() {
    Alert.alert('Log out', 'Are you sure you want to log out?', [
      { text: 'Cancel', style: 'cancel' },
      {
        text: 'Log Out',
        style: 'destructive',
        onPress: () => {
          useAuthStore.getState().logout();
          router.replace('/(auth)/login');
        },
      },
    ]);
  }

  // ── Loading ────────────────────────────────────────────────────────────────
  if (isLoading) {
    return (
      <SafeAreaView className="flex-1 bg-canvas" edges={['top', 'left', 'right']}>
        <View className="flex-1 items-center justify-center">
          <ActivityIndicator size="large" color={customerColors.coral.DEFAULT} />
        </View>
      </SafeAreaView>
    );
  }

  // Initials for avatar placeholder
  const initials =
    [profile?.firstName?.[0], profile?.lastName?.[0]]
      .filter(Boolean)
      .join('')
      .toUpperCase() || '?';

  return (
    <SafeAreaView className="flex-1 bg-canvas" edges={['top', 'left', 'right']}>
      <ScrollView contentContainerStyle={{ paddingBottom: dockClearance }}>

        {/* ── Geist-Bold header ── */}
        <View className="px-4 pt-3 pb-2">
          <Text className="text-2xl font-bold text-charcoal tracking-tight font-display">
            Profile
          </Text>
        </View>

        {/* ── Identity block ── */}
        <View className="items-center pt-4 pb-6 bg-canvas">
          {/* Avatar circle — coral bg, white initials */}
          <View
            className="w-[72px] h-[72px] rounded-full bg-coral items-center justify-center mb-3"
          >
            <Text
              className="text-[28px] font-bold text-canvas font-display"
              style={{ lineHeight: 34 }}
            >
              {initials}
            </Text>
          </View>
          {/* Name */}
          {(profile?.firstName || profile?.lastName) ? (
            <Text className="text-lg font-semibold text-charcoal font-display mb-1">
              {[profile?.firstName, profile?.lastName].filter(Boolean).join(' ')}
            </Text>
          ) : null}
          {/* Email */}
          <Text className="text-sm text-charcoal-soft">
            {profile?.email ?? ''}
          </Text>
        </View>

        {/* ── Hairline under identity block ── */}
        <View className="h-px bg-hairline mx-0" />

        {/* ═══════════════════════════════════════════════════════════════════
            Section — Personal Info
        ═══════════════════════════════════════════════════════════════════ */}
        <SectionLabel>Personal Info</SectionLabel>

        <View className="bg-canvas mx-4 rounded-xl overflow-hidden border border-hairline">
          {/* First Name */}
          <View className="px-4 pt-3 pb-1">
            <Text className="text-xs font-semibold text-charcoal-soft uppercase tracking-wide mb-1">
              First Name
            </Text>
            <Controller
              control={control}
              name="firstName"
              render={({ field: { onChange, value, onBlur } }) => (
                <TextInput
                  className="text-base text-charcoal bg-transparent pb-2"
                  style={{ borderBottomWidth: errors.firstName ? 1 : 0, borderBottomColor: customerColors.coral.pressed }}
                  value={value}
                  onChangeText={onChange}
                  onBlur={onBlur}
                  placeholder="First name"
                  placeholderTextColor={customerColors.charcoal.soft}
                  autoCapitalize="words"
                  accessibilityLabel="First name"
                />
              )}
            />
            {errors.firstName ? (
              <Text className="text-xs text-coral-pressed mb-1">{errors.firstName.message}</Text>
            ) : null}
          </View>

          <View className="h-px bg-hairline mx-4" />

          {/* Last Name */}
          <View className="px-4 pt-3 pb-1">
            <Text className="text-xs font-semibold text-charcoal-soft uppercase tracking-wide mb-1">
              Last Name
            </Text>
            <Controller
              control={control}
              name="lastName"
              render={({ field: { onChange, value, onBlur } }) => (
                <TextInput
                  className="text-base text-charcoal bg-transparent pb-2"
                  style={{ borderBottomWidth: errors.lastName ? 1 : 0, borderBottomColor: customerColors.coral.pressed }}
                  value={value}
                  onChangeText={onChange}
                  onBlur={onBlur}
                  placeholder="Last name"
                  placeholderTextColor={customerColors.charcoal.soft}
                  autoCapitalize="words"
                  accessibilityLabel="Last name"
                />
              )}
            />
            {errors.lastName ? (
              <Text className="text-xs text-coral-pressed mb-1">{errors.lastName.message}</Text>
            ) : null}
          </View>

          <View className="h-px bg-hairline mx-4" />

          {/* Phone */}
          <View className="px-4 pt-3 pb-3">
            <Text className="text-xs font-semibold text-charcoal-soft uppercase tracking-wide mb-1">
              Phone
            </Text>
            <Controller
              control={control}
              name="phone"
              render={({ field: { onChange, value, onBlur } }) => (
                <TextInput
                  className="text-base text-charcoal bg-transparent pb-2"
                  style={{ borderBottomWidth: errors.phone ? 1 : 0, borderBottomColor: customerColors.coral.pressed }}
                  value={value ?? ''}
                  onChangeText={onChange}
                  onBlur={onBlur}
                  placeholder="+91 9876543210"
                  placeholderTextColor={customerColors.charcoal.soft}
                  keyboardType="phone-pad"
                  accessibilityLabel="Phone number"
                />
              )}
            />
            {errors.phone ? (
              <Text className="text-xs text-coral-pressed mt-1">{errors.phone.message}</Text>
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
                >
                  <View
                    className={`px-4 py-2 rounded-full border ${
                      isSelected
                        ? 'bg-coral-tint border-coral'
                        : 'bg-canvas border-hairline'
                    }`}
                  >
                    <Text
                      className={`text-sm font-medium ${
                        isSelected ? 'text-coral font-semibold' : 'text-charcoal-soft'
                      }`}
                    >
                      {cuisine}
                    </Text>
                  </View>
                </Pressable>
              );
            })}
          </View>

          {/* Save Preferences CTA */}
          <Pressable
            onPress={saveCuisinePrefs}
            disabled={updateProfile.isPending}
            accessibilityRole="button"
            accessibilityLabel="Save preferences"
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
                >
                  <View
                    className={`px-4 py-2 rounded-full border ${
                      isSelected ? 'bg-coral-tint border-coral' : 'bg-canvas border-hairline'
                    }`}
                  >
                    <Text
                      className={`text-sm font-medium ${
                        isSelected ? 'text-coral font-semibold' : 'text-charcoal-soft'
                      }`}
                    >
                      {opt.label}
                    </Text>
                  </View>
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
                >
                  <View
                    className={`px-4 py-2 rounded-full border ${
                      isSelected ? 'bg-destructive-tint border-destructive' : 'bg-canvas border-hairline'
                    }`}
                  >
                    <Text
                      className={`text-sm font-medium ${
                        isSelected ? 'text-destructive font-semibold' : 'text-charcoal-soft'
                      }`}
                    >
                      {opt.label}
                    </Text>
                  </View>
                </Pressable>
              );
            })}
          </View>

          <Pressable
            onPress={saveDietaryProfile}
            disabled={updateProfile.isPending}
            accessibilityRole="button"
            accessibilityLabel="Save dietary profile"
          >
            {({ pressed }) => (
              <View
                className={`rounded-lg min-h-[52px] items-center justify-center bg-coral ${pressed ? 'opacity-90' : ''}`}
              >
                <Text className="text-canvas font-semibold text-base">Save Dietary Profile</Text>
              </View>
            )}
          </Pressable>
        </View>

        {/* ═══════════════════════════════════════════════════════════════════
            Section — More (iOS grouped nav rows)
        ═══════════════════════════════════════════════════════════════════ */}
        {/* MORE nav rows are each gated by a feature flag — everything deferred for
            v1 (wallet/rewards/referral/social/catering/tiffin) is hidden, and the
            whole section drops out when no row is enabled. Flip the flag in
            lib/features.ts (+ any backend flag) to bring a row back. */}
        {(() => {
          const moreRows = [
            WALLET_ENABLED && {
              icon: <Wallet size={18} color={customerColors.charcoal.soft} />,
              label: 'Wallet',
              route: '/wallet',
            },
            REWARDS_ENABLED && {
              icon: <Award size={18} color={customerColors.charcoal.soft} />,
              label: 'Rewards',
              route: '/loyalty',
            },
            REFERRAL_ENABLED && {
              icon: <Gift size={18} color={customerColors.charcoal.soft} />,
              label: 'Refer & Earn',
              route: '/referral',
            },
            SOCIAL_ENABLED && {
              icon: <MessageSquare size={18} color={customerColors.charcoal.soft} />,
              label: 'Social Feed',
              route: '/social',
            },
            CATERING_ENABLED && {
              icon: <UtensilsCrossed size={18} color={customerColors.charcoal.soft} />,
              label: 'Catering',
              route: '/catering',
            },
            TIFFIN_ENABLED && {
              icon: <CalendarDays size={18} color={customerColors.charcoal.soft} />,
              label: 'My meal plans',
              route: '/meal-plans',
            },
          ].filter(Boolean) as { icon: React.ReactNode; label: string; route: string }[];

          if (moreRows.length === 0) return null;

          return (
            <>
              <SectionLabel>More</SectionLabel>
              {/* Shadow on outer View, overflow+radius on clip View — iOS shadow gotcha */}
              <View
                className="mx-4"
                style={{
                  shadowColor: customerColors.charcoal.DEFAULT,
                  shadowOffset: { width: 0, height: 1 },
                  shadowOpacity: 0.06,
                  shadowRadius: 4,
                  elevation: 2,
                }}
              >
                <View className="rounded-xl overflow-hidden">
                  {moreRows.map((r, i) => (
                    <View key={r.label}>
                      {i > 0 ? <NavRowDivider /> : null}
                      <NavRow
                        icon={r.icon}
                        label={r.label}
                        onPress={() => router.push(r.route as never)}
                        isLast={i === moreRows.length - 1}
                      />
                    </View>
                  ))}
                </View>
              </View>
            </>
          );
        })()}

        {/* ═══════════════════════════════════════════════════════════════════
            Section — Legal (Terms · Privacy · Refund)
        ═══════════════════════════════════════════════════════════════════ */}
        <SectionLabel>Legal</SectionLabel>

        {/* Shadow on outer View, overflow+radius on clip View — iOS shadow gotcha */}
        <View
          className="mx-4"
          style={{
            shadowColor: customerColors.charcoal.DEFAULT,
            shadowOffset: { width: 0, height: 1 },
            shadowOpacity: 0.06,
            shadowRadius: 4,
            elevation: 2,
          }}
        >
          <View className="rounded-xl overflow-hidden">
            <NavRow
              icon={<FileText size={18} color={customerColors.charcoal.soft} />}
              label="Terms of Service"
              onPress={() => router.push('/terms')}
            />
            <NavRowDivider />
            <NavRow
              icon={<Shield size={18} color={customerColors.charcoal.soft} />}
              label="Privacy Policy"
              onPress={() => router.push('/privacy')}
            />
            <NavRowDivider />
            <NavRow
              icon={<DatabaseZap size={18} color={customerColors.charcoal.soft} />}
              label="Your Data"
              onPress={() => router.push('/data-privacy')}
            />
            <NavRowDivider />
            <NavRow
              icon={<Receipt size={18} color={customerColors.charcoal.soft} />}
              label="Refund Policy"
              onPress={() => router.push('/refund')}
            />
            <NavRowDivider />
            <NavRow
              icon={<ScrollText size={18} color={customerColors.charcoal.soft} />}
              label="End User Licence"
              onPress={() => router.push('/eula')}
              isLast
            />
          </View>
        </View>

        {/* ── Logout — destructive action ── */}
        <Pressable
          onPress={handleLogout}
          accessibilityRole="button"
          accessibilityLabel="Log out"
        >
          {({ pressed }) => (
            <View
              className={`mx-4 mt-6 rounded-lg min-h-[52px] items-center justify-center border border-hairline ${
                pressed ? 'bg-surface-soft' : 'bg-canvas'
              }`}
            >
              <Text className="text-base font-semibold text-destructive">
                Log Out
              </Text>
            </View>
          )}
        </Pressable>

      </ScrollView>
    </SafeAreaView>
  );
}
