import React, { useEffect, useState } from 'react';
import {
  ActivityIndicator,
  Alert,
  Platform,
  Pressable,
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
  RefreshCw,
  ChevronRight,
  MessageSquare,
  UtensilsCrossed,
  User,
  ScrollText,
  Wallet,
  Gift,
  Award,
  DatabaseZap,
  KeyRound,
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
import { KeyboardAwareScrollView } from '@homechef/mobile-shared/ui';
import { DIET_OPTIONS, ALLERGEN_OPTIONS } from '@homechef/mobile-shared/dietary';
import { hasPasswordProvider } from '@homechef/mobile-shared/auth';
import { useDockClearance } from '../../components/navigation/Dock';
import { ScreenTitle } from '../../components/shared/ScreenTitle';

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

// Android ripple tints — translucent tokens derived from existing colours,
// never a new literal colour (matches the ChefCard `withAlpha` convention).
const ROW_RIPPLE = `${customerColors.charcoal.DEFAULT}14`;
const CTA_RIPPLE = `${customerColors.canvas}33`;
const CHIP_RIPPLE = `${customerColors.charcoal.DEFAULT}14`;
const DESTRUCTIVE_RIPPLE = `${customerColors.destructive.DEFAULT}14`;

// ─── Section label (iOS grouped style) ───────────────────────────────────────

function SectionLabel({ children }: { children: string }) {
  return (
    <Text className="text-xs font-semibold text-charcoal-soft px-4 pt-5 pb-2">
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

function NavRow({ icon, label, onPress }: NavRowProps) {
  return (
    <Pressable
      onPress={onPress}
      accessibilityRole="button"
      accessibilityLabel={label}
      android_ripple={{ color: ROW_RIPPLE, borderless: false }}
    >
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

  // Only email/password accounts can change a password. Google/Apple (SSO)
  // accounts have no password credential, so the "Change password" row is
  // hidden for them. Read once on mount — providerData is stable for the
  // session and the user is already authenticated on this screen.
  const [canChangePassword] = useState(() => hasPasswordProvider());

  // Visible 2px coral focus ring on the inline-edit fields below (R9 / Input
  // parity) — which field (if any) currently has focus.
  const [focusedField, setFocusedField] = useState<'firstName' | 'lastName' | 'phone' | null>(null);

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

  // Dirty-tracking for the preference sections so their Save buttons only
  // appear when something actually changed (mirrors the Personal Info form).
  const sameSet = (a: string[], b: string[]) =>
    a.length === b.length && a.every((v) => b.includes(v));
  const cuisineDirty = profile
    ? !sameSet(cuisinePrefs, profile.cuisinePreferences ?? [])
    : false;
  const dietaryDirty = profile
    ? !sameSet(dietPrefs, profile.dietaryPreferences ?? []) ||
      !sameSet(allergyPrefs, profile.foodAllergies ?? [])
    : false;

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
      <KeyboardAwareScrollView
        contentContainerStyle={{ paddingBottom: dockClearance }}
      >

        <ScreenTitle title="Profile" />

        {/* ── Identity block — calm charcoal monogram (accent discipline: the
            avatar is identity, not a call to action; coral stays reserved for
            actions/selection). Compact left-aligned row, not a centered hero. ── */}
        <View className="flex-row items-center gap-4 px-4 pt-2 pb-5 bg-canvas">
          {/* Inline bg color — `bg-charcoal` isn't in the compiled class set */}
          <View
            className="w-16 h-16 rounded-full items-center justify-center"
            style={{ backgroundColor: customerColors.charcoal.DEFAULT }}
          >
            <Text
              className="text-[22px] font-bold text-canvas font-display"
              style={{ lineHeight: 28 }}
            >
              {initials}
            </Text>
          </View>
          <View className="flex-1">
            {(profile?.firstName || profile?.lastName) ? (
              <Text className="text-lg font-semibold text-charcoal font-display">
                {[profile?.firstName, profile?.lastName].filter(Boolean).join(' ')}
              </Text>
            ) : null}
            <Text className="text-sm text-charcoal-soft mt-0.5">
              {profile?.email ?? ''}
            </Text>
          </View>
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
            // /subscriptions was an ORPHAN route (#696): the only screen that can
            // pause or cancel a RECURRING charge, reachable solely via a one-shot
            // "View" button in the alert shown right after subscribing. Dismiss that
            // alert and the customer could never find it again — they'd have to call
            // support or charge back to stop being billed. This row is the whole fix;
            // the screen and its API already work.
            TIFFIN_ENABLED && {
              icon: <RefreshCw size={18} color={customerColors.charcoal.soft} />,
              label: 'My subscriptions',
              route: '/subscriptions',
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
            Section — Privacy & Legal. Two rows: "Your Data" stays top-level
            (DPDP action center — export/delete, functional not reference) and
            the four reference documents consolidate behind one "Legal" row
            (app/legal.tsx index).
        ═══════════════════════════════════════════════════════════════════ */}
        <SectionLabel>Privacy & Legal</SectionLabel>

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
              icon={<DatabaseZap size={18} color={customerColors.charcoal.soft} />}
              label="Your Data"
              onPress={() => router.push('/data-privacy')}
            />
            <NavRowDivider />
            <NavRow
              icon={<ScrollText size={18} color={customerColors.charcoal.soft} />}
              label="Legal"
              onPress={() => router.push('/legal')}
              isLast
            />
          </View>
        </View>

        {/* ── Account — password change, gated to email/password accounts.
            Google/Apple (SSO) accounts have no password credential, so this
            section is hidden for them (hasPasswordProvider). ── */}
        {canChangePassword ? (
          <>
            <SectionLabel>Account</SectionLabel>
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
                  icon={<KeyRound size={18} color={customerColors.charcoal.soft} />}
                  label="Change password"
                  onPress={() => router.push('/(auth)/forgot-password' as never)}
                  isLast
                />
              </View>
            </View>
          </>
        ) : null}

        {/* ── Logout — destructive action ── */}
        <Pressable
          onPress={handleLogout}
          accessibilityRole="button"
          accessibilityLabel="Log out"
          android_ripple={{ color: DESTRUCTIVE_RIPPLE, borderless: false }}
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

      </KeyboardAwareScrollView>
    </SafeAreaView>
  );
}
